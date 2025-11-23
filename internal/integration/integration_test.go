package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"elsa-quiz-service/internal/app"
	"elsa-quiz-service/internal/domain"
	pgloader "elsa-quiz-service/internal/infra/postgres"
	infraredis "elsa-quiz-service/internal/infra/redis"
	pgmigrations "elsa-quiz-service/migrations"
	"github.com/jackc/pgx/v4/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/migrate"
)

func TestSubmitAnswerEndToEnd(t *testing.T) {
	ctx := context.Background()
	requireDocker(t)

	pgURL, pgCleanup := startPostgres(t, ctx)
	defer pgCleanup()
	redisURL, redisCleanup := startRedis(t, ctx)
	defer redisCleanup()

	seedQuiz(t, ctx, pgURL, sampleQuiz())

	pool, err := pgxpool.Connect(ctx, pgURL)
	if err != nil {
		t.Fatalf("connect pg: %v", err)
	}
	defer pool.Close()

	loader := pgloader.NewQuizLoader(pool)

	redisClient, err := redisClientFromURL(redisURL)
	if err != nil {
		t.Fatalf("redis client: %v", err)
	}
	quizRepo := infraredis.NewQuizRepository(redisClient, loader, 5*time.Minute)
	sessionStore := infraredis.NewSessionStore(redisClient, 5*time.Minute)
	service := app.NewQuizService(sessionStore, quizRepo)

	if _, err := service.Join(ctx, "quiz-1", "u1", "Alice"); err != nil {
		t.Fatalf("join: %v", err)
	}
	if _, err := service.Join(ctx, "quiz-1", "u2", "Bob"); err != nil {
		t.Fatalf("join: %v", err)
	}

	lb, total, awarded, correct, err := service.SubmitAnswer(ctx, "quiz-1", "u2", domain.AnswerSubmission{
		QuestionID: "q1",
		OptionID:   "o2",
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if !correct || awarded != 1 || total != 1 {
		t.Fatalf("expected correct answer with 1 point, got correct=%v awarded=%d total=%d", correct, awarded, total)
	}
	if len(lb.Entries) != 2 || lb.Entries[0].UserID != "u2" {
		t.Fatalf("expected bob leading, got %+v", lb.Entries)
	}
}

func startPostgres(t *testing.T, ctx context.Context) (string, func()) {
	t.Helper()
	req := tc.ContainerRequest{
		Image:        "postgres:15-alpine",
		Env:          map[string]string{"POSTGRES_USER": "quiz", "POSTGRES_PASSWORD": "quizpass", "POSTGRES_DB": "quizdb"},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}
	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		if strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
			t.Skipf("docker not available: %v", err)
		}
		t.Fatalf("start postgres: %v", err)
	}
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("host: %v", err)
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("port: %v", err)
	}
	dsn := fmt.Sprintf("postgres://quiz:quizpass@%s:%s/quizdb?sslmode=disable", host, port.Port())
	return dsn, func() {
		_ = container.Terminate(ctx)
	}
}

func startRedis(t *testing.T, ctx context.Context) (string, func()) {
	t.Helper()
	req := tc.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(30 * time.Second),
	}
	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		if strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
			t.Skipf("docker not available: %v", err)
		}
		t.Fatalf("start redis: %v", err)
	}
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("redis host: %v", err)
	}
	port, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		t.Fatalf("redis port: %v", err)
	}
	url := fmt.Sprintf("redis://%s:%s", host, port.Port())
	return url, func() {
		_ = container.Terminate(ctx)
	}
}

func seedQuiz(t *testing.T, ctx context.Context, dsn string, quiz domain.Quiz) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())
	defer db.Close()

	migrator := migrate.NewMigrator(db, pgmigrations.Migrations)
	if err := migrator.Init(ctx); err != nil {
		t.Fatalf("migrator init: %v", err)
	}
	if _, err := migrator.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	data, err := json.Marshal(quiz)
	if err != nil {
		t.Fatalf("marshal quiz: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO quizzes (id, data) VALUES (? , ?::jsonb) ON CONFLICT (id) DO UPDATE SET data=EXCLUDED.data`, quiz.ID, string(data)); err != nil {
		t.Fatalf("insert quiz: %v", err)
	}
}

func sampleQuiz() domain.Quiz {
	return domain.Quiz{
		ID: "quiz-1",
		Questions: []domain.Question{
			{
				ID:     "q1",
				Prompt: "What is 2 + 2?",
				Options: []domain.Option{
					{ID: "o1", Text: "3", Correct: false},
					{ID: "o2", Text: "4", Correct: true},
					{ID: "o3", Text: "5", Correct: false},
				},
				Points: 1,
			},
		},
	}
}

func redisClientFromURL(url string) (*goredis.Client, error) {
	opts, err := goredis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	return goredis.NewClient(&goredis.Options{
		Addr:     opts.Addr,
		Password: opts.Password,
		DB:       opts.DB,
	}), nil
}

func requireDocker(t *testing.T) {
	t.Helper()
	if _, err := tc.NewDockerProvider(); err != nil {
		t.Skipf("docker not available: %v", err)
	}
}

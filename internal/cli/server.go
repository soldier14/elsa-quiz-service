package cli

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"elsa-quiz-service/internal/app"
	"elsa-quiz-service/internal/config"
	"elsa-quiz-service/internal/domain"
	"elsa-quiz-service/internal/infra/memory"
	pgloader "elsa-quiz-service/internal/infra/postgres"
	redissession "elsa-quiz-service/internal/infra/redis"
	transport "elsa-quiz-service/internal/transport/http"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
)

// NewStartCmd builds the CLI subcommand to start the server.
func NewStartCmd(configPath, port *string) *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the quiz server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(cmd.Context(), *configPath, *port)
		},
	}
}

func runServer(ctx context.Context, configPath, portFlag string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if cfg.Postgres.URL != "" {
		if err := runMigrationsWithConfig(ctx, cfg); err != nil {
			return err
		}
	}

	finalPort := portFlag
	if finalPort == "" {
		finalPort = cfg.Server.Port
	}
	if finalPort == "" {
		finalPort = "8080"
	}

	var redisClient *redis.Client
	if cfg.Redis.Addr != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
	}
	redisTTL := config.TTLDuration(cfg.Redis.TTL, 10*time.Minute)

	var pool *pgxpool.Pool
	if cfg.Postgres.URL != "" {
		pool, err = pgxpool.Connect(ctx, cfg.Postgres.URL)
		if err != nil {
			return err
		}
	}

	var loader memory.QuizLoader = memory.NewStaticQuizLoader(sampleQuizzes())
	if pool != nil {
		loader = pgloader.NewQuizLoader(pool)
	}

	quizTTL := config.TTLDuration(cfg.Quiz.TTL, 10*time.Minute)
	var quizRepo app.QuizRepository
	if redisClient != nil {
		quizRepo = redissession.NewQuizRepository(redisClient, loader, quizTTL)
	} else {
		quizRepo = memory.NewQuizRepository(loader, quizTTL)
	}

	var store app.SessionRepository
	if redisClient != nil {
		store = redissession.NewSessionStore(redisClient, redisTTL)
	} else {
		store = memory.NewSessionStore()
	}
	service := app.NewQuizService(store, quizRepo)
	wsHandler := transport.NewWSHandler(service)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/ws", wsHandler.ServeWS)

	server := &http.Server{
		Addr:         ":" + finalPort,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("starting quiz service on :%s", finalPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("failed to start server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-stop:
		log.Println("shutting down server...")
	case <-ctx.Done():
		log.Println("context canceled, shutting down server...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

// sampleQuizzes provides a minimal set of quiz data; swap this loader with a document DB-backed one in production.
func sampleQuizzes() map[string]domain.Quiz {
	return map[string]domain.Quiz{
		"quiz-1": {
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
		},
	}
}

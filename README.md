## Real-Time Quiz Service (Clean Architecture + Gorilla WebSocket)

This service provides a real-time quiz component: users join a quiz by ID, submit answers, and receive live leaderboard updates over WebSockets. It follows a clean architecture layout: transport (WebSocket), application use cases, domain entities, and swappable infrastructure (in-memory session store today).

### Component/Data Flow (C4 Component Diagram)
![Component diagram](docs/component-diagram.png)

### WebSocket Contract
- Connect:
  ```
  ws://localhost:8080/ws?quizId={quiz}&userId={user}&name={displayName}
  ```
- Messages:
  ```json
  // Client -> server
  {"type":"answer","payload":{"questionId":"q1","optionId":"o2"}}

  // Server -> client events
  {"type":"joined","payload":<leaderboard>}
  {"type":"leaderboard","payload":<leaderboard>}
  {"type":"answerResult","payload":{"questionId":"q1","correct":true,"awarded":1,"totalScore":5}}
  {"type":"error","payload":{"message":"..."}}
  ```
- Leaderboard shape:
  ```json
  {
    "quizId": "quiz-1",
    "updatedAt": "2024-01-01T00:00:00Z",
    "entries": [
      {"userId":"u2","displayName":"Bob","score":5},
      {"userId":"u1","displayName":"Alice","score":0}
    ]
  }
  ```

### Clean Architecture Layout
- `cmd/server`: wiring (HTTP server, routes, graceful shutdown).
- `internal/domain`: entities and domain errors.
- `internal/app`: quiz use cases and session orchestration (framework-agnostic).
- `internal/infra/memory`: in-memory `SessionStore` (swap with Redis/DB).
- `internal/transport/http`: Gorilla WebSocket handler.

### Quick Start
- Prereqs: Go 1.22+, internet (to fetch `github.com/gorilla/websocket` if not cached).
- Run the server:
  ```bash
  go run ./cmd/server
  ```
  Server listens on `:8080` or `$PORT` if set.
- Run tests:
  ```bash
  CGO_ENABLED=0 go test ./...
  ```

### Makefile Helpers
- `make build` — build binary to `bin/quiz-service`
- `make test` — run tests (`CGO_ENABLED=0`)
- `make lint` — run `go vet`
- `make docker` — build Docker image
- `make docker-run` — run image on port 8080

### Docker
- Build: `docker build -t elsa-quiz-service:latest .`
- Run: `docker run --rm -p 8080:8080 elsa-quiz-service:latest start`

### Seed Sample Quizzes
- Ensure Postgres is up (e.g., `docker-compose up -d` with the provided compose file).
- Seed fixtures:
  ```bash
  DATABASE_URL=postgres://quiz:quizpass@localhost:5432/quizdb?sslmode=disable ./scripts/seed_quizzes.sh
  ```
  This upserts sample quizzes (`quiz-1`, `quiz-2`) from `fixtures/quizzes.sql`.

### WebSocket Contract
- Connect:
  ```
  ws://localhost:8080/ws?quizId={quiz}&userId={user}&name={displayName}
  ```
- Messages:
  ```json
  // Client -> server
  {"type":"answer","payload":{"questionId":"q1","optionId":"o2"}}

  // Server -> client events
  {"type":"joined","payload":<leaderboard>}
  {"type":"leaderboard","payload":<leaderboard>}
  {"type":"answerResult","payload":{"questionId":"q1","correct":true,"awarded":1,"totalScore":5}}
  {"type":"error","payload":{"message":"..."}}
  ```
- Leaderboard shape:
  ```json
  {
    "quizId": "quiz-1",
    "updatedAt": "2024-01-01T00:00:00Z",
    "entries": [
      {"userId":"u2","displayName":"Bob","score":5},
      {"userId":"u1","displayName":"Alice","score":0}
    ]
  }
  ```

### Clean Architecture Layout
- `cmd/server`: wiring (HTTP server, routes, graceful shutdown).
- `internal/domain`: entities and domain errors.
- `internal/app`: quiz use cases and session orchestration (framework-agnostic).
- `internal/infra/memory`: in-memory `SessionStore` (swap with Redis/DB).
- `internal/transport/http`: Gorilla WebSocket handler.

### AI Collaboration Notes
- AI-assisted sections are called out inline (e.g., WebSocket goroutine wiring, broadcast backpressure handling). Verification steps include reasoning about single-writer semantics and unit tests covering join/score/subscribe flows.

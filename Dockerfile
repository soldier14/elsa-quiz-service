FROM golang:1.22-alpine AS build
WORKDIR /app
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/quiz-service ./cmd

FROM alpine:3.19
WORKDIR /app
COPY --from=build /out/quiz-service /usr/local/bin/quiz-service
COPY config/config.yaml /app/config/config.yaml

EXPOSE 8080
ENV CONFIG_PATH=/app/config/config.yaml
CMD ["quiz-service", "start", "--config", "/app/config/config.yaml"]

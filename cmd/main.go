package main

import (
	"os"

	"elsa-quiz-service/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}

package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	port       string
	configPath string
)

// Execute runs the CLI.
func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	envPort := os.Getenv("PORT")
	if envPort == "" {
		envPort = "8080"
	}
	envConfig := os.Getenv("CONFIG_PATH")
	if envConfig == "" {
		envConfig = "config/config.yaml"
	}

	cmd := &cobra.Command{
		Use:   "quiz-service",
		Short: "Real-time quiz service powered by Gorilla WebSocket",
	}

	cmd.PersistentFlags().StringVar(&port, "port", envPort, "port to listen on")
	cmd.PersistentFlags().StringVar(&configPath, "config", envConfig, "path to YAML config")
	cmd.AddCommand(NewStartCmd(&configPath, &port))
	cmd.AddCommand(NewMigrateCmd(&configPath))
	return cmd
}

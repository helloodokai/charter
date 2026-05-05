package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/helloodokai/charter/internal/config"
	"github.com/helloodokai/charter/internal/githubapp"
	"github.com/helloodokai/charter/internal/routing"
)

var appPort int

var appCmd = &cobra.Command{
	Use:   "app serve",
	Short: "Run the GitHub App webhook server",
	Long:  `Run the CHARTER GitHub App webhook server that listens for issue events and runs interactive dialogues in issue threads.`,
	RunE:  runApp,
}

func init() {
	appCmd.Flags().IntVar(&appPort, "addr", 8080, "address to listen on (port only)")
}

func runApp(cmd *cobra.Command, args []string) error {
	cfg, err := config.FindAndLoad(".")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	appID, err := strconv.ParseInt(os.Getenv(cfg.GitHub.AppIDEnv), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid GITHUB_APP_ID: %w", err)
	}

	privateKey, err := os.ReadFile(os.Getenv(cfg.GitHub.PrivateKeyEnv)) //nolint:gosec // expected: user-controlled config path
	if err != nil {
		return fmt.Errorf("reading private key: %w", err)
	}

	router, err := routing.NewRouter(cfg, cfg.Models.DefaultProfile)
	if err != nil {
		return fmt.Errorf("initializing model router: %w", err)
	}
	_ = router

	srv, err := githubapp.NewServer(githubapp.Config{
		Port:       appPort,
		AppID:      appID,
		PrivateKey: privateKey,
	})
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}

	return srv.Run(context.Background())
}

func main() {
	if err := appCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
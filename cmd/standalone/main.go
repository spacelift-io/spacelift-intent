package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"

	cliflags "spacelift-intent-mcp/internal/cli"
	"spacelift-intent-mcp/pkg/standalone"
)

func main() {
	app := &cli.App{
		Name:        "spacelift-intent-mcp-standalone",
		Usage:       "OpenTofu MCP Server (Standalone Mode)",
		Description: "Standalone mode OpenTofu MCP Server with all functionality in a single binary",
		Flags:       cliflags.StandaloneServerFlags(),
		Action: func(c *cli.Context) error {
			port := c.Int(cliflags.PortFlag.Name)
			serverType := c.String(cliflags.ServerTypeFlag.Name)
			tmpDir := c.String(cliflags.TmpDirFlag.Name)
			dbDir := c.String(cliflags.DBDirFlag.Name)

			// Create standalone server
			config := &standalone.Config{
				Port:       port,
				ServerType: serverType,
				TmpDir:     tmpDir,
				DBDir:      dbDir,
			}

			server, err := standalone.NewServer(config)
			if err != nil {
				return fmt.Errorf("failed to create standalone server: %w", err)
			}

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			errChan := make(chan error, 1)
			go func() {
				if err := server.Start(ctx); err != nil {
					errChan <- err
				}
			}()

			var serverErr error
			select {
			case <-ctx.Done():
				log.Println("Received signal, shutting down...")
			case serverErr = <-errChan:
				log.Println("Server error, shutting down...")
				stop()
			}

			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			server.Stop(ctx)

			log.Println("Standalone server shut down")

			return serverErr
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Failed to run application: %v", err)
	}
}

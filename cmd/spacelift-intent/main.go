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
)

func main() {
	app := &cli.App{
		Name:        "spacelift-intent-mcp-standalone",
		Usage:       "Spacelift Intent MCP Server",
		Description: "Infrastructure management server",
		Flags:       []cli.Flag{tmpDirFlag, dbDirFlag},
		Action: func(c *cli.Context) error {
			tmpDir := c.String(tmpDirFlag.Name)
			dbDir := c.String(dbDirFlag.Name)

			// Create standalone server
			config := &Config{
				TmpDir: tmpDir,
				DBDir:  dbDir,
			}

			server, err := newServer(config)
			if err != nil {
				return fmt.Errorf("failed to create standalone server: %w", err)
			}

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			errChan := make(chan error, 1)
			go func() {
				if err := server.start(ctx); err != nil {
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

			server.stop(ctx)

			log.Println("Standalone server shut down")

			return serverErr
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Failed to run application: %v", err)
	}
}

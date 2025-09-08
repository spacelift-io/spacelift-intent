package main

import "github.com/urfave/cli/v2"

var (
	tmpDirFlag = &cli.StringFlag{
		Name:    "tmp-dir",
		EnvVars: []string{"TMP_DIR"},
		Usage:   "Temporary directory for provider binaries and state",
		Value:   "/tmp/spacelift-intent-executor",
	}
	// Standalone-specific flags
	dbDirFlag = &cli.StringFlag{
		Name:    "db-dir",
		EnvVars: []string{"DB_DIR"},
		Usage:   "Directory containing DB files for persistent state",
		Value:   "./.state/",
	}
)

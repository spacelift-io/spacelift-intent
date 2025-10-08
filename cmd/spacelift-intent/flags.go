// Copyright 2025 Spacelift, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

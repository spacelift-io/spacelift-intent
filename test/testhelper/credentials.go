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

package testhelper

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// SetEnvVars sets environment variables from a map of key-value pairs
func SetEnvVars(t *testing.T, envVars map[string]string) {
	for key, value := range envVars {
		if err := os.Setenv(key, value); err != nil {
			t.Fatalf("Failed to set environment variable %s: %v", key, err)
		}
	}
}

// LoadAWSCredentials loads AWS credentials from .env.aws file
func LoadAWSCredentials(t *testing.T) map[string]string {
	file, err := os.Open("../.env.aws")
	if err != nil {
		t.Skip("Skipping test: .env.aws file not found")
		return nil
	}
	defer file.Close()

	credentials := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Remove surrounding quotes if present
			if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'')) {
				value = value[1 : len(value)-1]
			}

			credentials[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading .env.aws file: %v", err)
	}

	// Verify required AWS credentials are present
	requiredKeys := []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_REGION"}
	for _, key := range requiredKeys {
		if _, exists := credentials[key]; !exists || credentials[key] == "" {
			t.Skipf("Missing required AWS credential: %s", key)
		}
	}

	// Set environment variables
	SetEnvVars(t, credentials)

	return credentials
}

// LoadSpaceliftCredentials loads Spacelift credentials from .env.spacelift file
func LoadSpaceliftCredentials(t *testing.T) map[string]string {
	file, err := os.Open("../.env.spacelift")
	if err != nil {
		t.Skip("Skipping test: .env.spacelift file not found")
		return nil
	}
	defer file.Close()

	credentials := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Remove surrounding quotes if present
			if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'')) {
				value = value[1 : len(value)-1]
			}

			credentials[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading .env.spacelift file: %v", err)
	}

	// Verify required Spacelift credentials are present
	requiredKeys := []string{"SPACELIFT_API_KEY_ENDPOINT", "SPACELIFT_API_KEY_ID", "SPACELIFT_API_KEY_SECRET"}
	for _, key := range requiredKeys {
		if _, exists := credentials[key]; !exists || credentials[key] == "" {
			t.Skipf("Missing required Spacelift credential: %s", key)
		}
	}

	// Set environment variables
	SetEnvVars(t, credentials)

	return credentials
}

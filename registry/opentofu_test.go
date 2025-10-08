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

package registry

import (
	"os"
	"testing"
)

func TestNewOpenTofuClient_DefaultURLs(t *testing.T) {
	// Ensure environment variables are not set
	os.Unsetenv("OPENTOFU_REGISTRY_URL")
	os.Unsetenv("OPENTOFU_API_URL")

	client := NewOpenTofuClient()
	otfClient, ok := client.(*openTofuClient)
	if !ok {
		t.Fatal("expected *openTofuClient type")
	}

	expectedSearchURL := "https://api.opentofu.org/registry/docs/search?q=%s"
	expectedDownloadURL := "https://registry.opentofu.org/v1/providers/%s/%s/%s/download/%s/%s"
	expectedVersionsURL := "https://registry.opentofu.org/v1/providers/%s/%s/versions"

	if otfClient.searchURLTemplate != expectedSearchURL {
		t.Errorf("expected searchURLTemplate %q, got %q", expectedSearchURL, otfClient.searchURLTemplate)
	}

	if otfClient.downloadURLTemplate != expectedDownloadURL {
		t.Errorf("expected downloadURLTemplate %q, got %q", expectedDownloadURL, otfClient.downloadURLTemplate)
	}

	if otfClient.versionsURLTemplate != expectedVersionsURL {
		t.Errorf("expected versionsURLTemplate %q, got %q", expectedVersionsURL, otfClient.versionsURLTemplate)
	}
}

func TestNewOpenTofuClient_CustomRegistryURL(t *testing.T) {
	customRegistryURL := "https://custom-registry.example.com"
	os.Setenv("OPENTOFU_REGISTRY_URL", customRegistryURL)
	defer os.Unsetenv("OPENTOFU_REGISTRY_URL")
	os.Unsetenv("OPENTOFU_API_URL")

	client := NewOpenTofuClient()
	otfClient, ok := client.(*openTofuClient)
	if !ok {
		t.Fatal("expected *openTofuClient type")
	}

	expectedDownloadURL := customRegistryURL + "/v1/providers/%s/%s/%s/download/%s/%s"
	expectedVersionsURL := customRegistryURL + "/v1/providers/%s/%s/versions"

	if otfClient.downloadURLTemplate != expectedDownloadURL {
		t.Errorf("expected downloadURLTemplate %q, got %q", expectedDownloadURL, otfClient.downloadURLTemplate)
	}

	if otfClient.versionsURLTemplate != expectedVersionsURL {
		t.Errorf("expected versionsURLTemplate %q, got %q", expectedVersionsURL, otfClient.versionsURLTemplate)
	}
}

func TestNewOpenTofuClient_CustomAPIURL(t *testing.T) {
	customAPIURL := "https://custom-api.example.com"
	os.Unsetenv("OPENTOFU_REGISTRY_URL")
	os.Setenv("OPENTOFU_API_URL", customAPIURL)
	defer os.Unsetenv("OPENTOFU_API_URL")

	client := NewOpenTofuClient()
	otfClient, ok := client.(*openTofuClient)
	if !ok {
		t.Fatal("expected *openTofuClient type")
	}

	expectedSearchURL := customAPIURL + "/registry/docs/search?q=%s"

	if otfClient.searchURLTemplate != expectedSearchURL {
		t.Errorf("expected searchURLTemplate %q, got %q", expectedSearchURL, otfClient.searchURLTemplate)
	}
}

func TestNewOpenTofuClient_BothCustomURLs(t *testing.T) {
	customRegistryURL := "https://custom-registry.example.com"
	customAPIURL := "https://custom-api.example.com"

	os.Setenv("OPENTOFU_REGISTRY_URL", customRegistryURL)
	os.Setenv("OPENTOFU_API_URL", customAPIURL)
	defer func() {
		os.Unsetenv("OPENTOFU_REGISTRY_URL")
		os.Unsetenv("OPENTOFU_API_URL")
	}()

	client := NewOpenTofuClient()
	otfClient, ok := client.(*openTofuClient)
	if !ok {
		t.Fatal("expected *openTofuClient type")
	}

	expectedSearchURL := customAPIURL + "/registry/docs/search?q=%s"
	expectedDownloadURL := customRegistryURL + "/v1/providers/%s/%s/%s/download/%s/%s"
	expectedVersionsURL := customRegistryURL + "/v1/providers/%s/%s/versions"

	if otfClient.searchURLTemplate != expectedSearchURL {
		t.Errorf("expected searchURLTemplate %q, got %q", expectedSearchURL, otfClient.searchURLTemplate)
	}

	if otfClient.downloadURLTemplate != expectedDownloadURL {
		t.Errorf("expected downloadURLTemplate %q, got %q", expectedDownloadURL, otfClient.downloadURLTemplate)
	}

	if otfClient.versionsURLTemplate != expectedVersionsURL {
		t.Errorf("expected versionsURLTemplate %q, got %q", expectedVersionsURL, otfClient.versionsURLTemplate)
	}
}

func TestNewOpenTofuClient_EmptyEnvVars(t *testing.T) {
	// Test that empty environment variables fall back to defaults
	os.Setenv("OPENTOFU_REGISTRY_URL", "")
	os.Setenv("OPENTOFU_API_URL", "")
	defer func() {
		os.Unsetenv("OPENTOFU_REGISTRY_URL")
		os.Unsetenv("OPENTOFU_API_URL")
	}()

	client := NewOpenTofuClient()
	otfClient, ok := client.(*openTofuClient)
	if !ok {
		t.Fatal("expected *openTofuClient type")
	}

	expectedSearchURL := "https://api.opentofu.org/registry/docs/search?q=%s"
	expectedDownloadURL := "https://registry.opentofu.org/v1/providers/%s/%s/%s/download/%s/%s"
	expectedVersionsURL := "https://registry.opentofu.org/v1/providers/%s/%s/versions"

	if otfClient.searchURLTemplate != expectedSearchURL {
		t.Errorf("expected searchURLTemplate %q, got %q", expectedSearchURL, otfClient.searchURLTemplate)
	}

	if otfClient.downloadURLTemplate != expectedDownloadURL {
		t.Errorf("expected downloadURLTemplate %q, got %q", expectedDownloadURL, otfClient.downloadURLTemplate)
	}

	if otfClient.versionsURLTemplate != expectedVersionsURL {
		t.Errorf("expected versionsURLTemplate %q, got %q", expectedVersionsURL, otfClient.versionsURLTemplate)
	}
}

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

package test

import (
	"testing"
	"time"

	"github.com/spacelift-io/spacelift-intent/test/testhelper"
	"github.com/stretchr/testify/require"
)

// TestCloudFrontDistributionCreate tests creating a CloudFront distribution using lifecycle-resources-create
func TestCloudFrontDistributionCreate(t *testing.T) {
	// Load AWS credentials from .env.aws
	credentials := testhelper.LoadAWSCredentials(t)
	if credentials == nil {
		return // Test was skipped
	}

	// Set environment variables for the test
	for key, value := range credentials {
		t.Setenv(key, value)
	}

	// Use extended timeout for CloudFront operations (10 minutes)
	th := testhelper.NewTestHelperWithTimeout(t, 10*time.Minute)
	defer th.Cleanup()

	const resourceID = "test-cloudfront-distribution"
	const provider = "hashicorp/aws"
	const resourceType = "aws_cloudfront_distribution"

	// CloudFront distribution configuration
	cloudFrontConfig := map[string]any{
		"origin": []map[string]any{
			{
				"domain_name": "example.com",
				"origin_id":   "example",
				"custom_origin_config": []map[string]any{
					{
						"http_port":              80,
						"https_port":             443,
						"origin_protocol_policy": "https-only",
						"origin_ssl_protocols":   []string{"TLSv1.2"},
					},
				},
			},
		},
		"enabled": true,
		"default_cache_behavior": []map[string]any{
			{
				"target_origin_id":       "example",
				"allowed_methods":        []string{"GET", "HEAD"},
				"cached_methods":         []string{"GET", "HEAD"},
				"cache_policy_id":        "658327ea-f89d-4fab-a63d-7e88639e58f6",
				"viewer_protocol_policy": "redirect-to-https",
			},
		},
		"restrictions": []map[string]any{
			{
				"geo_restriction": []map[string]any{
					{
						"restriction_type": "none",
					},
				},
			},
		},
		"viewer_certificate": []map[string]any{
			{
				"cloudfront_default_certificate": true,
			},
		},
		"comment": "Example CloudFront distribution",
		"tags": map[string]string{
			"Environment": "production",
			"Name":        "example-distribution",
		},
	}

	t.Run("CreateCloudFrontDistribution", func(t *testing.T) {
		result, err := th.CallTool("lifecycle-resources-create", map[string]any{
			"resource_id":   resourceID,
			"provider":      provider,
			"resource_type": resourceType,
			"config":        cloudFrontConfig,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-create")

		content := th.GetTextContent(result)
		require.Contains(t, content, resourceID, "Create result should contain resource ID")
		require.Contains(t, content, "created", "Should show created status")

		t.Logf("CloudFront distribution created successfully: %s", content)
	})

	t.Run("RefreshCloudFrontDistribution", func(t *testing.T) {
		result, err := th.CallTool("lifecycle-resources-refresh", map[string]any{
			"resource_id": resourceID,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-refresh")

		content := th.GetTextContent(result)
		require.Contains(t, content, resourceID, "Refresh result should contain resource ID")

		t.Logf("CloudFront distribution refreshed: %s", content)
	})

	t.Run("GetCloudFrontDistributionState", func(t *testing.T) {
		result, err := th.CallTool("state-get", map[string]any{
			"resource_id": resourceID,
		})
		th.AssertToolSuccess(result, err, "state-get")

		content := th.GetTextContent(result)
		require.Contains(t, content, resourceID, "State should contain resource ID")
		require.Contains(t, content, "aws_cloudfront_distribution", "State should contain resource type")

		t.Logf("CloudFront distribution state: %s", content)
	})

	t.Run("DeleteCloudFrontDistribution", func(t *testing.T) {
		result, err := th.CallTool("lifecycle-resources-delete", map[string]any{
			"resource_id": resourceID,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-delete")

		content := th.GetTextContent(result)
		require.Contains(t, content, resourceID, "Delete result should contain resource ID")
		require.Contains(t, content, "deleted", "Should show deleted status")

		t.Logf("CloudFront distribution deleted: %s", content)
	})
}

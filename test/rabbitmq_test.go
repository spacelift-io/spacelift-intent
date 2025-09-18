package test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestRabbitMQBrokerLifecycle tests creating and deleting an AWS MQ RabbitMQ broker
func TestRabbitMQBrokerLifecycle(t *testing.T) {
	// Load AWS credentials from .env.aws
	credentials := loadAWSCredentials(t)
	if credentials == nil {
		return // Test was skipped
	}

	// Set environment variables for the test
	for key, value := range credentials {
		t.Setenv(key, value)
	}

	// Use extended timeout for MQ broker operations (10 minutes)
	th := NewTestHelperWithTimeout(t, 10*time.Minute)
	defer th.Cleanup()

	const resourceID = "ultimate-rabbit"
	const provider = "hashicorp/aws"
	const resourceType = "aws_mq_broker"

	// Original JSON config as provided
	configJSON := `{
		"tags": {
			"Name": "ultimate-rabbit",
			"Purpose": "Ultimate RabbitMQ Message Broker"
		},
		"user": [
			{
				"groups": [],
				"password": "UltimateRabbitPass123!",
				"username": "admin",
				"console_access": true,
				"replication_user": false
			}
		],
		"subnet_ids": [
			"subnet-0d8d1e3c9be867fbd"
		],
		"broker_name": "ultimate-rabbit",
		"engine_type": "RabbitMQ",
		"engine_version": "3.13",
		"deployment_mode": "SINGLE_INSTANCE",
		"host_instance_type": "mq.t3.micro",
		"publicly_accessible": true,
		"auto_minor_version_upgrade": true
	}`

	// Parse JSON config into Go map
	var config map[string]any
	err := json.Unmarshal([]byte(configJSON), &config)
	require.NoError(t, err, "Failed to parse config JSON")

	t.Run("CreateRabbitMQBroker", func(t *testing.T) {
		result, err := th.CallTool("lifecycle-resources-create", map[string]any{
			"resource_id":   resourceID,
			"provider":      provider,
			"resource_type": resourceType,
			"config":        config,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-create")

		content := th.GetTextContent(result)
		require.Contains(t, content, resourceID, "Create result should contain resource ID")
		require.Contains(t, content, "created", "Should show created status")

		t.Logf("RabbitMQ broker created successfully: %s", content)
	})

	t.Run("RefreshRabbitMQBroker", func(t *testing.T) {
		result, err := th.CallTool("lifecycle-resources-refresh", map[string]any{
			"resource_id": resourceID,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-refresh")

		content := th.GetTextContent(result)
		require.Contains(t, content, resourceID, "Refresh result should contain resource ID")

		t.Logf("RabbitMQ broker refreshed: %s", content)
	})

	t.Run("GetRabbitMQBrokerState", func(t *testing.T) {
		result, err := th.CallTool("state-get", map[string]any{
			"resource_id": resourceID,
		})
		th.AssertToolSuccess(result, err, "state-get")

		content := th.GetTextContent(result)
		require.Contains(t, content, resourceID, "State should contain resource ID")
		require.Contains(t, content, "aws_mq_broker", "State should contain resource type")

		t.Logf("RabbitMQ broker state: %s", content)
	})

	t.Run("DeleteRabbitMQBroker", func(t *testing.T) {
		result, err := th.CallTool("lifecycle-resources-delete", map[string]any{
			"resource_id": resourceID,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-delete")

		content := th.GetTextContent(result)
		require.Contains(t, content, resourceID, "Delete result should contain resource ID")
		require.Contains(t, content, "deleted", "Should show deleted status")

		t.Logf("RabbitMQ broker deleted: %s", content)
	})
}

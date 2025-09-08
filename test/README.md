# Test Suite

This directory contains integration tests for the spacelift-intent library.

## Prerequisites

### AWS Credentials

For AWS provider tests (like CloudFront), create a `.env.aws` file in the project root with your AWS credentials:

```bash
AWS_ACCESS_KEY_ID="your_access_key_id"
AWS_SECRET_ACCESS_KEY="your_secret_access_key"
AWS_REGION="us-east-1"
```

The test will automatically load and set these environment variables. Values can be quoted or unquoted.

## Running Tests

### All Integration Tests

```bash
go test ./test -v
```

### CloudFront Distribution Tests

CloudFront distributions take several minutes to create/delete, so use extended timeouts:

```bash
GOLANG_PROTOBUF_REGISTRATION_CONFLICT=warn USE_OPENTOFU_PROVIDER_LIB=true go test -timeout 1000s -run ^TestCloudFrontDistributionCreate github.com/spacelift-io/spacelift-intent/test -v
```

### Environment Variables

- `GOLANG_PROTOBUF_REGISTRATION_CONFLICT=warn` - Suppresses protobuf registration warnings
- `USE_OPENTOFU_PROVIDER_LIB=true` - Enables OpenTofu provider library mode
- `-timeout 1000s` - Sets test timeout to ~17 minutes (CloudFront operations can take 5-15 minutes)

### Test Structure

The CloudFront test (`TestCloudFrontDistributionCreate`) performs a complete lifecycle:

1. **Create** - Creates a CloudFront distribution with custom origin configuration
2. **Refresh** - Refreshes the resource state from AWS
3. **Get State** - Retrieves the current resource state
4. **Delete** - Cleans up the CloudFront distribution

Each operation is tested separately and logs detailed output for debugging.

## Test Configuration

The CloudFront test uses this configuration:

```json
{
  "origin": [
    {
      "domain_name": "example.com",
      "origin_id": "example",
      "custom_origin_config": [
        {
          "http_port": 80,
          "https_port": 443,
          "origin_protocol_policy": "https-only",
          "origin_ssl_protocols": ["TLSv1.2"]
        }
      ]
    }
  ],
  "enabled": true,
  "default_cache_behavior": [
    {
      "target_origin_id": "example",
      "allowed_methods": ["GET", "HEAD"],
      "cached_methods": ["GET", "HEAD"],
      "cache_policy_id": "658327ea-f89d-4fab-a63d-7e88639e58f6",
      "viewer_protocol_policy": "redirect-to-https"
    }
  ],
  "restrictions": [
    {
      "geo_restriction": [
        {
          "restriction_type": "none"
        }
      ]
    }
  ],
  "viewer_certificate": [
    {
      "cloudfront_default_certificate": true
    }
  ],
  "comment": "Example CloudFront distribution",
  "tags": {
    "Environment": "production",
    "Name": "example-distribution"
  }
}
```

## Troubleshooting

### Timeout Issues
- CloudFront distributions can take 5-15 minutes to deploy
- Use `-timeout 1000s` or higher for CloudFront tests
- The test helper uses a 10-minute context timeout internally

### Credential Issues
- Ensure `.env.aws` file exists in project root
- Verify AWS credentials have CloudFront permissions
- Check AWS region is set correctly

### Provider Issues
- Set `USE_OPENTOFU_PROVIDER_LIB=true` environment variable
- Use `GOLANG_PROTOBUF_REGISTRATION_CONFLICT=warn` to suppress warnings
- Ensure AWS provider is available in the test environment
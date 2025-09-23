package testhelper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
)

// GraphQLRequest represents a GraphQL request
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GraphQL response
type GraphQLResponse struct {
	Data   map[string]interface{} `json:"data,omitempty"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

// SpaceliftContext represents a Spacelift context from the API
type SpaceliftContext struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	Labels      []string `json:"labels"`
	Space       string   `json:"space"`
}

// getJWTToken exchanges API key credentials for a JWT token
func getJWTToken(endpoint, keyID, keySecret string) (string, error) {
	// Ensure endpoint has /graphql suffix
	if !strings.HasSuffix(endpoint, "/graphql") {
		endpoint = endpoint + "/graphql"
	}
	// Create GraphQL mutation to exchange API key for JWT (matching the working curl structure)
	mutation := `
		mutation GetSpaceliftToken($keyId: ID!, $keySecret: String!) {
			apiKeyUser(id: $keyId, secret: $keySecret) {
				jwt
			}
		}
	`

	// Prepare GraphQL request matching the working curl structure
	req := GraphQLRequest{
		Query:         mutation,
		OperationName: "GetSpaceliftToken",
		Variables: map[string]interface{}{
			"keyId":     keyID,
			"keySecret": keySecret,
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal GraphQL request: %v", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set headers (no authorization needed for this exchange)
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to execute GraphQL request: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	var gqlResp GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return "", fmt.Errorf("failed to decode GraphQL response: %v", err)
	}

	// Check for GraphQL errors
	if len(gqlResp.Errors) > 0 {
		return "", fmt.Errorf("GraphQL errors during JWT exchange: %s", gqlResp.Errors[0].Message)
	}

	// Extract JWT from response
	apiKeyUser, ok := gqlResp.Data["apiKeyUser"].(map[string]interface{})
	if !ok || apiKeyUser == nil {
		return "", fmt.Errorf("JWT exchange failed: no apiKeyUser in response")
	}

	jwt, ok := apiKeyUser["jwt"].(string)
	if !ok || jwt == "" {
		return "", fmt.Errorf("JWT exchange failed: no jwt token in response")
	}

	return jwt, nil
}

// ValidateContextExistsByName validates that a context with the given name exists by listing all contexts
func ValidateContextExistsByName(t *testing.T, contextName string) (*SpaceliftContext, error) {
	// Get required environment variables
	endpoint := os.Getenv("SPACELIFT_API_KEY_ENDPOINT")
	keyID := os.Getenv("SPACELIFT_API_KEY_ID")
	keySecret := os.Getenv("SPACELIFT_API_KEY_SECRET")

	if endpoint == "" || keyID == "" || keySecret == "" {
		return nil, fmt.Errorf("missing required Spacelift credentials (SPACELIFT_API_KEY_ENDPOINT, SPACELIFT_API_KEY_ID, SPACELIFT_API_KEY_SECRET)")
	}

	// Ensure endpoint has /graphql suffix
	if !strings.HasSuffix(endpoint, "/graphql") {
		endpoint = endpoint + "/graphql"
	}

	// Exchange API key for JWT token
	jwtToken, err := getJWTToken(endpoint, keyID, keySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWT token: %v", err)
	}

	// Create GraphQL query to list all contexts
	query := `
		query {
			contexts {
				id
				name
				description
				labels
				space
			}
		}
	`

	// Prepare GraphQL request
	req := GraphQLRequest{
		Query: query,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL request: %v", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set headers with JWT token
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GraphQL request: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	var gqlResp GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, fmt.Errorf("failed to decode GraphQL response: %v", err)
	}

	// Check for GraphQL errors
	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL errors: %s", gqlResp.Errors[0].Message)
	}

	// Extract contexts from response
	contexts, ok := gqlResp.Data["contexts"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response structure: contexts should be an array")
	}

	// Find context with matching name
	for _, ctxInterface := range contexts {
		ctxMap, ok := ctxInterface.(map[string]interface{})
		if !ok {
			continue
		}

		if name, ok := ctxMap["name"].(string); ok && name == contextName {
			// Convert to SpaceliftContext struct
			contextBytes, err := json.Marshal(ctxMap)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal context data: %v", err)
			}

			var context SpaceliftContext
			if err := json.Unmarshal(contextBytes, &context); err != nil {
				return nil, fmt.Errorf("failed to unmarshal context data: %v", err)
			}

			t.Logf("Successfully validated context exists: ID=%s, Name=%s", context.ID, context.Name)
			return &context, nil
		}
	}

	return nil, fmt.Errorf("context with name '%s' not found", contextName)
}

// ValidateContextExistsById validates that a context with the given ID exists
func ValidateContextExistsById(t *testing.T, contextID string) (*SpaceliftContext, error) {
	// Get required environment variables
	endpoint := os.Getenv("SPACELIFT_API_KEY_ENDPOINT")
	keyID := os.Getenv("SPACELIFT_API_KEY_ID")
	keySecret := os.Getenv("SPACELIFT_API_KEY_SECRET")

	if endpoint == "" || keyID == "" || keySecret == "" {
		return nil, fmt.Errorf("missing required Spacelift credentials (SPACELIFT_API_KEY_ENDPOINT, SPACELIFT_API_KEY_ID, SPACELIFT_API_KEY_SECRET)")
	}

	// Ensure endpoint has /graphql suffix
	if !strings.HasSuffix(endpoint, "/graphql") {
		endpoint = endpoint + "/graphql"
	}

	// Exchange API key for JWT token
	jwtToken, err := getJWTToken(endpoint, keyID, keySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWT token: %v", err)
	}

	// Create GraphQL query to get context by ID
	query := `
		query($id: ID!) {
			context(id: $id) {
				id
				name
				description
				labels
				space
			}
		}
	`

	// Prepare GraphQL request
	req := GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"id": contextID,
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL request: %v", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set headers with JWT token
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GraphQL request: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	var gqlResp GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, fmt.Errorf("failed to decode GraphQL response: %v", err)
	}

	// Check for GraphQL errors
	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL errors: %s", gqlResp.Errors[0].Message)
	}

	// Extract context from response
	contextData, ok := gqlResp.Data["context"]
	if !ok || contextData == nil {
		return nil, fmt.Errorf("context with ID '%s' not found", contextID)
	}

	// Convert to SpaceliftContext struct
	contextBytes, err := json.Marshal(contextData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal context data: %v", err)
	}

	var context SpaceliftContext
	if err := json.Unmarshal(contextBytes, &context); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context data: %v", err)
	}

	t.Logf("Successfully validated context exists: ID=%s, Name=%s", context.ID, context.Name)
	return &context, nil
}

// CleanupContextByName deletes a context by name via Spacelift API and validates cleanup succeeded
func CleanupContextByName(t *testing.T, contextName string) error {
	// First, find the context to get its ID
	context, err := ValidateContextExistsByName(t, contextName)
	if err != nil {
		// Context doesn't exist, consider it already cleaned up
		t.Logf("Context '%s' not found, assuming already cleaned up", contextName)
		return nil
	}

	return CleanupContextById(t, context.ID)
}

// CleanupContextById deletes a context by ID via Spacelift API and validates cleanup succeeded
func CleanupContextById(t *testing.T, contextID string) error {
	// Get required environment variables
	endpoint := os.Getenv("SPACELIFT_API_KEY_ENDPOINT")
	keyID := os.Getenv("SPACELIFT_API_KEY_ID")
	keySecret := os.Getenv("SPACELIFT_API_KEY_SECRET")

	if endpoint == "" || keyID == "" || keySecret == "" {
		return fmt.Errorf("missing required Spacelift credentials (SPACELIFT_API_KEY_ENDPOINT, SPACELIFT_API_KEY_ID, SPACELIFT_API_KEY_SECRET)")
	}

	// Ensure endpoint has /graphql suffix
	if !strings.HasSuffix(endpoint, "/graphql") {
		endpoint = endpoint + "/graphql"
	}

	// Exchange API key for JWT token
	jwtToken, err := getJWTToken(endpoint, keyID, keySecret)
	if err != nil {
		return fmt.Errorf("failed to get JWT token: %v", err)
	}

	// Create GraphQL mutation to delete context
	mutation := `
		mutation($id: ID!) {
			contextDelete(id: $id) {
				id
				name
			}
		}
	`

	// Prepare GraphQL request
	req := GraphQLRequest{
		Query: mutation,
		Variables: map[string]interface{}{
			"id": contextID,
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal GraphQL request: %v", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set headers with JWT token
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to execute GraphQL request: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	var gqlResp GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return fmt.Errorf("failed to decode GraphQL response: %v", err)
	}

	// Check for GraphQL errors
	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("GraphQL errors during deletion: %s", gqlResp.Errors[0].Message)
	}

	// Validate deletion succeeded
	deletedContext, ok := gqlResp.Data["contextDelete"]
	if !ok {
		return fmt.Errorf("context deletion failed: no 'contextDelete' field in response. Response data: %+v", gqlResp.Data)
	}

	// Spacelift returns null for successfully deleted contexts (which is expected)
	if deletedContext == nil {
		t.Logf("Context deletion successful (null response indicates deletion)")
	} else {
		t.Logf("Context deletion response: %+v", deletedContext)
	}

	t.Logf("Successfully deleted context via Spacelift API: ID=%s", contextID)

	// Validate that the context no longer exists
	_, err = ValidateContextExistsById(t, contextID)
	if err != nil {
		// Expected error - context should not exist anymore
		t.Logf("✅ Validated context cleanup: context with ID '%s' no longer exists", contextID)
		return nil
	}

	return fmt.Errorf("context cleanup validation failed: context with ID '%s' still exists after deletion", contextID)
}

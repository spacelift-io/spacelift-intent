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
	"bytes"
	"context"
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
		endpoint += "/graphql"
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
	httpReq, err := http.NewRequestWithContext(context.Background(), "POST", endpoint, bytes.NewBuffer(reqBody))
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
		endpoint += "/graphql"
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
	httpReq, err := http.NewRequestWithContext(context.Background(), "POST", endpoint, bytes.NewBuffer(reqBody))
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

// ValidateContextExistsByID validates that a context with the given ID exists
func ValidateContextExistsByID(t *testing.T, contextID string) (*SpaceliftContext, error) {
	// Get required environment variables
	endpoint := os.Getenv("SPACELIFT_API_KEY_ENDPOINT")
	keyID := os.Getenv("SPACELIFT_API_KEY_ID")
	keySecret := os.Getenv("SPACELIFT_API_KEY_SECRET")

	if endpoint == "" || keyID == "" || keySecret == "" {
		return nil, fmt.Errorf("missing required Spacelift credentials (SPACELIFT_API_KEY_ENDPOINT, SPACELIFT_API_KEY_ID, SPACELIFT_API_KEY_SECRET)")
	}

	// Ensure endpoint has /graphql suffix
	if !strings.HasSuffix(endpoint, "/graphql") {
		endpoint += "/graphql"
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
	httpReq, err := http.NewRequestWithContext(context.Background(), "POST", endpoint, bytes.NewBuffer(reqBody))
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

	return CleanupContextByID(t, context.ID)
}

// CleanupContextByID deletes a context by ID via Spacelift API and validates cleanup succeeded
func CleanupContextByID(t *testing.T, contextID string) error {
	// Get required environment variables
	endpoint := os.Getenv("SPACELIFT_API_KEY_ENDPOINT")
	keyID := os.Getenv("SPACELIFT_API_KEY_ID")
	keySecret := os.Getenv("SPACELIFT_API_KEY_SECRET")

	if endpoint == "" || keyID == "" || keySecret == "" {
		return fmt.Errorf("missing required Spacelift credentials (SPACELIFT_API_KEY_ENDPOINT, SPACELIFT_API_KEY_ID, SPACELIFT_API_KEY_SECRET)")
	}

	// Ensure endpoint has /graphql suffix
	if !strings.HasSuffix(endpoint, "/graphql") {
		endpoint += "/graphql"
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
	httpReq, err := http.NewRequestWithContext(context.Background(), "POST", endpoint, bytes.NewBuffer(reqBody))
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
	_, err = ValidateContextExistsByID(t, contextID)
	if err != nil {
		// Expected error - context should not exist anymore
		t.Logf("✅ Validated context cleanup: context with ID '%s' no longer exists", contextID)
		return nil
	}

	return fmt.Errorf("context cleanup validation failed: context with ID '%s' still exists after deletion", contextID)
}

// UpdateContextDescription updates a context's description via Spacelift API
func UpdateContextDescription(t *testing.T, contextID, newDescription string) error {
	// Get required environment variables
	endpoint := os.Getenv("SPACELIFT_API_KEY_ENDPOINT")
	keyID := os.Getenv("SPACELIFT_API_KEY_ID")
	keySecret := os.Getenv("SPACELIFT_API_KEY_SECRET")

	if endpoint == "" || keyID == "" || keySecret == "" {
		return fmt.Errorf("missing required Spacelift credentials (SPACELIFT_API_KEY_ENDPOINT, SPACELIFT_API_KEY_ID, SPACELIFT_API_KEY_SECRET)")
	}

	// Ensure endpoint has /graphql suffix
	if !strings.HasSuffix(endpoint, "/graphql") {
		endpoint += "/graphql"
	}

	// Exchange API key for JWT token
	jwtToken, err := getJWTToken(endpoint, keyID, keySecret)
	if err != nil {
		return fmt.Errorf("failed to get JWT token: %v", err)
	}

	// Create GraphQL mutation to update context description
	mutation := `
		mutation($id: ID!, $name: String!, $description: String) {
			contextUpdate(id: $id, name: $name, description: $description) {
				id
				name
				description
			}
		}
	`

	// First get the current context to get its name (required for update)
	currentContext, err := ValidateContextExistsByID(t, contextID)
	if err != nil {
		return fmt.Errorf("failed to get current context: %v", err)
	}

	// Prepare GraphQL request
	req := GraphQLRequest{
		Query: mutation,
		Variables: map[string]interface{}{
			"id":          contextID,
			"name":        currentContext.Name,
			"description": newDescription,
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal GraphQL request: %v", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(context.Background(), "POST", endpoint, bytes.NewBuffer(reqBody))
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
		return fmt.Errorf("GraphQL errors during update: %s", gqlResp.Errors[0].Message)
	}

	// Validate update succeeded
	updatedContext, ok := gqlResp.Data["contextUpdate"]
	if !ok || updatedContext == nil {
		return fmt.Errorf("context update failed: no context returned in response. Response data: %+v", gqlResp.Data)
	}

	t.Logf("Successfully updated context description via Spacelift API: ID=%s", contextID)
	return nil
}

// CreateContextViaAPI creates a context directly via Spacelift API (bypassing our tools)
func CreateContextViaAPI(t *testing.T, name, description string) (*SpaceliftContext, error) {
	// Get required environment variables
	endpoint := os.Getenv("SPACELIFT_API_KEY_ENDPOINT")
	keyID := os.Getenv("SPACELIFT_API_KEY_ID")
	keySecret := os.Getenv("SPACELIFT_API_KEY_SECRET")

	if endpoint == "" || keyID == "" || keySecret == "" {
		return nil, fmt.Errorf("missing required Spacelift credentials (SPACELIFT_API_KEY_ENDPOINT, SPACELIFT_API_KEY_ID, SPACELIFT_API_KEY_SECRET)")
	}

	// Ensure endpoint has /graphql suffix
	if !strings.HasSuffix(endpoint, "/graphql") {
		endpoint += "/graphql"
	}

	// Exchange API key for JWT token
	jwtToken, err := getJWTToken(endpoint, keyID, keySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWT token: %v", err)
	}

	// Create GraphQL mutation to create context
	mutation := `
		mutation($name: String!, $description: String, $labels: [String!]) {
			contextCreate(name: $name, description: $description, labels: $labels) {
				id
				name
				description
				labels
			}
		}
	`

	// Prepare GraphQL request
	req := GraphQLRequest{
		Query: mutation,
		Variables: map[string]interface{}{
			"name":        name,
			"description": description,
			"labels":      []string{"spacelift-intent-testing"},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL request: %v", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(context.Background(), "POST", endpoint, bytes.NewBuffer(reqBody))
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
		return nil, fmt.Errorf("GraphQL errors during creation: %s", gqlResp.Errors[0].Message)
	}

	// Validate creation succeeded
	createdContextData, ok := gqlResp.Data["contextCreate"]
	if !ok || createdContextData == nil {
		return nil, fmt.Errorf("context creation failed: no context returned in response. Response data: %+v", gqlResp.Data)
	}

	// Parse the created context data
	contextMap, ok := createdContextData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid context data format in response")
	}

	contextID, ok := contextMap["id"].(string)
	if !ok || contextID == "" {
		return nil, fmt.Errorf("missing or invalid context ID in response")
	}

	contextName, ok := contextMap["name"].(string)
	if !ok || contextName == "" {
		return nil, fmt.Errorf("missing or invalid context name in response")
	}

	var contextDescription *string
	if desc, exists := contextMap["description"]; exists && desc != nil {
		if descStr, ok := desc.(string); ok {
			contextDescription = &descStr
		}
	}

	var contextLabels []string
	if labels, exists := contextMap["labels"]; exists && labels != nil {
		if labelList, ok := labels.([]interface{}); ok {
			for _, label := range labelList {
				if labelStr, ok := label.(string); ok {
					contextLabels = append(contextLabels, labelStr)
				}
			}
		}
	}

	context := &SpaceliftContext{
		ID:          contextID,
		Name:        contextName,
		Description: contextDescription,
		Labels:      contextLabels,
	}

	t.Logf("Successfully created context via Spacelift API: ID=%s, Name=%s", context.ID, context.Name)
	return context, nil
}

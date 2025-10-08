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

package resources

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spacelift-io/spacelift-intent/types"
)

func newResourceOperation(input types.ResourceOperationInput) (types.ResourceOperation, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return types.ResourceOperation{}, fmt.Errorf("failed to generate operation ID: %v", err)
	}

	return types.ResourceOperation{
		ID:                     id.String(),
		ResourceOperationInput: input,
	}, nil
}

// RespondJSON returns a JSON response
func RespondJSON(response map[string]any) (*mcp.CallToolResult, error) {
	result, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

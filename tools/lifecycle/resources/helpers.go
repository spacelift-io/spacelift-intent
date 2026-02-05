// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"

	"github.com/google/uuid"

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

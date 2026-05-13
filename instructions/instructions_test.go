// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package instructions

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacelift-io/spacelift-intent/allowlist"
)

func TestGetWithAllowlist_DisabledReturnsBase(t *testing.T) {
	assert.Equal(t, Get(), GetWithAllowlist(allowlist.Disabled()))
	assert.Equal(t, Get(), GetWithAllowlist(nil))
}

func TestGetWithAllowlist_AppendsSummary(t *testing.T) {
	al, err := allowlist.LoadFromBytes([]byte(`
providers:
  - name: hashicorp/*
  - name: integrations/github
    versions: "~> 6.0"
`))
	require.NoError(t, err)

	out := GetWithAllowlist(al)
	assert.True(t, strings.HasPrefix(out, Get()), "base instructions must remain intact")
	assert.Contains(t, out, "Provider Allowlist")
	assert.Contains(t, out, "hashicorp/*")
	assert.Contains(t, out, "integrations/github")
}

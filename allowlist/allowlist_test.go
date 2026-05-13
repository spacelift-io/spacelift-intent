// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package allowlist

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacelift-io/spacelift-intent/types"
)

func TestDisabled_AllowsEverything(t *testing.T) {
	al := Disabled()
	assert.False(t, al.Enabled())
	assert.NoError(t, al.Allows(&types.ProviderConfig{Name: "evil/fake", Version: "1.0.0"}))
	assert.NoError(t, al.AllowsName("evil/fake"))
	assert.Empty(t, al.Summary())
}

func TestNilReceiver_AllowsEverything(t *testing.T) {
	var al *Allowlist
	assert.False(t, al.Enabled())
	assert.NoError(t, al.Allows(&types.ProviderConfig{Name: "evil/fake", Version: "1.0.0"}))
	assert.NoError(t, al.AllowsName("evil/fake"))
	assert.Empty(t, al.Summary())
}

func TestExactMatch(t *testing.T) {
	al := mustParse(t, `
providers:
  - name: hashicorp/aws
`)
	assert.NoError(t, al.Allows(provider("hashicorp/aws", "5.0.0")))
	assert.Error(t, al.Allows(provider("hashicorp/random", "3.0.0")))
	assert.Error(t, al.Allows(provider("kuwas/aws", "1.0.0")))
}

func TestNamespaceWildcard(t *testing.T) {
	al := mustParse(t, `
providers:
  - name: hashicorp/*
`)
	assert.NoError(t, al.Allows(provider("hashicorp/aws", "5.0.0")))
	assert.NoError(t, al.Allows(provider("hashicorp/random", "3.0.0")))
	assert.Error(t, al.Allows(provider("opentofu/aws", "5.0.0")))
	assert.Error(t, al.Allows(provider("kuwas/github", "4.3.0")))
}

func TestVersionConstraint_Allowed(t *testing.T) {
	al := mustParse(t, `
providers:
  - name: hashicorp/aws
    versions: ">= 5.0.0"
`)
	assert.NoError(t, al.Allows(provider("hashicorp/aws", "5.0.0")))
	assert.NoError(t, al.Allows(provider("hashicorp/aws", "6.42.0")))
}

func TestVersionConstraint_Denied(t *testing.T) {
	al := mustParse(t, `
providers:
  - name: hashicorp/aws
    versions: ">= 5.0.0"
`)
	err := al.Allows(provider("hashicorp/aws", "4.50.0"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "4.50.0")
}

func TestVersionConstraint_VPrefix(t *testing.T) {
	al := mustParse(t, `
providers:
  - name: hashicorp/aws
    versions: ">= 5.0.0"
`)
	assert.NoError(t, al.Allows(provider("hashicorp/aws", "v5.20.0")))
}

func TestMostSpecificWins_ExactDeniesViaConstraint(t *testing.T) {
	al := mustParse(t, `
providers:
  - name: hashicorp/*
  - name: hashicorp/aws
    versions: ">= 5.0.0"
`)
	// Exact entry exists for aws → exact tier wins; wildcard is shadowed.
	assert.Error(t, al.Allows(provider("hashicorp/aws", "4.50.0")), "exact rule must win over wildcard")
	assert.NoError(t, al.Allows(provider("hashicorp/aws", "5.20.0")))
	// random has no exact entry → wildcard tier applies.
	assert.NoError(t, al.Allows(provider("hashicorp/random", "3.0.0")))
}

func TestMultipleExactEntries_ORWithinTier(t *testing.T) {
	al := mustParse(t, `
providers:
  - name: hashicorp/aws
    versions: ">= 5.0.0"
  - name: hashicorp/aws
    versions: "< 4.0.0"
`)
	assert.NoError(t, al.Allows(provider("hashicorp/aws", "3.5.0")))
	assert.Error(t, al.Allows(provider("hashicorp/aws", "4.5.0")))
	assert.NoError(t, al.Allows(provider("hashicorp/aws", "5.20.0")))
}

func TestConstraintsDoNotInherit(t *testing.T) {
	// Wildcard's constraint must not leak to exact rule that has no constraint.
	al := mustParse(t, `
providers:
  - name: hashicorp/*
    versions: ">= 2.0.0"
  - name: hashicorp/aws
`)
	assert.NoError(t, al.Allows(provider("hashicorp/aws", "1.5.0")), "exact tier wins; no constraint = any version")
	assert.Error(t, al.Allows(provider("hashicorp/random", "1.5.0")), "wildcard constraint applies when no exact entry")
	assert.NoError(t, al.Allows(provider("hashicorp/random", "2.5.0")))
}

func TestAllowsName_IgnoresVersion(t *testing.T) {
	al := mustParse(t, `
providers:
  - name: hashicorp/aws
    versions: ">= 5.0.0"
`)
	assert.NoError(t, al.AllowsName("hashicorp/aws"))
	assert.Error(t, al.AllowsName("hashicorp/random"))
}

func TestAllowsName_WildcardMatches(t *testing.T) {
	al := mustParse(t, `
providers:
  - name: hashicorp/*
`)
	assert.NoError(t, al.AllowsName("hashicorp/aws"))
	assert.NoError(t, al.AllowsName("hashicorp/random"))
	assert.Error(t, al.AllowsName("opentofu/aws"))
}

func TestInvalidProviderName(t *testing.T) {
	al := mustParse(t, `
providers:
  - name: hashicorp/aws
`)
	for _, name := range []string{"", "noslash", "too/many/slashes", "/aws", "hashicorp/"} {
		assert.Error(t, al.AllowsName(name), name)
	}
}

func TestEmptyProvidersList_DeniesAll(t *testing.T) {
	al := mustParse(t, `providers: []`)
	assert.True(t, al.Enabled())
	assert.Error(t, al.Allows(provider("hashicorp/aws", "5.0.0")))
}

func TestParse_InvalidEntries(t *testing.T) {
	cases := map[string]string{
		"empty name":               `providers: [{name: ""}]`,
		"no namespace":              `providers: [{name: "/aws"}]`,
		"no type":                   `providers: [{name: "hashicorp/"}]`,
		"cross-namespace wildcard":  `providers: [{name: "*/aws"}]`,
		"too many slashes":          `providers: [{name: "a/b/c"}]`,
		"invalid version constraint": `providers: [{name: "hashicorp/aws", versions: "not a constraint"}]`,
	}
	for label, yamlSrc := range cases {
		t.Run(label, func(t *testing.T) {
			_, err := parse([]byte(yamlSrc))
			assert.Error(t, err)
		})
	}
}

func TestParse_MalformedYAML(t *testing.T) {
	_, err := parse([]byte("not: yaml: [["))
	assert.Error(t, err)
}

func TestLoad_BuiltinTrusted(t *testing.T) {
	al, err := Load("builtin:trusted")
	require.NoError(t, err)
	assert.True(t, al.Enabled())
	assert.NoError(t, al.Allows(provider("hashicorp/aws", "6.0.0")))
	assert.NoError(t, al.Allows(provider("opentofu/random", "3.0.0")))
	assert.Error(t, al.Allows(provider("kuwas/github", "4.3.0")))
	assert.Error(t, al.Allows(provider("integrations/github", "6.0.0")))
}

func TestLoad_UnknownBuiltin(t *testing.T) {
	_, err := Load("builtin:nonsense")
	assert.Error(t, err)
}

func TestLoad_FromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "allowlist.yaml")
	err := writeFile(path, `
providers:
  - name: hashicorp/aws
    versions: "~> 5.0"
`)
	require.NoError(t, err)

	al, err := Load(path)
	require.NoError(t, err)
	assert.NoError(t, al.Allows(provider("hashicorp/aws", "5.20.0")))
	assert.Error(t, al.Allows(provider("hashicorp/aws", "6.0.0")))
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/allowlist.yaml")
	assert.Error(t, err)
}

func TestSummary(t *testing.T) {
	al := mustParse(t, `
providers:
  - name: hashicorp/*
  - name: integrations/github
    versions: "~> 6.0"
`)
	assert.Equal(t, "hashicorp/*, integrations/github (~> 6.0)", al.Summary())
}

// helpers

func mustParse(t *testing.T, yaml string) *Allowlist {
	t.Helper()
	al, err := parse([]byte(yaml))
	require.NoError(t, err)
	return al
}

func provider(name, ver string) *types.ProviderConfig {
	return &types.ProviderConfig{Name: name, Version: ver}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}

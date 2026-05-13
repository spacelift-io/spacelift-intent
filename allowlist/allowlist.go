// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

// Package allowlist constrains which providers the server is willing to load
// and operate against. Used as a supply-chain trust boundary in front of the
// OpenTofu registry: the LLM cannot widen this boundary at runtime.
package allowlist

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/go-version"
	"gopkg.in/yaml.v3"

	"github.com/spacelift-io/spacelift-intent/types"
)

//go:embed builtin/trusted.yaml
var builtinTrusted []byte

const builtinPrefix = "builtin:"

// Allowlist permits a configured set of providers. A nil or disabled Allowlist
// permits everything (used when the deployer does not configure a list).
type Allowlist struct {
	enabled bool
	entries []entry
}

type entry struct {
	namespace   string
	// name is empty for a namespace wildcard ("hashicorp/*").
	name        string
	// constraints is nil when no version restriction is configured.
	constraints version.Constraints
}

type yamlFile struct {
	Providers []yamlEntry `yaml:"providers"`
}

type yamlEntry struct {
	Name     string `yaml:"name"`
	Versions string `yaml:"versions,omitempty"`
}

// Disabled returns an Allowlist that permits all providers.
func Disabled() *Allowlist {
	return &Allowlist{}
}

// Load reads an allowlist from a file path or the "builtin:<name>" sentinel.
// Currently the only builtin is "trusted" (hashicorp/* + opentofu/*).
func Load(path string) (*Allowlist, error) {
	data, err := readSource(path)
	if err != nil {
		return nil, err
	}
	return LoadFromBytes(data)
}

// LoadFromBytes parses an allowlist directly from YAML bytes.
func LoadFromBytes(data []byte) (*Allowlist, error) {
	return parse(data)
}

func readSource(path string) ([]byte, error) {
	if name, ok := strings.CutPrefix(path, builtinPrefix); ok {
		switch name {
		case "trusted":
			return builtinTrusted, nil
		default:
			return nil, fmt.Errorf("unknown builtin allowlist %q (available: trusted)", name)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read allowlist file: %w", err)
	}
	return data, nil
}

func parse(data []byte) (*Allowlist, error) {
	var f yamlFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse allowlist YAML: %w", err)
	}

	al := &Allowlist{enabled: true}
	for i, ye := range f.Providers {
		e, err := parseEntry(ye)
		if err != nil {
			return nil, fmt.Errorf("entry %d (%q): %w", i, ye.Name, err)
		}
		al.entries = append(al.entries, e)
	}
	return al, nil
}

func parseEntry(ye yamlEntry) (entry, error) {
	if ye.Name == "" {
		return entry{}, fmt.Errorf("name must not be empty")
	}

	parts := strings.Split(ye.Name, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return entry{}, fmt.Errorf("invalid name %q: expected 'namespace/type' or 'namespace/*'", ye.Name)
	}
	if parts[0] == "*" {
		return entry{}, fmt.Errorf("cross-namespace wildcard %q not supported", ye.Name)
	}

	e := entry{namespace: parts[0]}
	if parts[1] != "*" {
		e.name = parts[1]
	}

	if ye.Versions != "" {
		c, err := version.NewConstraint(ye.Versions)
		if err != nil {
			return entry{}, fmt.Errorf("invalid version constraint %q: %w", ye.Versions, err)
		}
		e.constraints = c
	}

	return e, nil
}

// Enabled reports whether any restriction is in effect.
func (a *Allowlist) Enabled() bool {
	return a != nil && a.enabled
}

// AllowsName checks name-only membership, ignoring version constraints.
// Used by the registry decorator to filter search results, which do not
// carry concrete versions.
func (a *Allowlist) AllowsName(name string) error {
	if !a.Enabled() {
		return nil
	}

	namespace, typ, err := parseProviderName(name)
	if err != nil {
		return err
	}

	for _, e := range a.entries {
		if e.namespace == namespace && (e.name == "" || e.name == typ) {
			return nil
		}
	}
	return fmt.Errorf("provider %s not in allowlist", name)
}

// Allows checks both name and version. Match semantics: most-specific-wins
// (exact name overrides namespace wildcard); within the chosen specificity
// tier, the provider is allowed if any entry permits the version (OR).
func (a *Allowlist) Allows(p *types.ProviderConfig) error {
	if !a.Enabled() {
		return nil
	}
	if p == nil {
		return fmt.Errorf("nil provider config")
	}

	namespace, typ, err := parseProviderName(p.Name)
	if err != nil {
		return err
	}

	var v *version.Version
	if p.Version != "" {
		v, err = version.NewVersion(p.Version)
		if err != nil {
			return fmt.Errorf("invalid provider version %q: %w", p.Version, err)
		}
	}

	var exact, wildcard []entry
	for _, e := range a.entries {
		if e.namespace != namespace {
			continue
		}
		switch e.name {
		case typ:
			exact = append(exact, e)
		case "":
			wildcard = append(wildcard, e)
		}
	}

	tier := exact
	if len(tier) == 0 {
		tier = wildcard
	}
	if len(tier) == 0 {
		return fmt.Errorf("provider %s not in allowlist", p.Name)
	}

	for _, e := range tier {
		if e.constraints == nil {
			return nil
		}
		if v != nil && e.constraints.Check(v) {
			return nil
		}
	}

	if p.Version == "" {
		return fmt.Errorf("provider %s not in allowlist: matching entries have version constraints but no version was provided", p.Name)
	}
	return fmt.Errorf("provider %s@%s not in allowlist: version does not satisfy any matching constraint", p.Name, p.Version)
}

// Summary returns a comma-separated list of configured entries, suitable for
// inclusion in MCP server instructions. Returns the empty string when disabled.
func (a *Allowlist) Summary() string {
	if !a.Enabled() {
		return ""
	}
	names := make([]string, 0, len(a.entries))
	for _, e := range a.entries {
		var name string
		if e.name == "" {
			name = e.namespace + "/*"
		} else {
			name = e.namespace + "/" + e.name
		}
		if len(e.constraints) > 0 {
			name += " (" + e.constraints.String() + ")"
		}
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

func parseProviderName(name string) (namespace, typ string, err error) {
	parts := strings.Split(name, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid provider name %q: expected 'namespace/type'", name)
	}
	return parts[0], parts[1], nil
}

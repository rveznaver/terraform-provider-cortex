package cortex

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSuppressYAMLDiff(t *testing.T) {
	originalYAML := `route:
  group_by: ['alertname']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 1h
  receiver: 'web.hook'
`
	equivalentYAML := `route:
  group_by:          ['alertname'    ]
  group_wait:      30s
  group_interval: "5m"
  repeat_interval:      1h
  receiver: web.hook
`
	badYAML := `route:
group_wait:      30s
- group_interval: "5m
`
	tests := []struct {
		name             string
		oldValue         string
		newValue         string
		expectedSuppress bool
	}{
		{"original vs original", originalYAML, originalYAML, true},
		{"original vs equivalent", originalYAML, equivalentYAML, true},
		{"original vs empty", originalYAML, "", false},
		{"bad vs bad", badYAML, badYAML, false},
		{"original vs bad", originalYAML, badYAML, false},
		{"boolean vs string equivalent", `some_bool: true`, `some_bool: "true"`, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expectedSuppress, suppressYAMLDiff("", test.oldValue, test.newValue, nil))
		})
	}
}

func TestSuppressRuleGroupDiff(t *testing.T) {
	ruleGroupYAMLWithBooleanAsBoolean := `name: test
rules:
- alert: test
  expr: 'test > 0'
  labels:
    test_1: true
    severity: critical
    team: platform
  annotations:
    title: 'some title'
    summary: 'some summary'
    description: |-
      Some description
      Link https://github.com
`
	ruleGroupYAMLWithBooleanAsString := `name: test
rules:
- alert: test
  expr: 'test > 0'
  labels:
    test_1: "true"
    severity: critical
    team: platform
  annotations:
    title: 'some title'
    summary: 'some summary'
    description: |-
      Some description
      Link https://github.com
`
	tests := []struct {
		name             string
		oldValue         string
		newValue         string
		expectedSuppress bool
	}{
		{"boolean vs string equivalent", `some_bool: true`, `some_bool: "true"`, true},
		{"RuleGroup with boolean as boolean vs RuleGroup with boolean as string", ruleGroupYAMLWithBooleanAsBoolean, ruleGroupYAMLWithBooleanAsString, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expectedSuppress, suppressRuleGroupDiff("", test.oldValue, test.newValue, nil))
		})
	}
}

// contentStateValue simulates what Terraform stores in state for the content
// attribute. If the resource schema has a StateFunc, it is applied (as
// Terraform would); otherwise the raw value is stored unchanged.
func contentStateValue(value string) string {
	sf := resourceRules().Schema["content"].StateFunc
	if sf != nil {
		return sf(value)
	}
	return value
}

// TestIdenticalContentProducesIdenticalState verifies that semantically
// identical YAML written in different key orders produces identical stored
// state. Without normalisation, struct-marshalled YAML from Read and
// user-authored YAML from config are stored in different formats, which
// causes noisy plan diffs whenever a real change exists.
func TestIdenticalContentProducesIdenticalState(t *testing.T) {
	// Same content, two different key orders.
	orderA := "foo: 1\nbar: 2\nbaz: 3\n"
	orderB := "baz: 3\nfoo: 1\nbar: 2\n"

	require.Equal(t, contentStateValue(orderA), contentStateValue(orderB),
		"Semantically identical YAML in different key orders must produce identical stored state")
}

// TestStoredStateIsStable verifies that processing YAML through the state
// storage path twice produces the same result as once. If the stored form
// is not stable, each plan/refresh cycle would produce a different state
// value, causing spurious diffs.
func TestStoredStateIsStable(t *testing.T) {
	input := "foo: 1\nbar: 2\nbaz: 3\n"

	first := contentStateValue(input)
	second := contentStateValue(first)
	require.Equal(t, first, second,
		"Storing the same content twice must produce identical state")
}

// TestSmallChangeProducesSmallDiff verifies that when a single value changes
// between two differently-ordered YAML documents, the stored state diff
// contains only the line that actually changed -- not a full rewrite caused
// by key reordering.
func TestSmallChangeProducesSmallDiff(t *testing.T) {
	// Old in one key order, new in a different order with one value changed.
	old := "alert: test\nexpr: x > 100\nfor: 1m\nlabels:\n  severity: warning\n"
	new := "labels:\n  severity: warning\nalert: test\nfor: 1m\nexpr: x > 200\n"

	oldState := contentStateValue(old)
	newState := contentStateValue(new)

	oldLines := strings.Split(strings.TrimSpace(oldState), "\n")
	newLines := strings.Split(strings.TrimSpace(newState), "\n")

	require.Equal(t, len(oldLines), len(newLines),
		"Changing one value should not add or remove lines")

	diffCount := 0
	for i := 0; i < len(oldLines) && i < len(newLines); i++ {
		if oldLines[i] != newLines[i] {
			diffCount++
		}
	}

	require.Equal(t, 1, diffCount,
		"Only the changed value should differ, got %d differing lines", diffCount)
}

// TestArrayOrderIsPreserved verifies that normalisation does not reorder
// YAML arrays. Array order is semantically significant (e.g., rule
// evaluation order in Prometheus).
func TestArrayOrderIsPreserved(t *testing.T) {
	input := "items:\n  - third\n  - first\n  - second\n"

	state := contentStateValue(input)

	require.Contains(t, state, "- third\n")
	require.True(t,
		strings.Index(state, "third") < strings.Index(state, "first") &&
			strings.Index(state, "first") < strings.Index(state, "second"),
		"Array element order must be preserved after normalisation")
}

// TestDeepNestedNormalisation verifies that map key sorting and array order
// preservation work correctly through multiple levels of nesting.
func TestDeepNestedNormalisation(t *testing.T) {
	// Keys deliberately in non-alphabetical order at every level.
	orderA := `z_outer:
  z_inner:
    z_deep: 1
    a_deep: 2
  a_inner:
    value: 3
a_outer:
  items:
    - third
    - first
`
	orderB := `a_outer:
  items:
    - third
    - first
z_outer:
  a_inner:
    value: 3
  z_inner:
    a_deep: 2
    z_deep: 1
`

	require.Equal(t, contentStateValue(orderA), contentStateValue(orderB),
		"Deeply nested YAML with different key orders must produce identical stored state")
}

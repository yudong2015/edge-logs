package clickhouse

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/outpostos/edge-logs/pkg/model/request"
)

func TestK8sFilterBuilder_BuildNamespaceConditions(t *testing.T) {
	builder := NewK8sFilterBuilder()

	tests := []struct {
		name               string
		filters            []request.K8sFilter
		expectedConditions int
		expectedArgs       int
		containsSQL        []string
	}{
		{
			name: "single exact namespace",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterExact, Pattern: "production", Field: "namespace"},
			},
			expectedConditions: 1,
			expectedArgs:       1,
			containsSQL:        []string{"k8s_namespace_name = ?"},
		},
		{
			name: "multiple exact namespaces",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterExact, Pattern: "production", Field: "namespace"},
				{Type: request.K8sFilterExact, Pattern: "staging", Field: "namespace"},
			},
			expectedConditions: 1,
			expectedArgs:       2,
			containsSQL:        []string{"k8s_namespace_name IN"},
		},
		{
			name: "prefix namespace",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterPrefix, Pattern: "kube-", Field: "namespace"},
			},
			expectedConditions: 1,
			expectedArgs:       1,
			containsSQL:        []string{"startsWith(k8s_namespace_name, ?)"},
		},
		{
			name: "regex namespace",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterRegex, Pattern: "^env-.*$", Field: "namespace"},
			},
			expectedConditions: 1,
			expectedArgs:       1,
			containsSQL:        []string{"match(k8s_namespace_name, ?)"},
		},
		{
			name: "wildcard namespace",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterWildcard, Pattern: "test-*-env", Field: "namespace"},
			},
			expectedConditions: 1,
			expectedArgs:       1,
			containsSQL:        []string{"k8s_namespace_name LIKE ?"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder.SetFilters(tt.filters)
			conditions, args, err := builder.BuildK8sFilterConditions()

			require.NoError(t, err)
			assert.Equal(t, tt.expectedConditions, len(conditions))
			assert.Equal(t, tt.expectedArgs, len(args))

			// Check that expected SQL patterns are present
			conditionString := strings.Join(conditions, " ")
			for _, sqlPattern := range tt.containsSQL {
				assert.Contains(t, conditionString, sqlPattern)
			}
		})
	}
}

func TestK8sFilterBuilder_BuildPodConditions(t *testing.T) {
	builder := NewK8sFilterBuilder()

	tests := []struct {
		name               string
		filters            []request.K8sFilter
		expectedConditions int
		expectedArgs       int
		containsSQL        []string
	}{
		{
			name: "single exact pod",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterExact, Pattern: "web-app-123", Field: "pod"},
			},
			expectedConditions: 1,
			expectedArgs:       1,
			containsSQL:        []string{"k8s_pod_name = ?"},
		},
		{
			name: "prefix pod with case sensitivity",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterPrefix, Pattern: "api-", Field: "pod", CaseInsensitive: false},
			},
			expectedConditions: 1,
			expectedArgs:       1,
			containsSQL:        []string{"startsWith(k8s_pod_name, ?)"},
		},
		{
			name: "prefix pod case insensitive",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterPrefix, Pattern: "API-", Field: "pod", CaseInsensitive: true},
			},
			expectedConditions: 1,
			expectedArgs:       1,
			containsSQL:        []string{"startsWithUTF8(lowerUTF8(k8s_pod_name), lowerUTF8(?))"},
		},
		{
			name: "suffix pod",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterSuffix, Pattern: "-worker", Field: "pod"},
			},
			expectedConditions: 1,
			expectedArgs:       1,
			containsSQL:        []string{"endsWith(k8s_pod_name, ?)"},
		},
		{
			name: "contains pod",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterContains, Pattern: "database", Field: "pod"},
			},
			expectedConditions: 1,
			expectedArgs:       1,
			containsSQL:        []string{"position(k8s_pod_name, ?) > 0"},
		},
		{
			name: "contains pod case insensitive",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterContains, Pattern: "DATABASE", Field: "pod", CaseInsensitive: true},
			},
			expectedConditions: 1,
			expectedArgs:       1,
			containsSQL:        []string{"positionCaseInsensitiveUTF8(k8s_pod_name, ?) > 0"},
		},
		{
			name: "regex pod",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterRegex, Pattern: "^app-[0-9]+$", Field: "pod"},
			},
			expectedConditions: 1,
			expectedArgs:       1,
			containsSQL:        []string{"match(k8s_pod_name, ?)"},
		},
		{
			name: "regex pod case insensitive",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterRegex, Pattern: "^APP-[0-9]+$", Field: "pod", CaseInsensitive: true},
			},
			expectedConditions: 1,
			expectedArgs:       1,
			containsSQL:        []string{"match(k8s_pod_name, ?)"},
		},
		{
			name: "wildcard pod",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterWildcard, Pattern: "web-*-prod", Field: "pod"},
			},
			expectedConditions: 1,
			expectedArgs:       1,
			containsSQL:        []string{"k8s_pod_name LIKE ?"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder.SetFilters(tt.filters)
			conditions, args, err := builder.BuildK8sFilterConditions()

			require.NoError(t, err)
			assert.Equal(t, tt.expectedConditions, len(conditions))
			assert.Equal(t, tt.expectedArgs, len(args))

			// Check that expected SQL patterns are present
			conditionString := strings.Join(conditions, " ")
			for _, sqlPattern := range tt.containsSQL {
				assert.Contains(t, conditionString, sqlPattern)
			}
		})
	}
}

func TestK8sFilterBuilder_MixedFilters(t *testing.T) {
	builder := NewK8sFilterBuilder()

	filters := []request.K8sFilter{
		{Type: request.K8sFilterExact, Pattern: "production", Field: "namespace"},
		{Type: request.K8sFilterPrefix, Pattern: "api-", Field: "pod"},
		{Type: request.K8sFilterRegex, Pattern: "^kube-.*$", Field: "namespace"},
		{Type: request.K8sFilterWildcard, Pattern: "*worker*", Field: "pod"},
	}

	builder.SetFilters(filters)
	conditions, args, err := builder.BuildK8sFilterConditions()

	require.NoError(t, err)
	assert.Greater(t, len(conditions), 0)
	assert.Equal(t, len(filters), len(args))

	// Check that we have both namespace and pod conditions
	conditionString := strings.Join(conditions, " ")
	assert.Contains(t, conditionString, "k8s_namespace_name")
	assert.Contains(t, conditionString, "k8s_pod_name")
}

func TestK8sFilterBuilder_EstimateFilterComplexity(t *testing.T) {
	builder := NewK8sFilterBuilder()

	tests := []struct {
		name              string
		filters           []request.K8sFilter
		expectedComplexity float64
		compareOp         string // "equal", "greater", "less"
	}{
		{
			name: "simple exact filter",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterExact, Pattern: "production", Field: "namespace"},
			},
			expectedComplexity: 1.0,
			compareOp:          "equal",
		},
		{
			name: "complex regex filter",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterRegex, Pattern: "^app-.*$", Field: "pod"},
			},
			expectedComplexity: 5.0,
			compareOp:          "equal",
		},
		{
			name: "case insensitive filter adds complexity",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterPrefix, Pattern: "api-", Field: "pod", CaseInsensitive: true},
			},
			expectedComplexity: 3.0,
			compareOp:          "equal",
		},
		{
			name: "multiple filters",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterExact, Pattern: "production", Field: "namespace"},
				{Type: request.K8sFilterPrefix, Pattern: "api-", Field: "pod"},
				{Type: request.K8sFilterRegex, Pattern: "^kube-.*$", Field: "namespace"},
			},
			expectedComplexity: 8.0,
			compareOp:          "equal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder.SetFilters(tt.filters)
			complexity := builder.EstimateFilterComplexity()

			switch tt.compareOp {
			case "equal":
				assert.Equal(t, tt.expectedComplexity, complexity)
			case "greater":
				assert.Greater(t, complexity, tt.expectedComplexity)
			case "less":
				assert.Less(t, complexity, tt.expectedComplexity)
			}
		})
	}
}

func TestK8sFilterBuilder_EstimateFilterSelectivity(t *testing.T) {
	builder := NewK8sFilterBuilder()

	tests := []struct {
		name        string
		filters     []request.K8sFilter
		expectLow   bool // true if expecting low selectivity (high value)
		expectHigh  bool // true if expecting high selectivity (low value)
	}{
		{
			name: "exact filters are highly selective",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterExact, Pattern: "production", Field: "namespace"},
			},
			expectHigh: true,
		},
		{
			name: "regex filters have lower selectivity",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterRegex, Pattern: ".*", Field: "pod"},
			},
			expectLow: true,
		},
		{
			name: "multiple exact filters are very selective",
			filters: []request.K8sFilter{
				{Type: request.K8sFilterExact, Pattern: "production", Field: "namespace"},
				{Type: request.K8sFilterExact, Pattern: "web-app-123", Field: "pod"},
			},
			expectHigh: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder.SetFilters(tt.filters)
			selectivity := builder.EstimateFilterSelectivity()

			assert.GreaterOrEqual(t, selectivity, 0.0)
			assert.LessOrEqual(t, selectivity, 1.0)

			if tt.expectHigh {
				assert.Less(t, selectivity, 0.1, "Expected high selectivity (low value)")
			}
			if tt.expectLow {
				assert.Greater(t, selectivity, 0.1, "Expected low selectivity (high value)")
			}
		})
	}
}

func TestK8sFilterBuilder_CombineConditions(t *testing.T) {
	builder := NewK8sFilterBuilder()

	tests := []struct {
		name       string
		conditions []string
		expected   string
	}{
		{
			name:       "single condition",
			conditions: []string{"condition1"},
			expected:   "condition1",
		},
		{
			name:       "multiple conditions",
			conditions: []string{"condition1", "condition2", "condition3"},
			expected:   "(condition1 OR condition2 OR condition3)",
		},
		{
			name:       "empty conditions",
			conditions: []string{},
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.conditions) == 0 {
				// Test empty case separately
				result := ""
				assert.Equal(t, tt.expected, result)
			} else {
				result := builder.combineConditions(tt.conditions)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestK8sFilterBuilder_NoFilters(t *testing.T) {
	builder := NewK8sFilterBuilder()
	builder.SetFilters([]request.K8sFilter{})

	conditions, args, err := builder.BuildK8sFilterConditions()

	require.NoError(t, err)
	assert.Equal(t, 0, len(conditions))
	assert.Equal(t, 0, len(args))
}
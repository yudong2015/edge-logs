package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatasetValidator_ValidateDataset(t *testing.T) {
	validator := NewDatasetValidator()

	tests := []struct {
		name        string
		dataset     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid production dataset",
			dataset:     "prod-cluster-01",
			expectError: false,
		},
		{
			name:        "valid edge dataset",
			dataset:     "edge-cn-hz01",
			expectError: false,
		},
		{
			name:        "valid staging dataset",
			dataset:     "staging-app",
			expectError: false,
		},
		{
			name:        "valid default dataset",
			dataset:     "default",
			expectError: false,
		},
		{
			name:        "empty dataset",
			dataset:     "",
			expectError: true,
			errorMsg:    "dataset parameter is required",
		},
		{
			name:        "blocked dataset - system",
			dataset:     "system",
			expectError: true,
			errorMsg:    "reserved and cannot be accessed",
		},
		{
			name:        "blocked dataset - admin",
			dataset:     "admin",
			expectError: true,
			errorMsg:    "reserved and cannot be accessed",
		},
		{
			name:        "invalid format - special chars",
			dataset:     "prod@cluster",
			expectError: true,
			errorMsg:    "dataset format invalid",
		},
		{
			name:        "too long dataset name",
			dataset:     "this-is-a-very-long-dataset-name-that-exceeds-the-maximum-allowed-length-of-sixty-four-characters",
			expectError: true,
			errorMsg:    "dataset format invalid",
		},
		{
			name:        "SQL injection attempt",
			dataset:     "prod'; DROP TABLE logs; --",
			expectError: true,
			errorMsg:    "dataset format invalid",
		},
		{
			name:        "disallowed pattern",
			dataset:     "hacker-dataset",
			expectError: true,
			errorMsg:    "does not match allowed patterns",
		},
		{
			name:        "case insensitive blocked check",
			dataset:     "System",
			expectError: true,
			errorMsg:    "reserved and cannot be accessed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateDataset(tt.dataset)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDatasetValidator_SanitizeDataset(t *testing.T) {
	validator := NewDatasetValidator()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean dataset name",
			input:    "prod-cluster",
			expected: "prod-cluster",
		},
		{
			name:     "remove single quotes",
			input:    "prod'cluster",
			expected: "prodcluster",
		},
		{
			name:     "remove double quotes",
			input:    "prod\"cluster",
			expected: "prodcluster",
		},
		{
			name:     "remove SQL injection patterns",
			input:    "prod; DROP TABLE logs; --",
			expected: "prod DROP TABLE logs ",
		},
		{
			name:     "remove comment patterns",
			input:    "prod/*comment*/cluster",
			expected: "prodcommentcluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.SanitizeDataset(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDatasetValidator_IsValidDatasetFormat(t *testing.T) {
	validator := NewDatasetValidator()

	tests := []struct {
		name     string
		dataset  string
		expected bool
	}{
		{"valid alphanumeric", "prod123", true},
		{"valid with hyphens", "prod-cluster", true},
		{"valid with underscores", "prod_cluster", true},
		{"valid mixed", "prod-cluster_01", true},
		{"empty string", "", false},
		{"special characters", "prod@cluster", false},
		{"spaces", "prod cluster", false},
		{"too long", "a234567890123456789012345678901234567890123456789012345678901234567890", false},
		{"dots not allowed", "prod.cluster", false},
		{"slash not allowed", "prod/cluster", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.IsValidDatasetFormat(tt.dataset)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDatasetValidator_GetAllowedPatterns(t *testing.T) {
	validator := NewDatasetValidator()
	patterns := validator.GetAllowedPatterns()

	assert.Contains(t, patterns, "prod-*")
	assert.Contains(t, patterns, "staging-*")
	assert.Contains(t, patterns, "edge-*")
	assert.Contains(t, patterns, "default")

	// Test that modifying returned slice doesn't affect internal state
	patterns[0] = "modified"
	originalPatterns := validator.GetAllowedPatterns()
	assert.NotEqual(t, "modified", originalPatterns[0])
}

func TestDatasetValidator_GetBlockedDatasets(t *testing.T) {
	validator := NewDatasetValidator()
	blocked := validator.GetBlockedDatasets()

	assert.Contains(t, blocked, "system")
	assert.Contains(t, blocked, "admin")
	assert.Contains(t, blocked, "root")

	// Test that modifying returned slice doesn't affect internal state
	blocked[0] = "modified"
	originalBlocked := validator.GetBlockedDatasets()
	assert.NotEqual(t, "modified", originalBlocked[0])
}

func TestDatasetValidator_SQLInjectionDetection(t *testing.T) {
	validator := NewDatasetValidator()

	maliciousInputs := []string{
		"prod'; DROP TABLE logs; --",
		"prod\" OR 1=1",
		"prod UNION SELECT * FROM users",
		"prod/*comment*/",
		"prod')",
		"prod';--",
	}

	for _, input := range maliciousInputs {
		t.Run(input, func(t *testing.T) {
			result := validator.containsSQLInjection(input)
			assert.True(t, result, "Should detect SQL injection in: %s", input)
		})
	}

	safeInputs := []string{
		"prod-cluster",
		"edge_deployment",
		"staging123",
		"default",
		"test-env-001",
	}

	for _, input := range safeInputs {
		t.Run(input, func(t *testing.T) {
			result := validator.containsSQLInjection(input)
			assert.False(t, result, "Should not detect SQL injection in: %s", input)
		})
	}
}

func BenchmarkDatasetValidator_ValidateDataset(b *testing.B) {
	validator := NewDatasetValidator()
	testDataset := "prod-cluster-01"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateDataset(testDataset)
	}
}

func BenchmarkDatasetValidator_SanitizeDataset(b *testing.B) {
	validator := NewDatasetValidator()
	testDataset := "prod'; DROP TABLE logs; --"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.SanitizeDataset(testDataset)
	}
}
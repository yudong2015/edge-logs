package query

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/outpostos/edge-logs/pkg/model/request"
)

func TestK8sResourceValidator_ValidateNamespaceFormat(t *testing.T) {
	validator := NewK8sResourceValidator()

	tests := []struct {
		name        string
		namespace   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid namespace",
			namespace:   "production",
			expectError: false,
		},
		{
			name:        "valid namespace with hyphens",
			namespace:   "kube-system",
			expectError: false,
		},
		{
			name:        "valid namespace with numbers",
			namespace:   "env-123",
			expectError: false,
		},
		{
			name:        "invalid namespace with uppercase",
			namespace:   "Production",
			expectError: true,
			errorMsg:    "DNS-1123 compliant",
		},
		{
			name:        "invalid namespace starting with hyphen",
			namespace:   "-invalid",
			expectError: true,
			errorMsg:    "DNS-1123 compliant",
		},
		{
			name:        "invalid namespace ending with hyphen",
			namespace:   "invalid-",
			expectError: true,
			errorMsg:    "DNS-1123 compliant",
		},
		{
			name:        "empty namespace",
			namespace:   "",
			expectError: true,
			errorMsg:    "cannot be empty",
		},
		{
			name:        "too long namespace",
			namespace:   "this-is-a-very-long-namespace-name-that-exceeds-the-maximum-allowed-length-of-63-characters",
			expectError: true,
			errorMsg:    "too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateNamespaceFormat(tt.namespace)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestK8sResourceValidator_ValidatePodNameFormat(t *testing.T) {
	validator := NewK8sResourceValidator()

	tests := []struct {
		name        string
		podName     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid pod name",
			podName:     "web-app-123",
			expectError: false,
		},
		{
			name:        "valid pod name with dots",
			podName:     "app.service.com-123",
			expectError: false,
		},
		{
			name:        "valid pod name with uppercase",
			podName:     "Web-App-123",
			expectError: false,
		},
		{
			name:        "valid pod name with underscores",
			podName:     "web_app_123",
			expectError: false,
		},
		{
			name:        "invalid pod name starting with hyphen",
			podName:     "-invalid",
			expectError: true,
			errorMsg:    "naming conventions",
		},
		{
			name:        "invalid pod name ending with hyphen",
			podName:     "invalid-",
			expectError: true,
			errorMsg:    "naming conventions",
		},
		{
			name:        "empty pod name",
			podName:     "",
			expectError: true,
			errorMsg:    "cannot be empty",
		},
		{
			name:        "too long pod name",
			podName:     "this-is-a-very-long-pod-name-that-exceeds-the-maximum-allowed-length-of-253-characters-" + "abcdefghijklmnopqrstuvwxyz-abcdefghijklmnopqrstuvwxyz-abcdefghijklmnopqrstuvwxyz-abcdefghijklmnopqrstuvwxyz-abcdefghijklmnopqrstuvwxyz-abcdefghijklmnopqrstuvwxyz-abcdefghijklmnopqrstuvwxyz",
			expectError: true,
			errorMsg:    "too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validatePodNameFormat(tt.podName)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestK8sResourceValidator_ParseNamespaceFilter(t *testing.T) {
	validator := NewK8sResourceValidator()

	tests := []struct {
		name           string
		input          string
		expectedType   request.K8sFilterType
		expectedPattern string
		expectError    bool
		errorMsg       string
	}{
		{
			name:            "exact namespace",
			input:           "production",
			expectedType:    request.K8sFilterExact,
			expectedPattern: "production",
			expectError:     false,
		},
		{
			name:            "prefix namespace",
			input:           "kube-*",
			expectedType:    request.K8sFilterPrefix,
			expectedPattern: "kube-",
			expectError:     false,
		},
		{
			name:            "wildcard namespace",
			input:           "env-?-prod",
			expectedType:    request.K8sFilterWildcard,
			expectedPattern: "env-?-prod",
			expectError:     false,
		},
		{
			name:            "regex namespace",
			input:           "regex:^test-.*$",
			expectedType:    request.K8sFilterRegex,
			expectedPattern: "^test-.*$",
			expectError:     false,
		},
		{
			name:        "invalid regex namespace",
			input:       "regex:[invalid",
			expectError: true,
			errorMsg:    "invalid regex pattern",
		},
		{
			name:        "invalid namespace format",
			input:       "Invalid-Namespace",
			expectError: true,
			errorMsg:    "invalid namespace name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := validator.parseNamespaceFilter(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedType, filter.Type)
				assert.Equal(t, tt.expectedPattern, filter.Pattern)
				assert.Equal(t, "namespace", filter.Field)
			}
		})
	}
}

func TestK8sResourceValidator_ParsePodFilter(t *testing.T) {
	validator := NewK8sResourceValidator()

	tests := []struct {
		name            string
		input           string
		expectedType    request.K8sFilterType
		expectedPattern string
		caseInsensitive bool
		expectError     bool
		errorMsg        string
	}{
		{
			name:            "exact pod name",
			input:           "web-app-123",
			expectedType:    request.K8sFilterExact,
			expectedPattern: "web-app-123",
			caseInsensitive: false,
			expectError:     false,
		},
		{
			name:            "prefix pod name",
			input:           "api-*",
			expectedType:    request.K8sFilterPrefix,
			expectedPattern: "api-",
			caseInsensitive: false,
			expectError:     false,
		},
		{
			name:            "suffix pod name",
			input:           "*-worker",
			expectedType:    request.K8sFilterSuffix,
			expectedPattern: "-worker",
			caseInsensitive: false,
			expectError:     false,
		},
		{
			name:            "contains pod name",
			input:           "*database*",
			expectedType:    request.K8sFilterContains,
			expectedPattern: "database",
			caseInsensitive: false,
			expectError:     false,
		},
		{
			name:            "wildcard pod name",
			input:           "web-??-prod",
			expectedType:    request.K8sFilterWildcard,
			expectedPattern: "web-??-prod",
			caseInsensitive: false,
			expectError:     false,
		},
		{
			name:            "regex pod name",
			input:           "regex:^app-[0-9]+$",
			expectedType:    request.K8sFilterRegex,
			expectedPattern: "^app-[0-9]+$",
			caseInsensitive: false,
			expectError:     false,
		},
		{
			name:            "case insensitive prefix",
			input:           "icase:WEB-*",
			expectedType:    request.K8sFilterPrefix,
			expectedPattern: "WEB-",
			caseInsensitive: true,
			expectError:     false,
		},
		{
			name:            "case insensitive regex",
			input:           "icase:regex:^APP-.*$",
			expectedType:    request.K8sFilterRegex,
			expectedPattern: "^APP-.*$",
			caseInsensitive: true,
			expectError:     false,
		},
		{
			name:        "invalid regex pod name",
			input:       "regex:[invalid",
			expectError: true,
			errorMsg:    "invalid regex pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := validator.parsePodFilter(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedType, filter.Type)
				assert.Equal(t, tt.expectedPattern, filter.Pattern)
				assert.Equal(t, "pod", filter.Field)
				assert.Equal(t, tt.caseInsensitive, filter.CaseInsensitive)
			}
		})
	}
}

func TestK8sResourceValidator_ParseK8sFilters(t *testing.T) {
	validator := NewK8sResourceValidator()

	tests := []struct {
		name          string
		namespaces    []string
		pods          []string
		expectedCount int
		expectError   bool
		errorMsg      string
	}{
		{
			name:          "valid mixed filters",
			namespaces:    []string{"production", "kube-*"},
			pods:          []string{"web-*", "api-service-123"},
			expectedCount: 4,
			expectError:   false,
		},
		{
			name:          "empty filters",
			namespaces:    []string{},
			pods:          []string{},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "filters with empty strings",
			namespaces:    []string{"production", "", "staging"},
			pods:          []string{"", "web-*"},
			expectedCount: 3,
			expectError:   false,
		},
		{
			name:        "invalid namespace filter",
			namespaces:  []string{"Invalid-Namespace"},
			pods:        []string{},
			expectError: true,
			errorMsg:    "invalid namespace filter",
		},
		{
			name:        "invalid pod filter",
			namespaces:  []string{},
			pods:        []string{"regex:[invalid"},
			expectError: true,
			errorMsg:    "invalid pod filter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters, err := validator.ParseK8sFilters(tt.namespaces, tt.pods)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(filters))
			}
		})
	}
}

func TestK8sResourceValidator_ValidateRegexPattern(t *testing.T) {
	validator := NewK8sResourceValidator()

	tests := []struct {
		name        string
		pattern     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid simple regex",
			pattern:     "^test-.*$",
			expectError: false,
		},
		{
			name:        "valid complex regex",
			pattern:     "^app-[0-9]+(-[a-z]+)?$",
			expectError: false,
		},
		{
			name:        "invalid regex syntax",
			pattern:     "[invalid",
			expectError: true,
			errorMsg:    "invalid regex pattern",
		},
		{
			name:        "expensive regex pattern",
			pattern:     ".*.*test.*.*",
			expectError: true,
			errorMsg:    "too expensive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateRegexPattern(tt.pattern)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestK8sResourceValidator_FilterComplexityValidation(t *testing.T) {
	validator := NewK8sResourceValidator()

	// Test maximum filter count validation
	namespaces := make([]string, 51)
	for i := 0; i < 51; i++ {
		namespaces[i] = fmt.Sprintf("ns-%d", i)
	}

	filters, err := validator.ParseK8sFilters(namespaces, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many K8s filters")
	assert.Nil(t, filters)
}

func TestK8sResourceValidator_ValidateWildcardPattern(t *testing.T) {
	validator := NewK8sResourceValidator()

	tests := []struct {
		name        string
		pattern     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid wildcard pattern",
			pattern:     "web-*-prod",
			expectError: false,
		},
		{
			name:        "valid single character wildcard",
			pattern:     "app-?-env",
			expectError: false,
		},
		{
			name:        "too many wildcards",
			pattern:     "*-*-*-*-*-*",
			expectError: true,
			errorMsg:    "too many wildcards",
		},
		{
			name:        "too many single character wildcards",
			pattern:     "???????????",
			expectError: true,
			errorMsg:    "too many single-character wildcards",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateWildcardPattern(tt.pattern)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
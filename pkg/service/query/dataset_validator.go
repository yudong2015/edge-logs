package query

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// DatasetValidator provides dataset validation and security features
type DatasetValidator struct {
	allowedPatterns []string
	blockedDatasets []string
	regex          *regexp.Regexp
}

// NewDatasetValidator creates a new dataset validator with security rules
func NewDatasetValidator() *DatasetValidator {
	return &DatasetValidator{
		// Allow alphanumeric, hyphens, underscores, max 64 chars
		regex: regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`),
		allowedPatterns: []string{
			"prod-*",
			"staging-*",
			"dev-*",
			"edge-*",
			"test-*",
			"default",
		},
		blockedDatasets: []string{
			"system",
			"internal",
			"admin",
			"root",
			"clickhouse",
		},
	}
}

// ValidateDataset performs comprehensive dataset validation
func (v *DatasetValidator) ValidateDataset(dataset string) error {
	// 1. Basic format validation
	if dataset == "" {
		return fmt.Errorf("dataset parameter is required")
	}

	if !v.regex.MatchString(dataset) {
		return fmt.Errorf("dataset format invalid: must be alphanumeric with hyphens/underscores, max 64 chars")
	}

	// 2. Blocked dataset check
	for _, blocked := range v.blockedDatasets {
		if strings.EqualFold(dataset, blocked) {
			return fmt.Errorf("dataset '%s' is reserved and cannot be accessed", dataset)
		}
	}

	// 3. Pattern allowlist check (if configured)
	if len(v.allowedPatterns) > 0 {
		allowed := false
		for _, pattern := range v.allowedPatterns {
			if matched, _ := filepath.Match(pattern, dataset); matched {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("dataset '%s' does not match allowed patterns", dataset)
		}
	}

	// 4. SQL injection prevention
	if v.containsSQLInjection(dataset) {
		return fmt.Errorf("dataset name contains invalid characters")
	}

	return nil
}

// SanitizeDataset sanitizes dataset name for safe use in queries
func (v *DatasetValidator) SanitizeDataset(dataset string) string {
	// Remove any potentially dangerous characters
	sanitized := strings.ReplaceAll(dataset, "'", "")
	sanitized = strings.ReplaceAll(sanitized, "\"", "")
	sanitized = strings.ReplaceAll(sanitized, ";", "")
	sanitized = strings.ReplaceAll(sanitized, "--", "")
	sanitized = strings.ReplaceAll(sanitized, "/*", "")
	sanitized = strings.ReplaceAll(sanitized, "*/", "")

	return sanitized
}

// IsValidDatasetFormat checks if dataset name has valid format
func (v *DatasetValidator) IsValidDatasetFormat(dataset string) bool {
	return v.regex.MatchString(dataset)
}

// containsSQLInjection performs SQL injection detection
func (v *DatasetValidator) containsSQLInjection(input string) bool {
	// SQL injection patterns specific to dataset names
	dangerousPatterns := []string{
		"'", "\"", ";", "--", "/*", "*/",
		"union", "select", "drop", "delete", "update", "insert",
		"or 1=1", "and 1=1", "' or", "\" or",
	}

	inputLower := strings.ToLower(input)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(inputLower, pattern) {
			return true
		}
	}
	return false
}

// GetAllowedPatterns returns the allowed dataset patterns for documentation
func (v *DatasetValidator) GetAllowedPatterns() []string {
	return append([]string{}, v.allowedPatterns...)
}

// GetBlockedDatasets returns the blocked dataset names for documentation
func (v *DatasetValidator) GetBlockedDatasets() []string {
	return append([]string{}, v.blockedDatasets...)
}
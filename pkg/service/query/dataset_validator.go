package query

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"k8s.io/klog/v2"
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
			// K8s system namespaces
			"kube-*",
			"istio-*",
			"linkerd-*",
			"calico-*",
			"tigera-*",
			// Edge system namespaces
			"edge-*",
			"logging-*",
			"monitoring-*",
			"traefik*",  // Changed from traefik-* to match both 'traefik' and 'traefik-*'
			"ingress-*",
			// Standard namespaces
			"prod-*",
			"staging-*",
			"dev-*",
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
	klog.InfoS("[DatasetValidator] 开始验证数据集 [DEBUG]",
		"dataset", dataset,
		"validator_type", "pattern_and_security")

	// 1. Basic format validation
	if dataset == "" {
		klog.ErrorS(fmt.Errorf("dataset parameter is required"), "[DEBUG] validation_failed", "empty_dataset")
		return fmt.Errorf("dataset parameter is required")
	}

	klog.V(2).InfoS("[DatasetValidator] 格式检查 [DEBUG]",
		"dataset", dataset,
		"regex", "^[a-zA-Z0-9_-]{1,64}$")

	if !v.regex.MatchString(dataset) {
		klog.ErrorS(fmt.Errorf("dataset format invalid"), "[DEBUG] validation_failed",
			"dataset", dataset,
			"reason", "format_mismatch")
		return fmt.Errorf("dataset format invalid: must be alphanumeric with hyphens/underscores, max 64 chars")
	}

	klog.V(2).InfoS("[DatasetValidator] 格式检查通过 [DEBUG]", "dataset", dataset)

	// 2. Blocked dataset check
	for _, blocked := range v.blockedDatasets {
		if strings.EqualFold(dataset, blocked) {
			klog.ErrorS(fmt.Errorf("dataset '%s' is reserved", dataset), "[DEBUG] validation_failed",
				"dataset", dataset,
				"blocked_by", blocked)
			return fmt.Errorf("dataset '%s' is reserved and cannot be accessed", dataset)
		}
	}

	klog.V(2).InfoS("[DatasetValidator] 阻塞列表检查通过 [DEBUG]", "dataset", dataset)

	// 3. Pattern allowlist check (if configured)
	klog.V(2).InfoS("[DatasetValidator] 开始模式匹配检查 [DEBUG]",
		"dataset", dataset,
		"allowed_patterns_count", len(v.allowedPatterns))

	if len(v.allowedPatterns) > 0 {
		allowed := false
		matchedPatterns := []string{}
		for _, pattern := range v.allowedPatterns {
			if matched, _ := filepath.Match(pattern, dataset); matched {
				allowed = true
				matchedPatterns = append(matchedPatterns, pattern)
				break
			}
		}

		klog.InfoS("[DatasetValidator] 模式匹配结果 [DEBUG]",
			"dataset", dataset,
			"allowed", allowed,
			"matched_patterns", matchedPatterns)

		if !allowed {
			klog.ErrorS(fmt.Errorf("dataset '%s' does not match allowed patterns", dataset), "[DEBUG] validation_failed",
				"dataset", dataset,
				"available_patterns", v.allowedPatterns)
			return fmt.Errorf("dataset '%s' does not match allowed patterns", dataset)
		}
	}

	// 4. SQL injection prevention
	if v.containsSQLInjection(dataset) {
		klog.ErrorS(fmt.Errorf("dataset name contains invalid characters"), "[DEBUG] validation_failed",
			"dataset", dataset,
			"reason", "sql_injection_detected")
		return fmt.Errorf("dataset name contains invalid characters")
	}

	klog.InfoS("[DatasetValidator] 数据集验证成功 [DEBUG]",
		"dataset", dataset,
		"all_checks_passed", true)

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
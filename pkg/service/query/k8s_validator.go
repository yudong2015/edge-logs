package query

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/outpostos/edge-logs/pkg/model/request"
)

// K8sResourceValidator validates K8s resource names and patterns with DNS-1123 compliance
type K8sResourceValidator struct {
	namespaceRegex   *regexp.Regexp
	podNameRegex     *regexp.Regexp
	maxFilterCount   int
	maxPatternLength int
}

// NewK8sResourceValidator creates a new K8s resource validator with DNS-1123 compliance
func NewK8sResourceValidator() *K8sResourceValidator {
	return &K8sResourceValidator{
		// DNS-1123 compliant validation for namespaces
		namespaceRegex: regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`),
		// More permissive for pod names (allows uppercase, dots)
		podNameRegex: regexp.MustCompile(`^[a-z0-9A-Z]([-a-z0-9A-Z._]*[a-z0-9A-Z])?$`),
		maxFilterCount:   50,  // Prevent overly complex queries
		maxPatternLength: 255, // Prevent extremely long patterns
	}
}

// ParseK8sFilters validates and parses K8s filtering parameters
func (v *K8sResourceValidator) ParseK8sFilters(namespaces, pods []string) ([]request.K8sFilter, error) {
	var filters []request.K8sFilter

	// Parse namespace filters
	for _, ns := range namespaces {
		if ns == "" {
			continue
		}

		filter, err := v.parseNamespaceFilter(ns)
		if err != nil {
			return nil, fmt.Errorf("invalid namespace filter '%s': %w", ns, err)
		}
		filters = append(filters, filter)
	}

	// Parse pod name filters
	for _, pod := range pods {
		if pod == "" {
			continue
		}

		filter, err := v.parsePodFilter(pod)
		if err != nil {
			return nil, fmt.Errorf("invalid pod filter '%s': %w", pod, err)
		}
		filters = append(filters, filter)
	}

	// Validate total filter count
	if len(filters) > v.maxFilterCount {
		return nil, fmt.Errorf("too many K8s filters (%d), maximum allowed: %d",
			len(filters), v.maxFilterCount)
	}

	return filters, nil
}

// parseNamespaceFilter parses and validates namespace filter patterns
func (v *K8sResourceValidator) parseNamespaceFilter(namespace string) (request.K8sFilter, error) {
	if len(namespace) > v.maxPatternLength {
		return request.K8sFilter{}, fmt.Errorf("namespace pattern too long (%d chars), max: %d",
			len(namespace), v.maxPatternLength)
	}

	// Detect filter type based on pattern
	filter := request.K8sFilter{Field: "namespace"}

	switch {
	case strings.HasPrefix(namespace, "regex:"):
		// Regex pattern: regex:^kube-.*
		filter.Type = request.K8sFilterRegex
		filter.Pattern = strings.TrimPrefix(namespace, "regex:")
		if err := v.validateRegexPattern(filter.Pattern); err != nil {
			return filter, fmt.Errorf("invalid regex pattern: %w", err)
		}

	case strings.Contains(namespace, "*") || strings.Contains(namespace, "?"):
		// Wildcard pattern: kube-* or test-?-env
		filter.Type = request.K8sFilterWildcard
		filter.Pattern = namespace
		if err := v.validateWildcardPattern(filter.Pattern); err != nil {
			return filter, fmt.Errorf("invalid wildcard pattern: %w", err)
		}

	case strings.HasSuffix(namespace, "*"):
		// Prefix pattern: kube-*
		filter.Type = request.K8sFilterPrefix
		filter.Pattern = strings.TrimSuffix(namespace, "*")
		if err := v.validateNamespaceFormat(filter.Pattern); err != nil {
			return filter, fmt.Errorf("invalid prefix pattern: %w", err)
		}

	default:
		// Exact match
		filter.Type = request.K8sFilterExact
		filter.Pattern = namespace
		if err := v.validateNamespaceFormat(filter.Pattern); err != nil {
			return filter, fmt.Errorf("invalid namespace name: %w", err)
		}
	}

	return filter, nil
}

// parsePodFilter parses and validates pod name filter patterns
func (v *K8sResourceValidator) parsePodFilter(podName string) (request.K8sFilter, error) {
	if len(podName) > v.maxPatternLength {
		return request.K8sFilter{}, fmt.Errorf("pod pattern too long (%d chars), max: %d",
			len(podName), v.maxPatternLength)
	}

	filter := request.K8sFilter{Field: "pod"}

	// Check for case-insensitive prefix
	if strings.HasPrefix(strings.ToLower(podName), "icase:") {
		filter.CaseInsensitive = true
		podName = podName[6:] // Remove "icase:" prefix
	}

	switch {
	case strings.HasPrefix(podName, "regex:"):
		// Regex pattern: regex:^app-.*-[0-9]+$
		filter.Type = request.K8sFilterRegex
		filter.Pattern = strings.TrimPrefix(podName, "regex:")
		if err := v.validateRegexPattern(filter.Pattern); err != nil {
			return filter, fmt.Errorf("invalid regex pattern: %w", err)
		}

	case strings.Contains(podName, "*") || strings.Contains(podName, "?"):
		// Wildcard pattern: app-* or web-??-prod
		filter.Type = request.K8sFilterWildcard
		filter.Pattern = podName

	case strings.HasPrefix(podName, "*") && strings.HasSuffix(podName, "*"):
		// Contains pattern: *web-server*
		filter.Type = request.K8sFilterContains
		filter.Pattern = strings.Trim(podName, "*")
		if err := v.validatePodNameFormat(filter.Pattern); err != nil {
			return filter, fmt.Errorf("invalid contains pattern: %w", err)
		}

	case strings.HasSuffix(podName, "*"):
		// Prefix pattern: web-app-*
		filter.Type = request.K8sFilterPrefix
		filter.Pattern = strings.TrimSuffix(podName, "*")
		if err := v.validatePodNameFormat(filter.Pattern); err != nil {
			return filter, fmt.Errorf("invalid prefix pattern: %w", err)
		}

	case strings.HasPrefix(podName, "*"):
		// Suffix pattern: *-worker
		filter.Type = request.K8sFilterSuffix
		filter.Pattern = strings.TrimPrefix(podName, "*")
		if err := v.validatePodNameFormat(filter.Pattern); err != nil {
			return filter, fmt.Errorf("invalid suffix pattern: %w", err)
		}

	default:
		// Exact match
		filter.Type = request.K8sFilterExact
		filter.Pattern = podName
		if err := v.validatePodNameFormat(filter.Pattern); err != nil {
			return filter, fmt.Errorf("invalid pod name: %w", err)
		}
	}

	return filter, nil
}

// validateNamespaceFormat ensures namespace follows DNS-1123 rules
func (v *K8sResourceValidator) validateNamespaceFormat(namespace string) error {
	if len(namespace) == 0 {
		return fmt.Errorf("namespace cannot be empty")
	}
	if len(namespace) > 63 {
		return fmt.Errorf("namespace too long (%d chars), max 63", len(namespace))
	}
	if !v.namespaceRegex.MatchString(namespace) {
		return fmt.Errorf("namespace format invalid: must be DNS-1123 compliant (lowercase alphanumeric and hyphens)")
	}
	return nil
}

// validatePodNameFormat ensures pod name follows K8s naming rules
func (v *K8sResourceValidator) validatePodNameFormat(podName string) error {
	if len(podName) == 0 {
		return fmt.Errorf("pod name cannot be empty")
	}
	if len(podName) > 253 {
		return fmt.Errorf("pod name too long (%d chars), max 253", len(podName))
	}
	if !v.podNameRegex.MatchString(podName) {
		return fmt.Errorf("pod name format invalid: must follow K8s naming conventions")
	}
	return nil
}

// validateRegexPattern ensures regex patterns are safe and valid
func (v *K8sResourceValidator) validateRegexPattern(pattern string) error {
	_, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Additional safety checks for potentially expensive regex patterns
	if strings.Contains(pattern, ".*.*") || strings.Contains(pattern, ".+.+") {
		return fmt.Errorf("regex pattern may be too expensive (multiple greedy quantifiers)")
	}

	return nil
}

// validateWildcardPattern ensures wildcard patterns are reasonable
func (v *K8sResourceValidator) validateWildcardPattern(pattern string) error {
	// Basic validation for wildcard patterns
	if strings.Count(pattern, "*") > 5 {
		return fmt.Errorf("too many wildcards in pattern (max 5)")
	}
	if strings.Count(pattern, "?") > 10 {
		return fmt.Errorf("too many single-character wildcards in pattern (max 10)")
	}
	return nil
}
package optimization

import (
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/outpostos/edge-logs/pkg/model/request"
)

// PaginationManager provides advanced pagination and memory management for queries
type PaginationManager struct {
	defaultPageSize      int
	maxPageSize          int
	maxResultSize        int64 // Maximum total result size in bytes
	enableStreaming      bool
	streamingChunkSize   int
}

// NewPaginationManager creates a new pagination manager
func NewPaginationManager() *PaginationManager {
	return &PaginationManager{
		defaultPageSize:    100,
		maxPageSize:        10000,
		maxResultSize:      100 * 1024 * 1024, // 100MB default limit
		enableStreaming:    true,
		streamingChunkSize: 1000,
	}
}

// PaginatedQueryResult contains paginated query results with metadata
type PaginatedQueryResult struct {
	Logs          interface{} `json:"logs"`
	TotalCount    int64       `json:"total_count"`
	Page          int         `json:"page"`
	PageSize      int         `json:"page_size"`
	TotalPages    int         `json:"total_pages"`
	HasMore       bool        `json:"has_more"`
	MemoryUsage   int64       `json:"memory_usage_bytes"`
	QueryTime     time.Duration `json:"query_time_ms"`
	ExecutionPlan string      `json:"execution_plan"`
}

// ValidateAndAdjustPagination validates and adjusts pagination parameters
func (pm *PaginationManager) ValidateAndAdjustPagination(req *request.LogQueryRequest) error {
	// Validate page number
	if req.Page < 1 {
		req.Page = 1
		klog.InfoS("页码已调整为默认值",
			"original_page", req.Page,
			"adjusted_page", 1)
	}

	// Validate page size
	if req.PageSize <= 0 {
		req.PageSize = pm.defaultPageSize
		klog.InfoS("页面大小已设置为默认值",
			"dataset", req.Dataset,
			"page_size", pm.defaultPageSize)
	}

	// Enforce maximum page size
	if req.PageSize > pm.maxPageSize {
		klog.InfoS("页面大小超过最大值，已调整",
			"dataset", req.Dataset,
			"requested_size", req.PageSize,
			"max_size", pm.maxPageSize,
			"adjusted_size", pm.maxPageSize)
		req.PageSize = pm.maxPageSize
	}

	// Validate offset doesn't exceed reasonable limits
	maxOffset := 100000 // Don't allow pagination beyond 100k records
	offset := (req.Page - 1) * req.PageSize
	if offset > maxOffset {
		return fmt.Errorf("分页偏移量超过最大限制: %d (最大: %d)", offset, maxOffset)
	}

	return nil
}

// PaginationMetadata contains pagination information
// NOTE: This is a simplified version for the optimization package
type PaginationMetadata struct {
	TotalCount int64 `json:"total_count"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
	HasMore    bool  `json:"has_more"`
}

// CalculatePaginationMetadata calculates pagination metadata
func (pm *PaginationManager) CalculatePaginationMetadata(
	totalCount int64,
	page int,
	pageSize int,
) *PaginationMetadata {
	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize > 0 {
		totalPages++
	}

	// Ensure totalPages is at least 1
	if totalPages < 1 {
		totalPages = 1
	}

	// Ensure page is within valid range
	if page > totalPages {
		page = totalPages
	}

	hasMore := page < totalPages

	return &PaginationMetadata{
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		HasMore:    hasMore,
	}
}

// EstimateResultSize estimates the memory size of query results
func (pm *PaginationManager) EstimateResultSize(logCount int, avgLogSize int) int64 {
	// Average log entry size estimation:
	// - timestamp: 8 bytes
	// - namespace: 20 bytes (average)
	// - pod_name: 30 bytes (average)
	// - container_name: 30 bytes (average)
	// - host_name: 30 bytes (average)
	// - message: 200 bytes (average)
	// - severity: 10 bytes (average)
	// - dataset: 20 bytes (average)
	// - metadata fields: 50 bytes (average)
	// Total average: ~400 bytes per log entry

	if avgLogSize <= 0 {
		avgLogSize = 400 // Default average size
	}

	estimatedSize := int64(logCount * avgLogSize)

	klog.V(4).InfoS("结果集大小估算",
		"log_count", logCount,
		"avg_log_size", avgLogSize,
		"estimated_size_bytes", estimatedSize,
		"estimated_size_mb", estimatedSize/(1024*1024))

	return estimatedSize
}

// CheckMemoryLimits checks if result size exceeds memory limits
func (pm *PaginationManager) CheckMemoryLimits(estimatedSize int64, dataset string) error {
	if estimatedSize > pm.maxResultSize {
		return fmt.Errorf("估算结果大小超过内存限制: %d MB (最大: %d MB)",
			estimatedSize/(1024*1024),
			pm.maxResultSize/(1024*1024))
	}

	// Warning if approaching limit
	warningThreshold := pm.maxResultSize * 80 / 100 // 80% of limit
	if estimatedSize > warningThreshold {
		klog.InfoS("结果大小接近内存限制",
			"dataset", dataset,
			"estimated_size_mb", estimatedSize/(1024*1024),
			"warning_threshold_mb", warningThreshold/(1024*1024),
			"max_limit_mb", pm.maxResultSize/(1024*1024))
	}

	return nil
}

// OptimizeForMemory optimizes query parameters to stay within memory limits
func (pm *PaginationManager) OptimizeForMemory(
	req *request.LogQueryRequest,
	estimatedTotalRecords int64,
) (*request.LogQueryRequest, []string) {
	optimizations := []string{}
	optimizedReq := *req // Create a copy

	// Estimate memory usage for current page size
	estimatedSize := pm.EstimateResultSize(int(optimizedReq.PageSize), 400)

	// If estimated size exceeds limits, reduce page size
	if estimatedSize > pm.maxResultSize {
		// Calculate safe page size
		safePageSize := int(pm.maxResultSize / 400) // 400 bytes per record
		if safePageSize < 10 {
			safePageSize = 10 // Minimum page size
		}

		oldPageSize := optimizedReq.PageSize
		optimizedReq.PageSize = safePageSize

		optimizations = append(optimizations,
			fmt.Sprintf("页面大小已减少以控制内存使用: %d -> %d", oldPageSize, safePageSize))

		klog.InfoS("页面大小已优化以符合内存限制",
			"dataset", optimizedReq.Dataset,
			"original_page_size", oldPageSize,
			"optimized_page_size", safePageSize,
			"estimated_size_mb", estimatedSize/(1024*1024),
			"max_limit_mb", pm.maxResultSize/(1024*1024))
	}

	// Suggest time range reduction if total records are very high
	if estimatedTotalRecords > 1000000 {
		optimizations = append(optimizations,
			"考虑缩小时间范围以减少结果集大小")
		klog.InfoS("查询结果数量过大，建议缩小时间范围",
			"dataset", optimizedReq.Dataset,
			"estimated_total_records", estimatedTotalRecords)
	}

	return &optimizedReq, optimizations
}

// BuildPaginationInfo builds comprehensive pagination information
func (pm *PaginationManager) BuildPaginationInfo(
	req *request.LogQueryRequest,
	totalCount int64,
	executionTime time.Duration,
	memoryUsage int64,
) *PaginatedQueryResult {
	paginationMeta := pm.CalculatePaginationMetadata(totalCount, req.Page, req.PageSize)

	return &PaginatedQueryResult{
		TotalCount:  totalCount,
		Page:        paginationMeta.Page,
		PageSize:    paginationMeta.PageSize,
		TotalPages:  paginationMeta.TotalPages,
		HasMore:     paginationMeta.HasMore,
		MemoryUsage: memoryUsage,
		QueryTime:   executionTime,
		Logs:        nil, // Will be populated by caller
	}
}

// GetDefaultPageSize returns the default page size
func (pm *PaginationManager) GetDefaultPageSize() int {
	return pm.defaultPageSize
}

// GetMaxPageSize returns the maximum page size
func (pm *PaginationManager) GetMaxPageSize() int {
	return pm.maxPageSize
}

// SetMaxResultSize updates the maximum result size limit
func (pm *PaginationManager) SetMaxResultSize(maxSize int64) {
	pm.maxResultSize = maxSize
	klog.InfoS("最大结果大小限制已更新",
		"max_size_mb", maxSize/(1024*1024))
}

// EnableStreaming enables or disables result streaming
func (pm *PaginationManager) EnableStreaming(enabled bool) {
	pm.enableStreaming = enabled
	klog.InfoS("结果流式传输已更新",
		"enabled", enabled)
}

// SetStreamingChunkSize sets the chunk size for streaming results
func (pm *PaginationManager) SetStreamingChunkSize(chunkSize int) {
	pm.streamingChunkSize = chunkSize
	klog.InfoS("流式传输块大小已更新",
		"chunk_size", chunkSize)
}

// ValidatePageRequest validates a page request for potential abuse
func (pm *PaginationManager) ValidatePageRequest(page, pageSize int) error {
	// Check for unreasonably large page numbers
	if page < 1 {
		return fmt.Errorf("页码必须大于 0: %d", page)
	}

	// Check for page size abuse (too large or too small)
	if pageSize < 1 {
		return fmt.Errorf("页面大小必须大于 0: %d", pageSize)
	}

	if pageSize > pm.maxPageSize {
		return fmt.Errorf("页面大小超过最大限制: %d (最大: %d)", pageSize, pm.maxPageSize)
	}

	// Check for offset abuse (deep pagination)
	maxOffset := 100000
	offset := (page - 1) * pageSize
	if offset > maxOffset {
		return fmt.Errorf("分页偏移量过大: %d (最大: %d)", offset, maxOffset)
	}

	return nil
}

// CalculateOptimalPageSize calculates the optimal page size based on memory constraints
func (pm *PaginationManager) CalculateOptimalPageSize(avgRecordSize int, targetMemoryMB int) int {
	targetMemoryBytes := int64(targetMemoryMB) * 1024 * 1024
	if targetMemoryBytes <= 0 {
		targetMemoryBytes = 10 * 1024 * 1024 // Default 10MB target
	}

	if avgRecordSize <= 0 {
		avgRecordSize = 400 // Default average record size
	}

	optimalSize := int(targetMemoryBytes / int64(avgRecordSize))

	// Apply constraints
	if optimalSize < 10 {
		optimalSize = 10 // Minimum page size
	}
	if optimalSize > pm.maxPageSize {
		optimalSize = pm.maxPageSize // Respect maximum
	}

	klog.V(4).InfoS("计算最佳页面大小",
		"avg_record_size", avgRecordSize,
		"target_memory_mb", targetMemoryMB,
		"optimal_page_size", optimalSize)

	return optimalSize
}
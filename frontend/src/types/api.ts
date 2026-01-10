/**
 * API response types for edge-logs backend integration
 * These types match the backend API response structures from Epic 1
 */

/**
 * Log entry structure matching ClickHouse schema and API responses
 */
export interface LogEntry {
  timestamp: string;           // ISO timestamp with milliseconds
  dataset: string;             // Dataset identifier (cluster/environment)
  namespace: string;           // Kubernetes namespace
  pod_name: string;           // Kubernetes pod name
  container_name: string;     // Container name within pod
  content: string;            // Log message content
  severity: string;           // Log severity level (info, warning, error, etc.)
  host_name?: string;         // Optional host name
  tags?: Record<string, string>; // Optional tags/metadata
}

/**
 * Log query parameters matching backend API endpoint
 */
export interface LogQueryParams {
  dataset: string;
  startTime: string;          // ISO timestamp with milliseconds
  endTime: string;            // ISO timestamp with milliseconds
  namespace?: string;
  podName?: string;
  containerName?: string;
  filter?: string;            // Content search filter
  severity?: string;          // Severity filter
  limit?: number;             // Result limit (default: 100, max: 1000)
  offset?: number;            // Pagination offset
}

/**
 * API response structure for log queries
 */
export interface LogQueryResponse {
  logs: LogEntry[];
  totalCount: number;
  page?: number;
  limit?: number;
  executionTime?: number;     // Query execution time in milliseconds
}

/**
 * Dataset information for navigation
 */
export interface Dataset {
  name: string;
  description?: string;
  cluster?: string;
  environment?: string;
}

/**
 * API error response structure
 */
export interface ApiError {
  message: string;
  code?: string;
  details?: string;
  timestamp?: string;
}

/**
 * Time range preset for quick selection
 */
export interface TimeRangePreset {
  label: string;
  value: string;
  subtract: { value: number; unit: 'seconds' | 'minutes' | 'hours' | 'days' };
}

/**
 * Severity level with color mapping
 */
export interface SeverityLevel {
  value: string;
  label: string;
  color: string;
  priority: number;
}

/**
 * Suggestion response for auto-complete fields
 */
export interface SuggestionResponse {
  values: string[];
  total: number;
}

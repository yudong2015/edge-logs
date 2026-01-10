/**
 * Log Query Service
 * Handles all API interactions with the edge-logs backend
 */

import type { LogQueryParams, LogQueryResponse, ApiError, Dataset } from '@/types/api'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'
const API_TIMEOUT = parseInt(import.meta.env.VITE_API_TIMEOUT || '10000')

/**
 * Custom error class for API errors
 */
class ApiServiceError extends Error {
  constructor(
    public statusCode: number,
    public apiError: ApiError,
    message?: string
  ) {
    super(message || apiError.message || 'API request failed')
    this.name = 'ApiServiceError'
  }
}

/**
 * Log Query Service Class
 */
export class LogQueryService {
  private baseUrl: string
  private timeout: number

  constructor(baseUrl: string = API_BASE_URL, timeout: number = API_TIMEOUT) {
    this.baseUrl = baseUrl
    this.timeout = timeout
  }

  /**
   * Execute log query with the given parameters
   */
  async queryLogs(params: LogQueryParams): Promise<LogQueryResponse> {
    const url = this.buildQueryUrl(params)
    return this.fetchWithTimeout<LogQueryResponse>(url, {
      method: 'GET',
      headers: this.getHeaders(),
    })
  }

  /**
   * Get available datasets
   */
  async getDatasets(): Promise<Dataset[]> {
    const url = `${this.baseUrl}/apis/log.theriseunion.io/v1alpha1/datasets`
    return this.fetchWithTimeout<Dataset[]>(url, {
      method: 'GET',
      headers: this.getHeaders(),
    })
  }

  /**
   * Health check for backend availability
   */
  async healthCheck(): Promise<{ status: string; timestamp: string }> {
    const url = `${this.baseUrl}/healthz`
    return this.fetchWithTimeout<{ status: string; timestamp: string }>(url, {
      method: 'GET',
      headers: this.getHeaders(),
    })
  }

  /**
   * Build query URL from parameters
   */
  private buildQueryUrl(params: LogQueryParams): string {
    const baseUrl = `${this.baseUrl}/apis/log.theriseunion.io/v1alpha1/logdatasets/${params.dataset}/logs`

    const queryParams = new URLSearchParams()

    if (params.startTime) queryParams.append('start_time', params.startTime)
    if (params.endTime) queryParams.append('end_time', params.endTime)
    if (params.namespace) queryParams.append('namespace', params.namespace)
    if (params.podName) queryParams.append('pod_name', params.podName)
    if (params.containerName) queryParams.append('container_name', params.containerName)
    if (params.filter) queryParams.append('filter', params.filter)
    if (params.severity) queryParams.append('severity', params.severity)
    if (params.limit) queryParams.append('limit', params.limit.toString())
    if (params.offset) queryParams.append('offset', params.offset.toString())

    const queryString = queryParams.toString()
    return queryString ? `${baseUrl}?${queryString}` : baseUrl
  }

  /**
   * Get default headers for API requests
   */
  private getHeaders(): HeadersInit {
    return {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    }
  }

  /**
   * Fetch with timeout and error handling
   */
  private async fetchWithTimeout<T>(
    url: string,
    options: RequestInit
  ): Promise<T> {
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), this.timeout)

    try {
      const response = await fetch(url, {
        ...options,
        signal: controller.signal,
      })

      clearTimeout(timeoutId)

      if (!response.ok) {
        const errorData: ApiError = await response.json().catch(() => ({
          message: `HTTP ${response.status}: ${response.statusText}`,
        }))
        throw new ApiServiceError(response.status, errorData)
      }

      return await response.json()
    } catch (error) {
      clearTimeout(timeoutId)

      if (error instanceof ApiServiceError) {
        throw error
      }

      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          throw new ApiServiceError(408, {
            message: 'Request timeout',
            details: `Request exceeded ${this.timeout}ms timeout`,
          })
        }
        throw new ApiServiceError(0, {
          message: 'Network error',
          details: error.message,
        })
      }

      throw new ApiServiceError(0, {
        message: 'Unknown error occurred',
      })
    }
  }
}

/**
 * Default singleton instance
 */
export const logQueryService = new LogQueryService()

/**
 * Convenience function for querying logs
 */
export const queryLogs = (params: LogQueryParams): Promise<LogQueryResponse> => {
  return logQueryService.queryLogs(params)
}

/**
 * Convenience function for getting datasets
 */
export const getDatasets = (): Promise<Dataset[]> => {
  return logQueryService.getDatasets()
}

/**
 * Convenience function for health check
 */
export const healthCheck = (): Promise<{ status: string; timestamp: string }> => {
  return logQueryService.healthCheck()
}

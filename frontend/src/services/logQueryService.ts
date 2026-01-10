/**
 * Log Query Service
 * Handles all API interactions with the edge-logs backend
 */

import type { LogQueryParams, LogQueryResponse, ApiError, Dataset } from '@/types/api'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'
const API_TIMEOUT = parseInt(import.meta.env.VITE_API_TIMEOUT || '10000')
const MAX_RETRIES = 3
const RETRY_DELAY_MS = 1000

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
 * Sleep utility for retry delay
 */
const sleep = (ms: number): Promise<void> => new Promise(resolve => setTimeout(resolve, ms))

/**
 * Log Query Service Class
 */
export class LogQueryService {
  private baseUrl: string
  private timeout: number
  private maxRetries: number

  constructor(baseUrl: string = API_BASE_URL, timeout: number = API_TIMEOUT, maxRetries: number = MAX_RETRIES) {
    this.baseUrl = baseUrl
    this.timeout = timeout
    this.maxRetries = maxRetries
  }

  /**
   * Execute log query with the given parameters (with retry logic)
   */
  async queryLogs(params: LogQueryParams): Promise<LogQueryResponse> {
    const url = this.buildQueryUrl(params)
    return this.fetchWithRetry<LogQueryResponse>(url, {
      method: 'GET',
      headers: this.getHeaders(),
    })
  }

  /**
   * Get available datasets (with retry logic)
   */
  async getDatasets(): Promise<Dataset[]> {
    // Use hardcoded dataset list as fallback since endpoint may not exist
    const defaultDatasets: Dataset[] = [
      { name: 'default', description: 'Default dataset', cluster: 'default', environment: 'production' },
    ]

    try {
      const url = `${this.baseUrl}/apis/log.theriseunion.io/v1alpha1/datasets`
      return await this.fetchWithRetry<Dataset[]>(url, {
        method: 'GET',
        headers: this.getHeaders(),
      })
    } catch (error) {
      // Return default datasets if endpoint fails
      console.warn('Dataset endpoint not available, using default:', error)
      return defaultDatasets
    }
  }

  /**
   * Health check for backend availability (with retry logic)
   */
  async healthCheck(): Promise<{ status: string; timestamp: string }> {
    const url = `${this.baseUrl}/healthz`
    return this.fetchWithRetry<{ status: string; timestamp: string }>(url, {
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
   * Fetch with timeout and retry logic
   */
  private async fetchWithRetry<T>(
    url: string,
    options: RequestInit
  ): Promise<T> {
    let lastError: Error | undefined

    for (let attempt = 1; attempt <= this.maxRetries; attempt++) {
      try {
        return await this.fetchWithTimeout<T>(url, options)
      } catch (error) {
        lastError = error as Error

        // Don't retry on client errors (4xx) except 408, 429
        if (error instanceof ApiServiceError) {
          if (error.statusCode >= 400 && error.statusCode < 500 &&
              error.statusCode !== 408 && error.statusCode !== 429) {
            throw error
          }
        }

        // Don't retry on the last attempt
        if (attempt === this.maxRetries) {
          break
        }

        // Wait before retry with exponential backoff
        await sleep(RETRY_DELAY_MS * attempt)
      }
    }

    throw lastError || new Error('Max retries exceeded')
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

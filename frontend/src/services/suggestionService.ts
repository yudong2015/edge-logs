/**
 * Suggestion Service
 * Provides field value suggestions for auto-complete functionality
 * Features: debouncing, caching, error handling
 */

import type { SuggestionResponse } from '@/types/api'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'
const API_TIMEOUT = parseInt(import.meta.env.VITE_API_TIMEOUT || '10000')

/**
 * Cache entry structure
 */
interface CacheEntry {
  data: string[]
  timestamp: number
}

/**
 * Suggestion Service Class
 * Provides debounced and cached API calls for field suggestions
 */
export class SuggestionService {
  private baseUrl: string
  private timeout: number
  private cache: Map<string, CacheEntry>
  private readonly CACHE_TTL = 5 * 60 * 1000 // 5 minutes
  private readonly MAX_CACHE_SIZE = 100

  constructor(baseUrl: string = API_BASE_URL, timeout: number = API_TIMEOUT) {
    this.baseUrl = baseUrl
    this.timeout = timeout
    this.cache = new Map()
  }

  /**
   * Get unique namespaces for a dataset
   */
  async getNamespaces(dataset: string): Promise<string[]> {
    return this.fetchWithCache(
      `namespaces:${dataset}`,
      `${this.baseUrl}/apis/log.theriseunion.io/v1alpha1/logdatasets/${dataset}/namespaces`
    )
  }

  /**
   * Get unique pod names for a dataset, optionally filtered by namespace
   */
  async getPods(dataset: string, namespace?: string): Promise<string[]> {
    const cacheKey = namespace
      ? `pods:${dataset}:${namespace}`
      : `pods:${dataset}`

    let url = `${this.baseUrl}/apis/log.theriseunion.io/v1alpha1/logdatasets/${dataset}/pods`
    if (namespace) {
      url += `?namespace=${encodeURIComponent(namespace)}`
    }

    return this.fetchWithCache(cacheKey, url)
  }

  /**
   * Get unique container names for a dataset, optionally filtered by namespace and pod
   */
  async getContainers(dataset: string, namespace?: string, pod?: string): Promise<string[]> {
    const params = new URLSearchParams()
    if (namespace) params.append('namespace', namespace)
    if (pod) params.append('pod', pod)

    const cacheKey = `containers:${dataset}:${namespace || ''}:${pod || ''}`
    const url = `${this.baseUrl}/apis/log.theriseunion.io/v1alpha1/logdatasets/${dataset}/containers?${params}`

    return this.fetchWithCache(cacheKey, url)
  }

  /**
   * Fetch data with caching support
   */
  private async fetchWithCache(cacheKey: string, url: string): Promise<string[]> {
    // Check cache first
    const cached = this.cache.get(cacheKey)
    const now = Date.now()

    if (cached && (now - cached.timestamp) < this.CACHE_TTL) {
      return cached.data
    }

    // Fetch from API
    const data = await this.fetchFromApi<SuggestionResponse>(url)

    // Update cache
    this.addToCache(cacheKey, data.values || [])

    return data.values || []
  }

  /**
   * Add entry to cache with size management
   */
  private addToCache(key: string, data: string[]): void {
    // Implement LRU eviction if cache is full
    if (this.cache.size >= this.MAX_CACHE_SIZE && !this.cache.has(key)) {
      // Remove oldest entry (first entry in Map)
      const firstKey = this.cache.keys().next().value
      if (firstKey) {
        this.cache.delete(firstKey)
      }
    }

    this.cache.set(key, {
      data,
      timestamp: Date.now(),
    })
  }

  /**
   * Fetch data from API with timeout and error handling
   */
  private async fetchFromApi<T>(url: string): Promise<T> {
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), this.timeout)

    try {
      const response = await fetch(url, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'application/json',
        },
        signal: controller.signal,
      })

      clearTimeout(timeoutId)

      if (!response.ok) {
        // Return empty array for 404 (endpoint may not exist yet)
        if (response.status === 404) {
          return [] as unknown as T
        }
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }

      return await response.json()
    } catch (error) {
      clearTimeout(timeoutId)

      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          console.warn('Suggestion request timeout:', url)
        } else {
          console.warn('Suggestion request failed:', error.message)
        }
      }

      // Return empty array on error (graceful degradation)
      return [] as unknown as T
    }
  }

  /**
   * Clear all cached suggestions
   */
  clearCache(): void {
    this.cache.clear()
  }

  /**
   * Clear cache for a specific key pattern
   */
  clearCachePattern(pattern: string): void {
    const keysToDelete: string[] = []

    for (const key of this.cache.keys()) {
      if (key.includes(pattern)) {
        keysToDelete.push(key)
      }
    }

    keysToDelete.forEach(key => this.cache.delete(key))
  }
}

/**
 * Default singleton instance
 */
export const suggestionService = new SuggestionService()

/**
 * Debounce utility function
 */
export function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: ReturnType<typeof setTimeout> | null = null

  return function executedFunction(...args: Parameters<T>) {
    const later = () => {
      timeout = null
      func(...args)
    }

    if (timeout) {
      clearTimeout(timeout)
    }

    timeout = setTimeout(later, wait)
  }
}


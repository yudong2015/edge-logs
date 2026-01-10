/**
 * Query History Service
 * Manages query history and saved queries using local storage
 * Features: LRU eviction, error handling, memory fallback
 */

import dayjs from 'dayjs'
import type { LogQueryParams } from '@/types/api'
import type {
  QueryHistoryEntry,
  SavedQueryEntry,
  QueryHistoryStorage,
} from '@/types/queryHistory'
import { STORAGE_KEYS } from '@/types/queryHistory'

/**
 * Query History Service Class
 * Manages query history with local storage persistence and graceful fallback
 */
export class QueryHistoryService {
  private readonly HISTORY_KEY = STORAGE_KEYS.HISTORY
  private readonly SAVED_KEY = STORAGE_KEYS.SAVED
  private readonly MAX_HISTORY = 10

  // In-memory fallback for when localStorage is unavailable
  private memoryHistory: QueryHistoryEntry[] = []
  private memorySaved: SavedQueryEntry[] = []
  private useMemoryStorage = false

  constructor() {
    this.detectStorageCapability()
  }

  /**
   * Detect if localStorage is available
   */
  private detectStorageCapability(): void {
    try {
      const testKey = '__storage_test__'
      localStorage.setItem(testKey, 'test')
      localStorage.removeItem(testKey)
      this.useMemoryStorage = false
    } catch (e) {
      console.warn('localStorage unavailable, using in-memory storage:', (e as Error).message)
      this.useMemoryStorage = true
    }
  }

  /**
   * Get item from storage with error handling
   */
  private getStorage<T>(key: string, defaultValue: T): T {
    if (this.useMemoryStorage) {
      return key === this.HISTORY_KEY
        ? (this.memoryHistory as unknown as T)
        : (this.memorySaved as unknown as T)
    }

    try {
      const item = localStorage.getItem(key)
      if (!item) return defaultValue
      return JSON.parse(item) as T
    } catch (e) {
      console.error(`Error reading from localStorage (${key}):`, (e as Error).message)
      return defaultValue
    }
  }

  /**
   * Set item in storage with error handling
   */
  private setStorage<T>(key: string, value: T): boolean {
    if (this.useMemoryStorage) {
      if (key === this.HISTORY_KEY) {
        this.memoryHistory = value as unknown as QueryHistoryEntry[]
      } else {
        this.memorySaved = value as unknown as SavedQueryEntry[]
      }
      return true
    }

    try {
      localStorage.setItem(key, JSON.stringify(value))
      return true
    } catch (e) {
      const error = e as Error
      if (error.name === 'QuotaExceededError') {
        console.warn('localStorage quota exceeded, attempting to prune old entries')
        return this.handleQuotaExceeded(key, value)
      }
      console.error(`Error writing to localStorage (${key}):`, error.message)
      return false
    }
  }

  /**
   * Handle quota exceeded error by pruning old entries
   */
  private handleQuotaExceeded<T>(key: string, value: T): boolean {
    try {
      if (Array.isArray(value) && value.length > 0) {
        // Remove oldest entry and try again
        const pruned = value.slice(1)
        localStorage.setItem(key, JSON.stringify(pruned))
        // Update in-memory state too
        if (key === this.HISTORY_KEY) {
          this.memoryHistory = pruned as unknown as QueryHistoryEntry[]
        }
        return true
      }
    } catch (e) {
      console.error('Failed to handle quota exceeded:', (e as Error).message)
    }
    return false
  }

  /**
   * Generate unique ID for an entry
   */
  private generateId(): string {
    return `query-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`
  }

  /**
   * Add query to history
   */
  addToHistory(params: LogQueryParams, resultCount?: number): void {
    const entry: QueryHistoryEntry = {
      id: this.generateId(),
      timestamp: Date.now(),
      params: JSON.parse(JSON.stringify(params)) as LogQueryParams, // Deep clone
      description: this.generateDescription(params),
      resultCount,
    }

    let history = this.getHistory()

    // Remove duplicate if same query exists
    history = history.filter(
      (h) => !this.isSameQuery(h.params, params)
    )

    // Add new entry at the beginning
    history.unshift(entry)

    // Limit to MAX_HISTORY entries
    if (history.length > this.MAX_HISTORY) {
      history = history.slice(0, this.MAX_HISTORY)
    }

    this.setStorage(this.HISTORY_KEY, history)
  }

  /**
   * Get history entries (most recent first)
   */
  getHistory(): QueryHistoryEntry[] {
    return this.getStorage<QueryHistoryEntry[]>(this.HISTORY_KEY, [])
  }

  /**
   * Remove specific entry from history
   */
  removeFromHistory(id: string): void {
    const history = this.getHistory().filter((entry) => entry.id !== id)
    this.setStorage(this.HISTORY_KEY, history)
  }

  /**
   * Clear all history
   */
  clearHistory(): void {
    this.setStorage(this.HISTORY_KEY, [])
  }

  /**
   * Save query with user-defined name
   */
  saveQuery(params: LogQueryParams, name: string): SavedQueryEntry {
    const now = Date.now()
    const entry: SavedQueryEntry = {
      id: this.generateId(),
      timestamp: now,
      createdAt: now,
      updatedAt: now,
      params: JSON.parse(JSON.stringify(params)) as LogQueryParams,
      description: this.generateDescription(params),
      name,
    }

    const saved = this.getSavedQueries()

    // Check for duplicate by name
    const existingIndex = saved.findIndex((s) => s.name === name)
    if (existingIndex >= 0) {
      // Update existing
      saved[existingIndex] = entry
    } else {
      // Add new
      saved.unshift(entry)
    }

    this.setStorage(this.SAVED_KEY, saved)
    return entry
  }

  /**
   * Get saved queries
   */
  getSavedQueries(): SavedQueryEntry[] {
    return this.getStorage<SavedQueryEntry[]>(this.SAVED_KEY, [])
  }

  /**
   * Delete saved query
   */
  deleteSavedQuery(id: string): void {
    const saved = this.getSavedQueries().filter((entry) => entry.id !== id)
    this.setStorage(this.SAVED_KEY, saved)
  }

  /**
   * Update saved query
   */
  updateSavedQuery(id: string, updates: Partial<Omit<SavedQueryEntry, 'id' | 'createdAt'>>): SavedQueryEntry | null {
    const saved = this.getSavedQueries()
    const index = saved.findIndex((entry) => entry.id === id)

    if (index < 0) return null

    const updated: SavedQueryEntry = {
      ...saved[index],
      ...updates,
      updatedAt: Date.now(),
    }

    saved[index] = updated
    this.setStorage(this.SAVED_KEY, saved)
    return updated
  }

  /**
   * Generate human-readable description from query params
   */
  private generateDescription(params: LogQueryParams): string {
    const parts: string[] = []

    // Time range description
    if (params.startTime && params.endTime) {
      const startTime = dayjs(params.startTime)
      const endTime = dayjs(params.endTime)
      const timeDiff = endTime.diff(startTime, 'minute')

      if (timeDiff < 60) {
        parts.push(`Last ${timeDiff} min`)
      } else if (timeDiff < 1440) {
        const hours = Math.round(timeDiff / 60)
        parts.push(`Last ${hours} hour${hours > 1 ? 's' : ''}`)
      } else {
        const days = Math.round(timeDiff / 1440)
        parts.push(`Last ${days} day${days > 1 ? 's' : ''}`)
      }
    }

    // Severity
    parts.push(params.severity ? params.severity : 'All levels')

    // Namespace
    if (params.namespace) {
      parts.push(`ns: ${params.namespace}`)
    }

    // Pod
    if (params.podName) {
      parts.push(`pod: ${params.podName}`)
    }

    // Container
    if (params.containerName) {
      parts.push(`container: ${params.containerName}`)
    }

    // Content filter
    if (params.filter) {
      const filterPreview = params.filter.length > 30
        ? `${params.filter.substring(0, 30)}...`
        : params.filter
      parts.push(`filter: ${filterPreview}`)
    }

    return parts.join(' | ')
  }

  /**
   * Compare two query params for equality
   */
  private isSameQuery(params1: LogQueryParams, params2: LogQueryParams): boolean {
    return (
      params1.dataset === params2.dataset &&
      params1.startTime === params2.startTime &&
      params1.endTime === params2.endTime &&
      params1.namespace === params2.namespace &&
      params1.podName === params2.podName &&
      params1.containerName === params2.containerName &&
      params1.filter === params2.filter &&
      params1.severity === params2.severity
    )
  }

  /**
   * Export all data for backup
   */
  exportData(): QueryHistoryStorage {
    return {
      history: this.getHistory(),
      saved: this.getSavedQueries(),
      version: STORAGE_KEYS.VERSION,
    }
  }

  /**
   * Import data from backup
   */
  importData(data: QueryHistoryStorage): boolean {
    try {
      if (data.version !== STORAGE_KEYS.VERSION) {
        console.warn('Version mismatch, importing anyway')
      }

      if (Array.isArray(data.history)) {
        this.setStorage(this.HISTORY_KEY, data.history.slice(0, this.MAX_HISTORY))
      }

      if (Array.isArray(data.saved)) {
        this.setStorage(this.SAVED_KEY, data.saved)
      }

      return true
    } catch (e) {
      console.error('Failed to import data:', (e as Error).message)
      return false
    }
  }
}

/**
 * Default singleton instance
 */
export const queryHistoryService = new QueryHistoryService()

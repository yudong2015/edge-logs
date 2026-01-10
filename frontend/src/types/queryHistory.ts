/**
 * Query History Types
 * Types for query history and saved queries functionality
 */

import type { LogQueryParams } from './api'

/**
 * Query history entry representing a single query in history
 */
export interface QueryHistoryEntry {
  /** Unique identifier for this entry */
  id: string
  /** Unix timestamp when query was executed */
  timestamp: number
  /** Query parameters used */
  params: LogQueryParams
  /** Human-readable description of the query */
  description: string
  /** Number of results returned (optional, for display) */
  resultCount?: number
}

/**
 * Saved query entry - a user-named bookmarked query
 */
export interface SavedQueryEntry extends QueryHistoryEntry {
  /** User-defined name for the saved query */
  name: string
  /** Creation timestamp */
  createdAt: number
  /** Last update timestamp */
  updatedAt: number
}

/**
 * Storage structure for local storage
 */
export interface QueryHistoryStorage {
  /** Query history entries (max 10) */
  history: QueryHistoryEntry[]
  /** User-saved queries */
  saved: SavedQueryEntry[]
  /** Storage format version for migration */
  version: string
}

/**
 * Storage keys for local storage
 */
export const STORAGE_KEYS = {
  HISTORY: 'edge-logs-query-history',
  SAVED: 'edge-logs-saved-queries',
  VERSION: '1.0.0',
} as const

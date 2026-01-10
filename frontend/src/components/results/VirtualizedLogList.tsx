/**
 * Virtualized Log List Component
 * High-performance log display using react-virtuoso for large datasets
 */

import React, { useCallback, useMemo } from 'react'
import { Virtuoso } from 'react-virtuoso'
import type { LogEntry } from '@/types/api'
import LogEntryRow from './LogEntryRow'

interface VirtualizedLogListProps {
  logs: LogEntry[]
  loading?: boolean
  highlightKeyword?: string
  onLoadMore?: () => void
  height?: number | string
}

/**
 * Virtualized list for efficient rendering of large log datasets
 * Only renders visible rows + buffer for optimal performance
 */
const VirtualizedLogList: React.FC<VirtualizedLogListProps> = ({
  logs,
  loading = false,
  highlightKeyword = '',
  onLoadMore,
  height = 600,
}) => {
  /**
   * Memoize item content renderer for performance
   */
  const itemContent = useCallback(
    (index: number, log: LogEntry) => (
      <LogEntryRow
        log={log}
        highlightKeyword={highlightKeyword}
        index={index}
      />
    ),
    [highlightKeyword]
  )

  /**
   * Memoize components to prevent re-creation
   */
  const components = useMemo(
    () => ({
      ScrollSeekPlaceholder: () => (
        <div
          style={{
            height: '48px',
            padding: '12px 16px',
            borderBottom: '1px solid rgba(255, 255, 255, 0.08)',
            background: 'rgba(0, 0, 0, 0.3)',
          }}
        >
          <div
            style={{
              height: '24px',
              width: '60%',
              background: 'rgba(255, 255, 255, 0.1)',
              borderRadius: '4px',
            }}
          />
        </div>
      ),
      Footer: loading
        ? () => (
            <div
              style={{
                padding: '16px',
                textAlign: 'center',
                color: 'rgba(255, 255, 255, 0.5)',
              }}
            >
              Loading more logs...
            </div>
          )
        : undefined,
    }),
    [loading]
  )

  /**
   * Handle end reached for infinite scroll
   */
  const endReached = useCallback(() => {
    if (!loading && onLoadMore && logs.length > 0) {
      onLoadMore()
    }
  }, [loading, onLoadMore, logs.length])

  /**
   * Compute total height
   */
  const containerHeight = useMemo(() => {
    if (typeof height === 'number') {
      return `${height}px`
    }
    return height
  }, [height])

  if (logs.length === 0 && !loading) {
    return (
      <div
        style={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          height: typeof height === 'number' ? height : 400,
          color: 'rgba(255, 255, 255, 0.45)',
        }}
      >
        <div style={{ textAlign: 'center' }}>
          <div style={{ fontSize: '48px', marginBottom: '16px' }}>📭</div>
          <div>No log entries found</div>
        </div>
      </div>
    )
  }

  return (
    <div style={{ height: containerHeight }}>
      <Virtuoso
        style={{ height: '100%' }}
        data={logs}
        itemContent={itemContent}
        components={components}
        endReached={endReached}
        overscan={200}
        defaultItemHeight={48}
        increaseViewportBy={300}
        aria-label="Log entries list"
        role="listbox"
      />
    </div>
  )
}

export default VirtualizedLogList

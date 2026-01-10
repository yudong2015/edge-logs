/**
 * Log Entry Row Component
 * Memoized individual log row for virtualized list
 */

import { memo, useMemo } from 'react'
import { Typography } from 'antd'
import dayjs from 'dayjs'
import type { LogEntry } from '@/types/api'
import LogEntryCell from './LogEntryCell'

const { Text } = Typography

interface LogEntryRowProps {
  log: LogEntry
  highlightKeyword?: string
  index: number
}

/**
 * Individual log row component with memoization for performance
 * Only re-renders when log content or highlight keyword changes
 */
const LogEntryRow = memo<LogEntryRowProps>(({ log, highlightKeyword, index }) => {
  /**
   * Format timestamp for display
   */
  const formattedTime = useMemo(
    () => dayjs(log.timestamp).format('HH:mm:ss.SSS'),
    [log.timestamp]
  )

  /**
   * Get severity color for badge
   */
  const severityColor = useMemo(() => {
    const colors: Record<string, string> = {
      debug: '#8c8c8c',
      info: '#1677ff',
      notice: '#52c41a',
      warning: '#faad14',
      error: '#ff4d4f',
      critical: '#f5222d',
      alert: '#722ed1',
      emergency: '#fa541c',
    }
    return colors[log.severity?.toLowerCase()] || '#8c8c8c'
  }, [log.severity])

  return (
    <div
      className="log-entry-row"
      style={{
        display: 'flex',
        alignItems: 'flex-start',
        padding: '8px 12px',
        borderBottom: '1px solid rgba(255, 255, 255, 0.06)',
        fontSize: '13px',
        fontFamily: 'SFMono-Regular, Consolas, "Liberation Mono", Menlo, monospace',
        lineHeight: '1.5',
        transition: 'background-color 0.1s ease',
      }}
      role="option"
      aria-label={`Log entry at ${formattedTime}, severity ${log.severity}`}
      aria-setsize={-1}
      aria-posinset={index + 1}
      data-log-id={`${log.timestamp}-${log.pod_name}-${index}`}
    >
      {/* Timestamp */}
      <div
        style={{
          width: '100px',
          flexShrink: 0,
          color: 'rgba(255, 255, 255, 0.45)',
        }}
      >
        <Text style={{ color: 'inherit', fontSize: 'inherit' }}>
          {formattedTime}
        </Text>
      </div>

      {/* Severity Badge */}
      <div
        style={{
          width: '70px',
          flexShrink: 0,
          marginRight: '12px',
        }}
      >
        <span
          style={{
            display: 'inline-block',
            padding: '2px 8px',
            borderRadius: '4px',
            fontSize: '11px',
            fontWeight: 500,
            backgroundColor: `${severityColor}20`,
            color: severityColor,
            textTransform: 'uppercase',
          }}
        >
          {log.severity || 'INFO'}
        </span>
      </div>

      {/* Namespace */}
      <div
        style={{
          width: '120px',
          flexShrink: 0,
          color: 'rgba(255, 255, 255, 0.65)',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
        }}
      >
        <Text
          style={{ color: 'inherit', fontSize: 'inherit' }}
          ellipsis={{ tooltip: log.namespace }}
        >
          {log.namespace || '-'}
        </Text>
      </div>

      {/* Pod Name */}
      <div
        style={{
          width: '150px',
          flexShrink: 0,
          color: 'rgba(255, 255, 255, 0.6)',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
        }}
      >
        <Text
          style={{ color: 'inherit', fontSize: 'inherit' }}
          ellipsis={{ tooltip: log.pod_name }}
        >
          {log.pod_name || '-'}
        </Text>
      </div>

      {/* Log Content with highlighting */}
      <div
        style={{
          flex: 1,
          minWidth: 0,
          color: 'rgba(255, 255, 255, 0.85)',
        }}
      >
        <LogEntryCell
          content={log.content || ''}
          highlightKeyword={highlightKeyword}
        />
      </div>
    </div>
  )
}, (prevProps, nextProps) => {
  // Custom comparison for optimal re-render performance
  return (
    prevProps.log.timestamp === nextProps.log.timestamp &&
    prevProps.log.content === nextProps.log.content &&
    prevProps.highlightKeyword === nextProps.highlightKeyword
  )
})

LogEntryRow.displayName = 'LogEntryRow'

export default LogEntryRow

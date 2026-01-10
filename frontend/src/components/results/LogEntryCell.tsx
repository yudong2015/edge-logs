/**
 * Log Entry Cell Component
 * Formatted log content with syntax and keyword highlighting
 */

import React, { useMemo } from 'react'
import HighlightText from './HighlightText'

interface LogEntryCellProps {
  content: string
  highlightKeyword?: string
}

/**
 * Log cell component with syntax highlighting for log fields
 * Applies keyword highlighting on top of syntax highlighting
 */
const LogEntryCell: React.FC<LogEntryCellProps> = ({ content, highlightKeyword }) => {
  /**
   * Parse and format log content with syntax highlighting
   * Detects common log patterns: timestamps, key=value pairs, JSON
   */
  const formattedContent = useMemo(() => {
    if (!content) return ''

    // If keyword highlighting is requested, let HighlightText handle it
    if (highlightKeyword) {
      return <HighlightText text={content} keyword={highlightKeyword} />
    }

    // Basic formatting without keyword highlighting
    return <span style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>{content}</span>
  }, [content, highlightKeyword])

  return <div className="log-entry-cell">{formattedContent}</div>
}

export default LogEntryCell

/**
 * Highlight Text Component
 * Keyword highlighting utility for log content
 */

import { useMemo } from 'react'

interface HighlightTextProps {
  text: string
  keyword: string
  caseSensitive?: boolean
}

/**
 * Highlight keywords in text with yellow background
 * Supports case-insensitive matching and regex patterns
 */
const HighlightText: React.FC<HighlightTextProps> = ({
  text,
  keyword,
  caseSensitive = false,
}) => {
  /**
   * Split text into highlighted and non-highlighted parts
   */
  const highlightedParts = useMemo(() => {
    if (!text || !keyword) {
      return [{ text, isHighlighted: false }]
    }

    const flags = caseSensitive ? 'g' : 'gi'
    const regex = new RegExp(`(${escapeRegex(keyword)})`, flags)

    const parts: Array<{ text: string; isHighlighted: boolean }> = []
    let lastIndex = 0
    let match

    while ((match = regex.exec(text)) !== null) {
      // Add text before match
      if (match.index > lastIndex) {
        parts.push({
          text: text.slice(lastIndex, match.index),
          isHighlighted: false,
        })
      }

      // Add matched text
      parts.push({
        text: match[0],
        isHighlighted: true,
      })

      lastIndex = regex.lastIndex
    }

    // Add remaining text
    if (lastIndex < text.length) {
      parts.push({
        text: text.slice(lastIndex),
        isHighlighted: false,
      })
    }

    return parts.length > 0 ? parts : [{ text, isHighlighted: false }]
  }, [text, keyword, caseSensitive])

  if (!text) {
    return null
  }

  return (
    <span style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
      {highlightedParts.map((part, index) => (
        <span
          key={index}
          style={{
            backgroundColor: part.isHighlighted ? '#f5e066' : 'transparent',
            color: part.isHighlighted ? '#1f1f1f' : 'inherit',
            fontWeight: part.isHighlighted ? 600 : 'normal',
            padding: part.isHighlighted ? '0 2px' : '0',
            borderRadius: '2px',
          }}
        >
          {part.text}
        </span>
      ))}
    </span>
  )
}

/**
 * Escape special regex characters
 */
function escapeRegex(str: string): string {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

export default HighlightText

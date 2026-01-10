/**
 * Query History Item Component
 * Individual history or saved query entry with replay and delete actions
 */

import React, { useCallback } from 'react'
import { List, Typography, Space, Button, Tag, Tooltip, Popconfirm } from 'antd'
import {
  HistoryOutlined,
  StarOutlined,
  StarFilled,
  PlayCircleOutlined,
  DeleteOutlined,
} from '@ant-design/icons'
import dayjs from 'dayjs'
import type { QueryHistoryEntry, SavedQueryEntry } from '@/types/queryHistory'
import relativeTime from 'dayjs/plugin/relativeTime'
import type { LogQueryParams } from '@/types/api'

dayjs.extend(relativeTime)

const { Text } = Typography

// Named constants for timestamp formatting
const MINUTES_PER_HOUR = 60
const MINUTES_PER_DAY = 1440
const MINUTES_PER_MONTH = 43200  // 30 days

export interface QueryHistoryItemProps {
  entry: QueryHistoryEntry | SavedQueryEntry
  isSaved: boolean
  onReplay: (params: LogQueryParams) => void
  onDelete: (id: string) => void
  onSave?: (params: LogQueryParams) => void
}

/**
 * Query History Item Component
 * Displays a single history or saved query entry with actions
 */
const QueryHistoryItem: React.FC<QueryHistoryItemProps> = ({
  entry,
  isSaved,
  onReplay,
  onDelete,
  onSave,
}) => {
  /**
   * Handle replay click
   */
  const handleReplay = useCallback(() => {
    onReplay(entry.params)
  }, [entry.params, onReplay])

  /**
   * Handle delete click
   */
  const handleDelete = useCallback(() => {
    onDelete(entry.id)
  }, [entry.id, onDelete])

  /**
   * Handle save click
   */
  const handleSave = useCallback(() => {
    onSave?.(entry.params)
  }, [entry.params, onSave])

  /**
   * Format timestamp for display
   */
  const formatTimestamp = (timestamp: number): string => {
    const now = dayjs()
    const time = dayjs(timestamp)
    const diffMinutes = now.diff(time, 'minute')

    if (diffMinutes < 1) {
      return 'Just now'
    } else if (diffMinutes < MINUTES_PER_HOUR) {
      return `${diffMinutes}m ago`
    } else if (diffMinutes < MINUTES_PER_DAY) {
      const hours = Math.floor(diffMinutes / MINUTES_PER_HOUR)
      return `${hours}h ago`
    } else if (diffMinutes < MINUTES_PER_MONTH) {
      const days = Math.floor(diffMinutes / MINUTES_PER_DAY)
      return `${days}d ago`
    }
    return time.format('MMM D')
  }

  const isSavedEntry = 'name' in entry

  return (
    <List.Item
      style={{
        padding: '8px 12px',
        borderBottom: '1px solid rgba(255, 255, 255, 0.06)',
        cursor: 'pointer',
      }}
      onClick={handleReplay}
      className="query-history-item"
      actions={[
        !isSaved && onSave && (
          <Tooltip key="save" title="Save as bookmark">
            <Button
              type="text"
              size="small"
              icon={<StarOutlined />}
              onClick={(e: React.MouseEvent) => {
                e.stopPropagation()
                handleSave()
              }}
              style={{ color: 'rgba(255, 255, 255, 0.45)' }}
            />
          </Tooltip>
        ),
        <Popconfirm
          key="delete"
          title="Delete this query?"
          description="This action cannot be undone."
          onConfirm={(e) => {
            e?.stopPropagation()
            handleDelete()
          }}
          okText="Delete"
          cancelText="Cancel"
          okButtonProps={{ danger: true }}
        >
          <Button
            type="text"
            size="small"
            icon={<DeleteOutlined />}
            onClick={(e: React.MouseEvent) => e.stopPropagation()}
            style={{ color: 'rgba(255, 255, 255, 0.45)' }}
          />
        </Popconfirm>,
      ]}
    >
      <List.Item.Meta
        avatar={
          isSavedEntry ? (
            <StarFilled style={{ color: '#faad14', fontSize: '16px' }} />
          ) : (
            <HistoryOutlined style={{ color: 'rgba(255, 255, 255, 0.45)', fontSize: '16px' }} />
          )
        }
        title={
          <Space size={8}>
            <Text
              ellipsis={{ tooltip: entry.description }}
              style={{
                color: 'rgba(255, 255, 255, 0.85)',
                fontSize: '13px',
                fontWeight: 500,
              }}
            >
              {isSavedEntry ? (entry as SavedQueryEntry).name : entry.description}
            </Text>
            {entry.resultCount !== undefined && (
              <Tag color="blue" style={{ fontSize: '11px', margin: 0 }}>
                {entry.resultCount} results
              </Tag>
            )}
          </Space>
        }
        description={
          <Space size={12} style={{ marginTop: '4px' }}>
            <Text type="secondary" style={{ fontSize: '12px' }}>
              {entry.description}
            </Text>
            <Text type="secondary" style={{ fontSize: '12px' }}>
              {formatTimestamp(entry.timestamp)}
            </Text>
            <Button
              type="primary"
              size="small"
              icon={<PlayCircleOutlined />}
              onClick={(e: React.MouseEvent) => {
                e.stopPropagation()
                handleReplay()
              }}
              style={{ fontSize: '11px', height: '20px' }}
            >
              Replay
            </Button>
          </Space>
        }
      />
    </List.Item>
  )
}

export default QueryHistoryItem

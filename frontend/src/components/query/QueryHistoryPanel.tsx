/**
 * Query History Panel Component
 * Combined panel for query history and saved queries
 */

import React, { useState, useCallback, useEffect } from 'react'
import { Tabs, Empty, Typography, Button } from 'antd'
import {
  HistoryOutlined,
  StarFilled,
  BookOutlined,
  DeleteOutlined,
  StarOutlined,
} from '@ant-design/icons'
import type { LogQueryParams } from '@/types/api'
import type { QueryHistoryEntry, SavedQueryEntry } from '@/types/queryHistory'
import { queryHistoryService } from '@/services/queryHistoryService'
import QueryHistoryItem from './QueryHistoryItem'
import SaveQueryDialog from './SaveQueryDialog'

const { Text } = Typography

export interface QueryHistoryPanelProps {
  onReplay: (params: LogQueryParams) => void
  onHistoryChange?: () => void
}

/**
 * Query History Panel Component
 * Displays history and saved queries in a tabbed panel
 */
const QueryHistoryPanel: React.FC<QueryHistoryPanelProps> = ({
  onReplay,
  onHistoryChange,
}) => {
  const [history, setHistory] = useState<QueryHistoryEntry[]>([])
  const [saved, setSaved] = useState<SavedQueryEntry[]>([])
  const [saveDialogOpen, setSaveDialogOpen] = useState(false)
  const [pendingParams, setPendingParams] = useState<LogQueryParams | null>(null)
  const [activeTab, setActiveTab] = useState('history')

  /**
   * Load data from storage
   */
  const loadData = useCallback(() => {
    setHistory(queryHistoryService.getHistory())
    setSaved(queryHistoryService.getSavedQueries())
  }, [])

  /**
   * Initial load and storage event listener
   */
  useEffect(() => {
    loadData()

    // Listen for storage changes from other tabs
    const handleStorageChange = () => {
      loadData()
    }

    window.addEventListener('storage', handleStorageChange)
    return () => window.removeEventListener('storage', handleStorageChange)
  }, [loadData])

  /**
   * Handle replay click
   */
  const handleReplay = useCallback(
    (params: LogQueryParams) => {
      onReplay(params)
    },
    [onReplay]
  )

  /**
   * Handle delete history entry
   */
  const handleDeleteHistory = useCallback(
    (id: string) => {
      queryHistoryService.removeFromHistory(id)
      loadData()
      onHistoryChange?.()
    },
    [loadData, onHistoryChange]
  )

  /**
   * Handle delete saved query
   */
  const handleDeleteSaved = useCallback(
    (id: string) => {
      queryHistoryService.deleteSavedQuery(id)
      loadData()
      onHistoryChange?.()
    },
    [loadData, onHistoryChange]
  )

  /**
   * Handle save query dialog
   */
  const handleSave = useCallback((params: LogQueryParams) => {
    setPendingParams(params)
    setSaveDialogOpen(true)
  }, [])

  /**
   * Handle confirm save
   */
  const handleConfirmSave = useCallback((name: string) => {
    if (pendingParams) {
      queryHistoryService.saveQuery(pendingParams, name)
      loadData()
      setSaveDialogOpen(false)
      setPendingParams(null)
      onHistoryChange?.()
    }
  }, [pendingParams, loadData, onHistoryChange])

  /**
   * Handle clear all history
   */
  const handleClearHistory = useCallback(() => {
    queryHistoryService.clearHistory()
    loadData()
    onHistoryChange?.()
  }, [loadData, onHistoryChange])

  const hasHistory = history.length > 0
  const hasSaved = saved.length > 0

  const tabItems = [
    {
      key: 'history',
      label: (
        <span>
          <HistoryOutlined /> History ({history.length})
        </span>
      ),
      children: hasHistory ? (
        <>
          {history.length > 1 && (
            <div style={{ marginBottom: '8px', textAlign: 'right' }}>
              <Button
                type="link"
                size="small"
                danger
                icon={<DeleteOutlined />}
                onClick={handleClearHistory}
                style={{ fontSize: '12px' }}
              >
                Clear All
              </Button>
            </div>
          )}
          {history.map((entry) => (
            <QueryHistoryItem
              key={entry.id}
              entry={entry}
              isSaved={false}
              onReplay={handleReplay}
              onDelete={handleDeleteHistory}
              onSave={handleSave}
            />
          ))}
        </>
      ) : (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={
            <span style={{ color: 'rgba(255, 255, 255, 0.45)' }}>
              No query history yet
            </span>
          }
        />
      ),
    },
    {
      key: 'saved',
      label: (
        <span>
          <StarFilled style={{ color: '#faad14' }} /> Saved ({saved.length})
        </span>
      ),
      children: hasSaved ? (
        saved.map((entry) => (
          <QueryHistoryItem
            key={entry.id}
            entry={entry}
            isSaved={true}
            onReplay={handleReplay}
            onDelete={handleDeleteSaved}
          />
        ))
      ) : (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={
            <div>
              <div style={{ color: 'rgba(255, 255, 255, 0.45)', marginBottom: '8px' }}>
                No saved queries yet
              </div>
              <div style={{ fontSize: '12px', color: 'rgba(255, 255, 255, 0.35)' }}>
                Run a query and click the <StarOutlined /> icon to save it
              </div>
            </div>
          }
        />
      ),
    },
  ]

  return (
    <>
      <div
        style={{
          borderTop: '1px solid rgba(255, 255, 255, 0.08)',
          paddingTop: '16px',
        }}
      >
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: '12px',
          }}
        >
          <Text strong style={{ color: 'rgba(255, 255, 255, 0.85)', fontSize: '13px' }}>
            <BookOutlined /> Query History
          </Text>
          <Text type="secondary" style={{ fontSize: '12px' }}>
            Click to replay any query
          </Text>
        </div>

        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={tabItems}
          size="small"
        />
      </div>

      <SaveQueryDialog
        open={saveDialogOpen}
        params={pendingParams}
        onSave={handleConfirmSave}
        onCancel={() => {
          setSaveDialogOpen(false)
          setPendingParams(null)
        }}
      />
    </>
  )
}

export default QueryHistoryPanel

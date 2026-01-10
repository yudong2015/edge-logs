/**
 * Log Results Table Component
 * Main table component for displaying log query results
 */

import React from 'react'
import { Table, Tag, Typography } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import type { LogEntry } from '@/types/api'
import { getSeverityTagColor } from '@/styles/theme'

const { Text } = Typography

interface LogResultsTableProps {
  logs: LogEntry[]
  loading?: boolean
  pageSize?: number
  currentPage?: number
  onPageChange?: (page: number, pageSize: number) => void
}

/**
 * Log Results Table component
 * Displays log entries in a sortable, paginated table with severity color coding
 */
const LogResultsTable: React.FC<LogResultsTableProps> = ({
  logs,
  loading = false,
  pageSize = 50,
  currentPage = 1,
  onPageChange,
}) => {
  /**
   * Format timestamp for display
   */
  const formatTimestamp = (timestamp: string): string => {
    try {
      return new Date(timestamp).toLocaleString('en-US', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false,
      })
    } catch {
      return timestamp
    }
  }

  /**
   * Truncate long content for table display
   */
  const truncateContent = (content: string, maxLength: number = 100): string => {
    if (content.length <= maxLength) return content
    return content.substring(0, maxLength) + '...'
  }

  /**
   * Table column definitions
   */
  const columns: ColumnsType<LogEntry> = [
    {
      title: 'Timestamp',
      dataIndex: 'timestamp',
      key: 'timestamp',
      width: 180,
      sorter: (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime(),
      render: (timestamp: string) => (
        <Text style={{ color: 'rgba(255, 255, 255, 0.85)', fontFamily: 'monospace' }}>
          {formatTimestamp(timestamp)}
        </Text>
      ),
    },
    {
      title: 'Severity',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      filters: [
        { text: 'Debug', value: 'debug' },
        { text: 'Info', value: 'info' },
        { text: 'Notice', value: 'notice' },
        { text: 'Warning', value: 'warning' },
        { text: 'Error', value: 'error' },
        { text: 'Critical', value: 'critical' },
      ],
      onFilter: (value, record) => record.severity === value,
      render: (severity: string) => (
        <Tag color={getSeverityTagColor(severity)}>
          {severity.toUpperCase()}
        </Tag>
      ),
    },
    {
      title: 'Namespace',
      dataIndex: 'namespace',
      key: 'namespace',
      width: 150,
      ellipsis: true,
      render: (namespace: string) => (
        <Text ellipsis style={{ color: 'rgba(255, 255, 255, 0.85)' }}>
          {namespace}
        </Text>
      ),
    },
    {
      title: 'Pod',
      dataIndex: 'pod_name',
      key: 'pod_name',
      width: 200,
      ellipsis: true,
      render: (podName: string) => (
        <Text
          ellipsis
          style={{ color: 'rgba(255, 255, 255, 0.85)', fontFamily: 'monospace' }}
        >
          {podName}
        </Text>
      ),
    },
    {
      title: 'Container',
      dataIndex: 'container_name',
      key: 'container_name',
      width: 150,
      ellipsis: true,
      render: (containerName: string) => (
        <Text ellipsis style={{ color: 'rgba(255, 255, 255, 0.85)' }}>
          {containerName}
        </Text>
      ),
    },
    {
      title: 'Content',
      dataIndex: 'content',
      key: 'content',
      ellipsis: true,
      render: (content: string) => (
        <Text
          ellipsis
          style={{ color: 'rgba(255, 255, 255, 0.65)', fontFamily: 'monospace' }}
        >
          {truncateContent(content, 200)}
        </Text>
      ),
    },
  ]

  return (
    <Table
      columns={columns}
      dataSource={logs}
      rowKey={(record: LogEntry) => `${record.timestamp}-${record.pod_name}-${record.namespace}`}
      loading={loading}
      pagination={{
        pageSize,
        current: currentPage,
        total: logs.length,
        showSizeChanger: true,
        showQuickJumper: true,
        showTotal: (total: number, range: [number, number]) =>
          `${range[0]}-${range[1]} of ${total} log entries`,
        pageSizeOptions: ['10', '25', '50', '100', '200'],
        onChange: onPageChange,
      }}
      size="small"
      bordered={false}
      style={{
        background: '#1f1f1f',
      }}
      rowClassName={(record: LogEntry) => `log-row-${record.severity.toLowerCase()}`}
    />
  )
}

export default LogResultsTable

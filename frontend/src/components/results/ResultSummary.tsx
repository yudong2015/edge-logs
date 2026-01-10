/**
 * Result Summary Component
 * Displays summary information about query results
 */

import React from 'react'
import { Card, Statistic, Row, Col, Typography, Space } from 'antd'
import {
  FileTextOutlined,
  ClockCircleOutlined,
  DatabaseOutlined,
} from '@ant-design/icons'

const { Text } = Typography

interface ResultSummaryProps {
  totalCount: number
  executionTime?: number
  dataset?: string
  timeRange?: { start: string; end: string }
}

/**
 * Result Summary component
 * Displays key metrics about the query results including count, execution time, and dataset info
 */
const ResultSummary: React.FC<ResultSummaryProps> = ({
  totalCount,
  executionTime,
  dataset,
  timeRange,
}) => {
  /**
   * Format execution time for display
   */
  const formatExecutionTime = (ms: number): string => {
    if (ms < 1000) {
      return `${ms.toFixed(0)}ms`
    }
    return `${(ms / 1000).toFixed(2)}s`
  }

  /**
   * Format time range for display
   */
  const formatTimeRange = (): string => {
    if (!timeRange) return ''
    try {
      const start = new Date(timeRange.start).toLocaleTimeString()
      const end = new Date(timeRange.end).toLocaleTimeString()
      return `${start} - ${end}`
    } catch {
      return ''
    }
  }

  return (
    <Card
      size="small"
      bordered={false}
      style={{
        background: '#1f1f1f',
        marginBottom: '16px',
      }}
    >
      <Row gutter={[16, 16]}>
        {/* Total Results */}
        <Col xs={24} sm={8} md={6}>
          <Statistic
            title={<Text style={{ color: 'rgba(255, 255, 255, 0.65)' }}>Total Results</Text>}
            value={totalCount}
            prefix={<FileTextOutlined />}
            valueStyle={{ color: '#1677ff' }}
          />
        </Col>

        {/* Execution Time */}
        {executionTime !== undefined && (
          <Col xs={24} sm={8} md={6}>
            <Statistic
              title={<Text style={{ color: 'rgba(255, 255, 255, 0.65)' }}>Execution Time</Text>}
              value={formatExecutionTime(executionTime)}
              prefix={<ClockCircleOutlined />}
              valueStyle={{
                color: executionTime < 1000 ? '#52c41a' : executionTime < 2000 ? '#faad14' : '#ff4d4f'
              }}
            />
          </Col>
        )}

        {/* Dataset */}
        {dataset && (
          <Col xs={24} sm={8} md={6}>
            <div>
              <Text
                type="secondary"
                style={{ fontSize: '14px', display: 'block', marginBottom: '4px' }}
              >
                Dataset
              </Text>
              <Space>
                <DatabaseOutlined style={{ color: 'rgba(255, 255, 255, 0.45)' }} />
                <Text strong style={{ color: 'rgba(255, 255, 255, 0.85)' }}>
                  {dataset}
                </Text>
              </Space>
            </div>
          </Col>
        )}

        {/* Time Range */}
        {timeRange && (
          <Col xs={24} sm={24} md={6}>
            <div>
              <Text
                type="secondary"
                style={{ fontSize: '14px', display: 'block', marginBottom: '4px' }}
              >
                Time Range
              </Text>
              <Text style={{ color: 'rgba(255, 255, 255, 0.65)' }}>
                {formatTimeRange()}
              </Text>
            </div>
          </Col>
        )}
      </Row>
    </Card>
  )
}

export default ResultSummary

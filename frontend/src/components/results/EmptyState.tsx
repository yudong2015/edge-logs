/**
 * Empty State Component
 * Displayed when no query results are available
 */

import React from 'react'
import { Empty, Typography } from 'antd'
import {
  SearchOutlined,
  FileTextOutlined,
} from '@ant-design/icons'

const { Text } = Typography

interface EmptyStateProps {
  type: 'initial' | 'no-results' | 'error'
  message?: string
}

/**
 * Empty State component
 * Provides appropriate empty states for different scenarios
 */
const EmptyState: React.FC<EmptyStateProps> = ({ type, message }) => {
  const getEmptyStateConfig = () => {
    switch (type) {
      case 'initial':
        return {
          icon: <SearchOutlined style={{ fontSize: 64, color: 'rgba(255, 255, 255, 0.25)' }} />,
          description: (
            <div>
              <Text style={{ fontSize: '16px', color: 'rgba(255, 255, 255, 0.85)' }}>
                Start by selecting a time range and filters
              </Text>
              <br />
              <Text type="secondary" style={{ fontSize: '14px' }}>
                Use the query form above to search logs
              </Text>
            </div>
          ),
        }
      case 'no-results':
        return {
          icon: <FileTextOutlined style={{ fontSize: 64, color: 'rgba(255, 255, 255, 0.25)' }} />,
          description: (
            <div>
              <Text style={{ fontSize: '16px', color: 'rgba(255, 255, 255, 0.85)' }}>
                No log entries found
              </Text>
              <br />
              <Text type="secondary" style={{ fontSize: '14px' }}>
                {message || 'Try adjusting your time range or filters'}
              </Text>
            </div>
          ),
        }
      case 'error':
        return {
          icon: <SearchOutlined style={{ fontSize: 64, color: 'rgba(255, 77, 79, 0.5)' }} />,
          description: (
            <div>
              <Text style={{ fontSize: '16px', color: 'rgba(255, 255, 255, 0.85)' }}>
                Query failed
              </Text>
              <br />
              <Text type="secondary" style={{ fontSize: '14px' }}>
                {message || 'An error occurred while fetching logs'}
              </Text>
            </div>
          ),
        }
      default:
        return {
          icon: <SearchOutlined style={{ fontSize: 64, color: 'rgba(255, 255, 255, 0.25)' }} />,
          description: (
            <Text style={{ fontSize: '16px', color: 'rgba(255, 255, 255, 0.85)' }}>
              No data available
            </Text>
          ),
        }
    }
  }

  const config = getEmptyStateConfig()

  return (
    <div style={{ padding: '48px 24px', textAlign: 'center' }}>
      <Empty
        image={config.icon}
        description={config.description}
      />
    </div>
  )
}

export default EmptyState

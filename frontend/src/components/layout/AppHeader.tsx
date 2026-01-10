/**
 * Application Header Component
 * Top application bar with branding and controls
 */

import React from 'react'
import { Layout, Button, Space, Typography } from 'antd'
import { MenuFoldOutlined, MenuUnfoldOutlined, ReloadOutlined } from '@ant-design/icons'

const { Header } = Layout
const { Text } = Typography

interface AppHeaderProps {
  collapsed: boolean
  onToggle: () => void
}

/**
 * Application header component
 * Displays branding and provides controls for sidebar toggle and refresh
 */
const AppHeader: React.FC<AppHeaderProps> = ({ collapsed, onToggle }) => {
  const handleRefresh = () => {
    window.location.reload()
  }

  return (
    <Header
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        background: '#1f1f1f',
        padding: '0 24px',
        borderBottom: '1px solid #424242',
      }}
    >
      <Space align="center" size="large">
        {/* Sidebar Toggle Button */}
        <Button
          type="text"
          icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
          onClick={onToggle}
          style={{
            fontSize: '16px',
            width: 48,
            height: 48,
            color: 'rgba(255, 255, 255, 0.85)',
          }}
        />

        {/* Application Branding */}
        <Space direction="vertical" size={0}>
          <Text
            strong
            style={{
              color: 'rgba(255, 255, 255, 0.85)',
              fontSize: '18px',
              lineHeight: '1.2',
            }}
          >
            Edge Logs
          </Text>
          <Text
            style={{
              color: 'rgba(255, 255, 255, 0.45)',
              fontSize: '12px',
              lineHeight: '1.2',
            }}
          >
            Log Aggregation System
          </Text>
        </Space>
      </Space>

      {/* Right Side Controls */}
      <Space>
        <Button
          type="text"
          icon={<ReloadOutlined />}
          onClick={handleRefresh}
          style={{
            color: 'rgba(255, 255, 255, 0.65)',
            fontSize: '16px',
          }}
          title="Refresh Application"
        />
      </Space>
    </Header>
  )
}

export default AppHeader

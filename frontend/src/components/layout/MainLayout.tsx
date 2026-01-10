/**
 * Main Layout Component
 * Primary application layout with header, sidebar, and content area
 */

import React from 'react'
import { Layout } from 'antd'
import { ConfigProvider } from 'antd'
import AppHeader from './AppHeader'
import DatasetNav from './DatasetNav'
import { darkTheme } from '@/styles/theme'

const { Content, Sider } = Layout

interface MainLayoutProps {
  children: React.ReactNode
}

/**
 * Main application layout component
 * Provides consistent structure with header, sidebar navigation, and content area
 */
const MainLayout: React.FC<MainLayoutProps> = ({ children }) => {
  const [collapsed, setCollapsed] = React.useState(false)

  return (
    <ConfigProvider
      theme={darkTheme}
    >
      <Layout style={{ minHeight: '100vh' }}>
        {/* Application Header */}
        <AppHeader collapsed={collapsed} onToggle={() => setCollapsed(!collapsed)} />

        <Layout>
          {/* Dataset Navigation Sidebar */}
          <Sider
            collapsible
            collapsed={collapsed}
            onCollapse={setCollapsed}
            theme="dark"
            width={240}
            style={{
              overflow: 'auto',
              height: 'calc(100vh - 64px)',
              position: 'fixed',
              left: 0,
              top: 64,
              bottom: 0,
            }}
          >
            <DatasetNav collapsed={collapsed} />
          </Sider>

          {/* Main Content Area */}
          <Layout
            style={{
              marginLeft: collapsed ? 80 : 240,
              transition: 'margin-left 0.2s',
            }}
          >
            <Content
              style={{
                margin: '24px',
                padding: '24px',
                background: darkTheme.token?.colorBgContainer,
                borderRadius: darkTheme.token?.borderRadius,
                minHeight: 'calc(100vh - 112px)',
              }}
            >
              {children}
            </Content>
          </Layout>
        </Layout>
      </Layout>
    </ConfigProvider>
  )
}

export default MainLayout

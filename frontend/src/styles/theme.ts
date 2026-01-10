/**
 * Ant Design Theme Configuration
 * Professional dark theme optimized for log analysis and monitoring
 */

import type { ThemeConfig } from 'antd'

/**
 * Custom dark theme configuration for edge-logs
 * Optimized for professional tool usage with high contrast and readability
 */
export const darkTheme: ThemeConfig = {
  token: {
    // Primary color scheme - professional blue for actions and highlights
    colorPrimary: '#1677ff',
    colorSuccess: '#52c41a',
    colorWarning: '#faad14',
    colorError: '#ff4d4f',
    colorInfo: '#1677ff',

    // Dark mode base colors
    colorBgBase: '#141414',
    colorBgContainer: '#1f1f1f',
    colorBgElevated: '#262626',
    colorBgLayout: '#000000',

    // Text colors for readability
    colorText: 'rgba(255, 255, 255, 0.85)',
    colorTextSecondary: 'rgba(255, 255, 255, 0.65)',
    colorTextTertiary: 'rgba(255, 255, 255, 0.45)',
    colorTextQuaternary: 'rgba(255, 255, 255, 0.25)',

    // Border colors
    colorBorder: '#424242',
    colorBorderSecondary: '#303030',

    // Typography
    fontSize: 14,
    fontSizeHeading1: 38,
    fontSizeHeading2: 30,
    fontSizeHeading3: 24,
    fontSizeHeading4: 20,
    fontSizeHeading5: 16,
    fontFamily: `-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial,
                  'Noto Sans', sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol',
                  'Noto Color Emoji'`,

    // Border radius
    borderRadius: 6,

    // Spacing
    marginXS: 8,
    marginSM: 12,
    margin: 16,
    marginMD: 20,
    marginLG: 24,
    marginXL: 32,

    // Line height
    lineHeight: 1.5715,
  },

  components: {
    // Layout components
    Layout: {
      headerBg: '#1f1f1f',
      headerHeight: 64,
      siderBg: '#141414',
      bodyBg: '#000000',
    },

    // Menu components
    Menu: {
      darkItemBg: '#1f1f1f',
      darkItemSelectedBg: '#1677ff',
      darkItemHoverBg: '#262626',
      itemActiveBg: '#1677ff',
      itemBorderRadius: 6,
    },

    // Table components for log display
    Table: {
      headerBg: '#1f1f1f',
      headerColor: 'rgba(255, 255, 255, 0.85)',
      rowHoverBg: '#262626',
      borderColor: '#424242',
      cellPaddingInline: 16,
      cellPaddingBlock: 12,
    },

    // Form components
    Input: {
      colorBgContainer: '#1f1f1f',
      colorBgElevated: '#262626',
      colorBorder: '#424242',
      colorPrimaryHover: '#4096ff',
      activeBorderColor: '#1677ff',
      hoverBorderColor: '#4096ff',
    },

    InputNumber: {
      colorBgContainer: '#1f1f1f',
      colorBgElevated: '#262626',
    },

    Select: {
      colorBgContainer: '#1f1f1f',
      colorBgElevated: '#262626',
      optionSelectedBg: '#1677ff',
    },

    DatePicker: {
      colorBgContainer: '#1f1f1f',
      colorBgElevated: '#262626',
    },

    // Button components
    Button: {
      colorPrimary: '#1677ff',
      colorPrimaryHover: '#4096ff',
      colorPrimaryActive: '#0958d9',
      defaultBg: '#1f1f1f',
      defaultColor: 'rgba(255, 255, 255, 0.85)',
      defaultBorderColor: '#424242',
    },

    // Card components
    Card: {
      colorBgContainer: '#1f1f1f',
      colorBorderSecondary: '#424242',
    },

    // Message and notification components
    Message: {
      colorText: 'rgba(255, 255, 255, 0.85)',
    },

    Notification: {
      colorText: 'rgba(255, 255, 255, 0.85)',
    },

    // Modal components
    Modal: {
      colorBgElevated: '#1f1f1f',
    },

    // Alert components
    Alert: {
      colorInfoBg: '#1677ff1a',
      colorInfoBorder: '#1677ff',
      colorSuccessBg: '#52c41a1a',
      colorSuccessBorder: '#52c41a',
      colorWarningBg: '#faad141a',
      colorWarningBorder: '#faad14',
      colorErrorBg: '#ff4d4f1a',
      colorErrorBorder: '#ff4d4f',
    },

    // Tag components for severity levels
    Tag: {
      defaultBg: '#1f1f1f',
      defaultColor: 'rgba(255, 255, 255, 0.85)',
    },
  },
}

/**
 * Severity color mapping for log levels
 */
export const severityColors = {
  debug: '#8c8c8c',      // Gray
  info: '#1677ff',       // Blue
  notice: '#52c41a',     // Green
  warning: '#faad14',    // Orange
  error: '#ff4d4f',      // Red
  critical: '#f5222d',   // Dark red
  alert: '#722ed1',      // Purple
  emergency: '#fa541c',  // Dark orange
} as const

/**
 * Get severity color for display
 */
export const getSeverityColor = (severity: string): string => {
  const lowerSeverity = severity.toLowerCase()
  return severityColors[lowerSeverity as keyof typeof severityColors] || '#8c8c8c'
}

/**
 * Get severity tag color for Ant Design Tag component
 */
export const getSeverityTagColor = (severity: string): string => {
  const colorMap: Record<string, string> = {
    debug: 'default',
    info: 'blue',
    notice: 'green',
    warning: 'orange',
    error: 'red',
    critical: 'red',
    alert: 'purple',
    emergency: 'magenta',
  }

  const lowerSeverity = severity.toLowerCase()
  return colorMap[lowerSeverity] || 'default'
}

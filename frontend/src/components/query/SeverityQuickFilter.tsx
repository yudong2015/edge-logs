/**
 * Severity Quick Filter Component
 * Quick filter buttons for severity levels with toggle behavior
 */

import React, { useState, useCallback } from 'react'
import { Button, Space, Typography, Tooltip } from 'antd'
import type { ButtonProps } from 'antd'

const { Text } = Typography

/**
 * Severity level configuration with colors and priorities
 */
export const SEVERITY_LEVELS = [
  { value: 'emergency', label: 'Emergency', color: '#fa541c', priority: 8 },
  { value: 'alert', label: 'Alert', color: '#722ed1', priority: 7 },
  { value: 'critical', label: 'Critical', color: '#f5222d', priority: 6 },
  { value: 'error', label: 'Error', color: '#ff4d4f', priority: 5 },
  { value: 'warning', label: 'Warning', color: '#faad14', priority: 4 },
  { value: 'notice', label: 'Notice', color: '#52c41a', priority: 3 },
  { value: 'info', label: 'Info', color: '#1677ff', priority: 2 },
  { value: 'debug', label: 'Debug', color: '#8c8c8c', priority: 1 },
]

/**
 * Quick access severity buttons (most commonly used)
 */
const QUICK_SEVERITY = ['error', 'warning', 'info', 'debug', 'notice']

export interface SeverityQuickFilterProps {
  /**
   * Current selected severity value
   */
  value?: string

  /**
   * Callback when severity selection changes
   */
  onChange?: (severity: string | undefined) => void

  /**
   * Allow multiple severity selection (not implemented yet, for future use)
   */
  multiSelect?: boolean

  /**
   * Display mode: buttons or segmented
   */
  mode?: 'buttons' | 'segmented'

  /**
   * Size of buttons
   */
  size?: ButtonProps['size']

  /**
   * Additional CSS class name
   */
  className?: string

  /**
   * Additional CSS style
   */
  style?: React.CSSProperties
}

/**
 * Severity Quick Filter Component
 * Provides quick filter buttons for common severity levels
 */
const SeverityQuickFilter: React.FC<SeverityQuickFilterProps> = ({
  value,
  onChange,
  multiSelect: _multiSelect = false,
  mode: _mode = 'buttons',
  size = 'middle',
  className,
  style,
}) => {
  // Use internal state for immediate UI feedback, sync with value prop on changes
  const [internalValue, setInternalValue] = useState<string | undefined>(value)

  /**
   * Handle severity button click
   */
  const handleSeverityClick = useCallback(
    (severity: string) => {
      const newValue = internalValue === severity ? undefined : severity
      setInternalValue(newValue)
      onChange?.(newValue)
    },
    [internalValue, onChange]
  )

  /**
   * Clear all filters
   */
  const handleClear = useCallback(() => {
    setInternalValue(undefined)
    onChange?.(undefined)
  }, [onChange])

  /**
   * Sync with external value changes
   */
  React.useEffect(() => {
    setInternalValue(value)
  }, [value])

  const hasSelection = internalValue !== undefined

  return (
    <div className={className} style={style}>
      <div style={{ marginBottom: '12px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Text strong style={{ color: 'rgba(255, 255, 255, 0.85)' }}>
          Severity
        </Text>
        {hasSelection && (
          <Button
            type="link"
            size="small"
            onClick={handleClear}
            style={{ padding: 0, height: 'auto', fontSize: '12px' }}
          >
            Clear
          </Button>
        )}
      </div>

      <Space wrap size={[8, 8]}>
        {QUICK_SEVERITY.map((severityKey) => {
          const severity = SEVERITY_LEVELS.find((s) => s.value === severityKey)
          if (!severity) return null

          const isSelected = internalValue === severity.value

          return (
            <Tooltip key={severity.value} title={severity.label}>
              <Button
                size={size}
                type={isSelected ? 'primary' : 'default'}
                danger={severity.value === 'error' || severity.value === 'critical' || severity.value === 'emergency'}
                style={{
                  ...(isSelected
                    ? {
                        backgroundColor: severity.color,
                        borderColor: severity.color,
                        color: '#fff',
                      }
                    : {
                        borderColor: `${severity.color}40`,
                        color: severity.color,
                      }),
                }}
                onClick={() => handleSeverityClick(severity.value)}
              >
                {severity.label}
              </Button>
            </Tooltip>
          )
        })}
      </Space>

      {/* Show selected severity indicator */}
      {hasSelection && (
        <div style={{ marginTop: '8px' }}>
          <Text type="secondary" style={{ fontSize: '12px' }}>
            Filtering by:{' '}
            <Text style={{ color: SEVERITY_LEVELS.find((s) => s.value === internalValue)?.color }}>
              {SEVERITY_LEVELS.find((s) => s.value === internalValue)?.label}
            </Text>
          </Text>
        </div>
      )}
    </div>
  )
}

export default SeverityQuickFilter

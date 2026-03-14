/**
 * Time Range Picker Component
 * Provides intuitive time range selection with quick presets and precise date/time pickers
 */

import React, { useEffect } from 'react'
import { Form, DatePicker, Radio, Space, Typography, Divider, Button } from 'antd'
import type { RadioChangeEvent } from 'antd'
import dayjs, { Dayjs } from 'dayjs'

const { RangePicker } = DatePicker
const { Text } = Typography

interface TimeRangePickerProps {
  form: any
  value?: [Dayjs, Dayjs]
  onChange?: (dates: [Dayjs, Dayjs] | null) => void
}

type RangeValue = [Dayjs | null, Dayjs | null] | null

/**
 * Time range quick presets
 */
const timeRangePresets = [
  { label: 'Last 15 minutes', value: 15, unit: 'minutes' },
  { label: 'Last 1 hour', value: 1, unit: 'hours' },
  { label: 'Last 6 hours', value: 6, unit: 'hours' },
  { label: 'Last 24 hours', value: 1, unit: 'days' },
  { label: 'Last 7 days', value: 7, unit: 'days' },
]

/**
 * Special time range presets for Today and Yesterday
 */
const specialTimeRanges = [
  {
    label: 'Today',
    calculate: () => {
      const now = dayjs()
      return [dayjs().startOf('day'), now] as [Dayjs, Dayjs]
    },
  },
  {
    label: 'Yesterday',
    calculate: () => {
      return [
        dayjs().subtract(1, 'day').startOf('day'),
        dayjs().subtract(1, 'day').endOf('day'),
      ] as [Dayjs, Dayjs]
    },
  },
]

/**
 * Time Range Picker component
 * Provides both quick time range presets and precise date/time selection
 */
const TimeRangePicker: React.FC<TimeRangePickerProps> = ({ form, value, onChange }) => {
  const [mode, setMode] = React.useState<'quick' | 'custom'>('quick')
  const [selectedRange, setSelectedRange] = React.useState<Dayjs[]>([
    dayjs().subtract(1, 'hour'),
    dayjs()
  ])

  // Initialize with default time range (last 1 hour)
  useEffect(() => {
    if (!value) {
      const endTime = dayjs()
      const startTime = dayjs().subtract(1, 'hours')
      const range = [startTime, endTime]
      setSelectedRange(range)
      form.setFieldsValue({
        timeRange: range,
        startTime: startTime.toISOString(),
        endTime: endTime.toISOString(),
      })
    } else {
      // Sync with prop value if provided
      setSelectedRange(value)
    }
  }, [])

  const handleQuickSelect = (preset: typeof timeRangePresets[0]) => {
    console.log('Quick select clicked:', preset.label)

    const endTime = dayjs()
    const startTime = dayjs().subtract(preset.value, preset.unit as dayjs.ManipulateType)
    const range = [startTime, endTime] as [Dayjs, Dayjs]

    setSelectedRange(range)
    form.setFieldsValue({
      timeRange: range,
      startTime: startTime.toISOString(),
      endTime: endTime.toISOString(),
    })

    console.log('Time range updated:', range)
    onChange?.(range)
  }

  const handleSpecialRange = (range: typeof specialTimeRanges[0]) => {
    console.log('Special range clicked:', range.label)

    const [startTime, endTime] = range.calculate()
    const timeRange = [startTime, endTime] as [Dayjs, Dayjs]

    setSelectedRange(timeRange)
    form.setFieldsValue({
      timeRange: timeRange,
      startTime: startTime.toISOString(),
      endTime: endTime.toISOString(),
    })

    console.log('Special range updated:', timeRange)
    onChange?.(timeRange)
  }

  const handleCustomRangeChange = (dates: RangeValue) => {
    console.log('Custom range changed:', dates)

    if (dates && dates[0] && dates[1]) {
      const range = [dates[0], dates[1]] as [Dayjs, Dayjs]

      setSelectedRange(range)
      form.setFieldsValue({
        startTime: dates[0].toISOString(),
        endTime: dates[1].toISOString(),
      })
      onChange?.(range)
    }
  }

  // Check if a preset is currently selected
  const isPresetSelected = (preset: typeof timeRangePresets[0]) => {
    if (!selectedRange || selectedRange.length !== 2) return false
    const expectedStart = dayjs().subtract(preset.value, preset.unit as dayjs.ManipulateType)
    return selectedRange[0].isSame(expectedStart, 'minute')
  }

  return (
    <div>
      <div style={{ marginBottom: '12px' }}>
        <Text strong style={{ color: 'rgba(255, 255, 255, 0.85)' }}>
          Time Range
        </Text>
      </div>

      <Form.Item
        name="timeRange"
        style={{ marginBottom: '16px' }}
      >
        <Radio.Group
          value={mode}
          onChange={(e: RadioChangeEvent) => setMode(e.target.value as 'quick' | 'custom')}
          buttonStyle="solid"
          style={{ marginBottom: '12px', display: 'flex', gap: '8px' }}
        >
          <Radio.Button
            value="quick"
            style={{
              cursor: 'pointer',
              backgroundColor: mode === 'quick' ? '#1677ff' : '#1f1f1f',
              borderColor: '#424242',
              color: 'rgba(255, 255, 255, 0.85)'
            }}
          >
            Quick Select
          </Radio.Button>
          <Radio.Button
            value="custom"
            style={{
              cursor: 'pointer',
              backgroundColor: mode === 'custom' ? '#1677ff' : '#1f1f1f',
              borderColor: '#424242',
              color: 'rgba(255, 255, 255, 0.85)'
            }}
          >
            Custom Range
          </Radio.Button>
        </Radio.Group>
      </Form.Item>

      {mode === 'quick' && (
        <>
          <Space wrap size={[8, 8]} style={{ width: '100%' }}>
            {timeRangePresets.map((preset) => (
              <Button
                key={preset.label}
                size="small"
                type={isPresetSelected(preset) ? 'primary' : 'default'}
                onClick={() => handleQuickSelect(preset)}
                style={{
                  cursor: 'pointer',
                  backgroundColor: isPresetSelected(preset) ? undefined : '#1f1f1f',
                  borderColor: '#424242',
                  color: 'rgba(255, 255, 255, 0.85)'
                }}
              >
                {preset.label}
              </Button>
            ))}
            {specialTimeRanges.map((range) => {
              const isSelected = selectedRange &&
                selectedRange.length === 2 &&
                selectedRange[0].isSame(range.calculate()[0], 'minute') &&
                selectedRange[1].isSame(range.calculate()[1], 'minute')

              return (
                <Button
                  key={range.label}
                  size="small"
                  type={isSelected ? 'primary' : 'default'}
                  onClick={() => handleSpecialRange(range)}
                  style={{
                    cursor: 'pointer',
                    backgroundColor: isSelected ? undefined : '#1f1f1f',
                    borderColor: '#424242',
                    color: 'rgba(255, 255, 255, 0.85)'
                  }}
                >
                  {range.label}
                </Button>
              )
            })}
          </Space>
          <div style={{ marginTop: '12px', padding: '8px', background: '#141414', borderRadius: '4px' }}>
            <Text type="secondary" style={{ fontSize: '12px' }}>
              Selected:{' '}
              {selectedRange && selectedRange.length === 2
                ? `${selectedRange[0].format('MMM DD, HH:mm')} - ${selectedRange[1].format('MMM DD, HH:mm')}`
                : 'Last 1 hour'}
            </Text>
          </div>
        </>
      )}

      {mode === 'custom' && (
        <Form.Item name="timeRange" style={{ marginBottom: 0 }}>
          <RangePicker
            showTime
            format="YYYY-MM-DD HH:mm:ss"
            style={{ width: '100%' }}
            placeholder={['Start Time', 'End Time']}
            onChange={handleCustomRangeChange}
          />
        </Form.Item>
      )}

      <Divider style={{ margin: '12px 0', borderColor: '#424242' }} />

      {/* Hidden fields for ISO timestamp values */}
      <Form.Item name="startTime" hidden>
        <input type="hidden" />
      </Form.Item>
      <Form.Item name="endTime" hidden>
        <input type="hidden" />
      </Form.Item>
    </div>
  )
}

export default TimeRangePicker

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

  // Initialize with default time range (last 1 hour)
  useEffect(() => {
    if (!value) {
      const endTime = dayjs()
      const startTime = dayjs().subtract(1, 'hours')
      form.setFieldsValue({
        timeRange: [startTime, endTime],
        startTime: startTime.toISOString(),
        endTime: endTime.toISOString(),
      })
    }
  }, [])

  const handleQuickSelect = (preset: typeof timeRangePresets[0]) => {
    const endTime = dayjs()
    const startTime = dayjs().subtract(preset.value, preset.unit as dayjs.ManipulateType)

    const timeRange: [Dayjs, Dayjs] = [startTime, endTime]
    form.setFieldsValue({
      timeRange,
      startTime: startTime.toISOString(),
      endTime: endTime.toISOString(),
    })

    onChange?.(timeRange)
  }

  const handleSpecialRange = (range: typeof specialTimeRanges[0]) => {
    const [startTime, endTime] = range.calculate()

    form.setFieldsValue({
      timeRange: [startTime, endTime],
      startTime: startTime.toISOString(),
      endTime: endTime.toISOString(),
    })

    onChange?.([startTime, endTime])
  }

  const handleCustomRangeChange = (dates: RangeValue) => {
    if (dates && dates[0] && dates[1]) {
      form.setFieldsValue({
        startTime: dates[0].toISOString(),
        endTime: dates[1].toISOString(),
      })
      onChange?.([dates[0], dates[1]])
    }
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
          style={{ marginBottom: '12px' }}
        >
          <Radio.Button value="quick">Quick Select</Radio.Button>
          <Radio.Button value="custom">Custom Range</Radio.Button>
        </Radio.Group>
      </Form.Item>

      {mode === 'quick' && (
        <>
          <Space wrap size={[8, 8]}>
            {timeRangePresets.map((preset) => (
              <Button
                key={preset.label}
                size="small"
                onClick={() => handleQuickSelect(preset)}
              >
                {preset.label}
              </Button>
            ))}
            {specialTimeRanges.map((range) => (
              <Button
                key={range.label}
                size="small"
                onClick={() => handleSpecialRange(range)}
              >
                {range.label}
              </Button>
            ))}
          </Space>
          <div style={{ marginTop: '8px' }}>
            <Text type="secondary" style={{ fontSize: '12px' }}>
              Selected:{' '}
              {form.getFieldValue('timeRange')?.[0]
                ? `${form.getFieldValue('timeRange')[0].format('MMM DD, HH:mm')} - ${form.getFieldValue('timeRange')[1].format('MMM DD, HH:mm')}`
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

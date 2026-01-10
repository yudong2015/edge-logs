/**
 * Query Form Component
 * Main form for constructing and executing log queries
 */

import React, { useState, useCallback } from 'react'
import {
  Form,
  Button,
  Space,
  Card,
  Typography,
  message,
  Spin,
} from 'antd'
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import TimeRangePicker from './TimeRangePicker'
import FilterInputs from './FilterInputs'
import SeverityQuickFilter from './SeverityQuickFilter'
import QueryHistoryPanel from './QueryHistoryPanel'
import type { LogQueryParams } from '@/types/api'
import { queryLogs } from '@/services/logQueryService'
import { queryHistoryService } from '@/services/queryHistoryService'

const { Title } = Typography

interface QueryFormProps {
  onQueryResults: (results: any, params: LogQueryParams) => void
  onLoadingChange: (loading: boolean) => void
}

interface FormValues {
  dataset: string
  startTime: string
  endTime: string
  namespace?: string
  podName?: string
  containerName?: string
  filter?: string
  severity?: string
}

/**
 * Query form component
 * Provides comprehensive form for constructing log queries with time range selection and filters
 */
const QueryForm: React.FC<QueryFormProps> = ({ onQueryResults, onLoadingChange }) => {
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [severity, setSeverity] = useState<string | undefined>()

  const handleSeverityChange = (newSeverity: string | undefined) => {
    setSeverity(newSeverity)
    form.setFieldsValue({ severity: newSeverity })
  }

  /**
   * Handle replay from history/saved queries
   */
  const handleReplayQuery = useCallback((params: LogQueryParams) => {
    // Set form values from replayed params
    form.setFieldsValue({
      dataset: params.dataset,
      timeRange: [
        params.startTime ? new Date(params.startTime) : undefined,
        params.endTime ? new Date(params.endTime) : undefined,
      ],
      startTime: params.startTime,
      endTime: params.endTime,
      namespace: params.namespace || '',
      podName: params.podName || '',
      containerName: params.containerName || '',
      filter: params.filter || '',
      severity: params.severity || '',
    })

    setSeverity(params.severity || undefined)

    // Auto-submit the replayed query
    setTimeout(() => {
      form.submit()
    }, 100)
  }, [form])

  const handleSubmit = async (values: FormValues) => {
    try {
      setLoading(true)
      onLoadingChange(true)

      const queryParams: LogQueryParams = {
        dataset: values.dataset,
        startTime: values.startTime,
        endTime: values.endTime,
        namespace: values.namespace || undefined,
        podName: values.podName || undefined,
        containerName: values.containerName || undefined,
        filter: values.filter || undefined,
        severity: values.severity || undefined,
        limit: 100, // Default page size
      }

      console.log('Executing query with params:', queryParams)

      const results = await queryLogs(queryParams)

      message.success(`Found ${results.totalCount} log entries`)

      // Save to history after successful query
      queryHistoryService.addToHistory(queryParams, results.totalCount)

      onQueryResults(results, queryParams)
    } catch (error) {
      console.error('Query error:', error)
      message.error(
        error instanceof Error
          ? error.message
          : 'Failed to execute query. Please check your parameters and try again.'
      )
    } finally {
      setLoading(false)
      onLoadingChange(false)
    }
  }

  const handleReset = () => {
    form.resetFields()
    message.info('Form reset')
  }

  return (
    <>
      <Card
        bordered={false}
        style={{
          background: '#1f1f1f',
          borderColor: '#424242',
          marginBottom: '24px',
        }}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={{
            dataset: 'default',
            // Time range will be initialized by TimeRangePicker component
          }}
        >
          <Space direction="vertical" size="large" style={{ width: '100%' }}>
            {/* Header */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Title level={4} style={{ margin: 0, color: 'rgba(255, 255, 255, 0.85)' }}>
                Log Query
              </Title>
              <Space>
                <Button onClick={handleReset} icon={<ReloadOutlined />}>
                  Reset
                </Button>
                <Button
                  type="primary"
                  htmlType="submit"
                  icon={loading ? <Spin size="small" /> : <SearchOutlined />}
                  loading={loading}
                  disabled={loading}
                >
                  {loading ? 'Searching...' : 'Search Logs'}
                </Button>
              </Space>
            </div>

            {/* Time Range Selection */}
            <TimeRangePicker form={form} />

            {/* Severity Quick Filter */}
            <SeverityQuickFilter
              value={severity}
              onChange={handleSeverityChange}
            />

            {/* Filter Inputs */}
            <FilterInputs form={form} />

            {/* Dataset Selection (Hidden for now, will be implemented later) */}
            <Form.Item name="dataset" hidden>
              <input type="hidden" />
            </Form.Item>
          </Space>
        </Form>
      </Card>

      {/* Query History Panel */}
      <QueryHistoryPanel
        onReplay={handleReplayQuery}
        onHistoryChange={() => {
          // Refresh history display when needed
        }}
      />
    </>
  )
}

export default QueryForm

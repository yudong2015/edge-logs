/**
 * Filter Inputs Component
 * Provides input fields for namespace, pod, container, content, and severity filtering
 */

import React from 'react'
import { Form, Input, Select, Typography, Col, Row } from 'antd'

const { TextArea } = Input
const { Option } = Select
const { Text } = Typography

interface FilterInputsProps {
  form: any
}

/**
 * Severity levels for filtering
 */
const severityLevels = [
  { value: '', label: 'All Severities' },
  { value: 'debug', label: 'Debug' },
  { value: 'info', label: 'Info' },
  { value: 'notice', label: 'Notice' },
  { value: 'warning', label: 'Warning' },
  { value: 'error', label: 'Error' },
  { value: 'critical', label: 'Critical' },
  { value: 'alert', label: 'Alert' },
  { value: 'emergency', label: 'Emergency' },
]

/**
 * Filter Inputs component
 * Provides comprehensive filtering options for log queries
 */
const FilterInputs: React.FC<FilterInputsProps> = () => {
  return (
    <div>
      <div style={{ marginBottom: '12px' }}>
        <Text strong style={{ color: 'rgba(255, 255, 255, 0.85)' }}>
          Filters
        </Text>
      </div>

      <Row gutter={[16, 16]}>
        {/* Namespace Filter */}
        <Col xs={24} sm={12} md={8}>
          <Form.Item
            name="namespace"
            label={<Text style={{ color: 'rgba(255, 255, 255, 0.65)' }}>Namespace</Text>}
            tooltip="Filter logs by Kubernetes namespace"
          >
            <Input
              placeholder="e.g., default, kube-system"
              allowClear
            />
          </Form.Item>
        </Col>

        {/* Pod Name Filter */}
        <Col xs={24} sm={12} md={8}>
          <Form.Item
            name="podName"
            label={<Text style={{ color: 'rgba(255, 255, 255, 0.65)' }}>Pod Name</Text>}
            tooltip="Filter logs by pod name"
          >
            <Input
              placeholder="e.g., my-app-12345-abcde"
              allowClear
            />
          </Form.Item>
        </Col>

        {/* Container Name Filter */}
        <Col xs={24} sm={12} md={8}>
          <Form.Item
            name="containerName"
            label={<Text style={{ color: 'rgba(255, 255, 255, 0.65)' }}>Container Name</Text>}
            tooltip="Filter logs by container name"
          >
            <Input
              placeholder="e.g., main-container"
              allowClear
            />
          </Form.Item>
        </Col>

        {/* Severity Filter */}
        <Col xs={24} sm={12} md={8}>
          <Form.Item
            name="severity"
            label={<Text style={{ color: 'rgba(255, 255, 255, 0.65)' }}>Severity</Text>}
            tooltip="Filter logs by severity level"
          >
            <Select
              placeholder="Select severity level"
              allowClear
            >
              {severityLevels.map((level) => (
                <Option key={level.value} value={level.value}>
                  {level.label}
                </Option>
              ))}
            </Select>
          </Form.Item>
        </Col>

        {/* Content Filter */}
        <Col xs={24}>
          <Form.Item
            name="filter"
            label={<Text style={{ color: 'rgba(255, 255, 255, 0.65)' }}>Content Search</Text>}
            tooltip="Search for specific text in log content"
          >
            <TextArea
              placeholder="Enter search terms to filter log content..."
              rows={2}
              allowClear
              showCount
              maxLength={1000}
            />
          </Form.Item>
        </Col>
      </Row>
    </div>
  )
}

export default FilterInputs

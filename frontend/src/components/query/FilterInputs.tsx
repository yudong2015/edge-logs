/**
 * Filter Inputs Component
 * Provides input fields for namespace, pod, container, content, and severity filtering
 * Enhanced with AutoComplete for better UX
 */

import React from 'react'
import { Form, Input, Typography, Col, Row } from 'antd'
import AutoCompleteInput from './AutoCompleteInput'
import { suggestionService } from '@/services/suggestionService'

const { TextArea } = Input
const { Text } = Typography

interface FilterInputsProps {
  form: any
  dataset?: string
}

/**
 * Filter Inputs component
 * Provides comprehensive filtering options for log queries with auto-complete
 */
const FilterInputs: React.FC<FilterInputsProps> = ({ form, dataset = 'default' }) => {
  return (
    <div>
      <div style={{ marginBottom: '12px' }}>
        <Text strong style={{ color: 'rgba(255, 255, 255, 0.85)' }}>
          Filters
        </Text>
      </div>

      <Row gutter={[16, 16]}>
        {/* Namespace Filter with AutoComplete */}
        <Col xs={24} sm={12} md={8}>
          <Form.Item
            name="namespace"
            label={<Text style={{ color: 'rgba(255, 255, 255, 0.65)' }}>Namespace</Text>}
            tooltip="Filter logs by Kubernetes namespace"
          >
            <AutoCompleteInput
              placeholder="e.g., default, kube-system"
              allowClear
              fetchSuggestions={() =>
                suggestionService.getNamespaces(dataset)
              }
              minSearchLength={0}
              debounceDelay={300}
              label="Namespace filter"
            />
          </Form.Item>
        </Col>

        {/* Pod Name Filter with AutoComplete */}
        <Col xs={24} sm={12} md={8}>
          <Form.Item
            name="podName"
            label={<Text style={{ color: 'rgba(255, 255, 255, 0.65)' }}>Pod Name</Text>}
            tooltip="Filter logs by pod name"
          >
            <AutoCompleteInput
              placeholder="e.g., my-app-12345-abcde"
              allowClear
              fetchSuggestions={() =>
                suggestionService.getPods(dataset, form.getFieldValue('namespace'))
              }
              minSearchLength={0}
              debounceDelay={300}
              label="Pod name filter"
            />
          </Form.Item>
        </Col>

        {/* Container Name Filter with AutoComplete */}
        <Col xs={24} sm={12} md={8}>
          <Form.Item
            name="containerName"
            label={<Text style={{ color: 'rgba(255, 255, 255, 0.65)' }}>Container Name</Text>}
            tooltip="Filter logs by container name"
          >
            <AutoCompleteInput
              placeholder="e.g., main-container"
              allowClear
              fetchSuggestions={() =>
                suggestionService.getContainers(
                  dataset,
                  form.getFieldValue('namespace'),
                  form.getFieldValue('podName')
                )
              }
              minSearchLength={0}
              debounceDelay={300}
              label="Container name filter"
            />
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

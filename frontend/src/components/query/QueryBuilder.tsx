/**
 * Query Builder Component
 * Visual filter construction with AND/OR logic support
 */

import React, { useState, useCallback } from 'react'
import {
  Card,
  Button,
  Select,
  Input,
  Space,
  Typography,
  Divider,
  Col,
  Row,
} from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'

const { Option } = Select
const { Text } = Typography

/**
 * Filter operator types
 */
export type FilterOperator = 'equals' | 'contains' | 'regex' | 'exists' | 'not-exists'

/**
 * Filter condition interface
 */
export interface FilterCondition {
  id: string
  field: 'namespace' | 'podName' | 'containerName' | 'content' | 'severity'
  operator: FilterOperator
  value: string
}

/**
 * Logic type for combining conditions
 */
export type LogicType = 'AND' | 'OR'

export interface QueryBuilderProps {
  /**
   * Current filter conditions
   */
  conditions?: FilterCondition[]

  /**
   * Logic type for combining conditions
   */
  logicType?: LogicType

  /**
   * Callback when conditions change
   */
  onChange?: (conditions: FilterCondition[], logicType: LogicType) => void

  /**
   * Available fields for filtering
   */
  availableFields?: Array<{ value: FilterCondition['field']; label: string }>

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
 * Filter operator options
 */
const OPERATOR_OPTIONS: Array<{ value: FilterOperator; label: string }> = [
  { value: 'equals', label: 'Equals' },
  { value: 'contains', label: 'Contains' },
  { value: 'regex', label: 'Regex' },
  { value: 'exists', label: 'Exists' },
  { value: 'not-exists', label: 'Not Exists' },
]

/**
 * Default field options
 */
const DEFAULT_FIELD_OPTIONS = [
  { value: 'namespace' as const, label: 'Namespace' },
  { value: 'podName' as const, label: 'Pod Name' },
  { value: 'containerName' as const, label: 'Container Name' },
  { value: 'content' as const, label: 'Content' },
  { value: 'severity' as const, label: 'Severity' },
]

/**
 * Query Builder Component
 * Provides visual filter construction with AND/OR logic
 */
const QueryBuilder: React.FC<QueryBuilderProps> = ({
  conditions: externalConditions = [],
  logicType: externalLogicType = 'AND',
  onChange,
  availableFields = DEFAULT_FIELD_OPTIONS,
  className,
  style,
}) => {
  const [conditions, setConditions] = useState<FilterCondition[]>(externalConditions)
  const [logicType, setLogicType] = useState<LogicType>(externalLogicType)

  /**
   * Sync with external conditions
   */
  React.useEffect(() => {
    setConditions(externalConditions)
  }, [externalConditions])

  /**
   * Sync with external logic type
   */
  React.useEffect(() => {
    setLogicType(externalLogicType)
  }, [externalLogicType])

  /**
   * Add new condition
   */
  const handleAddCondition = useCallback(() => {
    const newCondition: FilterCondition = {
      id: `condition-${Date.now()}`,
      field: 'namespace',
      operator: 'contains',
      value: '',
    }

    const newConditions = [...conditions, newCondition]
    setConditions(newConditions)
    onChange?.(newConditions, logicType)
  }, [conditions, logicType, onChange])

  /**
   * Remove condition
   */
  const handleRemoveCondition = useCallback(
    (id: string) => {
      const newConditions = conditions.filter((c) => c.id !== id)
      setConditions(newConditions)
      onChange?.(newConditions, logicType)
    },
    [conditions, logicType, onChange]
  )

  /**
   * Update condition field
   */
  const handleFieldChange = useCallback(
    (id: string, field: FilterCondition['field']) => {
      const newConditions = conditions.map((c) =>
        c.id === id ? { ...c, field } : c
      )
      setConditions(newConditions)
      onChange?.(newConditions, logicType)
    },
    [conditions, logicType, onChange]
  )

  /**
   * Update condition operator
   */
  const handleOperatorChange = useCallback(
    (id: string, operator: FilterOperator) => {
      const newConditions = conditions.map((c) =>
        c.id === id ? { ...c, operator } : c
      )
      setConditions(newConditions)
      onChange?.(newConditions, logicType)
    },
    [conditions, logicType, onChange]
  )

  /**
   * Update condition value
   */
  const handleValueChange = useCallback(
    (id: string, value: string) => {
      const newConditions = conditions.map((c) =>
        c.id === id ? { ...c, value } : c
      )
      setConditions(newConditions)
      onChange?.(newConditions, logicType)
    },
    [conditions, logicType, onChange]
  )

  /**
   * Update logic type
   */
  const handleLogicTypeChange = useCallback(
    (newLogicType: LogicType) => {
      setLogicType(newLogicType)
      onChange?.(conditions, newLogicType)
    },
    [conditions, onChange]
  )

  const hasConditions = conditions.length > 0

  return (
    <div className={className} style={style}>
      <div style={{ marginBottom: '12px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Text strong style={{ color: 'rgba(255, 255, 255, 0.85)' }}>
          Advanced Filters
        </Text>
        {hasConditions && (
          <Select
            value={logicType}
            onChange={handleLogicTypeChange}
            size="small"
            style={{ width: 80 }}
          >
            <Option value="AND">AND</Option>
            <Option value="OR">OR</Option>
          </Select>
        )}
      </div>

      {!hasConditions ? (
        <Button
          type="dashed"
          onClick={handleAddCondition}
          icon={<PlusOutlined />}
          block
          style={{ borderColor: '#424242', color: 'rgba(255, 255, 255, 0.45)' }}
        >
          Add Filter Condition
        </Button>
      ) : (
        <Space direction="vertical" size="small" style={{ width: '100%' }}>
          {conditions.map((condition) => {
            const isExistsOperator =
              condition.operator === 'exists' || condition.operator === 'not-exists'

            return (
              <Card
                key={condition.id}
                size="small"
                styles={{
                  body: { padding: '12px' },
                }}
              >
                <Row gutter={[8, 8]} align="middle">
                  {/* Field Selector */}
                  <Col xs={24} sm={6}>
                    <Select
                      value={condition.field}
                      onChange={(value: FilterCondition['field']) => handleFieldChange(condition.id, value)}
                      style={{ width: '100%' }}
                      size="small"
                      placeholder="Field"
                    >
                      {availableFields.map((field) => (
                        <Option key={field.value} value={field.value}>
                          {field.label}
                        </Option>
                      ))}
                    </Select>
                  </Col>

                  {/* Operator Selector */}
                  <Col xs={24} sm={6}>
                    <Select
                      value={condition.operator}
                      onChange={(value: FilterOperator) => handleOperatorChange(condition.id, value)}
                      style={{ width: '100%' }}
                      size="small"
                      placeholder="Operator"
                    >
                      {OPERATOR_OPTIONS.map((op) => (
                        <Option key={op.value} value={op.value}>
                          {op.label}
                        </Option>
                      ))}
                    </Select>
                  </Col>

                  {/* Value Input (not shown for exists operators) */}
                  {!isExistsOperator && (
                    <Col xs={24} sm={8}>
                      <Input
                        value={condition.value}
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => handleValueChange(condition.id, e.target.value)}
                        placeholder="Value"
                        size="small"
                        allowClear
                      />
                    </Col>
                  )}

                  {/* Actions */}
                  <Col xs={24} sm={isExistsOperator ? 12 : 4}>
                    <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
                      <Button
                        type="text"
                        danger
                        size="small"
                        icon={<DeleteOutlined />}
                        onClick={() => handleRemoveCondition(condition.id)}
                      />
                    </Space>
                  </Col>
                </Row>
              </Card>
            )
          })}

          <Button
            type="dashed"
            onClick={handleAddCondition}
            icon={<PlusOutlined />}
            block
            size="small"
            style={{ borderColor: '#424242', color: 'rgba(255, 255, 255, 0.65)' }}
          >
            Add Condition
          </Button>
        </Space>
      )}

      {hasConditions && (
        <>
          <Divider style={{ margin: '12px 0', borderColor: '#424242' }} />
          <Text type="secondary" style={{ fontSize: '12px' }}>
            {conditions.length} condition{conditions.length > 1 ? 's' : ''} with{' '}
            <Text code>{logicType}</Text> logic
          </Text>
        </>
      )}
    </div>
  )
}

export default QueryBuilder

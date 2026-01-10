/**
 * Save Query Dialog Component
 * Modal dialog for naming and saving a query
 */

import React, { useState, useEffect } from 'react'
import { Modal, Input, Form, message } from 'antd'
import type { LogQueryParams } from '@/types/api'

export interface SaveQueryDialogProps {
  open: boolean
  params: LogQueryParams | null
  existingName?: string
  onSave: (name: string) => void
  onCancel: () => void
}

/**
 * Save Query Dialog Component
 * Modal for entering a name when saving a query
 */
const SaveQueryDialog: React.FC<SaveQueryDialogProps> = ({
  open,
  params,
  existingName = '',
  onSave,
  onCancel,
}) => {
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)

  /**
   * Reset form when dialog opens/closes
   */
  useEffect(() => {
    if (open) {
      form.setFieldsValue({ name: existingName })
    }
  }, [open, existingName, form])

  /**
   * Handle save click
   */
  const handleOk = async () => {
    try {
      const values = await form.validateFields()
      setLoading(true)

      if (!values.name || values.name.trim().length === 0) {
        message.warning('Please enter a name for this query')
        setLoading(false)
        return
      }

      onSave(values.name.trim())
      form.resetFields()
      setLoading(false)
    } catch (error) {
      setLoading(false)
    }
  }

  /**
   * Handle cancel click
   */
  const handleCancel = () => {
    form.resetFields()
    onCancel()
  }

  /**
   * Generate default name from params
   */
  const generateDefaultName = (): string => {
    if (!params) return 'My Query'

    const parts: string[] = []

    if (params.severity) {
      parts.push(params.severity)
    }

    if (params.namespace) {
      parts.push(params.namespace)
    }

    if (params.filter) {
      parts.push(params.filter.split(' ')[0])
    }

    return parts.length > 0 ? parts.join(' - ') : 'My Query'
  }

  return (
    <Modal
      title="Save Query"
      open={open}
      onOk={handleOk}
      onCancel={handleCancel}
      okText="Save"
      cancelText="Cancel"
      confirmLoading={loading}
      destroyOnClose
    >
      <Form form={form} layout="vertical" style={{ marginTop: '16px' }}>
        <Form.Item
          name="name"
          label="Query Name"
          rules={[
            { required: true, message: 'Please enter a name' },
            { max: 100, message: 'Name must be less than 100 characters' },
          ]}
          initialValue={generateDefaultName()}
          extra="A descriptive name to help you identify this query later"
        >
          <Input
            placeholder="e.g., Production Errors - Last 24 Hours"
            autoFocus
            onPressEnter={(e: React.KeyboardEvent<HTMLInputElement>) => {
              e.preventDefault()
              handleOk()
            }}
          />
        </Form.Item>

        {params && (
          <div
            style={{
              marginTop: '16px',
              padding: '12px',
              background: 'rgba(255, 255, 255, 0.04)',
              borderRadius: '6px',
            }}
          >
            <div style={{ fontSize: '12px', color: 'rgba(255, 255, 255, 0.45)', marginBottom: '4px' }}>
              Query details:
            </div>
            <div style={{ fontSize: '13px', color: 'rgba(255, 255, 255, 0.65)' }}>
              {params.severity && (
                <div>Severity: {params.severity}</div>
              )}
              {params.namespace && (
                <div>Namespace: {params.namespace}</div>
              )}
              {params.podName && (
                <div>Pod: {params.podName}</div>
              )}
              {params.filter && (
                <div>Filter: {params.filter}</div>
              )}
            </div>
          </div>
        )}
      </Form>
    </Modal>
  )
}

export default SaveQueryDialog

/**
 * Dataset Navigation Component
 * Sidebar navigation for dataset selection and management
 */

import React, { useState, useEffect } from 'react'
import { Menu, Spin, Alert, Typography } from 'antd'
import {
  DatabaseOutlined,
  CloudServerOutlined,
  ExperimentOutlined,
} from '@ant-design/icons'
import type { MenuProps } from 'antd'
import { getDatasets } from '@/services/logQueryService'
import type { Dataset } from '@/types/api'

const { Text } = Typography

interface DatasetNavProps {
  collapsed: boolean
  onDatasetChange?: (datasetName: string) => void
  selectedDataset?: string
}

/**
 * Dataset navigation sidebar component
 * Provides hierarchical dataset selection organized by environment and cluster
 */
const DatasetNav: React.FC<DatasetNavProps> = ({ collapsed, onDatasetChange, selectedDataset: propSelectedDataset }) => {
  const [datasets, setDatasets] = useState<Dataset[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [internalSelectedDataset, setInternalSelectedDataset] = useState<string>(propSelectedDataset || 'edge-system')

  // Use prop-selected dataset if provided, otherwise use internal state
  const selectedDataset = propSelectedDataset || internalSelectedDataset

  useEffect(() => {
    loadDatasets()
  }, [])

  const loadDatasets = async () => {
    try {
      setLoading(true)
      setError(null)
      const data = await getDatasets()
      setDatasets(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load datasets')
    } finally {
      setLoading(false)
    }
  }

  // Group datasets by environment
  const groupedDatasets = datasets.reduce((acc, dataset) => {
    const environment = dataset.environment || 'default'
    if (!acc[environment]) {
      acc[environment] = []
    }
    acc[environment].push(dataset)
    return acc
  }, {} as Record<string, Dataset[]>)

  // Build menu items from grouped datasets
  const menuItems: MenuProps['items'] = Object.entries(groupedDatasets).map(
    ([environment, envDatasets]) => ({
      key: environment,
      icon: <CloudServerOutlined />,
      label: environment.charAt(0).toUpperCase() + environment.slice(1),
      children: envDatasets.map((dataset) => ({
        key: dataset.name,
        icon: <DatabaseOutlined />,
        label: collapsed ? null : (
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Text ellipsis style={{ color: 'rgba(255, 255, 255, 0.85)' }}>
              {dataset.name}
            </Text>
          </div>
        ),
        title: dataset.description || dataset.name,
      })),
    })
  )

  const handleMenuSelect: MenuProps['onSelect'] = ({ key }) => {
    const datasetName = key as string
    // Update internal state only if prop is not controlled
    if (!propSelectedDataset) {
      setInternalSelectedDataset(datasetName)
    }
    // Notify parent component of dataset change
    onDatasetChange?.(datasetName)
  }

  if (loading) {
    return (
      <div style={{ padding: '24px', textAlign: 'center' }}>
        <Spin tip="Loading datasets..." />
      </div>
    )
  }

  if (error) {
    return (
      <div style={{ padding: '16px' }}>
        <Alert
          message="Dataset Load Error"
          description={error}
          type="error"
          showIcon
          closable
        />
      </div>
    )
  }

  if (datasets.length === 0) {
    return (
      <div style={{ padding: '24px', textAlign: 'center' }}>
        <ExperimentOutlined style={{ fontSize: '32px', color: 'rgba(255, 255, 255, 0.25)' }} />
        <div style={{ marginTop: '16px' }}>
          <Text type="secondary">No datasets available</Text>
        </div>
      </div>
    )
  }

  return (
    <div style={{ padding: collapsed ? '16px 0' : '16px' }}>
      {!collapsed && (
        <div style={{ marginBottom: '16px', padding: '0 16px' }}>
          <Text
            strong
            style={{ color: 'rgba(255, 255, 255, 0.85)', fontSize: '12px' }}
          >
            DATASETS
          </Text>
        </div>
      )}
      <Menu
        mode="inline"
        selectedKeys={[selectedDataset]}
        items={menuItems}
        onSelect={handleMenuSelect}
        style={{ borderRight: 0 }}
      />
    </div>
  )
}

export default DatasetNav

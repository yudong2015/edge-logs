/**
 * Auto Complete Input Component
 * Reusable auto-complete wrapper with debouncing and loading states
 */

import React, { useState, useCallback, useEffect, useRef } from 'react'
import { AutoComplete, Spin } from 'antd'
import type { AutoCompleteProps } from 'antd'
import { debounce } from '@/services/suggestionService'

export interface AutoCompleteInputProps extends Omit<AutoCompleteProps, 'options' | 'onSearch'> {
  /**
   * Function to fetch suggestions
   * @param searchValue - The current input value
   * @returns Promise of string array with suggestions
   */
  fetchSuggestions: (searchValue: string) => Promise<string[]>

  /**
   * Debounce delay in milliseconds (default: 300ms)
   */
  debounceDelay?: number

  /**
   * Minimum characters before triggering search (default: 1)
   */
  minSearchLength?: number

  /**
   * Placeholder text when empty
   */
  placeholder?: string

  /**
   * Value for the input
   */
  value?: string

  /**
   * Callback when value changes
   */
  onChange?: (value: string) => void

  /**
   * Allow clearing the input
   */
  allowClear?: boolean

  /**
   * Whether the input is disabled
   */
  disabled?: boolean

  /**
   * Label for accessibility
   */
  label?: string
}

/**
 * Auto Complete Input Component
 * Provides debounced auto-complete functionality with loading states
 */
const AutoCompleteInput: React.FC<AutoCompleteInputProps> = ({
  fetchSuggestions,
  debounceDelay = 300,
  minSearchLength = 1,
  placeholder = 'Type to search...',
  value = '',
  onChange,
  allowClear = true,
  disabled = false,
  label,
  ...restProps
}) => {
  const [options, setOptions] = useState<{ value: string; label: string }[]>([])
  const [loading, setLoading] = useState(false)
  const [searchValue, setSearchValue] = useState(value)
  const abortControllerRef = useRef<AbortController | null>(null)

  // Update local state when value prop changes
  useEffect(() => {
    setSearchValue(value)
  }, [value])

  /**
   * Fetch suggestions with debouncing
   */
  const debouncedFetch = useCallback(
    debounce(async (searchText: string) => {
      // Cancel previous request if still pending
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }

      // Create new abort controller for this request
      const abortController = new AbortController()
      abortControllerRef.current = abortController

      // Check minimum search length
      if (searchText.length < minSearchLength) {
        setOptions([])
        setLoading(false)
        return
      }

      setLoading(true)

      try {
        const results = await fetchSuggestions(searchText)

        // Check if this request was aborted
        if (abortController.signal.aborted) {
          return
        }

        // Convert string array to options format
        const newOptions = results.map((item) => ({
          value: item,
          label: item,
        }))

        setOptions(newOptions)
      } catch (error) {
        if ((error as Error).name !== 'AbortError') {
          console.error('Failed to fetch suggestions:', error)
        }
      } finally {
        if (!abortController.signal.aborted) {
          setLoading(false)
        }
        abortControllerRef.current = null
      }
    }, debounceDelay),
    [fetchSuggestions, debounceDelay, minSearchLength]
  )

  /**
   * Handle search input change
   */
  const handleSearch = (searchText: string) => {
    setSearchValue(searchText)
    debouncedFetch(searchText)
  }

  /**
   * Handle selection change
   */
  const handleChange = (newValue: string) => {
    setSearchValue(newValue)
    onChange?.(newValue)
  }

  /**
   * Handle clear action
   */
  const handleClear = () => {
    setSearchValue('')
    setOptions([])
    onChange?.('')
  }

  /**
   * Cleanup abort controller on unmount
   */
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }
    }
  }, [])

  return (
    <AutoComplete
      value={searchValue}
      options={options}
      onSearch={handleSearch}
      onChange={handleChange}
      onClear={handleClear}
      placeholder={placeholder}
      allowClear={allowClear}
      disabled={disabled}
      notFoundContent={loading ? <Spin size="small" /> : undefined}
      filterOption={false} // Disable client-side filtering since we use server-side
      defaultActiveFirstOption={false}
      aria-label={label || placeholder}
      {...restProps}
    />
  )
}

export default AutoCompleteInput

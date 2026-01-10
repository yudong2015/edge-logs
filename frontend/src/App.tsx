/**
 * Main Application Component
 * Root component that manages application state and layout
 */

import { useState } from 'react'
import { ConfigProvider, App as AntdApp } from 'antd'
import MainLayout from '@/components/layout/MainLayout'
import QueryForm from '@/components/query/QueryForm'
import LogResultsTable from '@/components/results/LogResultsTable'
import ResultSummary from '@/components/results/ResultSummary'
import EmptyState from '@/components/results/EmptyState'
import ErrorBoundary from '@/components/ErrorBoundary'
import { darkTheme } from '@/styles/theme'
import type { LogQueryParams, LogQueryResponse } from '@/types/api'

/**
 * Main application component
 * Manages query state and coordinates between components
 */
function App() {
  const [queryResults, setQueryResults] = useState<LogQueryResponse | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [hasSearched, setHasSearched] = useState(false)

  /**
   * Handle query execution and results display
   */
  const handleQueryResults = (results: LogQueryResponse, _params: LogQueryParams) => {
    setQueryResults(results)
    setHasSearched(true)
  }

  /**
   * Handle loading state changes
   */
  const handleLoadingChange = (loading: boolean) => {
    setIsLoading(loading)
  }

  /**
   * Determine which content to display
   */
  const renderContent = () => {
    // Show initial state if no search has been performed
    if (!hasSearched) {
      return <EmptyState type="initial" />
    }

    // Show loading state during query execution
    if (isLoading) {
      return <EmptyState type="initial" />
    }

    // Show results if available
    if (queryResults && queryResults.logs.length > 0) {
      return (
        <>
          <ResultSummary
            totalCount={queryResults.totalCount}
            executionTime={queryResults.executionTime}
          />
          <LogResultsTable logs={queryResults.logs} loading={isLoading} />
        </>
      )
    }

    // Show no results state
    return <EmptyState type="no-results" />
  }

  return (
    <ErrorBoundary>
      <ConfigProvider theme={darkTheme}>
        <AntdApp>
          <MainLayout>
            <QueryForm
              onQueryResults={handleQueryResults}
              onLoadingChange={handleLoadingChange}
            />
            {renderContent()}
          </MainLayout>
        </AntdApp>
      </ConfigProvider>
    </ErrorBoundary>
  )
}

export default App

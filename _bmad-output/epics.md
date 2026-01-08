---
stepsCompleted: [1, 2, 3, 4]
inputDocuments: ["_bmad-output/architecture.md", "_bmad-output/ux-design-specification.md"]
---

# edge-logs - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for edge-logs, decomposing the requirements from the PRD, UX Design if it exists, and Architecture requirements into implementable stories.

## Requirements Inventory

### Functional Requirements

**From Architecture Document:**

FR1: Provide high-performance log query API for edge computing scenarios
FR2: Support cloud-native log collection using open source components (iLogtail)
FR3: Implement simplified architecture with single API service component
FR4: Support data isolation using dataset as independent field for multiple data sources
FR5: Enable log querying through K8s API Aggregation pattern
FR6: Support time-range based log filtering with millisecond precision
FR7: Support namespace, pod name, and content-based log filtering
FR8: Support log aggregation by dimensions (severity, namespace, etc.)
FR9: Provide metadata enrichment by correlating with K8s API data
FR10: Support multiple dataset routing and switching

### NonFunctional Requirements

**From Architecture Document:**

NFR1: Query response time must be under 2 seconds for typical queries
NFR2: Support high-volume log data storage and retrieval using ClickHouse
NFR3: Use Go 1.23 and go-restful/v3 framework for consistency with edge-apiserver
NFR4: Support edge disconnected scenarios using iLogtail CRI interface
NFR5: Provide data compression and storage optimization (70%+ savings with Delta+ZSTD)
NFR6: Support 30-day TTL for log data with automatic cleanup
NFR7: Implement proper logging using klog/v2 for structured logging
NFR8: Provide Prometheus metrics for monitoring
NFR9: Support Kubernetes client-go v0.31.2 for metadata correlation

**From UX Design Document:**

NFR10: Support desktop browser with responsive design for 1920px+ displays
NFR11: Support Chrome 90+, Firefox 88+, Safari 14+, Edge 90+ browsers
NFR12: Implement dark theme as default for professional tool usage
NFR13: Provide virtualized scrolling for large dataset rendering
NFR14: Support query history preservation and replay functionality
NFR15: Implement auto-complete and intelligent suggestions for query fields

### Additional Requirements

**From Architecture Document:**

- **Starter Template**: Initialize project structure with Go module and dependencies
- Infrastructure: ClickHouse table schema creation with proper partitioning and indexing
- Integration: iLogtail configuration for direct ClickHouse writing
- Security: Support API aggregation with proper authentication and authorization
- Monitoring: Implement health checks (/healthz) and metrics endpoints (/metrics)
- Deployment: Helm chart creation for Kubernetes deployment
- Development: Dockerfile and build scripts for containerization

**From UX Design Document:**

- User Experience: Implement Ant Design v5 component system with dark theme
- Performance: Virtual scrolling for log list display to handle large datasets
- Interaction: Query builder with visual filter conditions
- Navigation: Dataset selector with hierarchical structure
- Search: Syntax highlighting and keyword highlighting in log content
- History: Automatic query history saving (last 10 queries)
- Accessibility: Clear visual hierarchy and keyboard shortcuts support

### FR Coverage Map

FR1: Epic 1 - Foundation and Core Query API
FR2: Epic 1 - Foundation and Core Query API
FR3: Epic 1 - Foundation and Core Query API
FR4: Epic 2 - Dataset Management and Data Isolation
FR5: Epic 1 - Foundation and Core Query API
FR6: Epic 2 - Dataset Management and Data Isolation
FR7: Epic 2 - Dataset Management and Data Isolation
FR8: Epic 3 - Advanced Query and Analytics
FR9: Epic 3 - Advanced Query and Analytics
FR10: Epic 2 - Dataset Management and Data Isolation

## Epic List

### Epic 1: Foundation and Core Query API
Operators can deploy and access the edge-logs system to perform basic log queries, establishing the core infrastructure needed for log aggregation and retrieval in edge computing environments.
**FRs covered:** FR1, FR2, FR3, FR5

### Epic 2: Dataset Management and Data Isolation
Operators can organize, switch between, and securely query different datasets (clusters/environments), enabling proper data isolation and multi-tenant log management across edge deployments.
**FRs covered:** FR4, FR6, FR7, FR10

### Epic 3: Advanced Query and Analytics
Operators can perform complex log analysis, aggregations, and metadata-enriched queries, enabling deep troubleshooting and operational insights across edge infrastructure.
**FRs covered:** FR8, FR9

### Epic 4: Professional Web Interface
Operators can use an intuitive, high-performance web interface for log querying with advanced features like query history, auto-completion, and optimized data visualization for professional monitoring workflows.
**FRs covered:** NFR10-NFR15 (UX-focused requirements)

## Epic 1: Foundation and Core Query API

Operators can deploy and access the edge-logs system to perform basic log queries, establishing the core infrastructure needed for log aggregation and retrieval in edge computing environments.

### Story 1.1: Initialize Project Structure

As a developer,
I want to initialize the edge-logs project with the correct Go module structure and dependencies,
So that I can begin implementing the log aggregation system with proper foundations.

**Acceptance Criteria:**

**Given** I am starting a new edge-logs project
**When** I initialize the project structure
**Then** I have a complete Go module with edge-logs package name
**And** All required dependencies are added (go-restful, klog, clickhouse-go, client-go)
**And** The project follows the defined architecture structure with cmd/, pkg/, config/, and deploy/ directories
**And** A basic Makefile is created for building and testing
**And** README.md contains project setup instructions

### Story 1.2: ClickHouse Database Schema Setup

As a system operator,
I want to have ClickHouse tables properly configured for log storage,
So that I can store and query edge computing logs efficiently with proper indexing and partitioning.

**Acceptance Criteria:**

**Given** ClickHouse is available
**When** I run the database schema setup
**Then** The logs table is created with all required columns (timestamp, dataset, content, severity, etc.)
**And** The table uses MergeTree engine with proper partitioning by (dataset, date)
**And** Proper indexes are created for content search and tags filtering
**And** TTL is set to 30 days for automatic data cleanup
**And** The schema supports data compression with Delta+ZSTD codecs

### Story 1.3: ClickHouse Repository Layer

As a developer,
I want to implement a ClickHouse repository layer,
So that I can perform CRUD operations on log data with proper error handling and connection management.

**Acceptance Criteria:**

**Given** ClickHouse schema is set up
**When** I implement the repository layer
**Then** I can establish connections to ClickHouse with proper configuration
**And** I can insert log records into the logs table
**And** I can query logs by dataset, time range, and basic filters
**And** Connection pooling and error handling are implemented
**And** All database operations use structured logging with klog/v2

### Story 1.4: Basic Log Query Service

As a developer,
I want to implement a log query service layer,
So that I can provide business logic for log retrieval with proper data transformation and validation.

**Acceptance Criteria:**

**Given** ClickHouse repository layer is implemented
**When** I create the log query service
**Then** I can query logs by dataset with time range filtering
**And** I can apply basic content filtering and severity filtering
**And** Query results are properly formatted and paginated
**And** Input validation is performed on all query parameters
**And** Service layer properly handles and logs errors

### Story 1.5: Core API Handler with go-restful

As an operator,
I want to access logs through a REST API,
So that I can query edge computing logs using standard HTTP requests with proper response formatting.

**Acceptance Criteria:**

**Given** Log query service is implemented
**When** I implement the REST API handler
**Then** I can access logs via GET /apis/log.theriseunion.io/v1alpha1/logdatasets/{dataset}/logs
**And** API accepts query parameters for start_time, end_time, namespace, pod_name, filter, and limit
**And** Responses follow the defined JSON structure with proper HTTP status codes
**And** Request logging and metrics are implemented
**And** API documentation is generated with go-restful OpenAPI

## Epic 2: Dataset Management and Data Isolation

Operators can organize, switch between, and securely query different datasets (clusters/environments), enabling proper data isolation and multi-tenant log management across edge deployments.

### Story 2.1: Dataset-Based Query Routing

As an operator,
I want to query logs from specific datasets through URL routing,
So that I can access logs from different edge clusters or environments with proper data isolation.

**Acceptance Criteria:**

**Given** Core API is implemented
**When** I make a request with a dataset in the URL path
**Then** The API routes the query to the correct dataset in ClickHouse
**And** Dataset validation ensures only valid datasets are accessible
**And** Each query is properly scoped to the specified dataset
**And** Cross-dataset queries are prevented for security
**And** Dataset information is included in response metadata

### Story 2.2: Time-Range Filtering with Millisecond Precision

As an operator,
I want to filter logs by precise time ranges,
So that I can investigate incidents that happened at specific moments with millisecond-level accuracy.

**Acceptance Criteria:**

**Given** Dataset routing is implemented
**When** I specify start_time and end_time parameters
**Then** Only logs within the exact time range are returned
**And** Time filtering supports millisecond precision using DateTime64(9)
**And** Time parameters accept ISO 8601 format
**And** Proper error messages are shown for invalid time formats
**And** Time zone handling is consistent and documented

### Story 2.3: Namespace and Pod Filtering

As an operator,
I want to filter logs by Kubernetes namespace and pod names,
So that I can focus on specific applications or services when troubleshooting issues.

**Acceptance Criteria:**

**Given** Time-range filtering is implemented
**When** I specify namespace or pod_name query parameters
**Then** Only logs matching the specified Kubernetes metadata are returned
**And** Filtering uses the k8s_namespace_name and k8s_pod_name columns efficiently
**And** Partial matching is supported for pod names (contains functionality)
**And** Multiple namespaces can be specified in a single query
**And** Proper error handling for non-existent namespaces or pods

### Story 2.4: Content-Based Log Search

As an operator,
I want to search log content using keywords,
So that I can find specific log messages or error patterns across the log data.

**Acceptance Criteria:**

**Given** Kubernetes metadata filtering is implemented
**When** I specify a filter parameter for content search
**Then** Log content is searched using ClickHouse's full-text search capabilities
**And** Search uses the tokenbf_v1 index for efficient content matching
**And** Case-insensitive search is supported
**And** Multiple keywords can be searched with AND/OR logic
**And** Search highlighting indicates matched terms in results

## Epic 3: Advanced Query and Analytics

Operators can perform complex log analysis, aggregations, and metadata-enriched queries, enabling deep troubleshooting and operational insights across edge infrastructure.

### Story 3.1: Log Aggregation by Dimensions

As an operator,
I want to aggregate logs by different dimensions (severity, namespace, host),
So that I can understand patterns and trends in my edge computing infrastructure.

**Acceptance Criteria:**

**Given** Basic log querying is implemented
**When** I use the aggregation API endpoint
**Then** I can group logs by severity, namespace, host_name, or container_name
**And** Aggregation results include count, time ranges, and distribution data
**And** Multiple dimensions can be combined in a single aggregation
**And** Results are properly formatted for visualization
**And** Aggregation queries are optimized for ClickHouse performance

### Story 3.2: K8s Metadata Enrichment Service

As an operator,
I want log entries enriched with additional Kubernetes metadata,
So that I can get complete context about pods, labels, and annotations for better troubleshooting.

**Acceptance Criteria:**

**Given** Basic log queries are working
**When** I enable metadata enrichment for log queries
**Then** Log results include additional K8s metadata beyond what's stored in ClickHouse
**And** Pod labels and annotations are retrieved from the K8s API when available
**And** Metadata enrichment is optional and can be enabled per query
**And** K8s API errors don't break log queries (graceful degradation)
**And** Enriched metadata is cached for performance

### Story 3.3: Advanced Query Performance Optimization

As an operator,
I want log queries to complete in under 2 seconds,
So that I can efficiently troubleshoot issues without waiting for slow query responses.

**Acceptance Criteria:**

**Given** All basic query functionality is implemented
**When** I perform various types of log queries
**Then** Typical queries complete in under 2 seconds as specified in NFR1
**And** Query execution time is measured and logged for monitoring
**And** Database queries use proper indexing and optimization techniques
**And** Connection pooling prevents connection overhead
**And** Query result pagination limits memory usage

## Epic 4: Professional Web Interface

Operators can use an intuitive, high-performance web interface for log querying with advanced features like query history, auto-completion, and optimized data visualization for professional monitoring workflows.

### Story 4.1: Core Web Interface with Ant Design

As an operator,
I want to access logs through a professional web interface,
So that I can query and view logs efficiently using a modern browser-based interface with dark theme.

**Acceptance Criteria:**

**Given** The log API is working
**When** I access the web interface
**Then** I see a professional interface using Ant Design v5 with dark theme
**And** The interface is responsive and works on 1920px+ displays
**And** I can select datasets from a clear navigation component
**And** Time range selection uses intuitive date/time pickers
**And** The interface works on Chrome 90+, Firefox 88+, Safari 14+, Edge 90+

### Story 4.2: High-Performance Log Display with Virtual Scrolling

As an operator,
I want to view large amounts of log data without browser performance issues,
So that I can scroll through thousands of log entries smoothly without page freezing.

**Acceptance Criteria:**

**Given** The core web interface is implemented
**When** I query logs that return large result sets
**Then** Log entries are displayed using virtual scrolling for performance
**And** I can scroll through thousands of entries without browser lag
**And** Log syntax highlighting shows severity levels with appropriate colors
**And** Search keyword highlighting makes matches easy to identify
**And** Loading states indicate when data is being fetched

### Story 4.3: Query Builder and Auto-completion

As an operator,
I want intelligent assistance when building log queries,
So that I can quickly create complex queries without memorizing field names and syntax.

**Acceptance Criteria:**

**Given** High-performance log display is working
**When** I build queries using the interface
**Then** Auto-completion suggests available namespaces, pod names, and field values
**And** Query builder provides visual filter construction for complex conditions
**And** Quick filter buttons for severity levels (Error, Warning, Info, Debug)
**And** Time range shortcuts for common periods (15min, 1hour, today, yesterday)
**And** Field suggestions are based on actual data in the selected dataset

### Story 4.4: Query History and Saved Queries

As an operator,
I want to reuse previous queries and save frequently used searches,
So that I can efficiently repeat common troubleshooting workflows without rebuilding queries.

**Acceptance Criteria:**

**Given** Query builder is implemented
**When** I perform queries and want to reuse them
**Then** My last 10 queries are automatically saved and accessible
**And** I can replay any historical query with one click
**And** Query history persists across browser sessions using local storage
**And** I can bookmark frequently used queries for quick access
**And** Saved queries include both parameters and readable descriptions
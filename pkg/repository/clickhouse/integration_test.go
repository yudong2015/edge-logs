package clickhouse

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clickhouseContainer "github.com/testcontainers/testcontainers-go/modules/clickhouse"

	"github.com/outpostos/edge-logs/pkg/config"
	chModel "github.com/outpostos/edge-logs/pkg/model/clickhouse"
	"github.com/outpostos/edge-logs/pkg/model/request"
)

// IntegrationTestSuite provides integration tests using real ClickHouse
type IntegrationTestSuite struct {
	container *clickhouseContainer.ClickHouseContainer
	config    *config.ClickHouseConfig
	repo      Repository
}

// TestMain sets up integration test environment
func TestMain(m *testing.M) {
	// Skip integration tests if running in CI without Docker
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		fmt.Println("Skipping integration tests")
		os.Exit(0)
	}

	os.Exit(m.Run())
}

// TestIntegrationClickHouseRepository runs comprehensive integration tests
func TestIntegrationClickHouseRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	suite, cleanup := setupIntegrationTest(t)
	defer cleanup()

	t.Run("HealthCheck", func(t *testing.T) {
		ctx := context.Background()
		err := suite.repo.HealthCheck(ctx)
		assert.NoError(t, err)
	})

	t.Run("InsertAndQuerySingleLog", func(t *testing.T) {
		testInsertAndQuerySingleLog(t, suite)
	})

	t.Run("BatchInsertAndQuery", func(t *testing.T) {
		testBatchInsertAndQuery(t, suite)
	})

	t.Run("QueryWithFilters", func(t *testing.T) {
		testQueryWithFilters(t, suite)
	})

	t.Run("QueryPagination", func(t *testing.T) {
		testQueryPagination(t, suite)
	})

	t.Run("QueryPerformance", func(t *testing.T) {
		testQueryPerformance(t, suite)
	})

	t.Run("ConcurrentOperations", func(t *testing.T) {
		testConcurrentOperations(t, suite)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		testErrorHandling(t, suite)
	})
}

// setupIntegrationTest initializes the test environment
func setupIntegrationTest(t *testing.T) (*IntegrationTestSuite, func()) {
	ctx := context.Background()

	// Start ClickHouse container
	clickhouseC, err := clickhouseContainer.Run(ctx,
		"clickhouse/clickhouse-server:23.12",
		clickhouseContainer.WithDatabase("edge_logs_test"),
		clickhouseContainer.WithUsername("test_user"),
		clickhouseContainer.WithPassword("test_pass"),
	)
	require.NoError(t, err)

	// Get connection details
	host, err := clickhouseC.Host(ctx)
	require.NoError(t, err)

	nativePort, err := clickhouseC.MappedPort(ctx, "9000/tcp")
	require.NoError(t, err)

	// Create configuration
	cfg := &config.ClickHouseConfig{
		Host:            host,
		Port:            nativePort.Int(),
		Database:        "edge_logs_test",
		Username:        "test_user",
		Password:        "test_pass",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
		QueryTimeout:    30 * time.Second,
		ExecTimeout:     10 * time.Second,
		BlockSize:       65536,
		Compression:     true,
	}

	// Create repository
	repo, err := NewClickHouseRepository(cfg)
	require.NoError(t, err)

	// Setup test schema
	setupTestSchema(t, repo)

	suite := &IntegrationTestSuite{
		container: clickhouseC,
		config:    cfg,
		repo:      repo,
	}

	cleanup := func() {
		if repo != nil {
			repo.Close()
		}
		if clickhouseC != nil {
			clickhouseC.Terminate(ctx)
		}
	}

	return suite, cleanup
}

// setupTestSchema creates the test database schema
func setupTestSchema(t *testing.T, repo Repository) {
	chRepo, ok := repo.(*ClickHouseRepository)
	require.True(t, ok)

	ctx := context.Background()
	db := chRepo.cm.GetDB()

	// Create tables using the same schema as production
	schema := `
		CREATE TABLE IF NOT EXISTS logs (
			timestamp          DateTime64(9) CODEC(Delta(8), ZSTD(1)),
			dataset            LowCardinality(String) CODEC(ZSTD(1)),
			content            String CODEC(ZSTD(1)),
			severity           LowCardinality(String) CODEC(ZSTD(1)),
			container_id       String CODEC(ZSTD(1)),
			container_name     LowCardinality(String) CODEC(ZSTD(1)),
			pid                String CODEC(ZSTD(1)),
			host_ip            LowCardinality(String) CODEC(ZSTD(1)),
			host_name          LowCardinality(String) CODEC(ZSTD(1)),
			k8s_namespace_name LowCardinality(String) CODEC(ZSTD(1)),
			k8s_pod_name       LowCardinality(String) CODEC(ZSTD(1)),
			k8s_pod_uid        String CODEC(ZSTD(1)),
			k8s_node_name      LowCardinality(String) CODEC(ZSTD(1)),
			tags               Map(String, String) CODEC(ZSTD(1)),
			INDEX idx_content content TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 1,
			INDEX idx_tags tags TYPE bloom_filter GRANULARITY 1
		)
		ENGINE = MergeTree()
		PARTITION BY (dataset, toDate(timestamp))
		ORDER BY (dataset, host_ip, timestamp)
		SETTINGS index_granularity = 8192;

		CREATE TABLE IF NOT EXISTS datasets (
			name String,
			display_name String,
			description String,
			created_at DateTime DEFAULT now(),
			updated_at DateTime DEFAULT now(),
			retention_days UInt32 DEFAULT 30,
			is_active UInt8 DEFAULT 1
		) ENGINE = MergeTree()
		ORDER BY name;

		CREATE TABLE IF NOT EXISTS query_stats (
			query_id String,
			dataset String,
			user_id String DEFAULT '',
			query_type LowCardinality(String),
			query_params String,
			execution_time_ms UInt32,
			rows_examined UInt64,
			rows_returned UInt64,
			created_at DateTime DEFAULT now()
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(created_at)
		ORDER BY (dataset, created_at);

		INSERT INTO datasets (name, display_name, description)
		SELECT 'default', 'Default Dataset', 'Default dataset for tests'
		WHERE NOT EXISTS (SELECT 1 FROM datasets WHERE name = 'default');

		INSERT INTO datasets (name, display_name, description)
		SELECT 'test-dataset', 'Test Dataset', 'Dataset for integration tests'
		WHERE NOT EXISTS (SELECT 1 FROM datasets WHERE name = 'test-dataset');
	`

	_, err := db.ExecContext(ctx, schema)
	require.NoError(t, err, "Failed to create test schema")
}

// testInsertAndQuerySingleLog tests single log insertion and retrieval
func testInsertAndQuerySingleLog(t *testing.T, suite *IntegrationTestSuite) {
	ctx := context.Background()
	now := time.Now().Truncate(time.Microsecond) // ClickHouse precision

	// Create test log entry
	logEntry := &chModel.LogEntry{
		Timestamp:     now,
		Dataset:       "test-dataset",
		Content:       "Test log message for integration test",
		Severity:      "INFO",
		ContainerID:   "container-12345",
		ContainerName: "test-container",
		PID:           "1234",
		HostIP:        "192.168.1.100",
		HostName:      "test-host",
		K8sNamespace:  "default",
		K8sPodName:    "test-pod-123",
		K8sPodUID:     "pod-uid-123",
		K8sNodeName:   "node-1",
		Tags:          map[string]string{"cluster": "test-cluster", "region": "us-east-1"},
	}

	// Insert the log entry
	err := suite.repo.InsertLog(ctx, logEntry)
	require.NoError(t, err)

	// Wait for the data to be available (ClickHouse eventual consistency)
	time.Sleep(100 * time.Millisecond)

	// Query the log entry
	queryReq := &request.LogQueryRequest{
		Dataset:       "test-dataset",
		StartTime:     &[]time.Time{now.Add(-1 * time.Minute)}[0],
		EndTime:       &[]time.Time{now.Add(1 * time.Minute)}[0],
		ContainerName: "test-container",
		PageSize:      10,
	}

	results, total, err := suite.repo.QueryLogs(ctx, queryReq)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1)
	assert.GreaterOrEqual(t, len(results), 1)

	// Verify the retrieved data
	found := false
	for _, result := range results {
		if result.ContainerID == "container-12345" {
			found = true
			assert.Equal(t, logEntry.Dataset, result.Dataset)
			assert.Equal(t, logEntry.Content, result.Content)
			assert.Equal(t, logEntry.Severity, result.Severity)
			assert.Equal(t, logEntry.ContainerName, result.ContainerName)
			assert.Equal(t, logEntry.HostIP, result.HostIP)
			assert.Equal(t, logEntry.K8sNamespace, result.K8sNamespace)
			assert.Equal(t, logEntry.Tags, result.Tags)
			break
		}
	}
	assert.True(t, found, "Inserted log entry not found in query results")
}

// testBatchInsertAndQuery tests batch insertion and retrieval
func testBatchInsertAndQuery(t *testing.T, suite *IntegrationTestSuite) {
	ctx := context.Background()
	baseTime := time.Now().Truncate(time.Second)

	// Create batch of log entries
	batchSize := 100
	logs := make([]chModel.LogEntry, batchSize)
	for i := 0; i < batchSize; i++ {
		logs[i] = chModel.LogEntry{
			Timestamp:     baseTime.Add(time.Duration(i) * time.Millisecond),
			Dataset:       "test-dataset",
			Content:       fmt.Sprintf("Batch log message %d", i),
			Severity:      "INFO",
			ContainerID:   fmt.Sprintf("batch-container-%d", i),
			ContainerName: "batch-test-container",
			PID:           fmt.Sprintf("%d", 1000+i),
			HostIP:        "192.168.1.200",
			HostName:      "batch-test-host",
			K8sNamespace:  "batch-test",
			K8sPodName:    fmt.Sprintf("batch-pod-%d", i%10),
			K8sPodUID:     fmt.Sprintf("batch-uid-%d", i),
			K8sNodeName:   "batch-node-1",
			Tags:          map[string]string{"batch": "true", "index": fmt.Sprintf("%d", i)},
		}
	}

	// Insert batch
	err := suite.repo.InsertLogsBatch(ctx, logs)
	require.NoError(t, err)

	// Wait for data availability
	time.Sleep(500 * time.Millisecond)

	// Query the batch data
	queryReq := &request.LogQueryRequest{
		Dataset:   "test-dataset",
		StartTime: &[]time.Time{baseTime.Add(-1 * time.Minute)}[0],
		EndTime:   &[]time.Time{baseTime.Add(1 * time.Minute)}[0],
		Namespace: "batch-test",
		PageSize:  200, // Request more than batch size
	}

	results, total, err := suite.repo.QueryLogs(ctx, queryReq)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, batchSize)
	assert.GreaterOrEqual(t, len(results), batchSize)

	// Verify batch data integrity
	batchFound := 0
	for _, result := range results {
		if result.K8sNamespace == "batch-test" && result.ContainerName == "batch-test-container" {
			batchFound++
		}
	}
	assert.GreaterOrEqual(t, batchFound, batchSize, "Not all batch entries were found")
}

// testQueryWithFilters tests various query filtering options
func testQueryWithFilters(t *testing.T, suite *IntegrationTestSuite) {
	ctx := context.Background()
	baseTime := time.Now().Truncate(time.Second)

	// Insert test data with various attributes
	testLogs := []chModel.LogEntry{
		{
			Timestamp:     baseTime.Add(-5 * time.Minute),
			Dataset:       "test-dataset",
			Content:       "ERROR: Database connection failed",
			Severity:      "ERROR",
			ContainerName: "api-server",
			HostIP:        "192.168.1.10",
			K8sNamespace:  "production",
			K8sPodName:    "api-pod-1",
			Tags:          map[string]string{"service": "api", "env": "prod"},
		},
		{
			Timestamp:     baseTime.Add(-3 * time.Minute),
			Dataset:       "test-dataset",
			Content:       "INFO: User login successful",
			Severity:      "INFO",
			ContainerName: "auth-server",
			HostIP:        "192.168.1.20",
			K8sNamespace:  "production",
			K8sPodName:    "auth-pod-1",
			Tags:          map[string]string{"service": "auth", "env": "prod"},
		},
		{
			Timestamp:     baseTime.Add(-1 * time.Minute),
			Dataset:       "test-dataset",
			Content:       "WARN: High memory usage detected",
			Severity:      "WARN",
			ContainerName: "worker",
			HostIP:        "192.168.1.30",
			K8sNamespace:  "development",
			K8sPodName:    "worker-pod-1",
			Tags:          map[string]string{"service": "worker", "env": "dev"},
		},
	}

	// Insert test data
	err := suite.repo.InsertLogsBatch(ctx, testLogs)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	tests := []struct {
		name           string
		queryReq       *request.LogQueryRequest
		expectedMinRows int
		verifyFunc     func(results []chModel.LogEntry) bool
	}{
		{
			name: "filter by severity",
			queryReq: &request.LogQueryRequest{
				Dataset:   "test-dataset",
				Severity:  "ERROR",
				StartTime: &[]time.Time{baseTime.Add(-10 * time.Minute)}[0],
				EndTime:   &[]time.Time{baseTime.Add(1 * time.Minute)}[0],
				PageSize:  100,
			},
			expectedMinRows: 1,
			verifyFunc: func(results []chModel.LogEntry) bool {
				for _, r := range results {
					if r.Severity == "ERROR" && r.ContainerName == "api-server" {
						return true
					}
				}
				return false
			},
		},
		{
			name: "filter by namespace",
			queryReq: &request.LogQueryRequest{
				Dataset:   "test-dataset",
				Namespace: "production",
				StartTime: &[]time.Time{baseTime.Add(-10 * time.Minute)}[0],
				EndTime:   &[]time.Time{baseTime.Add(1 * time.Minute)}[0],
				PageSize:  100,
			},
			expectedMinRows: 2,
			verifyFunc: func(results []chModel.LogEntry) bool {
				prodCount := 0
				for _, r := range results {
					if r.K8sNamespace == "production" {
						prodCount++
					}
				}
				return prodCount >= 2
			},
		},
		{
			name: "filter by content (full-text search)",
			queryReq: &request.LogQueryRequest{
				Dataset:   "test-dataset",
				Filter:    "connection",
				StartTime: &[]time.Time{baseTime.Add(-10 * time.Minute)}[0],
				EndTime:   &[]time.Time{baseTime.Add(1 * time.Minute)}[0],
				PageSize:  100,
			},
			expectedMinRows: 1,
			verifyFunc: func(results []chModel.LogEntry) bool {
				for _, r := range results {
					if r.ContainerName == "api-server" && r.Severity == "ERROR" {
						return true
					}
				}
				return false
			},
		},
		{
			name: "filter by host IP",
			queryReq: &request.LogQueryRequest{
				Dataset:   "test-dataset",
				HostIP:    "192.168.1.20",
				StartTime: &[]time.Time{baseTime.Add(-10 * time.Minute)}[0],
				EndTime:   &[]time.Time{baseTime.Add(1 * time.Minute)}[0],
				PageSize:  100,
			},
			expectedMinRows: 1,
			verifyFunc: func(results []chModel.LogEntry) bool {
				for _, r := range results {
					if r.HostIP == "192.168.1.20" && r.ContainerName == "auth-server" {
						return true
					}
				}
				return false
			},
		},
		{
			name: "filter by tags",
			queryReq: &request.LogQueryRequest{
				Dataset:   "test-dataset",
				Tags:      map[string]string{"env": "prod"},
				StartTime: &[]time.Time{baseTime.Add(-10 * time.Minute)}[0],
				EndTime:   &[]time.Time{baseTime.Add(1 * time.Minute)}[0],
				PageSize:  100,
			},
			expectedMinRows: 2,
			verifyFunc: func(results []chModel.LogEntry) bool {
				prodEnvCount := 0
				for _, r := range results {
					if env, exists := r.Tags["env"]; exists && env == "prod" {
						prodEnvCount++
					}
				}
				return prodEnvCount >= 2
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, total, err := suite.repo.QueryLogs(ctx, tt.queryReq)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, total, tt.expectedMinRows)
			assert.GreaterOrEqual(t, len(results), tt.expectedMinRows)

			if tt.verifyFunc != nil {
				assert.True(t, tt.verifyFunc(results), "Verification function failed for query: %s", tt.name)
			}
		})
	}
}

// testQueryPagination tests pagination functionality
func testQueryPagination(t *testing.T, suite *IntegrationTestSuite) {
	ctx := context.Background()
	baseTime := time.Now().Truncate(time.Second)

	// Insert 50 test entries for pagination
	paginationLogs := make([]chModel.LogEntry, 50)
	for i := 0; i < 50; i++ {
		paginationLogs[i] = chModel.LogEntry{
			Timestamp:     baseTime.Add(time.Duration(i) * time.Second),
			Dataset:       "test-dataset",
			Content:       fmt.Sprintf("Pagination test log %d", i),
			Severity:      "INFO",
			ContainerName: "pagination-test",
			HostIP:        "192.168.2.100",
			K8sNamespace:  "pagination",
			K8sPodName:    fmt.Sprintf("pagination-pod-%d", i),
			Tags:          map[string]string{"test": "pagination", "index": fmt.Sprintf("%d", i)},
		}
	}

	err := suite.repo.InsertLogsBatch(ctx, paginationLogs)
	require.NoError(t, err)
	time.Sleep(300 * time.Millisecond)

	// Test pagination
	pageSize := 10
	queryReq := &request.LogQueryRequest{
		Dataset:   "test-dataset",
		Namespace: "pagination",
		StartTime: &[]time.Time{baseTime.Add(-1 * time.Minute)}[0],
		EndTime:   &[]time.Time{baseTime.Add(1 * time.Hour)}[0],
		PageSize:  pageSize,
		Page:      0,
		OrderBy:   "timestamp",
		Direction: "desc", // Most recent first
	}

	// Get first page
	results1, total1, err := suite.repo.QueryLogs(ctx, queryReq)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total1, 50)
	assert.LessOrEqual(t, len(results1), pageSize)

	// Get second page
	queryReq.Page = 1
	results2, total2, err := suite.repo.QueryLogs(ctx, queryReq)
	require.NoError(t, err)
	assert.Equal(t, total1, total2) // Total should remain the same
	assert.LessOrEqual(t, len(results2), pageSize)

	// Verify no overlap between pages (assuming stable sort)
	if len(results1) > 0 && len(results2) > 0 {
		// Since we're ordering by timestamp DESC, first page should have newer entries
		assert.True(t, results1[0].Timestamp.After(results2[0].Timestamp) ||
			results1[0].Timestamp.Equal(results2[0].Timestamp))
	}

	// Test edge cases
	queryReq.Page = 100 // Request page beyond available data
	results3, total3, err := suite.repo.QueryLogs(ctx, queryReq)
	require.NoError(t, err)
	assert.Equal(t, total1, total3)
	assert.Equal(t, 0, len(results3)) // Should return empty results
}

// testQueryPerformance tests query performance requirements
func testQueryPerformance(t *testing.T, suite *IntegrationTestSuite) {
	ctx := context.Background()

	// Test standard time range query performance (< 2 seconds requirement)
	queryReq := &request.LogQueryRequest{
		Dataset:   "test-dataset",
		StartTime: &[]time.Time{time.Now().Add(-1 * time.Hour)}[0],
		EndTime:   &[]time.Time{time.Now()}[0],
		PageSize:  100,
	}

	start := time.Now()
	_, _, err := suite.repo.QueryLogs(ctx, queryReq)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Less(t, duration, 2*time.Second, "Standard query exceeded 2 second performance requirement")

	t.Logf("Query performance: %v", duration)
}

// testConcurrentOperations tests concurrent repository operations
func testConcurrentOperations(t *testing.T, suite *IntegrationTestSuite) {
	ctx := context.Background()
	baseTime := time.Now().Truncate(time.Second)

	// Test concurrent inserts
	numGoroutines := 10
	logsPerGoroutine := 10
	doneCh := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			logs := make([]chModel.LogEntry, logsPerGoroutine)
			for j := 0; j < logsPerGoroutine; j++ {
				logs[j] = chModel.LogEntry{
					Timestamp:     baseTime.Add(time.Duration(goroutineID*1000+j) * time.Millisecond),
					Dataset:       "test-dataset",
					Content:       fmt.Sprintf("Concurrent log from goroutine %d, entry %d", goroutineID, j),
					Severity:      "INFO",
					ContainerName: fmt.Sprintf("concurrent-container-%d", goroutineID),
					HostIP:        fmt.Sprintf("192.168.3.%d", goroutineID+1),
					K8sNamespace:  "concurrent-test",
					K8sPodName:    fmt.Sprintf("concurrent-pod-%d", goroutineID),
					Tags:          map[string]string{"goroutine": fmt.Sprintf("%d", goroutineID)},
				}
			}
			doneCh <- suite.repo.InsertLogsBatch(ctx, logs)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		err := <-doneCh
		assert.NoError(t, err, "Concurrent insert failed")
	}

	// Wait for data availability
	time.Sleep(500 * time.Millisecond)

	// Verify all data was inserted
	queryReq := &request.LogQueryRequest{
		Dataset:   "test-dataset",
		Namespace: "concurrent-test",
		StartTime: &[]time.Time{baseTime.Add(-1 * time.Minute)}[0],
		EndTime:   &[]time.Time{baseTime.Add(1 * time.Hour)}[0],
		PageSize:  200,
	}

	results, total, err := suite.repo.QueryLogs(ctx, queryReq)
	require.NoError(t, err)
	expectedMinRows := numGoroutines * logsPerGoroutine
	assert.GreaterOrEqual(t, total, expectedMinRows)
	assert.GreaterOrEqual(t, len(results), expectedMinRows)
}

// testErrorHandling tests various error conditions
func testErrorHandling(t *testing.T, suite *IntegrationTestSuite) {
	ctx := context.Background()

	t.Run("invalid query request", func(t *testing.T) {
		// Query without required dataset
		invalidReq := &request.LogQueryRequest{
			PageSize: 100,
		}

		_, _, err := suite.repo.QueryLogs(ctx, invalidReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dataset is required")
	})

	t.Run("invalid log entry", func(t *testing.T) {
		// Insert log without required fields
		invalidLog := &chModel.LogEntry{
			Content: "test without dataset",
		}

		err := suite.repo.InsertLog(ctx, invalidLog)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dataset is required")
	})

	t.Run("context timeout", func(t *testing.T) {
		// Create context with very short timeout
		timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
		defer cancel()

		// This should timeout
		queryReq := &request.LogQueryRequest{
			Dataset:  "test-dataset",
			PageSize: 100,
		}

		_, _, err := suite.repo.QueryLogs(timeoutCtx, queryReq)
		assert.Error(t, err)
	})
}

// BenchmarkIntegrationInsert benchmarks insertion performance
func BenchmarkIntegrationInsert(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping integration benchmark in short mode")
	}

	suite, cleanup := setupIntegrationTest(&testing.T{})
	defer cleanup()

	ctx := context.Background()
	baseTime := time.Now().Truncate(time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logEntry := &chModel.LogEntry{
			Timestamp:     baseTime.Add(time.Duration(i) * time.Millisecond),
			Dataset:       "benchmark-dataset",
			Content:       fmt.Sprintf("Benchmark log entry %d", i),
			Severity:      "INFO",
			ContainerName: "benchmark-container",
			HostIP:        "192.168.255.1",
			K8sNamespace:  "benchmark",
			K8sPodName:    fmt.Sprintf("benchmark-pod-%d", i),
			Tags:          map[string]string{"benchmark": "true"},
		}

		err := suite.repo.InsertLog(ctx, logEntry)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkIntegrationBatchInsert benchmarks batch insertion performance
func BenchmarkIntegrationBatchInsert(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping integration benchmark in short mode")
	}

	suite, cleanup := setupIntegrationTest(&testing.T{})
	defer cleanup()

	ctx := context.Background()
	batchSize := 100

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logs := make([]chModel.LogEntry, batchSize)
		baseTime := time.Now().Truncate(time.Second)

		for j := 0; j < batchSize; j++ {
			logs[j] = chModel.LogEntry{
				Timestamp:     baseTime.Add(time.Duration(j) * time.Millisecond),
				Dataset:       "benchmark-dataset",
				Content:       fmt.Sprintf("Batch benchmark log %d-%d", i, j),
				Severity:      "INFO",
				ContainerName: "benchmark-batch-container",
				HostIP:        "192.168.255.2",
				K8sNamespace:  "benchmark-batch",
				K8sPodName:    fmt.Sprintf("benchmark-batch-pod-%d", j),
				Tags:          map[string]string{"benchmark": "batch"},
			}
		}

		err := suite.repo.InsertLogsBatch(ctx, logs)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ReportMetric(float64(batchSize), "logs/op")
}
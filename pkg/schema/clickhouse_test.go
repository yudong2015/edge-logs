package schema

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	clickhousecontainer "github.com/testcontainers/testcontainers-go/modules/clickhouse"
)

// TestClickHouseSchema validates the APO production schema implementation
// This integration test verifies all story acceptance criteria are met
func TestClickHouseSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Setup ClickHouse test container
	clickHouseContainer, err := clickhousecontainer.RunContainer(ctx,
		testcontainers.WithImage("clickhouse/clickhouse-server:23.8"),
		clickhousecontainer.WithUsername("default"),
		clickhousecontainer.WithPassword(""),
		clickhousecontainer.WithDatabase("edge_logs"),
		clickhousecontainer.WithInitScripts("../../sqlscripts/clickhouse/01_tables.sql"),
	)
	require.NoError(t, err)
	defer func() {
		if err := clickHouseContainer.Terminate(ctx); err != nil {
			log.Fatalf("failed to terminate container: %s", err)
		}
	}()

	// Get connection details
	connectionHost, err := clickHouseContainer.Host(ctx)
	require.NoError(t, err)

	connectionPort, err := clickHouseContainer.MappedPort(ctx, "9000/tcp")
	require.NoError(t, err)

	// Create ClickHouse connection
	conn := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", connectionHost, connectionPort.Port())},
		Auth: clickhouse.Auth{
			Database: "edge_logs",
			Username: "default",
			Password: "",
		},
	})
	require.NoError(t, conn.Ping())
	defer conn.Close()

	// Run schema validation tests
	t.Run("ValidateTableStructure", func(t *testing.T) {
		testValidateTableStructure(t, conn)
	})

	t.Run("ValidateIndexes", func(t *testing.T) {
		testValidateIndexes(t, conn)
	})

	t.Run("ValidatePartitioning", func(t *testing.T) {
		testValidatePartitioning(t, conn)
	})

	t.Run("ValidateCompression", func(t *testing.T) {
		testValidateCompression(t, conn)
	})

	t.Run("ValidateTTL", func(t *testing.T) {
		testValidateTTL(t, conn)
	})

	t.Run("ValidateDataIsolation", func(t *testing.T) {
		testValidateDataIsolation(t, conn)
	})

	t.Run("ValidatePerformanceIndexes", func(t *testing.T) {
		testValidatePerformanceIndexes(t, conn)
	})

	t.Run("ValidateILogtailFieldMappings", func(t *testing.T) {
		testValidateILogtailFieldMappings(t, conn)
	})
}

// testValidateTableStructure verifies main logs table structure matches APO specifications
func testValidateTableStructure(t *testing.T, conn *sql.DB) {
	query := `
		SELECT name, type, default_kind
		FROM system.columns
		WHERE table = 'logs' AND database = 'edge_logs'
		ORDER BY position
	`

	rows, err := conn.Query(query)
	require.NoError(t, err)
	defer rows.Close()

	expectedColumns := map[string]string{
		"timestamp":          "DateTime64(9)",
		"dataset":            "LowCardinality(String)",
		"content":            "String",
		"severity":           "LowCardinality(String)",
		"container_id":       "String",
		"container_name":     "LowCardinality(String)",
		"pid":                "String",
		"host_ip":            "LowCardinality(String)",
		"host_name":          "LowCardinality(String)",
		"k8s_namespace_name": "LowCardinality(String)",
		"k8s_pod_name":       "LowCardinality(String)",
		"k8s_pod_uid":        "String",
		"k8s_node_name":      "LowCardinality(String)",
		"tags":               "Map(String, String)",
	}

	actualColumns := make(map[string]string)
	for rows.Next() {
		var name, dataType, defaultKind string
		err := rows.Scan(&name, &dataType, &defaultKind)
		require.NoError(t, err)
		actualColumns[name] = dataType
	}

	// Verify all expected columns exist with correct types
	for expectedCol, expectedType := range expectedColumns {
		actualType, exists := actualColumns[expectedCol]
		assert.True(t, exists, "Column %s should exist", expectedCol)
		assert.Equal(t, expectedType, actualType, "Column %s should have type %s", expectedCol, expectedType)
	}
}

// testValidateIndexes verifies performance indexes are created correctly
func testValidateIndexes(t *testing.T, conn *sql.DB) {
	query := `
		SELECT name, type, granularity
		FROM system.data_skipping_indices
		WHERE table = 'logs' AND database = 'edge_logs'
		ORDER BY name
	`

	rows, err := conn.Query(query)
	require.NoError(t, err)
	defer rows.Close()

	expectedIndexes := map[string]string{
		"idx_content": "tokenbf_v1",
		"idx_tags":    "bloom_filter",
	}

	actualIndexes := make(map[string]string)
	for rows.Next() {
		var name, indexType string
		var granularity int
		err := rows.Scan(&name, &indexType, &granularity)
		require.NoError(t, err)
		actualIndexes[name] = indexType
	}

	// Verify required indexes exist with correct types
	for expectedIndex, expectedType := range expectedIndexes {
		actualType, exists := actualIndexes[expectedIndex]
		assert.True(t, exists, "Index %s should exist", expectedIndex)
		assert.Equal(t, expectedType, actualType, "Index %s should have type %s", expectedIndex, expectedType)
	}
}

// testValidatePartitioning verifies partitioning strategy (dataset, date)
func testValidatePartitioning(t *testing.T, conn *sql.DB) {
	// Insert test data to create partition
	insertQuery := `
		INSERT INTO logs (
			timestamp, dataset, content, severity, container_id, container_name, pid,
			host_ip, host_name, k8s_namespace_name, k8s_pod_name, k8s_pod_uid, k8s_node_name, tags
		) VALUES (
			now64(9), 'test-dataset', 'test message', 'info', 'c123', 'test-container', '1234',
			'192.168.1.1', 'test-host', 'test-namespace', 'test-pod', 'test-uid', 'test-node',
			map('cluster', 'test-cluster', 'region', 'test-region')
		)
	`
	_, err := conn.Exec(insertQuery)
	require.NoError(t, err)

	// Check partition structure
	query := `
		SELECT DISTINCT partition
		FROM system.parts
		WHERE table = 'logs' AND database = 'edge_logs' AND active = 1
	`

	rows, err := conn.Query(query)
	require.NoError(t, err)
	defer rows.Close()

	var foundPartition bool
	for rows.Next() {
		var partition string
		err := rows.Scan(&partition)
		require.NoError(t, err)
		// Partition should be in format: dataset-YYYYMM
		assert.Contains(t, partition, "test-dataset", "Partition should contain dataset name")
		foundPartition = true
	}
	assert.True(t, foundPartition, "Should have at least one partition created")
}

// testValidateCompression verifies Delta+ZSTD compression is applied
func testValidateCompression(t *testing.T, conn *sql.DB) {
	query := `
		SELECT name, compression_codec
		FROM system.columns
		WHERE table = 'logs' AND database = 'edge_logs' AND compression_codec != ''
		ORDER BY name
	`

	rows, err := conn.Query(query)
	require.NoError(t, err)
	defer rows.Close()

	compressionFound := false
	timestampHasDelta := false

	for rows.Next() {
		var name, codec string
		err := rows.Scan(&name, &codec)
		require.NoError(t, err)

		// All columns should have ZSTD compression
		assert.Contains(t, codec, "ZSTD", "Column %s should have ZSTD compression", name)
		compressionFound = true

		// Timestamp should have Delta compression
		if name == "timestamp" {
			assert.Contains(t, codec, "Delta", "Timestamp should have Delta compression")
			timestampHasDelta = true
		}
	}

	assert.True(t, compressionFound, "Should find compression codecs")
	assert.True(t, timestampHasDelta, "Timestamp should have Delta compression")
}

// testValidateTTL verifies 30-day TTL configuration
func testValidateTTL(t *testing.T, conn *sql.DB) {
	query := `
		SELECT ttl_info.expression
		FROM system.tables
		WHERE table = 'logs' AND database = 'edge_logs'
	`

	var ttlExpression sql.NullString
	err := conn.QueryRow(query).Scan(&ttlExpression)
	require.NoError(t, err)

	assert.True(t, ttlExpression.Valid, "TTL should be configured")
	assert.Contains(t, ttlExpression.String, "30", "TTL should be 30 days")
	assert.Contains(t, ttlExpression.String, "DAY", "TTL should be in days")
}

// testValidateDataIsolation verifies dataset-based data isolation
func testValidateDataIsolation(t *testing.T, conn *sql.DB) {
	// Insert test data for multiple datasets
	insertQueries := []string{
		`INSERT INTO logs (timestamp, dataset, content, severity, container_id, container_name, pid, host_ip, host_name, k8s_namespace_name, k8s_pod_name, k8s_pod_uid, k8s_node_name, tags) VALUES (now64(9), 'dataset-1', 'message 1', 'info', 'c1', 'container1', '1', '192.168.1.1', 'host1', 'ns1', 'pod1', 'uid1', 'node1', map('cluster', 'cluster1'))`,
		`INSERT INTO logs (timestamp, dataset, content, severity, container_id, container_name, pid, host_ip, host_name, k8s_namespace_name, k8s_pod_name, k8s_pod_uid, k8s_node_name, tags) VALUES (now64(9), 'dataset-2', 'message 2', 'info', 'c2', 'container2', '2', '192.168.1.2', 'host2', 'ns2', 'pod2', 'uid2', 'node2', map('cluster', 'cluster2'))`,
	}

	for _, query := range insertQueries {
		_, err := conn.Exec(query)
		require.NoError(t, err)
	}

	// Verify dataset isolation works
	query := `
		SELECT dataset, count() as count
		FROM logs
		WHERE dataset IN ('dataset-1', 'dataset-2')
		GROUP BY dataset
		ORDER BY dataset
	`

	rows, err := conn.Query(query)
	require.NoError(t, err)
	defer rows.Close()

	datasets := make(map[string]int)
	for rows.Next() {
		var dataset string
		var count int
		err := rows.Scan(&dataset, &count)
		require.NoError(t, err)
		datasets[dataset] = count
	}

	assert.Equal(t, 1, datasets["dataset-1"], "Dataset-1 should have 1 record")
	assert.Equal(t, 1, datasets["dataset-2"], "Dataset-2 should have 1 record")
}

// testValidatePerformanceIndexes verifies tokenbf_v1 and bloom_filter indexes work
func testValidatePerformanceIndexes(t *testing.T, conn *sql.DB) {
	// Insert test data with searchable content
	insertQuery := `
		INSERT INTO logs (
			timestamp, dataset, content, severity, container_id, container_name, pid,
			host_ip, host_name, k8s_namespace_name, k8s_pod_name, k8s_pod_uid, k8s_node_name, tags
		) VALUES (
			now64(9), 'perf-test', 'database connection established successfully', 'info',
			'c123', 'db-service', '1234', '192.168.1.100', 'db-host',
			'production', 'db-pod-123', 'db-uid-456', 'db-node',
			map('cluster', 'production-cluster', 'region', 'us-west')
		)
	`
	_, err := conn.Exec(insertQuery)
	require.NoError(t, err)

	// Test tokenbf_v1 index (content search)
	contentQuery := `
		SELECT count()
		FROM logs
		WHERE hasToken(content, 'database') AND dataset = 'perf-test'
	`

	var contentCount int
	err = conn.QueryRow(contentQuery).Scan(&contentCount)
	require.NoError(t, err)
	assert.Greater(t, contentCount, 0, "Content search should find records")

	// Test bloom_filter index (tags search)
	tagsQuery := `
		SELECT count()
		FROM logs
		WHERE tags['cluster'] = 'production-cluster' AND dataset = 'perf-test'
	`

	var tagsCount int
	err = conn.QueryRow(tagsQuery).Scan(&tagsCount)
	require.NoError(t, err)
	assert.Greater(t, tagsCount, 0, "Tags search should find records")
}

// testValidateILogtailFieldMappings verifies all required iLogtail fields are present
func testValidateILogtailFieldMappings(t *testing.T, conn *sql.DB) {
	// Required iLogtail field mappings from story specification
	requiredFields := []string{
		"timestamp",          // Log timestamp
		"dataset",            // ENV: LOG_DATASET
		"content",            // body (log message content)
		"severity",           // level (log level)
		"container_id",       // _container_id_
		"container_name",     // _container_name_
		"k8s_namespace_name", // k8s.namespace.name
		"k8s_pod_name",       // k8s.pod.name
		"k8s_pod_uid",        // k8s.pod.uid
		"k8s_node_name",      // k8s.node.name
		"tags",               // map for cluster/region via ENV
	}

	query := `
		SELECT name
		FROM system.columns
		WHERE table = 'logs' AND database = 'edge_logs'
		ORDER BY name
	`

	rows, err := conn.Query(query)
	require.NoError(t, err)
	defer rows.Close()

	actualFields := make(map[string]bool)
	for rows.Next() {
		var fieldName string
		err := rows.Scan(&fieldName)
		require.NoError(t, err)
		actualFields[fieldName] = true
	}

	// Verify all required iLogtail fields are present
	for _, field := range requiredFields {
		assert.True(t, actualFields[field], "Required iLogtail field %s should exist", field)
	}
}
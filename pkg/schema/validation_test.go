package schema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchemaFilesExist validates that all required schema files exist
func TestSchemaFilesExist(t *testing.T) {
	schemaFiles := []string{
		"../../sqlscripts/clickhouse/01_tables.sql",
		"../../sqlscripts/clickhouse/02_indexes.sql",
		"../../sqlscripts/clickhouse/test_data.sql",
	}

	for _, file := range schemaFiles {
		_, err := os.Stat(file)
		assert.NoError(t, err, "Schema file %s should exist", file)
	}
}

// TestMainTableSchema validates the main logs table definition matches APO specifications
func TestMainTableSchema(t *testing.T) {
	schemaFile := "../../sqlscripts/clickhouse/01_tables.sql"
	content, err := os.ReadFile(schemaFile)
	require.NoError(t, err)

	schemaContent := string(content)

	// Verify APO required fields are present
	requiredFields := []string{
		"timestamp          DateTime64(9)",
		"dataset            LowCardinality(String)",
		"content            String",
		"severity           LowCardinality(String)",
		"container_id       String",
		"container_name     LowCardinality(String)",
		"pid                String",
		"host_ip            LowCardinality(String)",
		"host_name          LowCardinality(String)",
		"k8s_namespace_name LowCardinality(String)",
		"k8s_pod_name       LowCardinality(String)",
		"k8s_pod_uid        String",
		"k8s_node_name      LowCardinality(String)",
		"tags               Map(String, String)",
	}

	for _, field := range requiredFields {
		assert.Contains(t, schemaContent, field, "Schema should contain required field: %s", field)
	}

	// Verify APO compression patterns
	compressionPatterns := []string{
		"CODEC(Delta(8), ZSTD(1))", // timestamp
		"CODEC(ZSTD(1))",           // other fields
	}

	for _, pattern := range compressionPatterns {
		assert.Contains(t, schemaContent, pattern, "Schema should contain compression pattern: %s", pattern)
	}

	// Verify APO engine and configuration
	enginePatterns := []string{
		"ENGINE = MergeTree()",
		"PARTITION BY (dataset, toDate(timestamp))",
		"ORDER BY (dataset, host_ip, timestamp)",
		"TTL timestamp + INTERVAL 30 DAY DELETE",
		"index_granularity = 8192",
		"ttl_only_drop_parts = 1",
	}

	for _, pattern := range enginePatterns {
		assert.Contains(t, schemaContent, pattern, "Schema should contain engine pattern: %s", pattern)
	}

	// Verify APO performance indexes
	indexPatterns := []string{
		"INDEX idx_content content TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 1",
		"INDEX idx_tags tags TYPE bloom_filter GRANULARITY 1",
	}

	for _, pattern := range indexPatterns {
		assert.Contains(t, schemaContent, pattern, "Schema should contain index pattern: %s", pattern)
	}
}

// TestIndexesSchema validates the performance indexes file
func TestIndexesSchema(t *testing.T) {
	indexFile := "../../sqlscripts/clickhouse/02_indexes.sql"
	content, err := os.ReadFile(indexFile)
	require.NoError(t, err)

	indexContent := string(content)

	// Verify APO performance indexes are present
	requiredIndexes := []string{
		"proj_k8s_namespace",
		"proj_k8s_pod",
		"proj_severity",
		"idx_container_name_set",
		"idx_k8s_node_set",
		"idx_host_ip_set",
		"idx_timestamp_minmax",
		"idx_k8s_pod_pattern",
		"idx_tags_bloom",
		"idx_content_ngram",
		"idx_severity_set",
	}

	for _, index := range requiredIndexes {
		assert.Contains(t, indexContent, index, "Indexes should contain: %s", index)
	}

	// Verify index types are correct
	indexTypes := map[string]string{
		"tokenbf_v1":   "idx_k8s_pod_pattern",
		"bloom_filter": "idx_tags_bloom",
		"set":          "idx_container_name_set",
		"minmax":       "idx_timestamp_minmax",
		"ngrambf_v1":   "idx_content_ngram",
	}

	for indexType, indexName := range indexTypes {
		assert.Contains(t, indexContent, indexType, "Should contain index type %s for %s", indexType, indexName)
	}
}

// TestDistributedTableSupport validates distributed table configuration exists
func TestDistributedTableSupport(t *testing.T) {
	schemaFile := "../../sqlscripts/clickhouse/01_tables.sql"
	content, err := os.ReadFile(schemaFile)
	require.NoError(t, err)

	schemaContent := string(content)

	// Verify distributed table configuration is present (even if commented)
	distributedPatterns := []string{
		"logs_distributed",
		"Distributed(",
		"edge_logs_cluster",
	}

	for _, pattern := range distributedPatterns {
		assert.Contains(t, schemaContent, pattern, "Schema should contain distributed pattern: %s", pattern)
	}
}

// TestILogtailFieldMappings validates all required iLogtail fields are present
func TestILogtailFieldMappings(t *testing.T) {
	schemaFile := "../../sqlscripts/clickhouse/01_tables.sql"
	content, err := os.ReadFile(schemaFile)
	require.NoError(t, err)

	schemaContent := string(content)

	// iLogtail field mappings as specified in story Dev Notes
	iLogtailMappings := map[string]string{
		"timestamp":          "timestamp", // Log timestamp
		"dataset":            "ENV: LOG_DATASET",
		"content":            "body", // Log message content
		"severity":           "level", // Log level
		"container_id":       "_container_id_",
		"container_name":     "_container_name_",
		"k8s_namespace_name": "k8s.namespace.name",
		"k8s_pod_name":       "k8s.pod.name",
		"k8s_pod_uid":        "k8s.pod.uid",
		"k8s_node_name":      "k8s.node.name",
		"tags":               "ENV: CLUSTER_NAME, REGION_NAME",
	}

	for clickhouseField := range iLogtailMappings {
		// Check that the field exists in the schema
		assert.Contains(t, schemaContent, clickhouseField, "Schema should contain iLogtail field: %s", clickhouseField)
	}
}

// TestTestDataFile validates the test data file contains proper validation queries
func TestTestDataFile(t *testing.T) {
	testFile := "../../sqlscripts/clickhouse/test_data.sql"
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)

	testContent := string(content)

	// Verify test data contains validation queries
	requiredTests := []string{
		"Schema Validation",
		"Insert Sample Data",
		"Partition Validation",
		"Index Validation",
		"Compression Validation",
		"Time Range Query Performance Test",
		"Full-text Search Test",
		"Tags Query Test",
		"Aggregation Query Test",
		"TTL Configuration Validation",
		"Storage Efficiency Test",
		"Dataset Isolation Test",
		"Performance Baseline Query",
	}

	for _, test := range requiredTests {
		assert.Contains(t, testContent, test, "Test data should contain test: %s", test)
	}

	// Verify sample data includes all required fields
	requiredSampleFields := []string{
		"timestamp",
		"dataset",
		"content",
		"severity",
		"container_id",
		"container_name",
		"pid",
		"host_ip",
		"host_name",
		"k8s_namespace_name",
		"k8s_pod_name",
		"k8s_pod_uid",
		"k8s_node_name",
		"tags",
	}

	for _, field := range requiredSampleFields {
		assert.Contains(t, testContent, field, "Test data should include field in sample data: %s", field)
	}
}

// TestSchemaCompliance validates the schema meets all story acceptance criteria
func TestSchemaCompliance(t *testing.T) {
	t.Run("AC1: Logs table created with MergeTree engine", func(t *testing.T) {
		schemaFile := "../../sqlscripts/clickhouse/01_tables.sql"
		content, err := os.ReadFile(schemaFile)
		require.NoError(t, err)

		schemaContent := string(content)
		assert.Contains(t, schemaContent, "CREATE TABLE IF NOT EXISTS logs", "Should create logs table")
		assert.Contains(t, schemaContent, "ENGINE = MergeTree()", "Should use MergeTree engine")
	})

	t.Run("AC2: Proper partitioning by dataset and date", func(t *testing.T) {
		schemaFile := "../../sqlscripts/clickhouse/01_tables.sql"
		content, err := os.ReadFile(schemaFile)
		require.NoError(t, err)

		schemaContent := string(content)
		assert.Contains(t, schemaContent, "PARTITION BY (dataset, toDate(timestamp))", "Should partition by dataset and date")
		assert.Contains(t, schemaContent, "ORDER BY (dataset, host_ip, timestamp)", "Should order by dataset, host_ip, timestamp")
	})

	t.Run("AC3: Performance indexes created", func(t *testing.T) {
		schemaFile := "../../sqlscripts/clickhouse/01_tables.sql"
		content, err := os.ReadFile(schemaFile)
		require.NoError(t, err)

		schemaContent := string(content)
		assert.Contains(t, schemaContent, "tokenbf_v1", "Should have tokenbf_v1 index for content search")
		assert.Contains(t, schemaContent, "bloom_filter", "Should have bloom_filter index for tags")
	})

	t.Run("AC4: 30-day TTL configured", func(t *testing.T) {
		schemaFile := "../../sqlscripts/clickhouse/01_tables.sql"
		content, err := os.ReadFile(schemaFile)
		require.NoError(t, err)

		schemaContent := string(content)
		assert.Contains(t, schemaContent, "TTL timestamp + INTERVAL 30 DAY DELETE", "Should have 30-day TTL")
		assert.Contains(t, schemaContent, "ttl_only_drop_parts = 1", "Should use ttl_only_drop_parts")
	})

	t.Run("AC5: Delta+ZSTD compression configured", func(t *testing.T) {
		schemaFile := "../../sqlscripts/clickhouse/01_tables.sql"
		content, err := os.ReadFile(schemaFile)
		require.NoError(t, err)

		schemaContent := string(content)
		assert.Contains(t, schemaContent, "CODEC(Delta(8), ZSTD(1))", "Should have Delta+ZSTD compression on timestamp")

		// Count occurrences of ZSTD compression
		zstdCount := strings.Count(schemaContent, "CODEC(ZSTD(1))")
		assert.Greater(t, zstdCount, 10, "Should have ZSTD compression on multiple fields")
	})
}

// TestSchemaFilesStructure validates proper file organization
func TestSchemaFilesStructure(t *testing.T) {
	// Check sqlscripts directory structure
	baseDir := "../../sqlscripts/clickhouse"

	expectedFiles := []string{
		"01_tables.sql",
		"02_indexes.sql",
		"test_data.sql",
	}

	for _, file := range expectedFiles {
		filePath := filepath.Join(baseDir, file)
		_, err := os.Stat(filePath)
		assert.NoError(t, err, "Expected file should exist: %s", filePath)
	}

	// Verify files are not empty
	for _, file := range expectedFiles {
		filePath := filepath.Join(baseDir, file)
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Greater(t, len(content), 100, "File should not be empty: %s", file)
	}
}

// TestSQLSyntaxBasic performs basic SQL syntax validation
func TestSQLSyntaxBasic(t *testing.T) {
	schemaFiles := []string{
		"../../sqlscripts/clickhouse/01_tables.sql",
		"../../sqlscripts/clickhouse/02_indexes.sql",
		"../../sqlscripts/clickhouse/test_data.sql",
	}

	for _, file := range schemaFiles {
		t.Run(file, func(t *testing.T) {
			content, err := os.ReadFile(file)
			require.NoError(t, err)

			// Basic syntax checks
			lines := strings.Split(string(content), "\n")

			openParens := 0
			for lineNum, line := range lines {
				// Skip comments
				if strings.HasPrefix(strings.TrimSpace(line), "--") || strings.TrimSpace(line) == "" {
					continue
				}

				// Check parentheses balance
				openParens += strings.Count(line, "(") - strings.Count(line, ")")

				// Parentheses should never go negative
				assert.GreaterOrEqual(t, openParens, 0, "Unbalanced parentheses at line %d: %s", lineNum+1, line)
			}

			// All parentheses should be closed at the end
			assert.Equal(t, 0, openParens, "Unbalanced parentheses in file %s", file)
		})
	}
}
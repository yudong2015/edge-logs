package clickhouse

import (
	"fmt"
	"testing"
	"time"

	chModel "github.com/outpostos/edge-logs/pkg/model/clickhouse"
	"github.com/outpostos/edge-logs/pkg/model/request"
)

// BenchmarkQueryBuilder benchmarks query building operations
func BenchmarkQueryBuilder(b *testing.B) {
	b.Run("BuildLogQuery-Simple", benchmarkBuildLogQuerySimple)
	b.Run("BuildLogQuery-Complex", benchmarkBuildLogQueryComplex)
	b.Run("BuildCountQuery", benchmarkBuildCountQuery)
	b.Run("BuildInsertQuery", benchmarkBuildInsertQuery)
	b.Run("ValidateQuery", benchmarkValidateQuery)
}

// benchmarkBuildLogQuerySimple benchmarks simple query building
func benchmarkBuildLogQuerySimple(b *testing.B) {
	req := &request.LogQueryRequest{
		Dataset:   "benchmark-dataset",
		PageSize:  100,
		OrderBy:   "timestamp",
		Direction: "desc",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qb := NewQueryBuilder()
		_, _, err := qb.BuildLogQuery(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// benchmarkBuildLogQueryComplex benchmarks complex query building with all filters
func benchmarkBuildLogQueryComplex(b *testing.B) {
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	req := &request.LogQueryRequest{
		Dataset:       "benchmark-dataset",
		StartTime:     &startTime,
		EndTime:       &endTime,
		Namespace:     "production",
		PodName:       "api-pod-123",
		NodeName:      "node-1",
		HostIP:        "192.168.1.100",
		HostName:      "api-server",
		ContainerName: "api-container",
		Filter:        "error",
		Severity:      "ERROR",
		Tags:          map[string]string{"cluster": "prod", "region": "us-east-1", "service": "api"},
		Page:          2,
		PageSize:      50,
		OrderBy:       "timestamp",
		Direction:     "asc",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qb := NewQueryBuilder()
		_, _, err := qb.BuildLogQuery(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// benchmarkBuildCountQuery benchmarks count query building
func benchmarkBuildCountQuery(b *testing.B) {
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	req := &request.LogQueryRequest{
		Dataset:   "benchmark-dataset",
		StartTime: &startTime,
		EndTime:   &endTime,
		Namespace: "production",
		Filter:    "error",
		Severity:  "ERROR",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qb := NewQueryBuilder()
		_, _, err := qb.BuildCountQuery(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// benchmarkBuildInsertQuery benchmarks insert query building
func benchmarkBuildInsertQuery(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qb := NewQueryBuilder()
		_, err := qb.BuildInsertQuery()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// benchmarkValidateQuery benchmarks query validation
func benchmarkValidateQuery(b *testing.B) {
	req := &request.LogQueryRequest{
		Dataset:   "benchmark-dataset",
		PageSize:  100,
		OrderBy:   "timestamp",
		Direction: "desc",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qb := NewQueryBuilder()
		err := qb.ValidateQuery(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValidation benchmarks various validation operations
func BenchmarkValidation(b *testing.B) {
	b.Run("LogEntryValidation", benchmarkLogEntryValidation)
	b.Run("RequestValidation", benchmarkRequestValidation)
}

// benchmarkLogEntryValidation benchmarks log entry validation
func benchmarkLogEntryValidation(b *testing.B) {
	repo := &ClickHouseRepository{}
	logEntry := &chModel.LogEntry{
		Timestamp:     time.Now(),
		Dataset:       "benchmark-dataset",
		Content:       "Benchmark log message with detailed content for validation testing",
		Severity:      "INFO",
		ContainerID:   "container-12345",
		ContainerName: "benchmark-container",
		PID:           "1234",
		HostIP:        "192.168.1.100",
		HostName:      "benchmark-host",
		K8sNamespace:  "benchmark",
		K8sPodName:    "benchmark-pod-123",
		K8sPodUID:     "pod-uid-12345",
		K8sNodeName:   "node-1",
		Tags:          map[string]string{"cluster": "benchmark", "region": "us-east-1", "service": "api"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := repo.validateLogEntry(logEntry)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// benchmarkRequestValidation benchmarks request validation
func benchmarkRequestValidation(b *testing.B) {
	req := &request.LogQueryRequest{
		Dataset:   "benchmark-dataset",
		PageSize:  100,
		OrderBy:   "timestamp",
		Direction: "desc",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := req.Validate()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkErrorMapping benchmarks error mapping operations
func BenchmarkErrorMapping(b *testing.B) {
	testError := fmt.Errorf("connection refused to host")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repoErr := MapClickHouseError(testError, "benchmark_op")
		if repoErr == nil {
			b.Fatal("Expected non-nil repository error")
		}
	}
}

// BenchmarkMetricsCollection benchmarks metrics collection operations
func BenchmarkMetricsCollection(b *testing.B) {
	b.Run("QueryMetricsCollector", benchmarkQueryMetricsCollector)
	b.Run("MetricsRecording", benchmarkMetricsRecording)
}

// benchmarkQueryMetricsCollector benchmarks query metrics collection
func benchmarkQueryMetricsCollector(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector := NewQueryMetricsCollector("benchmark-dataset", "search", "{\"test\":true}")
		// Simulate query execution time
		time.Sleep(1 * time.Microsecond)
		collector.Finish(nil, 100)
	}
}

// benchmarkMetricsRecording benchmarks metrics recording (without actual database)
func benchmarkMetricsRecording(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate metrics recording preparation
		collector := NewQueryMetricsCollector("benchmark-dataset", "search", "{\"benchmark\":true}")
		duration := time.Duration(i%100) * time.Millisecond
		rowsReturned := uint64(i % 1000)

		// This would normally record to database, but we're just benchmarking the setup
		_ = collector
		_ = duration
		_ = rowsReturned
	}
}

// BenchmarkConcurrentOperations benchmarks concurrent access patterns
func BenchmarkConcurrentOperations(b *testing.B) {
	b.Run("ConcurrentQueryBuilding", benchmarkConcurrentQueryBuilding)
	b.Run("ConcurrentValidation", benchmarkConcurrentValidation)
}

// benchmarkConcurrentQueryBuilding benchmarks concurrent query building
func benchmarkConcurrentQueryBuilding(b *testing.B) {
	req := &request.LogQueryRequest{
		Dataset:   "benchmark-dataset",
		PageSize:  100,
		OrderBy:   "timestamp",
		Direction: "desc",
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			qb := NewQueryBuilder()
			_, _, err := qb.BuildLogQuery(req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// benchmarkConcurrentValidation benchmarks concurrent validation
func benchmarkConcurrentValidation(b *testing.B) {
	repo := &ClickHouseRepository{}
	logEntry := &chModel.LogEntry{
		Timestamp:     time.Now(),
		Dataset:       "benchmark-dataset",
		Content:       "Concurrent benchmark log message",
		Severity:      "INFO",
		ContainerName: "concurrent-benchmark-container",
		HostIP:        "192.168.1.100",
		K8sNamespace:  "benchmark",
		K8sPodName:    "concurrent-pod",
		Tags:          map[string]string{"test": "concurrent"},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := repo.validateLogEntry(logEntry)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("QueryBuilderAllocation", benchmarkQueryBuilderAllocation)
	b.Run("LogEntryAllocation", benchmarkLogEntryAllocation)
	b.Run("ErrorAllocation", benchmarkErrorAllocation)
}

// benchmarkQueryBuilderAllocation benchmarks memory allocation in query building
func benchmarkQueryBuilderAllocation(b *testing.B) {
	req := &request.LogQueryRequest{
		Dataset:   "benchmark-dataset",
		PageSize:  100,
		OrderBy:   "timestamp",
		Direction: "desc",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qb := NewQueryBuilder()
		_, _, err := qb.BuildLogQuery(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// benchmarkLogEntryAllocation benchmarks memory allocation for log entries
func benchmarkLogEntryAllocation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logEntry := &chModel.LogEntry{
			Timestamp:     time.Now(),
			Dataset:       "benchmark-dataset",
			Content:       fmt.Sprintf("Allocation benchmark log %d", i),
			Severity:      "INFO",
			ContainerID:   fmt.Sprintf("container-%d", i),
			ContainerName: "allocation-benchmark",
			PID:           fmt.Sprintf("%d", 1000+i),
			HostIP:        "192.168.1.100",
			HostName:      "allocation-host",
			K8sNamespace:  "benchmark",
			K8sPodName:    fmt.Sprintf("allocation-pod-%d", i),
			K8sPodUID:     fmt.Sprintf("uid-%d", i),
			K8sNodeName:   "allocation-node",
			Tags:          map[string]string{"benchmark": "allocation", "index": fmt.Sprintf("%d", i)},
		}
		_ = logEntry
	}
}

// benchmarkErrorAllocation benchmarks memory allocation in error handling
func benchmarkErrorAllocation(b *testing.B) {
	testError := fmt.Errorf("benchmark error message")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repoErr := MapClickHouseError(testError, "benchmark_op")
		_ = repoErr
	}
}

// BenchmarkPerformanceRequirements validates performance requirements
func BenchmarkPerformanceRequirements(b *testing.B) {
	b.Run("QueryBuildingPerformance", func(b *testing.B) {
		// Requirement: Query building should complete in < 1ms
		req := &request.LogQueryRequest{
			Dataset:   "performance-test",
			PageSize:  100,
			OrderBy:   "timestamp",
			Direction: "desc",
		}

		b.ResetTimer()
		start := time.Now()
		for i := 0; i < b.N; i++ {
			qb := NewQueryBuilder()
			_, _, err := qb.BuildLogQuery(req)
			if err != nil {
				b.Fatal(err)
			}
		}
		duration := time.Since(start)
		avgDuration := duration / time.Duration(b.N)

		b.ReportMetric(avgDuration.Seconds()*1000, "ms/op")
		if avgDuration > 1*time.Millisecond {
			b.Logf("WARNING: Query building exceeded 1ms requirement: %v", avgDuration)
		}
	})

	b.Run("ValidationPerformance", func(b *testing.B) {
		// Requirement: Validation should complete in < 100μs
		repo := &ClickHouseRepository{}
		logEntry := &chModel.LogEntry{
			Timestamp:     time.Now(),
			Dataset:       "performance-test",
			Content:       "Performance test log message",
			Severity:      "INFO",
			ContainerName: "performance-container",
			K8sNamespace:  "performance",
			Tags:          map[string]string{"test": "performance"},
		}

		b.ResetTimer()
		start := time.Now()
		for i := 0; i < b.N; i++ {
			err := repo.validateLogEntry(logEntry)
			if err != nil {
				b.Fatal(err)
			}
		}
		duration := time.Since(start)
		avgDuration := duration / time.Duration(b.N)

		b.ReportMetric(avgDuration.Seconds()*1000000, "μs/op")
		if avgDuration > 100*time.Microsecond {
			b.Logf("WARNING: Validation exceeded 100μs requirement: %v", avgDuration)
		}
	})
}

// Benchmark scenarios based on iLogtail workload patterns
func BenchmarkILogtailWorkloads(b *testing.B) {
	b.Run("HighFrequencySmallBatches", func(b *testing.B) {
		// Simulate iLogtail sending small frequent batches
		batchSize := 10
		baseTime := time.Now()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logs := make([]chModel.LogEntry, batchSize)
			for j := 0; j < batchSize; j++ {
				logs[j] = chModel.LogEntry{
					Timestamp:     baseTime.Add(time.Duration(i*batchSize+j) * time.Millisecond),
					Dataset:       "ilogtail-dataset",
					Content:       fmt.Sprintf("iLogtail log batch %d entry %d", i, j),
					Severity:      "INFO",
					ContainerName: "ilogtail-container",
					HostIP:        "192.168.1.100",
					K8sNamespace:  "ilogtail",
					Tags:          map[string]string{"source": "ilogtail", "batch": fmt.Sprintf("%d", i)},
				}
			}
			// Simulate validation that would happen before insertion
			for _, log := range logs {
				repo := &ClickHouseRepository{}
				err := repo.validateLogEntry(&log)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
		b.ReportMetric(float64(batchSize), "logs/op")
	})

	b.Run("LargeBatchesLowFrequency", func(b *testing.B) {
		// Simulate iLogtail sending large infrequent batches
		batchSize := 1000
		baseTime := time.Now()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logs := make([]chModel.LogEntry, batchSize)
			for j := 0; j < batchSize; j++ {
				logs[j] = chModel.LogEntry{
					Timestamp:     baseTime.Add(time.Duration(i*batchSize+j) * time.Microsecond),
					Dataset:       "ilogtail-large-dataset",
					Content:       fmt.Sprintf("iLogtail large batch %d entry %d with more detailed content", i, j),
					Severity:      "INFO",
					ContainerName: "ilogtail-large-container",
					HostIP:        "192.168.1.100",
					K8sNamespace:  "ilogtail-large",
					Tags:          map[string]string{"source": "ilogtail", "type": "large_batch", "batch": fmt.Sprintf("%d", i)},
				}
			}
			// Validate the entire batch
			for _, log := range logs {
				repo := &ClickHouseRepository{}
				err := repo.validateLogEntry(&log)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
		b.ReportMetric(float64(batchSize), "logs/op")
	})
}
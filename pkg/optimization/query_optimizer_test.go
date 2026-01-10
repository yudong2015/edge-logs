package optimization

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/outpostos/edge-logs/pkg/model/request"
)

func TestNewQueryOptimizer(t *testing.T) {
	optimizer := NewQueryOptimizer()

	require.NotNil(t, optimizer)
	assert.True(t, optimizer.enablePrewhere)
	assert.True(t, optimizer.enableColumnPruning)
	assert.Equal(t, 100000, optimizer.maxResultRows)
	assert.Equal(t, 30*time.Second, optimizer.queryTimeout)
}

func TestOptimizeQuery(t *testing.T) {
	optimizer := NewQueryOptimizer()

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	req := &request.LogQueryRequest{
		Dataset:   "test-dataset",
		StartTime: &startTime,
		EndTime:   &endTime,
		Page:      1,
		PageSize:  100,
		Namespace: "default",
	}

	query := "SELECT * FROM logs WHERE namespace = 'default'"

	result, err := optimizer.OptimizeQuery(context.Background(), query, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.OptimizedQuery)
	assert.Equal(t, query, result.OriginalQuery)
	assert.NotNil(t, result.OptimizationsApplied)
	assert.GreaterOrEqual(t, result.EstimatedImprovement, 0.0)
	assert.LessOrEqual(t, result.EstimatedImprovement, 100.0)
}

func TestValidateQuery(t *testing.T) {
	optimizer := NewQueryOptimizer()

	tests := []struct {
		name        string
		query       string
		shouldError bool
	}{
		{"Valid SELECT", "SELECT * FROM logs WHERE namespace = 'default'", false},
		{"Dangerous DROP", "DROP TABLE logs", true},
		{"Dangerous DELETE", "DELETE FROM logs WHERE timestamp < now()", true},
		{"Missing WHERE", "SELECT * FROM logs", false}, // Warning but not error
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := optimizer.ValidateQuery(context.Background(), tt.query)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestApplyPrewhereOptimization(t *testing.T) {
	optimizer := NewQueryOptimizer()

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	req := &request.LogQueryRequest{
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	query := "SELECT * FROM logs WHERE timestamp >= now() - INTERVAL 1 HOUR"
	optimized := optimizer.applyPrewhereOptimization(query, req)

	// Should return a query (may or may not be modified depending on implementation)
	assert.NotEmpty(t, optimized)
}

func TestApplyColumnPruning(t *testing.T) {
	optimizer := NewQueryOptimizer()

	req := &request.LogQueryRequest{
		Namespace: "default",
	}

	query := "SELECT * FROM logs"
	optimized := optimizer.applyColumnPruning(query, req)

	// Should replace SELECT * with specific columns
	assert.NotEmpty(t, optimized)
	assert.NotContains(t, optimized, "SELECT *")
}

func TestApplyResultLimit(t *testing.T) {
	optimizer := NewQueryOptimizer()

	req := &request.LogQueryRequest{
		PageSize: 100,
	}

	query := "SELECT * FROM logs"
	optimized := optimizer.applyResultLimit(query, req)

	assert.NotEmpty(t, optimized)
	assert.Contains(t, optimized, "LIMIT")
}

func TestSetMaxResultRows(t *testing.T) {
	optimizer := NewQueryOptimizer()

	optimizer.SetMaxResultRows(50000)
	assert.Equal(t, 50000, optimizer.maxResultRows)
}

func TestSetQueryTimeout(t *testing.T) {
	optimizer := NewQueryOptimizer()

	newTimeout := 45 * time.Second
	optimizer.SetQueryTimeout(newTimeout)
	assert.Equal(t, newTimeout, optimizer.queryTimeout)
}

func TestEnablePrewhere(t *testing.T) {
	optimizer := NewQueryOptimizer()

	optimizer.EnablePrewhere(false)
	assert.False(t, optimizer.enablePrewhere)

	optimizer.EnablePrewhere(true)
	assert.True(t, optimizer.enablePrewhere)
}

func TestEnableColumnPruning(t *testing.T) {
	optimizer := NewQueryOptimizer()

	optimizer.EnableColumnPruning(false)
	assert.False(t, optimizer.enableColumnPruning)

	optimizer.EnableColumnPruning(true)
	assert.True(t, optimizer.enableColumnPruning)
}

func TestEstimateImprovement(t *testing.T) {
	optimizer := NewQueryOptimizer()

	tests := []struct {
		name              string
		optimizations     []string
		minImprovement    float64
		maxImprovement    float64
	}{
		{"No optimizations", []string{}, 0.0, 0.0},
		{"Single optimization", []string{"PREWHERE optimization"}, 10.0, 20.0},
		{"Multiple optimizations", []string{"PREWHERE optimization", "Column pruning", "Query hints"}, 35.0, 45.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			improvement := optimizer.estimateImprovement(tt.optimizations)
			assert.GreaterOrEqual(t, improvement, tt.minImprovement)
			assert.LessOrEqual(t, improvement, tt.maxImprovement)
		})
	}
}

func TestGetRequiredColumns(t *testing.T) {
	optimizer := NewQueryOptimizer()

	req := &request.LogQueryRequest{
		Namespace: "default",
	}

	columns := optimizer.getRequiredColumns(req)

	assert.NotEmpty(t, columns)
	assert.Contains(t, columns, "timestamp")
	assert.Contains(t, columns, "namespace")
	assert.Contains(t, columns, "message")
}
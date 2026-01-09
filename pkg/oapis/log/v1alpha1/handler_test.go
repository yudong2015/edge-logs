package v1alpha1

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"

	"github.com/outpostos/edge-logs/pkg/model/request"
)

// Test ParseQueryRequest
func TestParseQueryRequest(t *testing.T) {
	// Create a minimal handler for testing
	handler := &LogHandler{}

	now := time.Now()
	startTime := now.Add(-24 * time.Hour)
	startTimeStr := startTime.UTC().Format(time.RFC3339)

	tests := []struct {
		name    string
		url     string
		dataset string
		wantReq *request.LogQueryRequest
		wantErr bool
	}{
		{
			name:    "basic query",
			url:     "/apis/log.theriseunion.io/v1alpha1/logdatasets/test/logs",
			dataset: "test",
			wantReq: &request.LogQueryRequest{
				Dataset: "test",
				Tags:    make(map[string]string),
			},
			wantErr: false,
		},
		{
			name:    "query with time range",
			url:     "/apis/log.theriseunion.io/v1alpha1/logdatasets/test/logs?start_time=" + startTimeStr + "&end_time=" + now.UTC().Format(time.RFC3339),
			dataset: "test",
			wantReq: &request.LogQueryRequest{
				Dataset: "test",
				Tags:    make(map[string]string),
			},
			wantErr: false, // We'll check that times are parsed correctly
		},
		{
			name:    "query with filters",
			url:     "/apis/log.theriseunion.io/v1alpha1/logdatasets/test/logs?namespace=default&pod_name=test-pod&filter=error&severity=ERROR",
			dataset: "test",
			wantReq: &request.LogQueryRequest{
				Dataset:   "test",
				Namespace: "default",
				PodName:   "test-pod",
				Filter:    "error",
				Severity:  "ERROR",
				Tags:      make(map[string]string),
			},
			wantErr: false,
		},
		{
			name:    "query with pagination",
			url:     "/apis/log.theriseunion.io/v1alpha1/logdatasets/test/logs?page=2&page_size=50",
			dataset: "test",
			wantReq: &request.LogQueryRequest{
				Dataset:  "test",
				Page:     2,
				PageSize: 50,
				Tags:     make(map[string]string),
			},
			wantErr: false,
		},
		{
			name:    "invalid start_time format",
			url:     "/apis/log.theriseunion.io/v1alpha1/logdatasets/test/logs?start_time=invalid",
			dataset: "test",
			wantErr: true,
		},
		{
			name:    "invalid page parameter",
			url:     "/apis/log.theriseunion.io/v1alpha1/logdatasets/test/logs?page=abc",
			dataset: "test",
			wantErr: true,
		},
		{
			name:    "negative page",
			url:     "/apis/log.theriseunion.io/v1alpha1/logdatasets/test/logs?page=-1",
			dataset: "test",
			wantErr: true,
		},
		{
			name:    "page_size out of range",
			url:     "/apis/log.theriseunion.io/v1alpha1/logdatasets/test/logs?page_size=20000",
			dataset: "test",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create HTTP request
			httpReq, _ := http.NewRequest("GET", tt.url, nil)
			restfulReq := restful.NewRequest(httpReq)

			// Set path parameter
			restfulReq.Request.URL.Path = tt.url

			// Parse request
			result, err := handler.parseQueryRequest(restfulReq, tt.dataset)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantReq.Dataset, result.Dataset)
				assert.Equal(t, tt.wantReq.Namespace, result.Namespace)
				assert.Equal(t, tt.wantReq.PodName, result.PodName)
				assert.Equal(t, tt.wantReq.Filter, result.Filter)
				assert.Equal(t, tt.wantReq.Severity, result.Severity)
				assert.Equal(t, tt.wantReq.Page, result.Page)
				assert.Equal(t, tt.wantReq.PageSize, result.PageSize)
			}
		})
	}
}

// Test MapErrorToStatusCode
func TestMapErrorToStatusCode(t *testing.T) {
	handler := &LogHandler{}

	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: http.StatusOK,
		},
		{
			name:     "generic error",
			err:      assert.AnError,
			expected: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.mapErrorToStatusCode(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test ErrorResponseSerialization
func TestErrorResponseSerialization(t *testing.T) {
	errorResp := map[string]interface{}{
		"code":    400,
		"message": "参数解析失败: start_time 格式错误",
	}

	jsonData, err := json.Marshal(errorResp)
	assert.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	assert.NoError(t, err)

	assert.Equal(t, float64(400), result["code"])
	assert.Contains(t, result["message"], "参数解析失败")
}

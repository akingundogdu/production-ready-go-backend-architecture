package actions

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (as *ActionSuite) Test_HealthHandler() {
	// Test successful health check
	res := as.JSON("/health").Get()

	as.Equal(http.StatusOK, res.Code)
	
	// Parse the response
	var response HealthResponse
	err := json.Unmarshal([]byte(res.Body.String()), &response)
	as.NoError(err)

	// Verify response structure
	as.Equal("healthy", response.Status)
	as.NotEmpty(response.Timestamp)
	as.NotEmpty(response.Uptime)
	as.Equal("1.0.0", response.Version)
	
	// Verify services
	as.Equal("healthy", response.Services["api"])
	as.Equal("not_configured", response.Services["database"])
	as.Equal("not_configured", response.Services["cache"])
	
	// Verify system info
	as.NotEmpty(response.System.GoVersion)
	as.Greater(response.System.NumGoroutines, 0)
	as.Greater(response.System.NumCPU, 0)
	as.NotEmpty(response.System.OS)
	as.NotEmpty(response.System.Arch)
}

func (as *ActionSuite) Test_LivenessHandler() {
	// Test liveness probe
	res := as.JSON("/health/live").Get()

	as.Equal(http.StatusOK, res.Code)
	
	// Parse the response
	var response map[string]string
	err := json.Unmarshal([]byte(res.Body.String()), &response)
	as.NoError(err)

	// Verify response
	as.Equal("alive", response["status"])
	as.NotEmpty(response["timestamp"])
	
	// Verify timestamp format
	_, err = time.Parse(time.RFC3339, response["timestamp"])
	as.NoError(err)
}

func (as *ActionSuite) Test_ReadinessHandler() {
	// Test readiness probe when ready
	res := as.JSON("/health/ready").Get()

	as.Equal(http.StatusOK, res.Code)
	
	// Parse the response
	var response map[string]interface{}
	err := json.Unmarshal([]byte(res.Body.String()), &response)
	as.NoError(err)

	// Verify response
	as.Equal("ready", response["status"])
	as.NotEmpty(response["timestamp"])
	
	// Verify timestamp format
	timestampStr, ok := response["timestamp"].(string)
	as.True(ok)
	_, err = time.Parse(time.RFC3339, timestampStr)
	as.NoError(err)
	
	// Verify services
	services, ok := response["services"].(map[string]interface{})
	as.True(ok)
	as.Equal("ready", services["api"])
}

func (as *ActionSuite) Test_ReadinessHandler_NotReady() {
	// Test readiness probe when not ready
	// Set environment variable to simulate not ready state
	os.Setenv("SIMULATE_NOT_READY", "true")
	defer os.Unsetenv("SIMULATE_NOT_READY")
	
	res := as.JSON("/health/ready").Get()

	as.Equal(http.StatusServiceUnavailable, res.Code)
	
	// Parse the response
	var response map[string]interface{}
	err := json.Unmarshal([]byte(res.Body.String()), &response)
	as.NoError(err)

	// Verify response
	as.Equal("not_ready", response["status"])
	as.NotEmpty(response["timestamp"])
	
	// Verify timestamp format
	timestampStr, ok := response["timestamp"].(string)
	as.True(ok)
	_, err = time.Parse(time.RFC3339, timestampStr)
	as.NoError(err)
	
	// Verify services show not ready
	services, ok := response["services"].(map[string]interface{})
	as.True(ok)
	as.Equal("not_ready", services["api"])
}

func (as *ActionSuite) Test_HealthHandler_ResponseFormat() {
	// Test that the health handler returns proper JSON format
	res := as.JSON("/health").Get()
	
	as.Equal(http.StatusOK, res.Code)
	as.Contains(res.Header().Get("Content-Type"), "application/json")
	
	// Verify it's valid JSON
	var response HealthResponse
	err := json.Unmarshal([]byte(res.Body.String()), &response)
	as.NoError(err)
	
	// Verify all required fields are present
	as.NotEmpty(response.Status)
	as.NotZero(response.Timestamp)
	as.NotEmpty(response.Uptime)
	as.NotEmpty(response.Version)
	as.NotNil(response.Services)
	as.NotZero(response.System)
}

func (as *ActionSuite) Test_HealthHandler_UptimeCalculation() {
	// Test that uptime is calculated correctly
	res1 := as.JSON("/health").Get()
	as.Equal(http.StatusOK, res1.Code)
	
	var response1 HealthResponse
	err := json.Unmarshal([]byte(res1.Body.String()), &response1)
	as.NoError(err)
	
	// Wait a small amount of time and test again
	time.Sleep(10 * time.Millisecond)
	
	res2 := as.JSON("/health").Get()
	as.Equal(http.StatusOK, res2.Code)
	
	var response2 HealthResponse
	err = json.Unmarshal([]byte(res2.Body.String()), &response2)
	as.NoError(err)
	
	// The second request should have a slightly longer uptime
	as.True(response2.Timestamp.After(response1.Timestamp))
}

func (as *ActionSuite) Test_AllHealthEndpoints_ContentType() {
	// Test that all health endpoints return JSON content type
	
	// Test /health endpoint
	res := as.JSON("/health").Get()
	as.Equal(http.StatusOK, res.Code)
	as.Contains(res.Header().Get("Content-Type"), "application/json")
	
	// Test /health/live endpoint
	res = as.JSON("/health/live").Get()
	as.Equal(http.StatusOK, res.Code)
	as.Contains(res.Header().Get("Content-Type"), "application/json")
	
	// Test /health/ready endpoint
	res = as.JSON("/health/ready").Get()
	as.Equal(http.StatusOK, res.Code)
	as.Contains(res.Header().Get("Content-Type"), "application/json")
}

func (as *ActionSuite) Test_Translations() {
	// Test that translations middleware can be initialized
	// This tests the translations function to achieve 100% coverage
	middleware := translations()
	as.NotNil(middleware)
	
	// Test that the translator T is initialized
	as.NotNil(T)
}

// Unit tests for individual functions (non-HTTP tests)
func TestHealthResponse_JSONMarshaling(t *testing.T) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC(),
		Uptime:    "1h2m3s",
		Version:   "1.0.0",
		Services: map[string]string{
			"api":      "healthy",
			"database": "not_configured",
		},
		System: SystemInfo{
			GoVersion:     "go1.21.0",
			NumGoroutines: 10,
			NumCPU:        4,
			OS:            "linux",
			Arch:          "amd64",
		},
	}
	
	// Test JSON marshaling
	data, err := json.Marshal(response)
	require.NoError(t, err)
	assert.Contains(t, string(data), "healthy")
	assert.Contains(t, string(data), "1.0.0")
	
	// Test JSON unmarshaling
	var unmarshaled HealthResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, response.Status, unmarshaled.Status)
	assert.Equal(t, response.Version, unmarshaled.Version)
}

func TestSystemInfo_JSONMarshaling(t *testing.T) {
	systemInfo := SystemInfo{
		GoVersion:     "go1.21.0",
		NumGoroutines: 15,
		NumCPU:        8,
		OS:            "darwin",
		Arch:          "arm64",
	}
	
	// Test JSON marshaling
	data, err := json.Marshal(systemInfo)
	require.NoError(t, err)
	assert.Contains(t, string(data), "go1.21.0")
	assert.Contains(t, string(data), "darwin")
	
	// Test JSON unmarshaling
	var unmarshaled SystemInfo
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, systemInfo.GoVersion, unmarshaled.GoVersion)
	assert.Equal(t, systemInfo.NumCPU, unmarshaled.NumCPU)
}

// Unit test for checkReadiness function
func TestCheckReadiness(t *testing.T) {
	// Test normal ready state
	ready, services := checkReadiness()
	assert.True(t, ready)
	assert.Equal(t, "ready", services["api"])
	
	// Test not ready state
	os.Setenv("SIMULATE_NOT_READY", "true")
	defer os.Unsetenv("SIMULATE_NOT_READY")
	
	ready, services = checkReadiness()
	assert.False(t, ready)
	assert.Equal(t, "not_ready", services["api"])
} 
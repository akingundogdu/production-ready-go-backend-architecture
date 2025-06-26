package actions

import (
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/gobuffalo/buffalo"
)

// HealthResponse represents the health check response structure
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Uptime    string            `json:"uptime"`
	Version   string            `json:"version"`
	Services  map[string]string `json:"services"`
	System    SystemInfo        `json:"system"`
}

// SystemInfo represents system information
type SystemInfo struct {
	GoVersion     string `json:"go_version"`
	NumGoroutines int    `json:"num_goroutines"`
	NumCPU        int    `json:"num_cpu"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
}

var startTime = time.Now()

// HealthHandler provides comprehensive health check information
// GET /health
func HealthHandler(c buffalo.Context) error {
	uptime := time.Since(startTime)

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC(),
		Uptime:    uptime.String(),
		Version:   "1.0.0",
		Services: map[string]string{
			"api":      "healthy",
			"database": "not_configured", // Will be updated when database is added
			"cache":    "not_configured", // Will be updated when Redis is added
		},
		System: SystemInfo{
			GoVersion:     runtime.Version(),
			NumGoroutines: runtime.NumGoroutine(),
			NumCPU:        runtime.NumCPU(),
			OS:            runtime.GOOS,
			Arch:          runtime.GOARCH,
		},
	}

	return c.Render(http.StatusOK, r.JSON(response))
}

// LivenessHandler provides a simple liveness probe for Kubernetes
// GET /health/live
func LivenessHandler(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.JSON(map[string]string{
		"status":    "alive",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}))
}

// checkReadiness checks if the application is ready to serve traffic
func checkReadiness() (bool, map[string]string) {
	services := map[string]string{
		"api": "ready",
		// Add more service checks here as they are implemented
	}

	// Check if we're in a simulated not-ready state (for testing)
	if os.Getenv("SIMULATE_NOT_READY") == "true" {
		services["api"] = "not_ready"
		return false, services
	}

	// In the future, add real readiness checks here:
	// - Database connection check
	// - Cache connection check
	// - External service dependencies

	return true, services
}

// ReadinessHandler provides a readiness probe for Kubernetes
// GET /health/ready
func ReadinessHandler(c buffalo.Context) error {
	// Check if all required services are ready
	ready, services := checkReadiness()

	status := "ready"
	httpStatus := http.StatusOK

	if !ready {
		status = "not_ready"
		httpStatus = http.StatusServiceUnavailable
	}

	return c.Render(httpStatus, r.JSON(map[string]interface{}{
		"status":    status,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"services":  services,
	}))
}

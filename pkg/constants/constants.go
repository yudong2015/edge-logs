package constants

// API constants
const (
	APIVersion     = "v1alpha1"
	APIPath        = "/api/" + APIVersion
	HealthEndpoint = APIPath + "/health"
)

// Log query constants
const (
	DefaultPageSize = 100
	MaxPageSize     = 1000
)

// Error constants
const (
	ErrInvalidQuery = "invalid query parameters"
	ErrUnauthorized = "unauthorized access"
	ErrServerError  = "internal server error"
)
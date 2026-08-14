package config

import "testing"

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid configuration",
			config: Config{
				ServerPort:              8080,
				UpstreamURL:             "http://localhost:9000",
				RedisURL:                "redis://localhost:6379",
				RateLimitRequests:       100,
				RateLimitWindowSec:      60,
				CircuitFailureThreshold: 5,
				CircuitOpenTimeoutSec:   30,
				HealthCheckIntervalSec:  5,
			},
		},
		{
			name: "invalid server port",
			config: Config{
				ServerPort:  70000,
				UpstreamURL: "http://localhost:9000",
			},
			wantErr: true,
		},
		{
			name: "missing upstream URL",
			config: Config{
				ServerPort: 8080,
			},
			wantErr: true,
		},
		{
			name: "invalid rate limit",
			config: Config{
				ServerPort:        8080,
				UpstreamURL:       "http://localhost:9000",
				RateLimitRequests: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

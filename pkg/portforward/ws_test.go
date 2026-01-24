package portforward

import (
	"os"
	"testing"
)

func TestResolveWebSocketURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		envURL  string
		want    string
		wantErr bool
	}{
		{
			name:    "empty defaults to US",
			input:   "",
			want:    "wss://app.komodor.com",
			wantErr: false,
		},
		{
			name:    "us region lowercase",
			input:   "us",
			want:    "wss://app.komodor.com",
			wantErr: false,
		},
		{
			name:    "US region uppercase",
			input:   "US",
			want:    "wss://app.komodor.com",
			wantErr: false,
		},
		{
			name:    "eu region lowercase",
			input:   "eu",
			want:    "wss://app.eu.komodor.com",
			wantErr: false,
		},
		{
			name:    "EU region uppercase",
			input:   "EU",
			want:    "wss://app.eu.komodor.com",
			wantErr: false,
		},
		{
			name:    "custom wss URL",
			input:   "wss://custom.example.com",
			want:    "wss://custom.example.com",
			wantErr: false,
		},
		{
			name:    "custom ws URL",
			input:   "ws://localhost:8080",
			want:    "ws://localhost:8080",
			wantErr: false,
		},
		{
			name:    "invalid region",
			input:   "invalid",
			want:    "",
			wantErr: true,
		},
		{
			name:    "env var overrides region",
			input:   "eu",
			envURL:  "wss://env.override.com",
			want:    "wss://env.override.com",
			wantErr: false,
		},
		{
			name:    "env var overrides custom URL",
			input:   "wss://custom.com",
			envURL:  "wss://env.override.com",
			want:    "wss://env.override.com",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.envURL != "" {
				_ = os.Setenv("KOMOCLI_WS_URL", tt.envURL)
				defer func() {
					_ = os.Unsetenv("KOMOCLI_WS_URL")
				}()
			} else {
				_ = os.Unsetenv("KOMOCLI_WS_URL")
			}

			got, err := ResolveWebSocketURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveWebSocketURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveWebSocketURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegionURLsMap(t *testing.T) {
	expectedRegions := map[string]string{
		"us": "wss://app.komodor.com",
		"eu": "wss://app.eu.komodor.com",
	}

	for region, expectedURL := range expectedRegions {
		if url, ok := RegionURLs[region]; !ok {
			t.Errorf("Region '%s' not found in RegionURLs map", region)
		} else if url != expectedURL {
			t.Errorf("Region '%s' has URL '%s', expected '%s'", region, url, expectedURL)
		}
	}
}

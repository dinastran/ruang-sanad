package config

import "testing"

func TestListenAddressKeepsProductionDefault(t *testing.T) {
	cfg := Config{AppPort: "8080"}
	if got := cfg.ListenAddress(); got != ":8080" {
		t.Fatalf("default binding changed: %q", got)
	}
}

func TestListenAddressSupportsLoopbackStaging(t *testing.T) {
	cfg := Config{AppPort: "18081", AppBindHost: "127.0.0.1"}
	if got := cfg.ListenAddress(); got != "127.0.0.1:18081" {
		t.Fatalf("expected staging to bind only loopback, got %q", got)
	}
}

func TestListenAddressSupportsIPv6(t *testing.T) {
	cfg := Config{AppPort: "18081", AppBindHost: "::1"}
	if got := cfg.ListenAddress(); got != "[::1]:18081" {
		t.Fatalf("expected valid IPv6 loopback, got %q", got)
	}
}

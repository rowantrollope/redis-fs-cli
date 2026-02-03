package embedding

import (
	"math"
	"testing"
)

func TestFloat32ToBytes(t *testing.T) {
	vec := []float32{1.0, 2.0, 3.0, -1.5}
	bytes := Float32ToBytes(vec)

	if len(bytes) != len(vec)*4 {
		t.Fatalf("expected %d bytes, got %d", len(vec)*4, len(bytes))
	}

	// Round-trip
	result := BytesToFloat32(bytes)
	if len(result) != len(vec) {
		t.Fatalf("expected %d floats, got %d", len(vec), len(result))
	}
	for i := range vec {
		if math.Abs(float64(result[i]-vec[i])) > 1e-6 {
			t.Errorf("index %d: got %f, want %f", i, result[i], vec[i])
		}
	}
}

func TestFloat32ToBytesEmpty(t *testing.T) {
	bytes := Float32ToBytes(nil)
	if len(bytes) != 0 {
		t.Fatalf("expected 0 bytes for nil input, got %d", len(bytes))
	}

	result := BytesToFloat32(nil)
	if len(result) != 0 {
		t.Fatalf("expected 0 floats for nil input, got %d", len(result))
	}
}

func TestConfigIsConfigured(t *testing.T) {
	cfg := &Config{}
	if cfg.IsConfigured() {
		t.Error("expected not configured with empty key")
	}

	cfg.APIKey = "sk-test"
	if !cfg.IsConfigured() {
		t.Error("expected configured with key set")
	}
}

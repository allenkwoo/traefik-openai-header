package traefik_openai_header

import (
	"context"
	"testing"
)

// Traefik builds one handler per router, so without a shared counter
// maxInFlightBytes would bound each router rather than the process. Handlers
// built from the same middleware name must draw on one budget.
func TestSharedInFlight_SameNameSharesOnePool(t *testing.T) {
	cfg := cappedConfig(64 << 20)
	cfg.MaxInFlightBytes = 128 << 20
	first, err := New(context.Background(), &nextRecorder{}, cfg, "shared-pool")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	second, err := New(context.Background(), &nextRecorder{}, cfg, "shared-pool")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	a, b := first.(*Handler), second.(*Handler)

	if !a.acquire(100 << 20) {
		t.Fatal("acquire within the budget was rejected")
	}
	if b.acquire(100 << 20) {
		t.Fatal("second handler admitted 200 MiB against a 128 MiB budget")
	}
	if got, want := b.inFlightBytes(), int64(100<<20); got != want {
		t.Fatalf("in-flight bytes = %d, want %d", got, want)
	}

	// A release by one handler frees the budget for the other.
	a.release(100 << 20)
	if !b.acquire(100 << 20) {
		t.Fatal("budget released by one handler was not available to the other")
	}
	b.release(100 << 20)
}

// Middlewares configured separately keep separate budgets, so the audio
// variant cannot be starved by completions traffic.
func TestSharedInFlight_DifferentNamesAreIndependent(t *testing.T) {
	cfg := cappedConfig(64 << 20)
	cfg.MaxInFlightBytes = 128 << 20
	first, err := New(context.Background(), &nextRecorder{}, cfg, "completions")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	second, err := New(context.Background(), &nextRecorder{}, cfg, "audio")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	a, b := first.(*Handler), second.(*Handler)

	if !a.acquire(100<<20) || !b.acquire(100<<20) {
		t.Fatal("handlers with different middleware names shared a budget")
	}
	a.release(100 << 20)
	b.release(100 << 20)
}

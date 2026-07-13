package webhooks

import (
	"testing"
	"time"
)

func TestBackoffDelay(t *testing.T) {
	tests := []struct {
		attempt  int
		minDelay time.Duration
		maxDelay time.Duration
	}{
		{0, 2 * time.Second, 4*time.Second + 2*time.Second},
		{1, 4 * time.Second, 6*time.Second + 2*time.Second},
		{2, 8 * time.Second, 10*time.Second + 2*time.Second},
		{3, 16 * time.Second, 18*time.Second + 2*time.Second},
	}

	for _, tc := range tests {
		d := backoffDelay(tc.attempt)
		if d < tc.minDelay {
			t.Errorf("attempt %d: delay %v is less than min %v", tc.attempt, d, tc.minDelay)
		}
	}
}

func TestBackoffDelay_Max(t *testing.T) {
	// After many attempts, delay should cap at MaxDelay + jitter
	for attempt := 10; attempt < 20; attempt++ {
		d := backoffDelay(attempt)
		if d > MaxDelay+2*time.Second {
			t.Errorf("attempt %d: delay %v exceeds max %v", attempt, d, MaxDelay+2*time.Second)
		}
	}
}

func TestRetryableStatusCodes(t *testing.T) {
	retryable := []int{408, 425, 429, 500, 502, 503, 504}
	nonRetryable := []int{200, 201, 400, 401, 403, 404, 410, 422, 451}

	for _, code := range retryable {
		if !retryableStatusCodes[code] {
			t.Errorf("status %d should be retryable", code)
		}
	}
	for _, code := range nonRetryable {
		if retryableStatusCodes[code] {
			t.Errorf("status %d should not be retryable", code)
		}
	}
}

func TestBuildPayload(t *testing.T) {
	// We can't easily test buildPayload without DB, but we can test the
	// event type constants and emit channel behavior.
	if EventObjectCreated != "beamdrop.object.created" {
		t.Fatalf("unexpected event constant: %q", EventObjectCreated)
	}
	if EventBucketCreated != "beamdrop.bucket.created" {
		t.Fatalf("unexpected event constant: %q", EventBucketCreated)
	}
}

func TestEmitQueueSize(t *testing.T) {
	if emitQueueSize != 1000 {
		t.Fatalf("expected emitQueueSize 1000, got %d", emitQueueSize)
	}
}

func TestWorker_StartStop(t *testing.T) {
	w := NewWorker()
	w.Start()
	w.Stop()

	// Stopping twice should be safe
	w.Stop()
}

func TestWorker_StopBeforeStart(t *testing.T) {
	w := NewWorker()
	w.Stop()
}

func TestMaxAttempts(t *testing.T) {
	if MaxAttempts != 8 {
		t.Fatalf("expected MaxAttempts 8, got %d", MaxAttempts)
	}
}

func TestBaseDelay(t *testing.T) {
	if BaseDelay != 2*time.Second {
		t.Fatalf("expected BaseDelay 2s, got %v", BaseDelay)
	}
}

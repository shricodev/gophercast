package client

import (
	"testing"
	"time"

	"github.com/shricodev/gophercast/internal/testutil"
)

func TestDriftCorrectorReset(t *testing.T) {
	d := &driftCorrector{}
	now := time.Now()
	d.reset(44100, 2, now)

	if d.sampleRate != 44100 {
		t.Fatalf("expected sampleRate 44100, got %d", d.sampleRate)
	}
	if d.bytesPerSample != 4 {
		t.Fatalf("expected bytesPerSample 4, got %d", d.bytesPerSample)
	}
	if d.samplesWritten != 0 {
		t.Fatalf("expected samplesWritten 0, got %d", d.samplesWritten)
	}
	if d.startTime != now {
		t.Fatal("startTime mismatch")
	}
}

func TestDriftCorrectorWritten(t *testing.T) {
	d := &driftCorrector{}
	d.reset(44100, 2, time.Now())

	// 4096 bytes / 4 bytes per sample = 1024 samples
	d.written(4096)
	if d.samplesWritten != 1024 {
		t.Fatalf("expected 1024 samples, got %d", d.samplesWritten)
	}

	d.written(4096)
	if d.samplesWritten != 2048 {
		t.Fatalf("expected 2048 samples, got %d", d.samplesWritten)
	}
}

func TestDriftCorrectorNoCheckPassthrough(t *testing.T) {
	d := &driftCorrector{}
	d.reset(44100, 2, time.Now())

	payload := make([]byte, 4096)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	// shouldCheck=false should return payload unchanged
	result := d.correct(payload, false)
	testutil.AssertLen(t, len(result), len(payload))
}

func TestDriftCorrectorNoCorrectionWithinThreshold(t *testing.T) {
	d := &driftCorrector{}
	// Start time = now, so drift is ~0
	d.reset(44100, 2, time.Now())

	payload := make([]byte, 4096)
	result := d.correct(payload, true)

	// Drift is within threshold, payload unchanged
	testutil.AssertLen(t, len(result), len(payload))
}

func TestDriftCorrectorTrimWhenBehind(t *testing.T) {
	d := &driftCorrector{}
	// Set start time 50ms in the past but report 0 samples written.
	// This means we're 50ms behind schedule.
	d.reset(44100, 2, time.Now().Add(-50*time.Millisecond))
	d.samplesWritten = 0

	payload := make([]byte, 4096)
	result := d.correct(payload, true)

	// Should trim samples from the payload (shorter output)
	if len(result) >= len(payload) {
		t.Fatalf("expected shorter payload when behind, got %d bytes (original %d)", len(result), len(payload))
	}

	// Trimmed amount should be aligned to bytesPerSample (4 bytes)
	trimmed := len(payload) - len(result)
	if trimmed%d.bytesPerSample != 0 {
		t.Fatalf("trim %d bytes is not sample-aligned (bytesPerSample=%d)", trimmed, d.bytesPerSample)
	}
}

func TestDriftCorrectorDuplicateWhenAhead(t *testing.T) {
	d := &driftCorrector{}
	// Set start time 50ms in the future but report samples as if we've been
	// playing for 50ms. This means we're 50ms ahead of schedule.
	d.reset(44100, 2, time.Now().Add(50*time.Millisecond))
	d.samplesWritten = uint64(44100 * 50 / 1000) // 50ms worth of samples

	payload := make([]byte, 4096)
	result := d.correct(payload, true)

	// Should duplicate samples (longer output)
	if len(result) <= len(payload) {
		t.Fatalf("expected longer payload when ahead, got %d bytes (original %d)", len(result), len(payload))
	}

	// Extra amount should be sample-aligned
	extra := len(result) - len(payload)
	if extra%d.bytesPerSample != 0 {
		t.Fatalf("extra %d bytes is not sample-aligned (bytesPerSample=%d)", extra, d.bytesPerSample)
	}
}

func TestDriftCorrectorMaxCorrection(t *testing.T) {
	d := &driftCorrector{}
	// Set start time way in the past (1 second behind).
	// Even with huge drift, correction should be capped at maxCorrectionSamples.
	d.reset(44100, 2, time.Now().Add(-1*time.Second))
	d.samplesWritten = 0

	payload := make([]byte, 4096)
	result := d.correct(payload, true)

	trimmed := len(payload) - len(result)
	maxTrimBytes := maxCorrectionSamples * d.bytesPerSample
	if trimmed > maxTrimBytes {
		t.Fatalf("trimmed %d bytes exceeds max %d bytes", trimmed, maxTrimBytes)
	}
}

func TestDriftCorrectorUninitializedPassthrough(t *testing.T) {
	d := &driftCorrector{}
	// Not reset — sampleRate=0, startTime is zero

	payload := make([]byte, 4096)
	result := d.correct(payload, true)

	// Should pass through unchanged
	testutil.AssertLen(t, len(result), len(payload))
}

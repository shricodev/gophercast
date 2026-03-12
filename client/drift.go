package client

import "time"

const (
	// driftCheckInterval is how often (in frames) to measure and correct drift.
	// At 44.1kHz with 4096-byte chunks (~23ms each), 50 frames ≈ 1.15 seconds.
	driftCheckInterval uint32 = 50

	// driftThreshold is the minimum drift before correction is applied.
	// Below this, the difference is imperceptible.
	driftThreshold = 2 * time.Millisecond

	// maxCorrectionSamples caps the per-check correction to avoid audible artifacts.
	// 32 samples at 44.1kHz ≈ 0.7ms — inaudible as a single skip/duplicate.
	maxCorrectionSamples = 32
)

// driftCorrector tracks the relationship between wall-clock time and samples
// written to the audio sink, applying micro-corrections to keep playback
// aligned with the server's timeline.
type driftCorrector struct {
	sampleRate     int
	bytesPerSample int
	samplesWritten uint64
	startTime      time.Time
}

// reset prepares the corrector for a new track.
func (d *driftCorrector) reset(sampleRate, channels int, startTime time.Time) {
	d.sampleRate = sampleRate
	d.bytesPerSample = channels * 2 // 16-bit signed LE
	d.samplesWritten = 0
	d.startTime = startTime
}

// written records that n bytes of PCM were sent to the audio sink.
func (d *driftCorrector) written(n int) {
	d.samplesWritten += uint64(n / d.bytesPerSample)
}

// correct checks the drift between wall-clock elapsed time and expected
// playback position, returning a possibly adjusted payload.
//
// If the audio hardware consumes samples slower than the declared rate,
// we fall behind schedule — correct by trimming samples from the payload.
// If the hardware is faster, we get ahead — correct by duplicating samples.
//
// shouldCheck indicates whether this frame should trigger a drift check.
func (d *driftCorrector) correct(payload []byte, shouldCheck bool) []byte {
	if !shouldCheck || d.sampleRate == 0 || d.startTime.IsZero() {
		return payload
	}

	expectedElapsed := time.Duration(
		float64(d.samplesWritten) / float64(d.sampleRate) * float64(time.Second),
	)
	actualElapsed := time.Since(d.startTime)
	drift := actualElapsed - expectedElapsed

	if drift > driftThreshold {
		// Behind schedule (late) — trim samples from the end of the payload.
		correction := int(drift.Seconds() * float64(d.sampleRate))
		if correction > maxCorrectionSamples {
			correction = maxCorrectionSamples
		}
		bytesToTrim := correction * d.bytesPerSample
		if bytesToTrim > 0 && bytesToTrim < len(payload) {
			return payload[:len(payload)-bytesToTrim]
		}
	} else if drift < -driftThreshold {
		// Ahead of schedule (early) — duplicate samples at the end of the payload.
		correction := int((-drift).Seconds() * float64(d.sampleRate))
		if correction > maxCorrectionSamples {
			correction = maxCorrectionSamples
		}
		bytesToDup := correction * d.bytesPerSample
		if bytesToDup > 0 && bytesToDup <= len(payload) {
			extra := make([]byte, bytesToDup)
			copy(extra, payload[len(payload)-bytesToDup:])
			return append(payload, extra...)
		}
	}

	return payload
}

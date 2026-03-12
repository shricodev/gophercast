package client

import "time"

const (
	// check every ~1.15s at 44.1kHz (50 frames * ~23ms each)
	driftCheckInterval uint32 = 50

	// anything less than this is basically inaudible, don't bother correcting
	driftThreshold = 2 * time.Millisecond

	// cap correction to 32 samples (~0.7ms) so we don't get audible glitches
	maxCorrectionSamples = 32
)

// driftCorrector nudges playback to stay aligned with the server's clock.
type driftCorrector struct {
	sampleRate     int
	bytesPerSample int
	samplesWritten uint64
	startTime      time.Time
}

func (d *driftCorrector) reset(sampleRate, channels int, startTime time.Time) {
	d.sampleRate = sampleRate
	d.bytesPerSample = channels * 2 // 16-bit signed LE
	d.samplesWritten = 0
	d.startTime = startTime
}

func (d *driftCorrector) written(n int) {
	d.samplesWritten += uint64(n / d.bytesPerSample)
}

// correct trims or duplicates a few samples to compensate for drift.
// If we're behind, trim from the end. If we're ahead, duplicate from the end.
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
		// behind - trim some samples off the end
		correction := int(drift.Seconds() * float64(d.sampleRate))
		if correction > maxCorrectionSamples {
			correction = maxCorrectionSamples
		}
		bytesToTrim := correction * d.bytesPerSample
		if bytesToTrim > 0 && bytesToTrim < len(payload) {
			return payload[:len(payload)-bytesToTrim]
		}
	} else if drift < -driftThreshold {
		// ahead - pad with a few duplicate samples at the end
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

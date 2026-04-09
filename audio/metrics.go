package audio

import "math"

// AudioSnapshot holds one frame of visualization data, sent to the TUI.
type AudioSnapshot struct {
	PreLeft  []float64
	PreRight []float64

	PostLeft  []float64
	PostRight []float64

	PrePeakLeft   float64
	PrePeakRight  float64
	PostPeakLeft  float64
	PostPeakRight float64
}

// MetricsCollector accumulates audio samples and produces snapshots
// at a configured interval for TUI visualization.
type MetricsCollector struct {
	samplesPerSnap int
	count          int

	preLBuf  []float64
	preRBuf  []float64
	postLBuf []float64
	postRBuf []float64

	prePeakL  float64
	prePeakR  float64
	postPeakL float64
	postPeakR float64
}

const waveformPoints = 128

func NewMetricsCollector(sampleRate float64, snapshotFPS int) *MetricsCollector {
	cap := int(sampleRate) / snapshotFPS
	return &MetricsCollector{
		samplesPerSnap: cap,
		preLBuf:        make([]float64, 0, cap),
		preRBuf:        make([]float64, 0, cap),
		postLBuf:       make([]float64, 0, cap),
		postRBuf:       make([]float64, 0, cap),
	}
}

func (mc *MetricsCollector) RecordPre(left, right float32) {
	mc.preLBuf = append(mc.preLBuf, float64(left))
	mc.preRBuf = append(mc.preRBuf, float64(right))

	if al := math.Abs(float64(left)); al > mc.prePeakL {
		mc.prePeakL = al
	}
	if ar := math.Abs(float64(right)); ar > mc.prePeakR {
		mc.prePeakR = ar
	}
}

func (mc *MetricsCollector) RecordPost(left, right float32) {
	mc.postLBuf = append(mc.postLBuf, float64(left))
	mc.postRBuf = append(mc.postRBuf, float64(right))

	if al := math.Abs(float64(left)); al > mc.postPeakL {
		mc.postPeakL = al
	}
	if ar := math.Abs(float64(right)); ar > mc.postPeakR {
		mc.postPeakR = ar
	}
}

// Advance increments the sample counter. Returns a snapshot if enough
// samples have accumulated, otherwise nil.
func (mc *MetricsCollector) Advance() *AudioSnapshot {
	mc.count++
	if mc.count < mc.samplesPerSnap {
		return nil
	}

	snap := &AudioSnapshot{
		PreLeft:       downsample(mc.preLBuf, waveformPoints),
		PreRight:      downsample(mc.preRBuf, waveformPoints),
		PostLeft:      downsample(mc.postLBuf, waveformPoints),
		PostRight:     downsample(mc.postRBuf, waveformPoints),
		PrePeakLeft:   mc.prePeakL,
		PrePeakRight:  mc.prePeakR,
		PostPeakLeft:  mc.postPeakL,
		PostPeakRight: mc.postPeakR,
	}

	// Reset accumulators.
	mc.count = 0
	mc.preLBuf = mc.preLBuf[:0]
	mc.preRBuf = mc.preRBuf[:0]
	mc.postLBuf = mc.postLBuf[:0]
	mc.postRBuf = mc.postRBuf[:0]
	mc.prePeakL = 0
	mc.prePeakR = 0
	mc.postPeakL = 0
	mc.postPeakR = 0

	return snap
}

// downsample reduces a slice to n points using max-abs within each bin.
func downsample(data []float64, n int) []float64 {
	if len(data) <= n {
		out := make([]float64, len(data))
		copy(out, data)
		return out
	}

	out := make([]float64, n)
	binSize := float64(len(data)) / float64(n)
	for i := 0; i < n; i++ {
		start := int(float64(i) * binSize)
		end := int(float64(i+1) * binSize)
		if end > len(data) {
			end = len(data)
		}

		maxVal := 0.0
		maxAbs := 0.0
		for _, v := range data[start:end] {
			if a := math.Abs(v); a > maxAbs {
				maxAbs = a
				maxVal = v
			}
		}
		out[i] = maxVal
	}
	return out
}

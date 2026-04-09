package dsp

import "math"

// LowPassFilter is a single-pole IIR low-pass filter.
//
//	y[n] = alpha * x[n] + (1 - alpha) * y[n-1]
//
// where alpha = 1 - exp(-2*pi*fc/fs)
type LowPassFilter struct {
	alpha      float32
	prev       float32
	cutoff     float32
	sampleRate float64
}

func NewLowPassFilter(cutoffHz float32, sampleRate float64) *LowPassFilter {
	f := &LowPassFilter{}
	f.SetCutoff(cutoffHz, sampleRate)
	return f
}

func (f *LowPassFilter) SetCutoff(cutoffHz float32, sampleRate float64) {
	if cutoffHz == f.cutoff && sampleRate == f.sampleRate {
		return
	}
	f.cutoff = cutoffHz
	f.sampleRate = sampleRate
	f.alpha = float32(1.0 - math.Exp(-2.0*math.Pi*float64(cutoffHz)/sampleRate))
}

func (f *LowPassFilter) Process(x float32) float32 {
	f.prev = f.alpha*x + (1.0-f.alpha)*f.prev
	return f.prev
}

package dsp

func (proc *Processor) processRealistic(lIn, rIn float32, p *ParamSnapshot) (float32, float32) {
	straight := 1.0 - p.Blend
	delaySamples := p.DelayMs * float32(p.SampleRate) / 1000.0

	// Write current opposite-channel samples into delay lines.
	// The left-cross delay line holds R samples (what crosses to the left ear).
	// The right-cross delay line holds L samples (what crosses to the right ear).
	proc.delayLine.WriteStereo(rIn, lIn)

	// Read delayed crossed signals with fractional delay (linear interpolation).
	crossedToL := proc.delayLine.ReadLeft(delaySamples)
	crossedToR := proc.delayLine.ReadRight(delaySamples)

	// Low-pass filter the crossed signals (simulates head shadow).
	proc.lpfLeft.SetCutoff(p.FilterCutoffHz, p.SampleRate)
	proc.lpfRight.SetCutoff(p.FilterCutoffHz, p.SampleRate)
	crossedToL = proc.lpfLeft.Process(crossedToL)
	crossedToR = proc.lpfRight.Process(crossedToR)

	// Mix direct and crossed signals.
	lOut := lIn*straight + crossedToL*p.Blend
	rOut := rIn*straight + crossedToR*p.Blend

	return lOut, rOut
}

// DelayLine is a stereo circular buffer for fractional delay with linear interpolation.
type DelayLine struct {
	bufL, bufR []float32
	writePos   int
	mask       int
}

func NewDelayLine(maxSamples int) *DelayLine {
	size := nextPow2(maxSamples)
	return &DelayLine{
		bufL: make([]float32, size),
		bufR: make([]float32, size),
		mask: size - 1,
	}
}

func (dl *DelayLine) WriteStereo(toL, toR float32) {
	dl.bufL[dl.writePos] = toL
	dl.bufR[dl.writePos] = toR
	dl.writePos = (dl.writePos + 1) & dl.mask
}

// ReadLeft reads from the left-cross buffer with fractional delay.
func (dl *DelayLine) ReadLeft(delaySamples float32) float32 {
	return dl.readFrom(dl.bufL, delaySamples)
}

// ReadRight reads from the right-cross buffer with fractional delay.
func (dl *DelayLine) ReadRight(delaySamples float32) float32 {
	return dl.readFrom(dl.bufR, delaySamples)
}

func (dl *DelayLine) readFrom(buf []float32, delaySamples float32) float32 {
	// Integer and fractional parts of the delay.
	delayInt := int(delaySamples)
	frac := delaySamples - float32(delayInt)

	// Read two adjacent samples for linear interpolation.
	idx0 := (dl.writePos - 1 - delayInt) & dl.mask
	idx1 := (dl.writePos - 2 - delayInt) & dl.mask
	return buf[idx0]*(1-frac) + buf[idx1]*frac
}

func nextPow2(n int) int {
	v := 1
	for v < n {
		v <<= 1
	}
	return v
}

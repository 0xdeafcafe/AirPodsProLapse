package dsp

// Processor holds stateful DSP components for crossfeed processing.
type Processor struct {
	delayLine *DelayLine
	lpfLeft   *LowPassFilter // filter on the crossed-to-left signal
	lpfRight  *LowPassFilter // filter on the crossed-to-right signal
}

func NewProcessor(sampleRate float64) *Processor {
	// Max delay: 1ms at 48kHz = 48 samples. Allocate 64 for power-of-2 masking.
	maxDelay := int(sampleRate*0.001) + 16
	return &Processor{
		delayLine: NewDelayLine(maxDelay),
		lpfLeft:   NewLowPassFilter(700, sampleRate),
		lpfRight:  NewLowPassFilter(700, sampleRate),
	}
}

// Process takes a stereo sample pair and returns the crossfed output.
func (proc *Processor) Process(lIn, rIn float32, p *ParamSnapshot) (lOut, rOut float32) {
	switch p.Mode {
	case ModeSimple:
		return proc.processSimple(lIn, rIn, p.Blend)
	case ModeRealistic:
		return proc.processRealistic(lIn, rIn, p)
	}
	return lIn, rIn
}

func (proc *Processor) processSimple(lIn, rIn, blend float32) (float32, float32) {
	straight := 1.0 - blend
	lOut := lIn*straight + rIn*blend
	rOut := rIn*straight + lIn*blend
	return lOut, rOut
}

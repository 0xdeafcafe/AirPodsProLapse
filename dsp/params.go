package dsp

import "sync"

// Mode selects the crossfeed algorithm.
type Mode int

const (
	ModeSimple    Mode = iota
	ModeRealistic
)

func (m Mode) String() string {
	switch m {
	case ModeSimple:
		return "Simple"
	case ModeRealistic:
		return "Realistic"
	default:
		return "Unknown"
	}
}

// ParamSnapshot is an immutable copy of all DSP parameters,
// safe to use without holding any lock.
type ParamSnapshot struct {
	Mode           Mode
	Blend          float32 // 0.0 (no crossfeed) to 0.5 (mono)
	DelayMs        float32 // 0.0 to 1.0 ms (realistic mode only)
	FilterCutoffHz float32 // low-pass cutoff in Hz (realistic mode only)
	Bypass         bool
	SampleRate     float64
}

// Params is a thread-safe container for DSP parameters.
// The audio goroutine reads via Snapshot(), the TUI goroutine writes via setters.
type Params struct {
	mu   sync.RWMutex
	snap ParamSnapshot
}

func NewParams(sampleRate float64) *Params {
	return &Params{
		snap: ParamSnapshot{
			Mode:           ModeSimple,
			Blend:          0.15,
			DelayMs:        0.3,
			FilterCutoffHz: 700,
			Bypass:         false,
			SampleRate:     sampleRate,
		},
	}
}

// Snapshot returns an immutable copy of current parameters.
func (p *Params) Snapshot() ParamSnapshot {
	p.mu.RLock()
	s := p.snap
	p.mu.RUnlock()
	return s
}

func (p *Params) SetBlend(v float32) {
	p.mu.Lock()
	p.snap.Blend = clamp32(v, 0, 0.5)
	p.mu.Unlock()
}

func (p *Params) AdjustBlend(delta float32) {
	p.mu.Lock()
	p.snap.Blend = clamp32(p.snap.Blend+delta, 0, 0.5)
	p.mu.Unlock()
}

func (p *Params) ToggleMode() {
	p.mu.Lock()
	if p.snap.Mode == ModeSimple {
		p.snap.Mode = ModeRealistic
	} else {
		p.snap.Mode = ModeSimple
	}
	p.mu.Unlock()
}

func (p *Params) SetDelayMs(v float32) {
	p.mu.Lock()
	p.snap.DelayMs = clamp32(v, 0, 1.0)
	p.mu.Unlock()
}

func (p *Params) AdjustDelayMs(delta float32) {
	p.mu.Lock()
	p.snap.DelayMs = clamp32(p.snap.DelayMs+delta, 0, 1.0)
	p.mu.Unlock()
}

func (p *Params) SetFilterCutoff(v float32) {
	p.mu.Lock()
	p.snap.FilterCutoffHz = clamp32(v, 200, 5000)
	p.mu.Unlock()
}

func (p *Params) AdjustFilterCutoff(delta float32) {
	p.mu.Lock()
	p.snap.FilterCutoffHz = clamp32(p.snap.FilterCutoffHz+delta, 200, 5000)
	p.mu.Unlock()
}

func (p *Params) ToggleBypass() {
	p.mu.Lock()
	p.snap.Bypass = !p.snap.Bypass
	p.mu.Unlock()
}

func clamp32(v, min, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

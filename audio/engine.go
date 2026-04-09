package audio

import (
	"fmt"
	"sync"
	"time"

	"github.com/0xdeafcafe/AirPodsProLapse/dsp"
	"github.com/gordonklaus/portaudio"
)

const (
	DefaultSampleRate   = 48000
	DefaultFramesPerBuf = 256
	NumChannels         = 2
	MetricsSnapshotFPS  = 30
	maxConsecutiveErrs  = 50
)

type Engine struct {
	inputDev  *portaudio.DeviceInfo
	outputDev *portaudio.DeviceInfo
	params    *dsp.Params
	metricsCh chan AudioSnapshot
	processor *dsp.Processor

	sampleRate   float64
	framesPerBuf int
	stopOnce     sync.Once
	stopCh       chan struct{}
	wg           sync.WaitGroup

	// Buffers shared between stream and processing loop.
	inBuf  []float32
	outBuf []float32
}

func NewEngine(
	inputDev, outputDev *portaudio.DeviceInfo,
	params *dsp.Params,
	metricsCh chan AudioSnapshot,
	sampleRate float64,
) *Engine {
	if sampleRate == 0 {
		sampleRate = DefaultSampleRate
	}
	framesPerBuf := DefaultFramesPerBuf
	return &Engine{
		inputDev:     inputDev,
		outputDev:    outputDev,
		params:       params,
		metricsCh:    metricsCh,
		processor:    dsp.NewProcessor(sampleRate),
		sampleRate:   sampleRate,
		framesPerBuf: framesPerBuf,
		stopCh:       make(chan struct{}),
		inBuf:        make([]float32, framesPerBuf*NumChannels),
		outBuf:       make([]float32, framesPerBuf*NumChannels),
	}
}

func (e *Engine) Start() error {
	inStream, err := e.openInputStream()
	if err != nil {
		return fmt.Errorf("opening input stream: %w", err)
	}
	outStream, err := e.openOutputStream()
	if err != nil {
		inStream.Close()
		return fmt.Errorf("opening output stream: %w", err)
	}

	if err := inStream.Start(); err != nil {
		inStream.Close()
		outStream.Close()
		return fmt.Errorf("starting input stream: %w", err)
	}
	if err := outStream.Start(); err != nil {
		inStream.Stop()
		inStream.Close()
		outStream.Close()
		return fmt.Errorf("starting output stream: %w", err)
	}

	e.wg.Add(1)
	go e.run(inStream, outStream)
	return nil
}

// Stop safely shuts down the engine. Safe to call multiple times.
func (e *Engine) Stop() {
	e.stopOnce.Do(func() {
		close(e.stopCh)
	})
	e.wg.Wait()
}

func (e *Engine) run(inStream, outStream *portaudio.Stream) {
	defer e.wg.Done()
	defer inStream.Stop()
	defer inStream.Close()
	defer outStream.Stop()
	defer outStream.Close()

	metrics := NewMetricsCollector(e.sampleRate, MetricsSnapshotFPS)
	consecutiveErrs := 0

	for {
		select {
		case <-e.stopCh:
			return
		default:
		}

		if err := inStream.Read(); err != nil {
			consecutiveErrs++
			if consecutiveErrs >= maxConsecutiveErrs {
				time.Sleep(10 * time.Millisecond)
			}
			continue
		}

		p := e.params.Snapshot()

		for i := 0; i < e.framesPerBuf; i++ {
			lIn := e.inBuf[i*2]
			rIn := e.inBuf[i*2+1]

			metrics.RecordPre(lIn, rIn)

			var lOut, rOut float32
			if p.Bypass {
				lOut, rOut = lIn, rIn
			} else {
				lOut, rOut = e.processor.Process(lIn, rIn, &p)
			}

			e.outBuf[i*2] = lOut
			e.outBuf[i*2+1] = rOut

			metrics.RecordPost(lOut, rOut)

			if snap := metrics.Advance(); snap != nil {
				select {
				case e.metricsCh <- *snap:
				default:
				}
			}
		}

		if err := outStream.Write(); err != nil {
			consecutiveErrs++
			if consecutiveErrs >= maxConsecutiveErrs {
				time.Sleep(10 * time.Millisecond)
			}
			continue
		}

		consecutiveErrs = 0
	}
}

func (e *Engine) openInputStream() (*portaudio.Stream, error) {
	p := portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   e.inputDev,
			Channels: NumChannels,
			Latency:  e.inputDev.DefaultLowInputLatency,
		},
		SampleRate:      e.sampleRate,
		FramesPerBuffer: e.framesPerBuf,
	}
	return portaudio.OpenStream(p, e.inBuf)
}

func (e *Engine) openOutputStream() (*portaudio.Stream, error) {
	p := portaudio.StreamParameters{
		Output: portaudio.StreamDeviceParameters{
			Device:   e.outputDev,
			Channels: NumChannels,
			Latency:  e.outputDev.DefaultLowOutputLatency,
		},
		SampleRate:      e.sampleRate,
		FramesPerBuffer: e.framesPerBuf,
	}
	return portaudio.OpenStream(p, e.outBuf)
}

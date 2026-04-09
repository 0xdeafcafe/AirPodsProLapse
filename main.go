package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/0xdeafcafe/AirPodsProLapse/audio"
	"github.com/0xdeafcafe/AirPodsProLapse/dsp"
	"github.com/0xdeafcafe/AirPodsProLapse/tui"
	"github.com/gordonklaus/portaudio"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	inputName := flag.String("input", "", "input device name (substring match, skips selection)")
	outputName := flag.String("output", "", "output device name (substring match, skips selection)")
	sampleRate := flag.Float64("sample-rate", float64(audio.DefaultSampleRate), "sample rate in Hz")
	listDevices := flag.Bool("list-devices", false, "list available audio devices and exit")
	noAutoRoute := flag.Bool("no-auto-route", false, "don't automatically set BlackHole as system output")
	flag.Parse()

	if err := portaudio.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing PortAudio: %v\n", err)
		os.Exit(1)
	}
	defer portaudio.Terminate()

	devices, err := audio.ListDevices()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing devices: %v\n", err)
		os.Exit(1)
	}

	if *listDevices {
		audio.PrintDevices(devices)
		return
	}

	var inputDev, outputDev *portaudio.DeviceInfo

	if *inputName != "" && *outputName != "" {
		inputDev, err = audio.FindDevice(devices, *inputName, audio.Input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		outputDev, err = audio.FindDevice(devices, *outputName, audio.Output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		selectModel := tui.NewDeviceSelectModel(devices)
		p := tea.NewProgram(selectModel, tea.WithAltScreen())
		result, err := p.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running device selector: %v\n", err)
			os.Exit(1)
		}

		m := result.(tui.DeviceSelectModel)
		if !m.Done() {
			return
		}

		sel := m.Selection()
		inputDev = sel.Input
		outputDev = sel.Output
	}

	// Auto-route: set system output to BlackHole so all audio flows through us.
	var autoRouteEnabled bool
	var autoRouteRestore func()
	if !*noAutoRoute {
		restoreFn, err := setupAutoRoute(inputDev.Name, outputDev)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: auto-route failed: %v\n", err)
			fmt.Fprintln(os.Stderr, "You may need to manually set system output to BlackHole.")
		} else {
			autoRouteEnabled = true
			autoRouteRestore = restoreFn
		}
	}

	params := dsp.NewParams(*sampleRate)
	metricsCh := make(chan audio.AudioSnapshot, 4)

	engine := audio.NewEngine(inputDev, outputDev, params, metricsCh, *sampleRate)
	if err := engine.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting audio engine: %v\n", err)
		os.Exit(1)
	}

	model := tui.NewModel(tui.ModelConfig{
		Engine:           engine,
		Params:           params,
		MetricsCh:        metricsCh,
		InputDev:         inputDev,
		OutputDev:        outputDev,
		Devices:          devices,
		SampleRate:       *sampleRate,
		AutoRouteEnabled: autoRouteEnabled,
		AutoRouteRestore: autoRouteRestore,
	})

	// Bubbletea handles SIGINT/SIGTERM internally and triggers tea.Quit,
	// so we don't need a separate signal handler. Cleanup runs via the
	// final model state after Run() returns.
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}

	if fm, ok := finalModel.(tui.Model); ok {
		fm.Cleanup()
	}
}

func setupAutoRoute(inputDevName string, outputDev *portaudio.DeviceInfo) (restore func(), err error) {
	blackholeID, bhName, err := audio.FindCoreAudioDevice(inputDevName)
	if err != nil {
		return nil, fmt.Errorf("finding BlackHole device: %w", err)
	}

	originalID := audio.GetDefaultOutputDeviceID()
	originalName := audio.GetDeviceName(originalID)

	if originalID == blackholeID {
		restoreID, _, findErr := audio.FindCoreAudioDevice(outputDev.Name)
		if findErr == nil {
			originalID = restoreID
			originalName = outputDev.Name
		}
	}

	if err := audio.SetDefaultOutputDevice(blackholeID); err != nil {
		return nil, fmt.Errorf("setting output to %s: %w", bhName, err)
	}

	restore = func() {
		if err := audio.SetDefaultOutputDevice(originalID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to restore output to %s: %v\n", originalName, err)
		}
	}

	return restore, nil
}

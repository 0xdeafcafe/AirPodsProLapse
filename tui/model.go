package tui

import (
	"fmt"
	"time"

	"github.com/0xdeafcafe/AirPodsProLapse/audio"
	"github.com/0xdeafcafe/AirPodsProLapse/dsp"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gordonklaus/portaudio"
)

type FocusedParam int

const (
	FocusBlend FocusedParam = iota
	FocusDelay
	FocusCutoff
	focusParamCount
)

type subView int

const (
	viewMain subView = iota
	viewPickInput
	viewPickOutput
)

type tickMsg time.Time

type ModelConfig struct {
	Engine     *audio.Engine
	Params     *dsp.Params
	MetricsCh  chan audio.AudioSnapshot // bidirectional — TUI reads, engine writes
	InputDev   *portaudio.DeviceInfo
	OutputDev  *portaudio.DeviceInfo
	Devices    []audio.DeviceEntry
	SampleRate float64

	AutoRouteEnabled bool
	AutoRouteRestore func()
}

type Model struct {
	// DSP
	params *dsp.Params

	// Audio engine
	engine     *audio.Engine
	metricsCh  chan audio.AudioSnapshot
	snapshot   audio.AudioSnapshot
	sampleRate float64

	// Devices
	allDevices []audio.DeviceEntry
	inputDev   *portaudio.DeviceInfo
	outputDev  *portaudio.DeviceInfo

	// Auto-route
	autoRouteEnabled bool
	autoRouteRestore func()

	// Main view state
	focused FocusedParam
	width   int
	height  int

	// Inline device picker
	subView      subView
	pickerItems  []audio.DeviceEntry
	pickerCursor int

	// Status message (shown briefly after actions)
	statusMsg      string
	statusExpireAt time.Time
}

func NewModel(cfg ModelConfig) Model {
	return Model{
		params:           cfg.Params,
		engine:           cfg.Engine,
		metricsCh:        cfg.MetricsCh,
		sampleRate:       cfg.SampleRate,
		allDevices:       cfg.Devices,
		inputDev:         cfg.InputDev,
		outputDev:        cfg.OutputDev,
		autoRouteEnabled: cfg.AutoRouteEnabled,
		autoRouteRestore: cfg.AutoRouteRestore,
		subView:          viewMain,
	}
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(33*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		if !m.statusExpireAt.IsZero() && time.Now().After(m.statusExpireAt) {
			m.statusMsg = ""
			m.statusExpireAt = time.Time{}
		}
		for {
			select {
			case snap := <-m.metricsCh:
				m.snapshot = snap
			default:
				return m, tickCmd()
			}
		}

	case tea.KeyMsg:
		if m.subView != viewMain {
			return m.updatePicker(msg)
		}
		return m.updateMain(msg)
	}

	return m, nil
}

func (m Model) updateMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case " ":
		m.params.ToggleBypass()
	case "m":
		m.params.ToggleMode()
	case "tab":
		m.focused = (m.focused + 1) % focusParamCount
	case "shift+tab":
		m.focused = (m.focused - 1 + focusParamCount) % focusParamCount
	case "up", "+", "=":
		m.adjustFocused(+1)
	case "down", "-":
		m.adjustFocused(-1)
	case "i":
		m.openPicker(viewPickInput)
	case "o":
		m.openPicker(viewPickOutput)
	case "r":
		m.toggleAutoRoute()
	}
	return m, nil
}

func (m *Model) openPicker(view subView) {
	m.subView = view
	m.pickerCursor = 0

	switch view {
	case viewPickInput:
		m.pickerItems = nil
		for _, d := range m.allDevices {
			if d.Direction == audio.Input {
				m.pickerItems = append(m.pickerItems, d)
			}
		}
	case viewPickOutput:
		m.pickerItems = nil
		for _, d := range m.allDevices {
			if d.Direction == audio.Output && d.Info.Name != m.inputDev.Name {
				m.pickerItems = append(m.pickerItems, d)
			}
		}
	}
}

func (m Model) updatePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "i", "o":
		m.subView = viewMain
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.pickerCursor > 0 {
			m.pickerCursor--
		}
	case "down", "j":
		if m.pickerCursor < len(m.pickerItems)-1 {
			m.pickerCursor++
		}
	case "enter":
		if m.pickerCursor < len(m.pickerItems) {
			m.selectPickerDevice()
		}
	}
	return m, nil
}

func (m *Model) selectPickerDevice() {
	selected := m.pickerItems[m.pickerCursor].Info

	switch m.subView {
	case viewPickInput:
		if selected.Name == m.inputDev.Name {
			m.subView = viewMain
			return
		}
		m.inputDev = selected

		// Re-enable auto-route for new input if it was on.
		if m.autoRouteEnabled {
			if m.autoRouteRestore != nil {
				m.autoRouteRestore()
			}
			m.enableAutoRoute()
		}

	case viewPickOutput:
		if selected.Name == m.outputDev.Name {
			m.subView = viewMain
			return
		}
		m.outputDev = selected
	}

	// Restart engine with new devices.
	m.restartEngine()
	m.subView = viewMain
}

func (m *Model) restartEngine() {
	m.engine.Stop()

	m.engine = audio.NewEngine(m.inputDev, m.outputDev, m.params, m.metricsCh, m.sampleRate)
	if err := m.engine.Start(); err != nil {
		m.setStatus(fmt.Sprintf("Engine error: %v", err))
		return
	}
	m.setStatus(fmt.Sprintf("Switched: %s -> %s", m.inputDev.Name, m.outputDev.Name))
}

func (m *Model) toggleAutoRoute() {
	if m.autoRouteEnabled {
		// Disable: restore original output.
		if m.autoRouteRestore != nil {
			m.autoRouteRestore()
			m.autoRouteRestore = nil
		}
		m.autoRouteEnabled = false
		m.setStatus("Auto-route OFF (set system output manually)")
	} else {
		m.enableAutoRoute()
	}
}

func (m *Model) enableAutoRoute() {
	blackholeID, _, err := audio.FindCoreAudioDevice(m.inputDev.Name)
	if err != nil {
		m.setStatus(fmt.Sprintf("Auto-route failed: %v", err))
		return
	}

	// Save current default output for restore, but skip if it's already BlackHole.
	originalID := audio.GetDefaultOutputDeviceID()
	restoreID := originalID
	restoreName := audio.GetDeviceName(originalID)
	if originalID == blackholeID {
		// Already BlackHole — restore to the selected output device instead.
		if rid, _, err := audio.FindCoreAudioDevice(m.outputDev.Name); err == nil {
			restoreID = rid
			restoreName = m.outputDev.Name
		}
	}

	if err := audio.SetDefaultOutputDevice(blackholeID); err != nil {
		m.setStatus(fmt.Sprintf("Auto-route failed: %v", err))
		return
	}

	m.autoRouteRestore = func() {
		if err := audio.SetDefaultOutputDevice(restoreID); err != nil {
			// Best effort.
			_ = err
		}
		_ = restoreName
	}
	m.autoRouteEnabled = true
	m.setStatus("Auto-route ON (system output -> BlackHole)")
}

func (m *Model) setStatus(msg string) {
	m.statusMsg = msg
	m.statusExpireAt = time.Now().Add(2 * time.Second)
}

// Cleanup should be called after the TUI exits to restore system state.
func (m Model) Cleanup() {
	m.engine.Stop()
	if m.autoRouteEnabled && m.autoRouteRestore != nil {
		m.autoRouteRestore()
	}
}

func (m Model) adjustFocused(direction int) {
	p := m.params.Snapshot()
	switch m.focused {
	case FocusBlend:
		m.params.AdjustBlend(float32(direction) * 0.01)
	case FocusDelay:
		if p.Mode == dsp.ModeRealistic {
			m.params.AdjustDelayMs(float32(direction) * 0.05)
		}
	case FocusCutoff:
		if p.Mode == dsp.ModeRealistic {
			m.params.AdjustFilterCutoff(float32(direction) * 50)
		}
	}
}

func (m Model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	p := m.params.Snapshot()

	// Title
	title := titleStyle.Width(m.width).Render("AirPodsProLapse v0.1.0")

	// Device info + auto-route status
	routeIcon := dimParamStyle.Render("[route: off]")
	if m.autoRouteEnabled {
		routeIcon = meterBarStyle.Render("[route: on]")
	}
	devInfo := dimParamStyle.Render("  "+m.inputDev.Name+"  ->  "+m.outputDev.Name) + "  " + routeIcon

	// Status message
	var statusLine string
	if m.statusMsg != "" {
		statusLine = bypassOnStyle.Render("  " + m.statusMsg)
	}

	// Waveforms
	halfWidth := m.width/2 - 2
	waveHeight := 6

	preWave := boxStyle.Width(halfWidth).Render(
		renderWaveformPair(
			m.snapshot.PreLeft, m.snapshot.PreRight,
			halfWidth-2, waveHeight,
			"Input (Pre-Crossfeed)",
		),
	)
	postWave := boxStyle.Width(halfWidth).Render(
		renderWaveformPair(
			m.snapshot.PostLeft, m.snapshot.PostRight,
			halfWidth-2, waveHeight,
			"Output (Post-Crossfeed)",
		),
	)
	waveRow := lipgloss.JoinHorizontal(lipgloss.Top, preWave, postWave)

	// Level meters
	metersContent := renderAllMeters(m.snapshot, m.width-6)
	meters := boxStyle.Width(m.width - 2).Render(
		headerStyle.Render("Levels") + "\n" + metersContent,
	)

	// Parameters
	paramsContent := renderParamsTable(p, m.focused)
	params := boxStyle.Width(m.width - 2).Render(
		headerStyle.Render("Parameters") + "\n" + paramsContent,
	)

	// Help
	help := helpStyle.Width(m.width).Render(
		"Tab: param  Up/Down: adjust  Space: bypass  M: mode  I/O: devices  R: route  Q: quit",
	)

	parts := []string{title, devInfo}
	if statusLine != "" {
		parts = append(parts, statusLine)
	}
	parts = append(parts, waveRow, meters, params, help)

	main := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// Overlay picker if active.
	if m.subView != viewMain {
		return m.renderWithPicker(main)
	}

	return main
}

func (m Model) renderWithPicker(background string) string {
	var title string
	switch m.subView {
	case viewPickInput:
		title = "Select Input Device"
	case viewPickOutput:
		title = "Select Output Device"
	}

	list := ""
	for i, d := range m.pickerItems {
		ch := d.Info.MaxInputChannels
		if d.Direction == audio.Output {
			ch = d.Info.MaxOutputChannels
		}
		label := fmt.Sprintf("%s  (%dch, %.0f Hz)", d.Info.Name, ch, d.Info.DefaultSampleRate)

		// Mark current device.
		var current string
		if (m.subView == viewPickInput && d.Info.Name == m.inputDev.Name) ||
			(m.subView == viewPickOutput && d.Info.Name == m.outputDev.Name) {
			current = " *"
		}

		if i == m.pickerCursor {
			list += activeParamStyle.Render("  > "+label+current) + "\n"
		} else {
			list += normalParamStyle.Render("    "+label+current) + "\n"
		}
	}

	pickerHelp := dimParamStyle.Render("  Up/Down: navigate  Enter: select  Esc: cancel")

	picker := boxStyle.
		Width(m.width - 4).
		BorderForeground(lipgloss.Color("170")).
		Render(
			headerStyle.Render(title) + "\n\n" +
				list + "\n" +
				pickerHelp,
		)

	// Place picker over the background — just show picker centered.
	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Width(m.width).Render("AirPodsProLapse v0.1.0"),
		"",
		lipgloss.Place(m.width, m.height-2, lipgloss.Center, lipgloss.Center, picker),
	)
}

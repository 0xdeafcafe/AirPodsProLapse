package tui

import (
	"fmt"

	"github.com/0xdeafcafe/AirPodsProLapse/audio"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gordonklaus/portaudio"
)

type selectPhase int

const (
	phaseSelectInput selectPhase = iota
	phaseSelectOutput
	phaseDone
)

// DeviceSelection holds the user's chosen devices.
type DeviceSelection struct {
	Input  *portaudio.DeviceInfo
	Output *portaudio.DeviceInfo
}

// DeviceSelectModel is a bubbletea model for interactive device selection.
type DeviceSelectModel struct {
	allDevices      []audio.DeviceEntry
	inputs          []audio.DeviceEntry
	outputs         []audio.DeviceEntry
	filteredOutputs []audio.DeviceEntry // outputs minus the selected input device
	phase           selectPhase
	cursor          int
	selection       DeviceSelection
	width           int
	height          int
}

func NewDeviceSelectModel(devices []audio.DeviceEntry) DeviceSelectModel {
	var inputs, outputs []audio.DeviceEntry
	for _, d := range devices {
		switch d.Direction {
		case audio.Input:
			inputs = append(inputs, d)
		case audio.Output:
			outputs = append(outputs, d)
		}
	}
	return DeviceSelectModel{
		allDevices: devices,
		inputs:     inputs,
		outputs:    outputs,
		phase:      phaseSelectInput,
	}
}

func (m DeviceSelectModel) Selection() DeviceSelection { return m.selection }
func (m DeviceSelectModel) Done() bool                 { return m.phase == phaseDone }

func (m DeviceSelectModel) Init() tea.Cmd { return nil }

func (m DeviceSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			m.cursor = min(m.cursor+1, m.listLen()-1)
		case "enter":
			return m.selectCurrent()
		}
	}
	return m, nil
}

func (m DeviceSelectModel) selectCurrent() (tea.Model, tea.Cmd) {
	switch m.phase {
	case phaseSelectInput:
		if m.cursor < len(m.inputs) {
			m.selection.Input = m.inputs[m.cursor].Info
			// Filter outputs: remove any device sharing the same name as input.
			m.filteredOutputs = nil
			for _, d := range m.outputs {
				if d.Info.Name != m.selection.Input.Name {
					m.filteredOutputs = append(m.filteredOutputs, d)
				}
			}
			m.phase = phaseSelectOutput
			m.cursor = 0
		}
	case phaseSelectOutput:
		list := m.outputList()
		if m.cursor < len(list) {
			m.selection.Output = list[m.cursor].Info
			m.phase = phaseDone
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m DeviceSelectModel) outputList() []audio.DeviceEntry {
	if m.filteredOutputs != nil {
		return m.filteredOutputs
	}
	return m.outputs
}

func (m DeviceSelectModel) listLen() int {
	switch m.phase {
	case phaseSelectInput:
		return len(m.inputs)
	case phaseSelectOutput:
		return len(m.outputList())
	}
	return 0
}

func (m DeviceSelectModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	title := titleStyle.Width(m.width).Render("AirPodsProLapse v0.1.0")

	var prompt string
	var items []audio.DeviceEntry

	switch m.phase {
	case phaseSelectInput:
		prompt = "Select input device (audio source):"
		items = m.inputs
	case phaseSelectOutput:
		prompt = "Select output device (headphones):"
		items = m.outputList()
	}

	promptStr := headerStyle.Render("  " + prompt)

	// Show selected input when choosing output.
	var selectedInfo string
	if m.phase == phaseSelectOutput && m.selection.Input != nil {
		selectedInfo = dimParamStyle.Render(fmt.Sprintf("  Input: %s", m.selection.Input.Name))
	}

	// Render device list.
	list := ""
	for i, d := range items {
		ch := d.Info.MaxInputChannels
		if d.Direction == audio.Output {
			ch = d.Info.MaxOutputChannels
		}
		label := fmt.Sprintf("%s  (%dch, %.0f Hz)", d.Info.Name, ch, d.Info.DefaultSampleRate)

		if i == m.cursor {
			line := activeParamStyle.Render("  > " + label)
			list += line + "\n"
		} else {
			line := normalParamStyle.Render("    " + label)
			list += line + "\n"
		}
	}

	listBox := boxStyle.Width(m.width - 2).Render(list)

	help := helpStyle.Width(m.width).Render("Up/Down: navigate  Enter: select  Q: quit")

	parts := []string{title, ""}
	if selectedInfo != "" {
		parts = append(parts, selectedInfo, "")
	}
	parts = append(parts, promptStr, "", listBox, "", help)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

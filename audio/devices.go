package audio

import (
	"fmt"
	"strings"

	"github.com/gordonklaus/portaudio"
)

type Direction int

const (
	Input Direction = iota
	Output
)

type DeviceEntry struct {
	Info      *portaudio.DeviceInfo
	Index     int
	Direction Direction
}

// ListDevices returns all available audio devices classified by direction.
func ListDevices() ([]DeviceEntry, error) {
	devices, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("enumerating devices: %w", err)
	}

	var entries []DeviceEntry
	for i, d := range devices {
		if d.MaxInputChannels > 0 {
			entries = append(entries, DeviceEntry{Info: d, Index: i, Direction: Input})
		}
		if d.MaxOutputChannels > 0 {
			entries = append(entries, DeviceEntry{Info: d, Index: i, Direction: Output})
		}
	}
	return entries, nil
}

// FindDevice searches for a device by name substring and direction.
// Matching is case-insensitive contains.
func FindDevice(devices []DeviceEntry, name string, dir Direction) (*portaudio.DeviceInfo, error) {
	nameLower := strings.ToLower(name)
	for _, d := range devices {
		if d.Direction == dir && strings.Contains(strings.ToLower(d.Info.Name), nameLower) {
			return d.Info, nil
		}
	}
	dirStr := "input"
	if dir == Output {
		dirStr = "output"
	}
	return nil, fmt.Errorf("no %s device matching %q found", dirStr, name)
}

// PrintDevices prints a formatted table of all devices to stdout.
func PrintDevices(devices []DeviceEntry) {
	fmt.Println()
	fmt.Printf("  %-4s  %-6s  %-40s  %-4s  %s\n", "IDX", "DIR", "NAME", "CH", "SAMPLE RATE")
	fmt.Printf("  %-4s  %-6s  %-40s  %-4s  %s\n", "---", "------", "----", "--", "-----------")
	for _, d := range devices {
		dir := "IN"
		ch := d.Info.MaxInputChannels
		if d.Direction == Output {
			dir = "OUT"
			ch = d.Info.MaxOutputChannels
		}
		fmt.Printf("  %-4d  %-6s  %-40s  %-4d  %.0f Hz\n",
			d.Index, dir, d.Info.Name, ch, d.Info.DefaultSampleRate)
	}
	fmt.Println()
}

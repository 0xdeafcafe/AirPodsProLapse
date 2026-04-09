package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/0xdeafcafe/AirPodsProLapse/audio"
)

func renderMeter(peak float64, width int, label string) string {
	dbfs := 20.0 * math.Log10(math.Max(peak, 1e-10))
	dbStr := fmt.Sprintf("%6.1f dB", dbfs)

	barWidth := width - len(label) - len(dbStr) - 4
	if barWidth < 1 {
		barWidth = 1
	}

	barLen := int(peak * float64(barWidth))
	if barLen < 0 {
		barLen = 0
	}
	if barLen > barWidth {
		barLen = barWidth
	}

	filled := strings.Repeat("█", barLen)
	empty := strings.Repeat("░", barWidth-barLen)

	if peak > 0.95 {
		filled = meterClipStyle.Render(filled)
	} else {
		filled = meterBarStyle.Render(filled)
	}

	return fmt.Sprintf("%s %s%s %s", label, filled, empty, dbStr)
}

func renderAllMeters(snap audio.AudioSnapshot, width int) string {
	lines := []string{
		renderMeter(snap.PrePeakLeft, width, "Pre  L"),
		renderMeter(snap.PrePeakRight, width, "Pre  R"),
		renderMeter(snap.PostPeakLeft, width, "Post L"),
		renderMeter(snap.PostPeakRight, width, "Post R"),
	}
	return strings.Join(lines, "\n")
}

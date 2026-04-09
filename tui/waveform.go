package tui

import (
	"github.com/guptarohit/asciigraph"
)

func renderWaveformPair(left, right []float64, width, height int, title string) string {
	if len(left) == 0 && len(right) == 0 {
		return ""
	}

	// Ensure we have data for both channels.
	if len(left) == 0 {
		left = make([]float64, len(right))
	}
	if len(right) == 0 {
		right = make([]float64, len(left))
	}

	// Clamp width to avoid negative values.
	graphWidth := width - 10
	if graphWidth < 10 {
		graphWidth = 10
	}

	return asciigraph.PlotMany(
		[][]float64{left, right},
		asciigraph.Width(graphWidth),
		asciigraph.Height(height),
		asciigraph.Caption(title),
		asciigraph.LowerBound(-1.0),
		asciigraph.UpperBound(1.0),
		asciigraph.SeriesColors(asciigraph.Cyan, asciigraph.Red),
		asciigraph.Precision(1),
	)
}

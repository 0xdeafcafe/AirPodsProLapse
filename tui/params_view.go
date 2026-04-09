package tui

import (
	"fmt"

	"github.com/0xdeafcafe/AirPodsProLapse/dsp"
)

func renderParamsTable(p dsp.ParamSnapshot, focused FocusedParam) string {
	modeStr := p.Mode.String()

	blendStr := fmt.Sprintf("%.1f%%", p.Blend*100)
	delayStr := fmt.Sprintf("%.2f ms", p.DelayMs)
	cutoffStr := fmt.Sprintf("%.0f Hz", p.FilterCutoffHz)

	bypassStr := "OFF"
	bypassSt := bypassOffStyle
	if p.Bypass {
		bypassStr = "ON"
		bypassSt = bypassOnStyle
	}

	// Style each parameter based on focus and mode applicability.
	stMode := normalParamStyle
	stBlend := normalParamStyle
	stDelay := normalParamStyle
	stCutoff := normalParamStyle

	// Delay and cutoff are dimmed in simple mode.
	if p.Mode == dsp.ModeSimple {
		stDelay = dimParamStyle
		stCutoff = dimParamStyle
	}

	// Highlight focused parameter.
	switch focused {
	case FocusBlend:
		stBlend = activeParamStyle
		blendStr = ">" + blendStr + "<"
	case FocusDelay:
		if p.Mode == dsp.ModeRealistic {
			stDelay = activeParamStyle
		}
		delayStr = ">" + delayStr + "<"
	case FocusCutoff:
		if p.Mode == dsp.ModeRealistic {
			stCutoff = activeParamStyle
		}
		cutoffStr = ">" + cutoffStr + "<"
	}

	line1 := fmt.Sprintf("  %s  %-12s   %s  %-14s",
		headerStyle.Render("Mode:"), stMode.Render(modeStr),
		headerStyle.Render("Delay:"), stDelay.Render(delayStr),
	)
	line2 := fmt.Sprintf("  %s %-14s   %s %-14s",
		headerStyle.Render("Blend:"), stBlend.Render(blendStr),
		headerStyle.Render("Cutoff:"), stCutoff.Render(cutoffStr),
	)
	line3 := fmt.Sprintf("  %s %s",
		headerStyle.Render("Bypass:"), bypassSt.Render(bypassStr),
	)

	return line1 + "\n" + line2 + "\n" + line3
}

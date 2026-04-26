package uikit

// PanelSize returns (width, height) for a centered modal panel.
// 70% of terminal width (min 80) and 65% of terminal height (min 20) leaves
// visible margins on all sides so lipgloss.Place() produces a centred effect.
func PanelSize(termW, termH int) (w, h int) {
	w = termW * 70 / 100
	if w < 80 {
		w = 80
	}
	h = termH * 65 / 100
	if h < 20 {
		h = 20
	}
	return
}

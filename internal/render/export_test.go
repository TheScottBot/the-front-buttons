package render

// FitLabelForTest exposes label fitting so its result can be measured.
func FitLabelForTest(renderer *Renderer, label string) (string, int) {
	return renderer.fitLabel(label)
}

// LabelWidthForTest exposes the width a label must fit within.
func LabelWidthForTest(renderer *Renderer) int {
	return renderer.labelWidth()
}

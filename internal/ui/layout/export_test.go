package layout

// SetActivePresetIndex forces the active preset index for the current page,
// bypassing bounds checks. Only for use in tests.
func (m *Manager) SetActivePresetIndex(idx int) {
	m.activePreset[m.activePage] = idx
}

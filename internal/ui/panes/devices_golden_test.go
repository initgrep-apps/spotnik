package panes_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/goldentest"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// TestDevicesPane_View_Devices verifies the golden snapshot of DeviceOverlay
// with 3 devices listed, active device marked ✓, at 80×24.
func TestDevicesPane_View_Devices(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	overlay := panes.NewDeviceOverlay(s, th)
	overlay.SetSize(80, 24)

	// Populate devices via DevicesLoadedMsg (simulates root app dispatching after fetch).
	overlay.Update(panes.DevicesLoadedMsg{
		Devices: []panes.DeviceInfo{
			{ID: "dev1", Name: "MacBook Pro", Type: "Computer", IsActive: true},
			{ID: "dev2", Name: "iPhone 15", Type: "Smartphone", IsActive: false},
			{ID: "dev3", Name: "Living Room Speaker", Type: "Speaker", IsActive: false},
		},
	})

	tm := goldentest.NewPaneTest(t, overlay, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestDevicesPane_View_Empty verifies the golden snapshot of DeviceOverlay
// when no devices are available at 80×24.
func TestDevicesPane_View_Empty(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	overlay := panes.NewDeviceOverlay(s, th)
	overlay.SetSize(80, 24)

	// Send empty device list to trigger empty state.
	overlay.Update(panes.DevicesLoadedMsg{
		Devices: []panes.DeviceInfo{},
	})

	tm := goldentest.NewPaneTest(t, overlay, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestDevicesPane_View_Narrow verifies the golden snapshot of DeviceOverlay
// at narrow terminal width (40×24).
func TestDevicesPane_View_Narrow(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	overlay := panes.NewDeviceOverlay(s, th)
	overlay.SetSize(40, 24)

	// Populate devices via DevicesLoadedMsg.
	overlay.Update(panes.DevicesLoadedMsg{
		Devices: []panes.DeviceInfo{
			{ID: "dev1", Name: "MacBook Pro", Type: "Computer", IsActive: true},
			{ID: "dev2", Name: "iPhone 15", Type: "Smartphone", IsActive: false},
			{ID: "dev3", Name: "Living Room Speaker", Type: "Speaker", IsActive: false},
		},
	})

	tm := goldentest.NewPaneTest(t, overlay, 40, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

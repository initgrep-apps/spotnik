// Package panes — DeviceOverlay is the floating device switcher overlay.
// It lists all available Spotify Connect devices and allows the user to transfer
// playback with a single keypress. It never imports api/ directly.
package panes

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// DeviceOverlay is the floating device-switcher overlay model.
// It renders a navigable list of Spotify Connect devices and dispatches
// a TransferPlaybackMsg when the user selects a non-active device.
type DeviceOverlay struct {
	store   state.StateReader
	theme   theme.Theme
	devices []DeviceInfo
	cursor  int
	width   int
	height  int

	// statusMsg is a transient message shown in the overlay (e.g. "Already playing").
	statusMsg string
}

// NewDeviceOverlay constructs a DeviceOverlay wired to the given store and theme.
func NewDeviceOverlay(store state.StateReader, t theme.Theme) *DeviceOverlay {
	return &DeviceOverlay{
		store: store,
		theme: t,
	}
}

// SetSize updates the render dimensions for the overlay.
func (d *DeviceOverlay) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// Init returns a command that fetches the current device list from the store.
// NOTE: The actual API call is dispatched by a command returned from here; panes
// never call the API directly. The root app provides a FetchDevicesRequestMsg handler.
func (d *DeviceOverlay) Init() tea.Cmd {
	return func() tea.Msg {
		return FetchDevicesRequestMsg{}
	}
}

// Update handles messages for the DeviceOverlay.
func (d *DeviceOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case DevicesLoadedMsg:
		// Store mutations (SetDevicesError, ClearDevicesError, SetDevicesFetchedAt) are
		// handled by root app.Update() per Elm purity rule — panes must not mutate store.
		// This handler only updates the local devices list for rendering.
		if m.Err == nil {
			d.devices = m.Devices
			// Clamp the cursor so it stays in bounds when the list shrinks
			// (e.g. a device goes offline between refreshes). Without this,
			// the next Enter keypress panics with index out of range.
			if len(d.devices) == 0 {
				d.cursor = 0
			} else if d.cursor >= len(d.devices) {
				d.cursor = len(d.devices) - 1
			}
		}
		return d, nil

	case tea.KeyMsg:
		return d.handleKey(m)
	}
	return d, nil
}

// handleKey processes keyboard input for the device overlay.
func (d *DeviceOverlay) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyRunes && string(msg.Runes) == "j",
		msg.Type == tea.KeyDown:
		if d.cursor < len(d.devices)-1 {
			d.cursor++
		}
		return d, nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "k",
		msg.Type == tea.KeyUp:
		if d.cursor > 0 {
			d.cursor--
		}
		return d, nil

	case msg.Type == tea.KeyEnter:
		return d.handleEnter()

	case msg.Type == tea.KeyEsc:
		return d, func() tea.Msg { return DeviceOverlayClosedMsg{} }
	}
	return d, nil
}

// handleEnter processes the Enter key: transfers playback or shows "Already playing".
func (d *DeviceOverlay) handleEnter() (tea.Model, tea.Cmd) {
	if len(d.devices) == 0 {
		return d, nil
	}
	selected := d.devices[d.cursor]
	if selected.IsActive {
		d.statusMsg = "Already playing on this device"
		return d, nil
	}
	// Return a command that emits the transfer message (root app dispatches API call).
	deviceID := selected.ID
	deviceName := selected.Name
	return d, func() tea.Msg {
		return TransferPlaybackMsg{DeviceID: deviceID, DeviceName: deviceName}
	}
}

// View renders the device overlay with a btop-style border.
// The border title is "Devices" and the action shortcut "Enter select" is embedded
// in the top border line, consistent with the main grid pane borders.
func (d *DeviceOverlay) View() string {
	totalWidth := d.overlayWidth()
	innerWidth := totalWidth - 2
	if innerWidth < 2 {
		innerWidth = 2
	}

	var lines []string

	if d.statusMsg != "" {
		msgStyle := lipgloss.NewStyle().
			Foreground(d.theme.TextMuted())
		lines = append(lines, msgStyle.Render(d.statusMsg))
	}

	// NOTE: Device errors are routed through toast notifications (app.go).
	// store.DevicesError() is preserved for retry logic but never read in View().
	if len(d.devices) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(d.theme.TextMuted())
		lines = append(lines, emptyStyle.Render("No devices found"))
	} else {
		for i, dev := range d.devices {
			lines = append(lines, d.renderDevice(i, dev))
		}
	}

	inner := strings.Join(lines, "\n")
	inner = lipgloss.NewStyle().
		Width(innerWidth).MaxWidth(innerWidth).
		Render(inner)

	// Count actual rendered lines after width-constrained wrapping.
	renderedLines := strings.Split(inner, "\n")
	totalHeight := len(renderedLines) + 2 // +2 for top and bottom border rows
	if totalHeight < 4 {
		totalHeight = 4
	}

	cfg := layout.BorderConfig{
		Width:  totalWidth,
		Height: totalHeight,
		Title:  "Devices",
		Actions: []layout.Action{
			{Key: "Enter", Label: "select"},
		},
		AccentColor: d.theme.ActiveBorder(),
		Focused:     true, // overlays are always focused
		Theme:       d.theme,
	}

	return layout.RenderPaneBorder(inner, cfg)
}

// overlayWidth computes the overlay width based on device name lengths.
// Minimum 32 columns, maximum d.width (or large default).
func (d *DeviceOverlay) overlayWidth() int {
	minWidth := 32
	for _, dev := range d.devices {
		needed := len(dev.Name) + 12 // prefix icons + spacing + padding
		if needed > minWidth {
			minWidth = needed
		}
	}
	if d.width > 0 && minWidth > d.width {
		minWidth = d.width
	}
	return minWidth
}

// renderDevice renders a single device row with the appropriate symbol and label.
func (d *DeviceOverlay) renderDevice(idx int, dev DeviceInfo) string {
	isCursor := idx == d.cursor

	var bullet string
	var bulletStyle lipgloss.Style
	var nameStyle lipgloss.Style

	// Only cursor rows get Background(SelectedBg()); non-cursor rows have NO explicit
	// background so they blend with the composited overlay background rather than
	// rendering as opaque rectangles over the dimmed grid behind the overlay.
	if isCursor {
		bg := d.theme.SelectedBg()
		if dev.IsActive {
			bulletStyle = lipgloss.NewStyle().Foreground(d.theme.HeaderChipFg()).Background(bg)
			bullet = "◉"
		} else {
			bulletStyle = lipgloss.NewStyle().Foreground(d.theme.InactiveBorder()).Background(bg)
			bullet = "○"
		}
		nameStyle = lipgloss.NewStyle().Foreground(d.theme.TextPrimary()).Background(bg)
	} else {
		// Non-cursor: no Background() at all.
		if dev.IsActive {
			bulletStyle = lipgloss.NewStyle().Foreground(d.theme.HeaderChipFg())
			bullet = "◉"
		} else {
			bulletStyle = lipgloss.NewStyle().Foreground(d.theme.InactiveBorder())
			bullet = "○"
		}
		nameStyle = lipgloss.NewStyle().Foreground(d.theme.TextPrimary())
	}

	typeIcon := deviceTypeIcon(dev.Type)
	label := ""
	if dev.IsActive {
		if isCursor {
			label = lipgloss.NewStyle().
				Foreground(d.theme.Success()).
				Background(d.theme.SelectedBg()).
				Render(" [active]")
		} else {
			label = lipgloss.NewStyle().
				Foreground(d.theme.Success()).
				Render(" [active]")
		}
	}

	typeStyle := lipgloss.NewStyle().Foreground(d.theme.TextMuted())
	if isCursor {
		typeStyle = typeStyle.Background(d.theme.SelectedBg())
	}

	var capSuffix string
	if dev.IsRestricted {
		capSuffix = lipgloss.NewStyle().Foreground(d.theme.TextMuted()).Render(" [restricted]")
	} else if !dev.SupportsVolume {
		capSuffix = lipgloss.NewStyle().Foreground(d.theme.TextMuted()).Render(" (no vol)")
	}

	return bulletStyle.Render(bullet) + " " +
		typeStyle.Render(typeIcon) + " " +
		nameStyle.Render(dev.Name) +
		label +
		capSuffix
}

// deviceTypeIcon returns the display icon prefix for a Spotify device type.
// Falls back to "○" for any unrecognized type.
func deviceTypeIcon(deviceType string) string {
	switch deviceType {
	case "Computer":
		return "⊡"
	case "Smartphone":
		return "⊞"
	case "Speaker":
		return "⊟"
	case "TV":
		return "⊠"
	default:
		return "○"
	}
}

// SetTheme updates the theme reference for runtime theme switching.
func (d *DeviceOverlay) SetTheme(th theme.Theme) {
	d.theme = th
}

// FetchDevicesRequestMsg is emitted by DeviceOverlay.Init() to signal the root
// app model to fetch the device list and then deliver a DevicesLoadedMsg back.
type FetchDevicesRequestMsg struct{}

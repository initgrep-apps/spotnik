// Package panes — DeviceOverlay is the floating device switcher overlay.
// It lists all available Spotify Connect devices and allows the user to transfer
// playback with a single keypress. It never imports api/ directly.
package panes

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
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
	// err is set when a DevicesLoadedMsg carries an error; cleared on success.
	// When non-nil, View() renders the error state instead of device list or empty state.
	err error
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

// Err returns the last fetch error, or nil if the devices loaded successfully.
// Exported for test helpers.
func (d *DeviceOverlay) Err() error {
	return d.err
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
			d.err = nil
			d.devices = m.Devices
			// Clamp the cursor so it stays in bounds when the list shrinks
			// (e.g. a device goes offline between refreshes). Without this,
			// the next Enter keypress panics with index out of range.
			if len(d.devices) == 0 {
				d.cursor = 0
			} else if d.cursor >= len(d.devices) {
				d.cursor = len(d.devices) - 1
			}
		} else {
			d.err = m.Err // preserve last known device list; show error state
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

	// Error state takes priority: shows "Failed to load devices" with a connection hint,
	// distinct from the legitimate empty-devices state.
	if d.err != nil {
		return d.renderEmptyChrome("Failed to load devices", "Check your connection.")
	}
	if len(d.devices) == 0 {
		return d.renderEmptyChrome("No devices found", "Open Spotify on a device to see it here")
	}
	for i, dev := range d.devices {
		lines = append(lines, d.renderDevice(i, dev))
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

	chrome := uikit.OverlayChrome{
		Width:  totalWidth,
		Height: totalHeight,
		Title:  "Devices",
		Theme:  d.theme,
	}
	return chrome.Render(inner)
}

// renderEmptyChrome renders the overlay chrome around an EmptyState with the
// given text and hint. Used for both error and empty states to avoid duplication.
func (d *DeviceOverlay) renderEmptyChrome(text, hint string) string {
	totalWidth := d.overlayWidth()
	innerW := totalWidth - 2
	if innerW < 2 {
		innerW = 2
	}
	// Reserve enough height for the empty-state content (2 text lines + padding).
	const emptyStateHeight = 6
	inner := uikit.EmptyState{
		Text:   text,
		Hint:   hint,
		Width:  innerW,
		Height: emptyStateHeight - 2, // subtract border rows from inner height
		Theme:  d.theme,
	}.Render()
	chrome := uikit.OverlayChrome{
		Width:  totalWidth,
		Height: emptyStateHeight,
		Title:  "Devices",
		Theme:  d.theme,
	}
	return chrome.Render(inner)
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
	m := uikit.ActiveMode()
	if isCursor {
		bg := d.theme.SelectedBg()
		if dev.IsActive {
			bulletStyle = lipgloss.NewStyle().Foreground(d.theme.HeaderChipFg()).Background(bg)
			bullet = uikit.GlyphFor(uikit.GlyphActive, m)
		} else {
			bulletStyle = lipgloss.NewStyle().Foreground(d.theme.InactiveBorder()).Background(bg)
			bullet = uikit.GlyphFor(uikit.GlyphAvailable, m)
		}
		nameStyle = lipgloss.NewStyle().Foreground(d.theme.TextPrimary()).Background(bg)
	} else {
		// Non-cursor: no Background() at all.
		if dev.IsActive {
			bulletStyle = lipgloss.NewStyle().Foreground(d.theme.HeaderChipFg())
			bullet = uikit.GlyphFor(uikit.GlyphActive, m)
		} else {
			bulletStyle = lipgloss.NewStyle().Foreground(d.theme.InactiveBorder())
			bullet = uikit.GlyphFor(uikit.GlyphAvailable, m)
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

	return bulletStyle.Render(bullet) + " " +
		typeStyle.Render(typeIcon) + " " +
		nameStyle.Render(dev.Name) +
		label
}

// deviceTypeIcon returns the display icon prefix for a Spotify device type.
// Falls back to GlyphAvailable (○) for any unrecognized type, preserving the
// pre-migration visual appearance of "an available but uncategorised device".
func deviceTypeIcon(deviceType string) string {
	m := uikit.ActiveMode()
	switch deviceType {
	case "Computer":
		return uikit.GlyphFor(uikit.GlyphDeviceComputer, m)
	case "Smartphone":
		return uikit.GlyphFor(uikit.GlyphDevicePhone, m)
	case "Speaker":
		return uikit.GlyphFor(uikit.GlyphDeviceSpeaker, m)
	case "TV":
		return uikit.GlyphFor(uikit.GlyphDeviceTV, m)
	default:
		return uikit.GlyphFor(uikit.GlyphAvailable, m)
	}
}

// SetTheme updates the theme reference for runtime theme switching.
func (d *DeviceOverlay) SetTheme(th theme.Theme) {
	d.theme = th
}

// FetchDevicesRequestMsg is emitted by DeviceOverlay.Init() to signal the root
// app model to fetch the device list and then deliver a DevicesLoadedMsg back.
type FetchDevicesRequestMsg struct{}
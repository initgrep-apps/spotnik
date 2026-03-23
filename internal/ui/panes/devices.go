// Package panes — DeviceOverlay is the floating device switcher overlay.
// It lists all available Spotify Connect devices and allows the user to transfer
// playback with a single keypress. It never imports api/ directly.
package panes

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// devicesLoadedMsg is sent by the fetchDevices command after the device list
// has been fetched from the Spotify API. It is unexported because it is only
// ever produced by the command returned from DeviceOverlay.Init().
type devicesLoadedMsg struct {
	devices []api.Device
	err     error
}

// DeviceOverlay is the floating device-switcher overlay model.
// It renders a navigable list of Spotify Connect devices and dispatches
// a TransferPlaybackMsg when the user selects a non-active device.
type DeviceOverlay struct {
	store   *state.Store
	theme   theme.Theme
	devices []api.Device
	cursor  int
	width   int
	height  int

	// statusMsg is a transient message shown in the overlay (e.g. "Already playing").
	statusMsg string
}

// NewDeviceOverlay constructs a DeviceOverlay wired to the given store and theme.
func NewDeviceOverlay(store *state.Store, t theme.Theme) *DeviceOverlay {
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
	case devicesLoadedMsg:
		if m.err == nil {
			d.devices = m.devices
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

// View renders the device overlay as a bordered list.
func (d *DeviceOverlay) View() string {
	minWidth := 32
	// Compute width: fit the longest device name plus padding.
	for _, dev := range d.devices {
		needed := len(dev.Name) + 12 // prefix + padding
		if needed > minWidth {
			minWidth = needed
		}
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(d.theme.ActiveBorder()).
		Background(d.theme.SurfaceAlt()).
		Padding(0, 1).
		Width(minWidth)

	titleStyle := lipgloss.NewStyle().
		Foreground(d.theme.TextPrimary()).
		Background(d.theme.SurfaceAlt()).
		Bold(true)

	dividerStyle := lipgloss.NewStyle().
		Foreground(d.theme.TextMuted()).
		Background(d.theme.SurfaceAlt())

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("DEVICES"))
	sb.WriteString("\n")
	sb.WriteString(dividerStyle.Render(strings.Repeat("┄", minWidth-2)))
	sb.WriteString("\n")

	if d.statusMsg != "" {
		msgStyle := lipgloss.NewStyle().
			Foreground(d.theme.TextMuted()).
			Background(d.theme.SurfaceAlt())
		sb.WriteString(msgStyle.Render(d.statusMsg))
		sb.WriteString("\n")
	}

	if err := d.store.DevicesError(); err != nil {
		errView := components.RenderError(d.theme, minWidth-4, 4,
			"Failed to load devices", "Press d to retry")
		sb.WriteString(errView)
	} else if len(d.devices) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(d.theme.TextMuted()).
			Background(d.theme.SurfaceAlt())
		sb.WriteString(emptyStyle.Render("No devices found"))
	} else {
		for i, dev := range d.devices {
			sb.WriteString(d.renderDevice(i, dev))
			if i < len(d.devices)-1 {
				sb.WriteString("\n")
			}
		}
	}

	return borderStyle.Render(sb.String())
}

// renderDevice renders a single device row with the appropriate symbol and label.
func (d *DeviceOverlay) renderDevice(idx int, dev api.Device) string {
	isCursor := idx == d.cursor

	var bullet string
	var bulletStyle lipgloss.Style
	var nameStyle lipgloss.Style

	bg := d.theme.SurfaceAlt()
	if isCursor {
		bg = d.theme.SelectedBg()
	}

	if dev.IsActive {
		bulletStyle = lipgloss.NewStyle().
			Foreground(d.theme.DeviceActive()).
			Background(bg)
		bullet = "◉"
	} else {
		bulletStyle = lipgloss.NewStyle().
			Foreground(d.theme.InactiveBorder()).
			Background(bg)
		bullet = "○"
	}

	nameStyle = lipgloss.NewStyle().
		Foreground(d.theme.TextPrimary()).
		Background(bg)

	typeIcon := deviceTypeIcon(dev.Type)
	label := ""
	if dev.IsActive {
		label = lipgloss.NewStyle().
			Foreground(d.theme.Success()).
			Background(bg).
			Render(" [active]")
	}

	return bulletStyle.Render(bullet) + " " +
		lipgloss.NewStyle().Foreground(d.theme.TextMuted()).Background(bg).Render(typeIcon) + " " +
		nameStyle.Render(dev.Name) +
		label
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

// FetchDevicesRequestMsg is emitted by DeviceOverlay.Init() to signal the root
// app model to fetch the device list and then deliver a devicesLoadedMsg back.
type FetchDevicesRequestMsg struct{}

// NewDevicesLoadedMsg creates a devicesLoadedMsg to be dispatched by the root app
// after fetching the device list. This constructor allows the root app to create
// the unexported message type without importing internal pane details.
func NewDevicesLoadedMsg(devices []api.Device, err error) tea.Msg {
	return devicesLoadedMsg{devices: devices, err: err}
}

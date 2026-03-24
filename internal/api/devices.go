// Package api provides the Spotify HTTP client and all typed API response models.
// This file implements the device listing and playback transfer API calls.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// DevicesClient handles Spotify Connect device listing and playback transfer.
// It embeds BaseClient for shared HTTP functionality.
// It never imports ui/ — data flows through messages and the central store.
type DevicesClient struct {
	BaseClient
}

// NewDevicesClient returns a DevicesClient configured with the given base URL and token.
// In production, baseURL is "https://api.spotify.com"; in tests it is the mock server URL.
func NewDevicesClient(baseURL, token string) *DevicesClient {
	return &DevicesClient{BaseClient: NewBaseClient(baseURL, token)}
}

// SetHTTPClient overrides the default HTTP client used for API calls.
func (c *DevicesClient) SetHTTPClient(cl *http.Client) {
	c.setHTTPClient(cl)
}

// devicesResponse is the JSON envelope returned by GET /me/player/devices.
type devicesResponse struct {
	Devices []Device `json:"devices"`
}

// GetDevices fetches all available Spotify Connect devices for the current user.
// Returns an empty (non-nil) slice when Spotify reports no devices.
func (c *DevicesClient) GetDevices(ctx context.Context) ([]Device, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/v1/me/player/devices", nil)
	if err != nil {
		return nil, fmt.Errorf("getting devices: creating request: %w", err)
	}

	var result devicesResponse
	if err := c.doJSON(req, &result); err != nil {
		return nil, fmt.Errorf("getting devices: %w", err)
	}

	if result.Devices == nil {
		return []Device{}, nil
	}
	return result.Devices, nil
}

// transferPlaybackBody is the JSON body for PUT /me/player.
type transferPlaybackBody struct {
	DeviceIDs []string `json:"device_ids"`
	Play      bool     `json:"play"`
}

// TransferPlayback transfers Spotify playback to the device identified by deviceID.
// When play is true, playback starts immediately on the new device.
func (c *DevicesClient) TransferPlayback(ctx context.Context, deviceID string, play bool) error {
	payload := transferPlaybackBody{
		DeviceIDs: []string{deviceID},
		Play:      play,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("transferring playback: marshaling body: %w", err)
	}

	req, err := c.newRequest(ctx, http.MethodPut, "/v1/me/player", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("transferring playback: creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if err := c.doNoContent(req); err != nil {
		return fmt.Errorf("transferring playback: %w", err)
	}
	return nil
}

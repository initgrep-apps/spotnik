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
// It never imports ui/ — data flows through messages and the central store.
type DevicesClient struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewDevicesClient returns a DevicesClient configured with the given base URL and token.
// In production, baseURL is "https://api.spotify.com"; in tests it is the mock server URL.
func NewDevicesClient(baseURL, token string) *DevicesClient {
	return &DevicesClient{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{},
	}
}

// SetHTTPClient overrides the default HTTP client used for API calls.
func (c *DevicesClient) SetHTTPClient(cl *http.Client) {
	c.http = cl
}

// devicesResponse is the JSON envelope returned by GET /me/player/devices.
type devicesResponse struct {
	Devices []Device `json:"devices"`
}

// GetDevices fetches all available Spotify Connect devices for the current user.
// Returns an empty (non-nil) slice when Spotify reports no devices.
func (c *DevicesClient) GetDevices(ctx context.Context) ([]Device, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/me/player/devices", nil)
	if err != nil {
		return nil, fmt.Errorf("getting devices: creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting devices: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getting devices: unexpected status %d", resp.StatusCode)
	}

	var result devicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("getting devices: decoding response: %w", err)
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/v1/me/player", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("transferring playback: creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("transferring playback: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 204 No Content is the success response for transfer
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("transferring playback: unexpected status %d", resp.StatusCode)
	}

	return nil
}

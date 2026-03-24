package api

import "context"

// DevicesAPI defines all Spotify Connect device operations.
// Concrete implementation: *DevicesClient.
type DevicesAPI interface {
	Devices(ctx context.Context) ([]Device, error)
	TransferPlayback(ctx context.Context, deviceID string, play bool) error
}

// Compile-time assertion: *DevicesClient must implement DevicesAPI.
var _ DevicesAPI = (*DevicesClient)(nil)

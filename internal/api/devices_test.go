package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDevices_Success(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/fixtures/devices_response.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/player/devices", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	client := NewDevicesClient(srv.URL, "test-token")
	devices, err := client.GetDevices(context.Background())
	require.NoError(t, err)
	require.Len(t, devices, 3)

	// Verify parsed fields
	assert.Equal(t, "abc123", devices[0].ID)
	assert.Equal(t, "MacBook Pro Speakers", devices[0].Name)
	assert.Equal(t, "Computer", devices[0].Type)
	assert.True(t, devices[0].IsActive)
	assert.Equal(t, 80, devices[0].VolumePercent)

	assert.Equal(t, "def456", devices[1].ID)
	assert.Equal(t, "iPhone 14", devices[1].Name)
	assert.Equal(t, "Smartphone", devices[1].Type)
	assert.False(t, devices[1].IsActive)

	assert.Equal(t, "ghi789", devices[2].ID)
	assert.Equal(t, "Kitchen Speaker", devices[2].Name)
	assert.Equal(t, "Speaker", devices[2].Type)
}

func TestGetDevices_Empty(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/fixtures/devices_empty.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	client := NewDevicesClient(srv.URL, "test-token")
	devices, err := client.GetDevices(context.Background())
	require.NoError(t, err)
	assert.Empty(t, devices, "expected empty device slice, got non-empty")
}

func TestGetDevices_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewDevicesClient(srv.URL, "test-token")
	devices, err := client.GetDevices(context.Background())
	require.Error(t, err, "expected error on server error")
	assert.Nil(t, devices)
	assert.Contains(t, err.Error(), "getting devices")
}

func TestTransferPlayback_Success(t *testing.T) {
	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/me/player", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewDevicesClient(srv.URL, "test-token")
	err := client.TransferPlayback(context.Background(), "device-abc", true)
	require.NoError(t, err)

	// Verify body contains both device_ids and play fields
	ids, ok := gotBody["device_ids"].([]interface{})
	require.True(t, ok, "expected device_ids to be an array")
	require.Len(t, ids, 1)
	assert.Equal(t, "device-abc", ids[0])
	assert.Equal(t, true, gotBody["play"])
}

func TestTransferPlayback_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"error": {"message": "Premium required"}}`)
	}))
	defer srv.Close()

	client := NewDevicesClient(srv.URL, "test-token")
	err := client.TransferPlayback(context.Background(), "device-abc", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transferring playback")
}

func TestDevice_Unmarshal(t *testing.T) {
	raw := `{
		"id": "xyz789",
		"is_active": true,
		"is_private_session": false,
		"is_restricted": true,
		"name": "Living Room TV",
		"type": "TV",
		"volume_percent": 45
	}`

	var d Device
	err := json.Unmarshal([]byte(raw), &d)
	require.NoError(t, err)

	assert.Equal(t, "xyz789", d.ID)
	assert.True(t, d.IsActive)
	assert.False(t, d.IsPrivateSession)
	assert.True(t, d.IsRestricted)
	assert.Equal(t, "Living Room TV", d.Name)
	assert.Equal(t, "TV", d.Type)
	assert.Equal(t, 45, d.VolumePercent)
}

func TestNewDevicesClient_Fields(t *testing.T) {
	client := NewDevicesClient("http://example.com", "my-token")
	assert.Equal(t, "http://example.com", client.baseURL)
	assert.Equal(t, "my-token", client.accessToken)
	assert.NotNil(t, client.http)
}

// TestGetDevices_InvalidJSON ensures errors are wrapped with context.
func TestGetDevices_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "not-valid-json")
	}))
	defer srv.Close()

	client := NewDevicesClient(srv.URL, "test-token")
	devices, err := client.GetDevices(context.Background())
	require.Error(t, err)
	assert.Nil(t, devices)
	assert.True(t, strings.Contains(err.Error(), "decoding") || strings.Contains(err.Error(), "getting devices"),
		"expected error to contain 'decoding' or 'getting devices', got: %s", err.Error())
}

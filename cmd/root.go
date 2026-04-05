// Package cmd provides the CLI entry point for Spotnik via Cobra.
// It wires configuration, auth flow, theme loading, and application startup.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/keychain"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/spf13/cobra"
)

// spotifyClientID is the Spotify application client ID, embedded at build time
// via -ldflags "-X github.com/initgrep-apps/spotnik/cmd.spotifyClientID=...".
// Users can override this by setting client_id in their config.toml.
var spotifyClientID string

var rootCmd = &cobra.Command{
	Use:   "spotnik",
	Short: "A terminal Spotify client for developers",
	Long:  "Spotnik — keyboard-driven Spotify client for developers who live in the terminal.",
	RunE:  runApp,
}

// RootCommand returns the root cobra command.
// Exported for testing.
func RootCommand() *cobra.Command {
	return rootCmd
}

// Execute is the entry point called from main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Register the auth subcommand and its children.
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
}

// authCmd is the `spotnik auth` subcommand — forces a fresh re-authentication.
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Spotify",
	Long:  "Force a fresh Spotify authentication, overwriting any existing stored tokens.",
	RunE:  runAuth,
}

// authLogoutCmd is the `spotnik auth logout` subcommand.
var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove all stored Spotify credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		store := keychain.NewKeychainTokenStore()
		if err := LogoutTokens(store); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Logged out. All stored credentials removed.")
		return nil
	},
}

// authStatusCmd is the `spotnik auth status` subcommand.
var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		store := keychain.NewKeychainTokenStore()
		return PrintAuthStatus(store, cmd.OutOrStdout())
	},
}

// LogoutTokens removes all three token keys from the token store.
// Exported for testing.
func LogoutTokens(store keychain.TokenStore) error {
	if err := store.Delete(); err != nil {
		return fmt.Errorf("logging out: %w", err)
	}
	return nil
}

// PrintAuthStatus writes the current authentication status to w.
// Exported for testing.
func PrintAuthStatus(store keychain.TokenStore, w io.Writer) error {
	access, err := store.Get(keychain.KeyAccessToken)
	if err != nil {
		_, _ = fmt.Fprintln(w, "Status: not authenticated")
		return nil
	}
	if access == "" {
		_, _ = fmt.Fprintln(w, "Status: not authenticated")
		return nil
	}

	expiry, err := store.GetExpiry()
	if err != nil {
		_, _ = fmt.Fprintln(w, "Status: authenticated (expiry unknown)")
		return nil
	}

	_, _ = fmt.Fprintf(w, "Status: authenticated\n")
	_, _ = fmt.Fprintf(w, "Token expiry: %s\n", expiry.Format(time.RFC1123))

	expiringSoon, _ := store.IsExpiringSoon()
	if expiringSoon {
		_, _ = fmt.Fprintln(w, "Note: token is expiring soon and will be refreshed automatically")
	}

	return nil
}

// LoadConfig reads the config file at path, bootstraps it if missing, and
// applies the embedded client ID fallback. Exported for testing.
func LoadConfig(path string) (*config.Config, error) {
	return loadConfigFromPath(path, spotifyClientID)
}

// LoadConfigWithEmbedded is the testable variant of LoadConfig that accepts
// an explicit embedded client ID, allowing tests to inject values without
// relying on build-time ldflags. Exported for testing.
func LoadConfigWithEmbedded(path string, embeddedClientID string) (*config.Config, error) {
	return loadConfigFromPath(path, embeddedClientID)
}

// init registers the theme registry validator with the config package so that
// config.Load() can clamp unknown theme IDs without importing ui/theme directly.
func init() {
	config.ThemeValidator = func(id string) bool {
		for _, valid := range theme.Available() {
			if valid == id {
				return true
			}
		}
		return false
	}
}

// loadConfigFromPath is the testable implementation of LoadConfig that accepts
// an explicit embedded client ID so tests can inject values without build flags.
func loadConfigFromPath(path string, embeddedClientID string) (*config.Config, error) {
	// Bootstrap config file on first launch.
	if err := config.Bootstrap(path); err != nil {
		return nil, fmt.Errorf("bootstrapping config: %w", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}

	// Config client_id overrides embedded. Use embedded as fallback.
	if cfg.ClientID == "" {
		cfg.ClientID = embeddedClientID
	}

	// If still empty (no embedded, no config), show setup instructions.
	if cfg.ClientID == "" {
		printSetupInstructions()
		return nil, fmt.Errorf("no client_id available — see setup instructions above")
	}

	return cfg, nil
}

// EnsureAuthenticated checks the token state and runs the auth flow if needed.
// The tokenBaseURL parameter allows tests to override the Spotify token endpoint.
// Pass "" for production (uses the real Spotify endpoint).
// Exported for testing.
func EnsureAuthenticated(cfg *config.Config, store keychain.TokenStore, tokenBaseURL string) error {
	access, err := store.Get(keychain.KeyAccessToken)
	if err != nil || access == "" {
		// No token — start auth flow.
		return RunAuthFlow(cfg, store, tokenBaseURL)
	}

	expiringSoon, err := store.IsExpiringSoon()
	if err != nil {
		// Cannot determine expiry — start fresh auth.
		return RunAuthFlow(cfg, store, tokenBaseURL)
	}

	if expiringSoon {
		// Proactively refresh the token before it expires.
		refreshToken, err := store.Get(keychain.KeyRefreshToken)
		if err != nil || refreshToken == "" {
			return RunAuthFlow(cfg, store, tokenBaseURL)
		}

		// Refresh using the configured token endpoint.
		if err := api.Refresh(context.Background(), tokenBaseURL, refreshToken, cfg.ClientID, store); err != nil {
			if errors.Is(err, api.ErrInvalidGrant) {
				// Refresh token rejected — delete tokens and force re-auth.
				fmt.Fprintln(os.Stderr, "Session expired. Please re-authenticate.")
				_ = store.Delete()
				return RunAuthFlow(cfg, store, tokenBaseURL)
			}
			return fmt.Errorf("refreshing token: %w", err)
		}
	}

	return nil
}

// RunAuthFlow executes the full OAuth PKCE authorization flow.
// It generates PKCE credentials, starts the local callback server,
// opens the browser, waits for the callback, and exchanges the code for tokens.
// The tokenBaseURL parameter allows tests to override the Spotify token endpoint.
// Exported for testing.
func RunAuthFlow(cfg *config.Config, store keychain.TokenStore, tokenBaseURL string) error {
	// Generate PKCE verifier and challenge.
	verifier, err := api.GenerateCodeVerifier()
	if err != nil {
		return fmt.Errorf("generating PKCE verifier: %w", err)
	}
	challenge := api.ComputeCodeChallenge(verifier)

	// Start local callback server on a random port.
	callbackSrv, codeCh, err := api.StartCallbackServer()
	if err != nil {
		return fmt.Errorf("starting callback server: %w", err)
	}
	defer callbackSrv.Close()

	redirectURI := callbackSrv.URL + "/callback"

	// Build the authorization URL.
	authURL := api.BuildAuthURL(cfg.ClientID, redirectURI, challenge, api.SpotifyScopes)

	// Print auth URL for the CLI auth subcommand.
	fmt.Printf("\nVisit this URL to authorize:\n  %s\n\nWaiting for authorization...\n", authURL)

	// Open browser (best-effort — failure does not abort auth).
	if err := api.OpenBrowser(authURL); err != nil {
		// Browser open failed — user can still visit URL manually.
		_ = err
	}

	// Wait for callback with 5-minute timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	select {
	case result := <-codeCh:
		if result.Err != nil {
			return fmt.Errorf("authorization failed: %w", result.Err)
		}

		// Exchange code for tokens using the configured endpoint.
		_, err := api.ExchangeCode(
			context.Background(),
			tokenBaseURL,
			result.Code,
			verifier,
			redirectURI,
			cfg.ClientID,
			store,
		)
		if err != nil {
			return fmt.Errorf("exchanging authorization code: %w", err)
		}

		fmt.Println("Authorization successful! Starting spotnik...")
		return nil

	case <-ctx.Done():
		return fmt.Errorf("authorization timed out after 5 minutes — please try again")
	}
}

// CheckAuthState performs a non-blocking check of the token state.
// Returns true if authentication is required (no valid token or refresh failed),
// and false if a valid token exists or was successfully refreshed.
// Exported for testing.
func CheckAuthState(cfg *config.Config, store keychain.TokenStore) bool {
	access, err := store.Get(keychain.KeyAccessToken)
	if err != nil || access == "" {
		return true
	}

	expiringSoon, err := store.IsExpiringSoon()
	if err != nil {
		return true
	}

	if expiringSoon {
		refreshToken, err := store.Get(keychain.KeyRefreshToken)
		if err != nil || refreshToken == "" {
			return true
		}
		if err := api.Refresh(context.Background(), "", refreshToken, cfg.ClientID, store); err != nil {
			_ = store.Delete()
			return true
		}
	}

	return false
}

// runApp is the main command handler. It loads config, checks auth state,
// and launches the Bubble Tea application. The TUI starts immediately in
// all cases — auth happens inside the TUI if needed.
func runApp(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	store := keychain.NewKeychainTokenStore()
	needsAuth := CheckAuthState(cfg, store)

	opts := app.AppOptions{
		NeedsAuth:  needsAuth,
		ClientID:   cfg.ClientID,
		TokenStore: store,
	}
	a := app.New(cfg, opts)

	if !needsAuth {
		accessToken, _ := store.Get(keychain.KeyAccessToken)
		a.InitAPIClients(accessToken)
	}

	// Start the Bubble Tea program.
	// tea.WithMouseCellMotion() enables mouse wheel scroll events (Feature 52).
	p := tea.NewProgram(a, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()
	return err
}

// runAuth forces a fresh re-authentication flow.
func runAuth(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	store := keychain.NewKeychainTokenStore()
	// Delete existing tokens to force a fresh login.
	_ = store.Delete()

	return RunAuthFlow(cfg, store, "")
}

// loadConfig reads the config file from the default path, bootstraps it if
// missing, applies the embedded client ID fallback, and prints setup
// instructions if no client ID is available.
func loadConfig() (*config.Config, error) {
	return loadConfigFromPath(config.DefaultConfigPath(), spotifyClientID)
}

// PrintMissingClientIDInstructions writes setup instructions when client_id is missing.
// Exported for testing.
func PrintMissingClientIDInstructions(w io.Writer) error {
	lines := []string{
		"╭─────────────────────────────────────────────────────╮",
		"│  Spotnik setup required                             │",
		"│                                                     │",
		"│  1. Create a Spotify app:                           │",
		"│     https://developer.spotify.com/dashboard         │",
		"│                                                     │",
		"│  2. Add your client_id to:                          │",
		"│     ~/.config/spotnik/config.toml                   │",
		"│                                                     │",
		"│  Example config.toml:                               │",
		"│    [spotify]                                        │",
		"│    client_id = \"your-client-id-here\"              │",
		"╰─────────────────────────────────────────────────────╯",
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

// HandleMissingClientID prints setup instructions and returns an error.
// Exported for testing. The error signals the CLI to exit with code 1.
func HandleMissingClientID() error {
	_ = PrintMissingClientIDInstructions(os.Stdout)
	return fmt.Errorf("missing client_id — see setup instructions above")
}

// printSetupInstructions prints clear guidance when client_id is missing.
func printSetupInstructions() {
	_ = PrintMissingClientIDInstructions(os.Stdout)
}

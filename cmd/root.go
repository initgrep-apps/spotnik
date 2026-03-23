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

	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/keychain"
	"github.com/spf13/cobra"
)

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

// runApp is the main command handler. It loads config, checks auth state,
// and launches the Bubble Tea application.
func runApp(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	store := keychain.NewKeychainTokenStore()
	if err := ensureAuthenticated(cfg, store); err != nil {
		return err
	}

	// App wires the theme into pane constructors at startup.
	_ = app.New(cfg)

	// TODO(03-playback): start the Bubble Tea program here.
	fmt.Fprintln(os.Stderr, "spotnik: TUI not yet implemented — coming in Feature 03")
	return nil
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

	return runAuthFlow(cfg, store)
}

// loadConfig reads the config file and validates it.
// Prints setup instructions and exits if client_id is missing.
func loadConfig() (*config.Config, error) {
	path := config.DefaultConfigPath()
	cfg, err := config.Load(path)
	if err != nil {
		// If client_id is missing, print clear setup instructions.
		printSetupInstructions()
		return nil, err
	}
	return cfg, nil
}

// ensureAuthenticated checks the token state and runs auth flow if needed.
func ensureAuthenticated(cfg *config.Config, store keychain.TokenStore) error {
	access, err := store.Get(keychain.KeyAccessToken)
	if err != nil || access == "" {
		// No token — start auth flow.
		return runAuthFlow(cfg, store)
	}

	expiringSoon, err := store.IsExpiringSoon()
	if err != nil {
		// Cannot determine expiry — start fresh auth.
		return runAuthFlow(cfg, store)
	}

	if expiringSoon {
		// Proactively refresh the token before it expires.
		refreshToken, err := store.Get(keychain.KeyRefreshToken)
		if err != nil || refreshToken == "" {
			return runAuthFlow(cfg, store)
		}

		// Refresh using the production Spotify token endpoint ("" = default URL).
		if err := api.Refresh(context.Background(), "", refreshToken, cfg.ClientID, store); err != nil {
			if errors.Is(err, api.ErrInvalidGrant) {
				// Refresh token rejected — delete tokens and force re-auth.
				fmt.Fprintln(os.Stderr, "Session expired. Please re-authenticate.")
				_ = store.Delete()
				return runAuthFlow(cfg, store)
			}
			return fmt.Errorf("refreshing token: %w", err)
		}
	}

	return nil
}

// runAuthFlow executes the full OAuth PKCE authorization flow.
// It generates PKCE credentials, starts the local callback server,
// opens the browser, waits for the callback, and exchanges the code for tokens.
func runAuthFlow(cfg *config.Config, store keychain.TokenStore) error {
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

	// Print the first-run UX prompt.
	fmt.Println("╭─────────────────────────────────────────────────────╮")
	fmt.Println("│  Opening Spotify login in your browser...           │")
	fmt.Println("│                                                     │")
	fmt.Println("│  If it doesn't open automatically, visit:          │")
	fmt.Printf("│  %s\n", authURL)
	fmt.Println("│                                                     │")
	fmt.Println("│  Waiting for authorization...                       │")
	fmt.Println("╰─────────────────────────────────────────────────────╯")

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

		// Exchange code for tokens using the production endpoint.
		_, err := api.ExchangeCode(
			context.Background(),
			"", // production endpoint
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

// PrintMissingClientIDInstructions writes setup instructions when client_id is missing.
// Exported for testing.
func PrintMissingClientIDInstructions(w io.Writer) error {
	lines := []string{
		"╭─────────────────────────────────────────────────────╮",
		"│  Spotnik setup required                             │",
		"│                                                     │",
		"│  1. Create a Spotify app:                          │",
		"│     https://developer.spotify.com/dashboard        │",
		"│                                                     │",
		"│  2. Add your client_id to:                         │",
		"│     ~/.config/spotnik/config.toml                  │",
		"│                                                     │",
		"│  Example config.toml:                              │",
		"│    [spotify]                                        │",
		"│    client_id = \"your-client-id-here\"               │",
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

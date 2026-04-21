// Package cmd provides the CLI entry point for Spotnik via Cobra.
// It wires configuration, auth flow, theme loading, and application startup.
package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/keychain"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
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

// appVersion holds the value injected by main.go via cmd.Execute(version).
// It is package-level so runApp can forward it into AppOptions without
// threading through cobra commands.
var appVersion string

// Execute is the entry point called from main.go.
// version is injected at build time via LDFLAGS and forwarded into the root
// command's Version field and the TUI app options.
func Execute(version string) {
	appVersion = version
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Register the auth subcommand and its children.
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authRegisterCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authForgetCmd)
	authCmd.AddCommand(authStatusCmd)
}

// authCmd is the `spotnik auth` subcommand group.
// It has no RunE — running `spotnik auth` alone prints usage listing all subcommands.
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Spotify authentication",
	Long: "Manage Spotify authentication. Use a subcommand:\n\n" +
		"  register  — Set up your Spotify app credentials and authenticate\n" +
		"  login     — Re-authenticate (requires client_id already in config)\n" +
		"  logout    — Remove stored tokens only\n" +
		"  forget    — Remove tokens and client_id from config\n" +
		"  status    — Show current authentication state",
}

// authRegisterCmd is the `spotnik auth register` subcommand.
var authRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Set up your Spotify app credentials and authenticate",
	Long: "Show setup instructions, prompt for your Spotify Developer app client ID, " +
		"save it to config, and run the OAuth authorization flow.",
	RunE: func(c *cobra.Command, args []string) error {
		return runRegister(c, os.Stdin)
	},
}

// authLoginCmd is the `spotnik auth login` subcommand.
var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Re-authenticate with Spotify (clears existing tokens)",
	Long:  "Force a fresh Spotify authentication, overwriting any existing stored tokens. Requires client_id to be set in config.",
	RunE:  runAuthLogin,
}

// authLogoutCmd is the `spotnik auth logout` subcommand.
var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored Spotify tokens (keeps client_id)",
	RunE: func(c *cobra.Command, args []string) error {
		store := keychain.NewKeychainTokenStore()
		if err := LogoutTokens(store); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(c.OutOrStdout(), "Logged out. Stored tokens removed.")
		return nil
	},
}

// authForgetCmd is the `spotnik auth forget` subcommand.
var authForgetCmd = &cobra.Command{
	Use:   "forget",
	Short: "Remove stored tokens and client_id from config",
	RunE: func(c *cobra.Command, args []string) error {
		store := keychain.NewKeychainTokenStore()
		if err := RunForget(store, config.DefaultConfigPath()); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(c.OutOrStdout(), "Forgotten. Tokens and client_id removed from config.")
		return nil
	},
}

// authStatusCmd is the `spotnik auth status` subcommand.
var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE: func(c *cobra.Command, args []string) error {
		store := keychain.NewKeychainTokenStore()
		return PrintAuthStatus(store, config.DefaultConfigPath(), c.OutOrStdout())
	},
}

// LogoutTokens removes all stored token keys from the token store.
// Exported for testing.
func LogoutTokens(store keychain.TokenStore) error {
	if err := store.Delete(); err != nil {
		return fmt.Errorf("logging out: %w", err)
	}
	return nil
}

// RunForget clears tokens AND removes client_id from config at path.
// Exported for testing.
func RunForget(store keychain.TokenStore, configPath string) error {
	if err := store.Delete(); err != nil {
		return fmt.Errorf("removing tokens: %w", err)
	}
	if err := config.ClearClientID(configPath); err != nil {
		return fmt.Errorf("removing client_id: %w", err)
	}
	return nil
}

// PrintAuthStatus writes current auth + registration state to w using lipgloss
// styling. Labels are muted, values are bold primary, status uses success/warning
// colours. Always uses the black theme so colours are safe in any terminal.
// Exported for testing.
func PrintAuthStatus(store keychain.TokenStore, configPath string, w io.Writer) error {
	th := theme.Load(theme.DefaultThemeID)

	labelStyle := lipgloss.NewStyle().Foreground(th.TextMuted())
	valueStyle := lipgloss.NewStyle().Foreground(th.TextPrimary()).Bold(true)
	okStyle := lipgloss.NewStyle().Foreground(th.Success())
	warnStyle := lipgloss.NewStyle().Foreground(th.Warning())
	mutedStyle := lipgloss.NewStyle().Foreground(th.TextMuted())

	// Load config to check whether a client_id is present.
	cfg, err := loadConfigFromPath(configPath)
	if err != nil {
		// Non-fatal: continue and just report no client_id.
		cfg = config.Default()
	}

	if cfg.ClientID != "" {
		_, _ = fmt.Fprintf(w, "%s  %s\n",
			labelStyle.Render("Client ID:"),
			valueStyle.Render("present"),
		)
	} else {
		_, _ = fmt.Fprintf(w, "%s  %s\n",
			labelStyle.Render("Client ID:"),
			warnStyle.Render("not set  (run: spotnik auth register)"),
		)
	}

	access, err := store.Get(keychain.KeyAccessToken)
	if err != nil || access == "" {
		_, _ = fmt.Fprintf(w, "%s  %s\n",
			labelStyle.Render("Status:  "),
			mutedStyle.Render("not authenticated"),
		)
		return nil
	}

	_, _ = fmt.Fprintf(w, "%s  %s\n",
		labelStyle.Render("Status:  "),
		okStyle.Render("authenticated"),
	)

	expiry, err := store.GetExpiry()
	if err == nil {
		_, _ = fmt.Fprintf(w, "%s  %s\n",
			labelStyle.Render("Expires: "),
			mutedStyle.Render(expiry.Format(time.RFC1123)),
		)
	}

	expiringSoon, _ := store.IsExpiringSoon()
	if expiringSoon {
		_, _ = fmt.Fprintf(w, "%s\n",
			warnStyle.Render("⚠  Token expiring soon — will refresh automatically"),
		)
	}

	return nil
}

// CheckAuthState returns (needsRegister, needsAuth).
//   - needsRegister: no client_id in config.
//   - needsAuth: client_id present but no valid token.
//
// Exported for testing.
func CheckAuthState(cfg *config.Config, store keychain.TokenStore) (needsRegister, needsAuth bool) {
	if cfg.ClientID == "" {
		return true, false
	}

	access, err := store.Get(keychain.KeyAccessToken)
	if err != nil || access == "" {
		return false, true
	}

	expiringSoon, err := store.IsExpiringSoon()
	if err != nil {
		return false, true
	}

	if expiringSoon {
		refreshToken, err := store.Get(keychain.KeyRefreshToken)
		if err != nil || refreshToken == "" {
			return false, true
		}
		if err := api.Refresh(context.Background(), http.DefaultClient, "", refreshToken, cfg.ClientID, store); err != nil {
			_ = store.Delete()
			return false, true
		}
	}

	return false, false
}

// LoadConfig reads the config file at path, bootstraps it if missing, and
// returns the parsed Config. An empty ClientID is not an error — the caller
// uses CheckAuthState to determine what flow is needed. Exported for testing.
func LoadConfig(path string) (*config.Config, error) {
	return loadConfigFromPath(path)
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

// loadConfigFromPath is the testable implementation of LoadConfig.
func loadConfigFromPath(path string) (*config.Config, error) {
	// Bootstrap config file on first launch.
	if err := config.Bootstrap(path); err != nil {
		return nil, fmt.Errorf("bootstrapping config: %w", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
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
		if err := api.Refresh(context.Background(), http.DefaultClient, tokenBaseURL, refreshToken, cfg.ClientID, store); err != nil {
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
// It generates PKCE credentials, starts the local callback server on cfg.CallbackPort,
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

	// Start local callback server on the configured port.
	callbackSrv, codeCh, err := api.StartCallbackServer(cfg.CallbackPort)
	if err != nil {
		return fmt.Errorf("starting callback server on port %d: %w", cfg.CallbackPort, err)
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
			http.DefaultClient,
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

// runRegister shows setup instructions, prompts for a client ID via r (stdin in prod),
// saves it to config, and runs the OAuth flow.
func runRegister(c *cobra.Command, r io.Reader) error {
	w := c.OutOrStdout()

	_, _ = fmt.Fprintln(w, "╭─────────────────────────────────────────────────────╮")
	_, _ = fmt.Fprintln(w, "│  Spotnik — first-time setup                         │")
	_, _ = fmt.Fprintln(w, "│                                                     │")
	_, _ = fmt.Fprintln(w, "│  1. Go to https://developer.spotify.com/dashboard   │")
	_, _ = fmt.Fprintln(w, "│  2. Create (or pick) a Spotify app.                 │")
	_, _ = fmt.Fprintln(w, "│  3. In Redirect URIs, add:                          │")
	_, _ = fmt.Fprintln(w, "│     http://127.0.0.1:8888/callback                  │")
	_, _ = fmt.Fprintln(w, "│     (change the port if you set callback_port)      │")
	_, _ = fmt.Fprintln(w, "╰─────────────────────────────────────────────────────╯")
	_, _ = fmt.Fprintln(w, "")

	_, _ = fmt.Fprint(w, "Enter your Spotify client_id: ")

	scanner := bufio.NewScanner(r)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading client_id: %w", err)
	}
	clientID := strings.TrimSpace(scanner.Text())
	if clientID == "" {
		return fmt.Errorf("client_id cannot be empty")
	}

	configPath := config.DefaultConfigPath()
	if err := config.Bootstrap(configPath); err != nil {
		return fmt.Errorf("bootstrapping config: %w", err)
	}
	if err := config.SetClientID(configPath, clientID); err != nil {
		return fmt.Errorf("saving client_id to config: %w", err)
	}
	_, _ = fmt.Fprintln(w, "client_id saved to ~/.config/spotnik/config.toml")

	cfg, err := loadConfigFromPath(configPath)
	if err != nil {
		return err
	}

	store := keychain.NewKeychainTokenStore()
	return RunAuthFlow(cfg, store, "")
}

// runAuthLogin forces a fresh re-authentication flow.
// Errors if no client_id is set in config.
func runAuthLogin(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if cfg.ClientID == "" {
		return fmt.Errorf("no client_id in config — run: spotnik auth register")
	}

	store := keychain.NewKeychainTokenStore()
	// Delete existing tokens to force a fresh login.
	_ = store.Delete()

	return RunAuthFlow(cfg, store, "")
}

// runApp is the main command handler. It loads config, checks auth state,
// and launches the Bubble Tea application.
func runApp(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	store := keychain.NewKeychainTokenStore()
	needsRegister, needsAuth := CheckAuthState(cfg, store)

	opts := app.AppOptions{
		NeedsRegister: needsRegister,
		NeedsAuth:     needsAuth,
		ClientID:      cfg.ClientID,
		TokenStore:    store,
		Version:       appVersion,
	}

	// When auth is needed, start the callback server early so the redirect URI
	// is known before the TUI is displayed.
	if needsRegister || needsAuth {
		srv, codeCh, err := api.StartCallbackServer(cfg.CallbackPort)
		if err != nil {
			return fmt.Errorf("port %d is busy — set a different callback_port in "+
				"~/.config/spotnik/config.toml: %w", cfg.CallbackPort, err)
		}
		defer srv.Close() // safety net: fires even if app exits before auth completes
		opts.CallbackPort = cfg.CallbackPort
		opts.CallbackCodeCh = codeCh
		opts.CallbackClose = srv.Close
	}

	a := app.New(cfg, opts)

	if !needsRegister && !needsAuth {
		accessToken, _ := store.Get(keychain.KeyAccessToken)
		a.InitAPIClients(accessToken)
	}

	// Start the Bubble Tea program.
	// tea.WithMouseCellMotion() enables mouse wheel scroll events (Feature 52).
	p := tea.NewProgram(a, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()
	return err
}

// loadConfig reads the config file from the default path and bootstraps it if missing.
func loadConfig() (*config.Config, error) {
	return loadConfigFromPath(config.DefaultConfigPath())
}

// PrintMissingClientIDInstructions writes setup instructions when client_id is missing.
// It directs the user to run `spotnik auth register`.
// Exported for testing.
func PrintMissingClientIDInstructions(w io.Writer) error {
	lines := []string{
		"╭─────────────────────────────────────────────────────╮",
		"│  Spotnik setup required                             │",
		"│                                                     │",
		"│  1. Create a Spotify app:                           │",
		"│     https://developer.spotify.com/dashboard         │",
		"│                                                     │",
		"│  2. Run: spotnik auth register                      │",
		"│     Follow the prompts to set client_id and auth.   │",
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
	return fmt.Errorf("missing client_id — run: spotnik auth register")
}

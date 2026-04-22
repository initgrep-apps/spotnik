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

// CLI output colours — fixed, not theme-dependent.
// AdaptiveColor keeps output readable on light terminals.
var (
	cliGreen  = lipgloss.AdaptiveColor{Dark: "#1DB954", Light: "#1A8C41"}
	cliRed    = lipgloss.AdaptiveColor{Dark: "#FF5555", Light: "#CC0000"}
	cliYellow = lipgloss.AdaptiveColor{Dark: "#F1C40F", Light: "#B8860B"}
	cliDim    = lipgloss.AdaptiveColor{Dark: "#6C7083", Light: "#888888"}
)

var (
	cliAccentS = lipgloss.NewStyle().Foreground(cliGreen).Bold(true)
	cliDimS    = lipgloss.NewStyle().Foreground(cliDim)
	cliErrS    = lipgloss.NewStyle().Foreground(cliRed).Bold(true)
	cliWarnS   = lipgloss.NewStyle().Foreground(cliYellow)
	// cliWrap applies left+right indentation to all CLI output. No top/bottom
	// padding — cliOut inserts a single leading blank line explicitly so that
	// adjacent cliLine calls remain compact while cliOut sections are separated.
	cliWrap = lipgloss.NewStyle().Padding(0, 2)
)

// errAlreadyPrinted is returned when a RunE handler has already printed a
// styled error block to stderr. Execute() recognizes it and exits 1 without
// printing again.
var errAlreadyPrinted = errors.New("")

// cliOut writes a blank line then the joined lines with standard left+right
// indentation. The leading blank separates output sections without adding a
// trailing blank that would double-space adjacent cliLine progress output.
func cliOut(w io.Writer, lines ...string) {
	block := lipgloss.JoinVertical(lipgloss.Left, lines...)
	_, _ = fmt.Fprintln(w, "\n"+cliWrap.Render(block))
}

// cliLine writes a single inline progress line with the standard indentation
// and no leading/trailing blank lines. Use for sequential step output where
// top/bottom spacing from cliOut would break the compact sequence.
func cliLine(w io.Writer, text string) {
	_, _ = fmt.Fprintln(w, cliWrap.Render(text))
}

// cliSpin starts an animated braille spinner on the current output line and
// returns a stop function. The stop function blocks until the goroutine exits.
// Use this for the gap between CLI auth completion and TUI startup.
func cliSpin(w io.Writer, label string) (stop func()) {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		tick := time.NewTicker(80 * time.Millisecond)
		defer tick.Stop()
		i := 0
		for {
			select {
			case <-done:
				return
			case <-tick.C:
				_, _ = fmt.Fprintf(w, "\r  %s %s",
					cliAccentS.Render(frames[i%len(frames)]),
					label)
				i++
			}
		}
	}()
	return func() {
		close(done)
		<-stopped
	}
}

// cliKV renders aligned key-value pairs. Labels are dim; values are default foreground.
func cliKV(pairs [][2]string) string {
	maxKey := 0
	for _, p := range pairs {
		if len(p[0]) > maxKey {
			maxKey = len(p[0])
		}
	}
	lines := make([]string, len(pairs))
	for i, p := range pairs {
		pad := strings.Repeat(" ", maxKey-len(p[0]))
		lines[i] = cliDimS.Render(p[0]+pad) + "  " + p[1]
	}
	return strings.Join(lines, "\n")
}

var rootCmd = &cobra.Command{
	Use:   "spotnik",
	Short: "A terminal Spotify client",
	Long:  "Spotnik — keyboard-driven Spotify client.",
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
		// cobra is silenced (SilenceErrors=true); we print once, styled.
		// errAlreadyPrinted means the handler already wrote a styled block to stderr.
		if !errors.Is(err, errAlreadyPrinted) {
			_, _ = fmt.Fprintln(os.Stderr, "\n"+cliWrap.Render(cliErrS.Render("✗")+" "+err.Error()))
		}
		os.Exit(1)
	}
}

func init() {
	// Silence cobra's built-in error/usage printing so Execute() owns the single
	// styled print. SilenceUsage prevents usage output on runtime errors (wrong
	// state, not wrong flags).
	rootCmd.SilenceErrors = true

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
	Use:          "register",
	Short:        "Set up your Spotify app credentials and authenticate",
	SilenceUsage: true,
	Long: "Show setup instructions, prompt for your Spotify Developer app client ID, " +
		"save it to config, and run the OAuth authorization flow.",
	RunE: func(c *cobra.Command, args []string) error {
		return runRegister(c, os.Stdin)
	},
}

// authLoginCmd is the `spotnik auth login` subcommand.
var authLoginCmd = &cobra.Command{
	Use:          "login",
	Short:        "Re-authenticate with Spotify (clears existing tokens)",
	SilenceUsage: true,
	Long:         "Force a fresh Spotify authentication, overwriting any existing stored tokens. Requires client_id to be set in config.",
	RunE:         runAuthLogin,
}

// authLogoutCmd is the `spotnik auth logout` subcommand.
var authLogoutCmd = &cobra.Command{
	Use:          "logout",
	Short:        "Remove stored Spotify tokens (keeps client_id)",
	SilenceUsage: true,
	RunE: func(c *cobra.Command, args []string) error {
		store := keychain.NewKeychainTokenStore()
		if err := LogoutTokens(store); err != nil {
			return err
		}
		PrintLogoutSuccess(c.OutOrStdout())
		return nil
	},
}

// authForgetCmd is the `spotnik auth forget` subcommand.
var authForgetCmd = &cobra.Command{
	Use:          "forget",
	Short:        "Remove stored tokens and client_id from config",
	SilenceUsage: true,
	RunE: func(c *cobra.Command, args []string) error {
		store := keychain.NewKeychainTokenStore()
		if err := RunForget(store, config.DefaultConfigPath()); err != nil {
			return err
		}
		PrintForgetSuccess(c.OutOrStdout())
		return nil
	},
}

// authStatusCmd is the `spotnik auth status` subcommand.
var authStatusCmd = &cobra.Command{
	Use:          "status",
	Short:        "Show current authentication status",
	SilenceUsage: true,
	RunE: func(c *cobra.Command, args []string) error {
		store := keychain.NewKeychainTokenStore()
		return PrintAuthStatus(store, config.DefaultConfigPath(), c.OutOrStdout())
	},
}

// PrintLogoutSuccess writes the styled "Signed out" confirmation block to w.
// Exported for testing.
func PrintLogoutSuccess(w io.Writer) {
	cliOut(w, cliAccentS.Render("✓")+" Signed out")
}

// PrintForgetSuccess writes the styled "Session ended" confirmation block to w.
// Exported for testing.
func PrintForgetSuccess(w io.Writer) {
	cliOut(w,
		cliAccentS.Render("✓")+" Session ended",
		cliDimS.Render("Tokens and client ID removed"),
		cliAccentS.Render("→")+" Run "+cliAccentS.Render("spotnik auth register")+" to set up again",
	)
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

// PrintAuthStatus writes current auth + registration state to w using the shared CLI
// style vars. Always uses the standard CLI colour palette (not theme-dependent).
// Exported for testing.
func PrintAuthStatus(store keychain.TokenStore, configPath string, w io.Writer) error {
	cfg, err := loadConfigFromPath(configPath)
	if err != nil {
		cfg = config.Default()
	}

	switch cfg.ClientID {
	case "":
		// Not registered — no client_id in config.
		cliOut(w,
			cliDimS.Render("◎ Spotnik  ")+"not registered",
			cliAccentS.Render("→")+" Run "+cliAccentS.Render("spotnik auth register")+" to connect your Spotify account",
		)
		return nil

	default:
		access, _ := store.Get(keychain.KeyAccessToken)
		if access == "" {
			// Registered but not authenticated.
			cliOut(w,
				cliDimS.Render("◎ Spotnik  ")+"not authenticated",
				cliKV([][2]string{{"Client ID", "present"}}),
				cliAccentS.Render("→")+" Run "+cliAccentS.Render("spotnik auth login")+" to connect",
			)
			return nil
		}

		expiringSoon, expiryErr := store.IsExpiringSoon()
		if expiryErr != nil {
			// Cannot read token state — show warning, don't claim healthy.
			cliOut(w,
				cliWarnS.Render("⚠")+" Spotnik  session state unknown",
				cliDimS.Render("Could not read token state from keychain"),
				cliAccentS.Render("→")+" Run "+cliAccentS.Render("spotnik auth login")+" to re-authenticate",
			)
			return nil
		}
		var expiry time.Time
		expiry, expiryErr = store.GetExpiry()

		var expiryVal string
		if expiryErr == nil {
			expiryVal = expiry.Format("Mon, 02 Jan 2006 15:04 UTC")
		}
		if expiringSoon {
			expiryVal += "  ·  auto-refresh pending"
		}

		kvPairs := [][2]string{{"Client ID", "present"}}
		if expiryVal != "" {
			kvPairs = append(kvPairs, [2]string{"Expires", expiryVal})
		}

		if expiringSoon {
			cliOut(w,
				cliWarnS.Render("⚠")+" Spotnik  session expiring",
				cliKV(kvPairs),
				cliAccentS.Render("→")+" Run "+cliAccentS.Render("spotnik auth login")+" to re-authenticate if auto-refresh fails",
			)
		} else {
			cliOut(w,
				cliAccentS.Render("◉")+" Spotnik  authenticated",
				cliKV(kvPairs),
			)
		}
		return nil
	}
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
		// No token — start auth flow. Output goes to io.Discard because
		// EnsureAuthenticated is called from the TUI path, not the CLI path.
		return RunAuthFlow(cfg, store, tokenBaseURL, io.Discard)
	}

	expiringSoon, err := store.IsExpiringSoon()
	if err != nil {
		// Cannot determine expiry — start fresh auth.
		return RunAuthFlow(cfg, store, tokenBaseURL, io.Discard)
	}

	if expiringSoon {
		// Proactively refresh the token before it expires.
		refreshToken, err := store.Get(keychain.KeyRefreshToken)
		if err != nil || refreshToken == "" {
			return RunAuthFlow(cfg, store, tokenBaseURL, io.Discard)
		}

		// Refresh using the configured token endpoint.
		if err := api.Refresh(context.Background(), http.DefaultClient, tokenBaseURL, refreshToken, cfg.ClientID, store); err != nil {
			if errors.Is(err, api.ErrInvalidGrant) {
				// Refresh token rejected — delete tokens and force re-auth.
				_, _ = fmt.Fprintln(os.Stderr, "\n"+cliWrap.Render(cliWarnS.Render("⚠")+" Session expired — please re-authenticate"))
				_ = store.Delete()
				return RunAuthFlow(cfg, store, tokenBaseURL, io.Discard)
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
// w receives progress output (URL block, step confirmations). Pass io.Discard for silent operation.
// Exported for testing.
func RunAuthFlow(cfg *config.Config, store keychain.TokenStore, tokenBaseURL string, w io.Writer) error {
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
	// cliOut provides the section-separating blank line via its leading \n;
	// the URL and waiting line use cliLine so they stay compact with the
	// step confirmations printed below.
	cliOut(w, cliDimS.Render("Visit this URL to authorize:"))
	cliLine(w, cliAccentS.Render(authURL))
	cliLine(w, cliDimS.Render("Waiting for callback…"))

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

		cliLine(w, cliAccentS.Render("✓")+" Browser authentication complete")

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

		cliLine(w, cliAccentS.Render("✓")+" Token exchange successful")

		return nil

	case <-ctx.Done():
		return fmt.Errorf("authorization timed out after 5 minutes — please try again")
	}
}

// runRegister shows setup instructions, prompts for a client ID via r (stdin in prod),
// saves it to config, and runs the OAuth flow.
func runRegister(c *cobra.Command, r io.Reader) error {
	w := c.OutOrStdout()

	// Load config first so we know the actual callback port to display.
	configPath := config.DefaultConfigPath()
	if err := config.Bootstrap(configPath); err != nil {
		return fmt.Errorf("bootstrapping config: %w", err)
	}
	cfg, err := loadConfigFromPath(configPath)
	if err != nil {
		cfg = config.Default()
	}

	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", cfg.CallbackPort)

	cliOut(w,
		cliDimS.Render("◎ Spotnik  ")+"not registered",
		cliKV([][2]string{
			{"1", "Go to developer.spotify.com/dashboard"},
			{"2", "Create or select a Spotify app"},
			{"3", "Add this redirect URI: " + cliAccentS.Render(redirectURI)},
		}),
	)
	_, _ = fmt.Fprint(w, "  Client ID: ")

	scanner := bufio.NewScanner(r)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading client_id: %w", err)
	}
	clientID := strings.TrimSpace(scanner.Text())
	if clientID == "" {
		return fmt.Errorf("client_id cannot be empty")
	}

	if err := config.SetClientID(configPath, clientID); err != nil {
		return fmt.Errorf("saving client_id to config: %w", err)
	}
	cliLine(w, cliAccentS.Render("✓")+" Client ID saved")

	cfg, err = loadConfigFromPath(configPath)
	if err != nil {
		return err
	}

	store := keychain.NewKeychainTokenStore()
	if err := RunAuthFlow(cfg, store, "", w); err != nil {
		cliOut(c.ErrOrStderr(),
			cliErrS.Render("✗")+" Authorization failed",
			cliKV([][2]string{
				{"Reason", err.Error()},
			}),
			cliAccentS.Render("→")+" Run "+cliAccentS.Render("spotnik auth register")+" to try again",
		)
		return errAlreadyPrinted
	}

	// Authorization succeeded — confirm sign-in then spin while the TUI loads.
	cliOut(w, cliAccentS.Render("◉")+" Signed in")
	stop := cliSpin(w, cliDimS.Render("Launching spotnik…"))
	err = runApp(c, []string{})
	stop()
	return err
}

// runAuthLogin forces a fresh re-authentication flow.
// Errors if no client_id is set in config.
func runAuthLogin(c *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if cfg.ClientID == "" {
		cliOut(c.ErrOrStderr(),
			cliErrS.Render("✗")+" Authentication failed",
			cliKV([][2]string{{"Reason", "no client_id configured"}}),
			cliAccentS.Render("→")+" Run "+cliAccentS.Render("spotnik auth register")+" to set up your Spotify app",
		)
		return errAlreadyPrinted
	}

	store := keychain.NewKeychainTokenStore()
	// Delete existing tokens to force a fresh login.
	if err := store.Delete(); err != nil {
		return fmt.Errorf("clearing existing tokens: %w", err)
	}

	if err := RunAuthFlow(cfg, store, "", c.OutOrStdout()); err != nil {
		cliOut(c.ErrOrStderr(),
			cliErrS.Render("✗")+" Authentication failed",
			cliKV([][2]string{{"Reason", err.Error()}}),
			cliAccentS.Render("→")+" Run "+cliAccentS.Render("spotnik auth login")+" to try again",
		)
		return errAlreadyPrinted
	}

	cliOut(c.OutOrStdout(), cliAccentS.Render("◉")+" Signed in")
	stop := cliSpin(c.OutOrStdout(), cliDimS.Render("Launching spotnik…"))
	err = runApp(c, []string{})
	stop()
	return err
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
	cliOut(w,
		cliDimS.Render("◎ Spotnik  ")+"not registered",
		"",
		cliKV([][2]string{
			{"1", "Create a Spotify app at developer.spotify.com/dashboard"},
			{"2", "Run: spotnik auth register"},
		}),
		"",
		cliAccentS.Render("→")+" Follow the prompts to set client_id and authenticate",
	)
	return nil
}

// HandleMissingClientID prints setup instructions and returns an error.
// Exported for testing. The error signals the CLI to exit with code 1.
func HandleMissingClientID() error {
	_ = PrintMissingClientIDInstructions(os.Stdout)
	return fmt.Errorf("missing client_id — run: spotnik auth register")
}

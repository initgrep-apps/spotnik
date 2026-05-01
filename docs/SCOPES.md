# Spotify OAuth Scopes Used by Spotnik

Spotnik authenticates with Spotify using the [PKCE OAuth flow][pkce] — no client
secret is ever stored, generated, or transmitted. At first launch you are
redirected to Spotify to grant the 14 scopes listed below; after consent the
access and refresh tokens are persisted in your operating system's keychain
(macOS Keychain, Linux Secret Service, or Windows Credential Manager).

This document lists every scope spotnik requests, what each one grants, what
spotnik does *not* do with that access, and how to revoke it at any time.

[pkce]: https://datatracker.ietf.org/doc/html/rfc7636

## Read-only access (9 scopes)

These scopes let spotnik display state. They do not allow any modification.

- **Playback state** — `user-read-playback-state`, `user-read-currently-playing`
  Read the currently active device, the track playing on it, the playback
  position, shuffle/repeat state, and volume. Powers the Now Playing pane,
  Queue pane, and the developer-facing Gateway and Polling Traffic panes.

- **Library** — `user-library-read`
  List your saved (liked) tracks and saved albums. Powers the Liked Songs and
  Albums panes.

- **Playlists** — `playlist-read-private`, `playlist-read-collaborative`
  List your private and collaborative playlists and their tracks. Powers the
  Playlists pane.

- **Profile** — `user-read-private`, `user-read-email`
  Read your display name, account country, product tier (Premium/Free), and
  email address. Used to confirm Premium and to render the profile overlay.
  The email address is never sent off-device, never logged, and never
  transmitted to any host other than Spotify.

- **Listening history** — `user-read-recently-played`, `user-top-read`
  Read your recently played tracks and your top tracks/artists across short,
  medium, and long term ranges. Powers the Recently Played, Top Tracks, and
  Top Artists panes.

- **Following** — `user-follow-read`
  Read the list of artists you follow. Used by the Top Artists pane to mark
  followed artists.

## Write actions (5 scopes)

These scopes allow spotnik to modify state on your account when you press a
control. Every write action is initiated by a user keypress; spotnik never
writes in the background.

- **Playback control** — `user-modify-playback-state`
  Play, pause, skip, seek, set volume, toggle shuffle/repeat, and transfer
  playback between Spotify Connect devices. Powers all transport controls and
  the Device switcher.

- **Library modification** — `user-library-modify`
  Like or unlike tracks, save or remove albums. Triggered only when you press
  the like/unlike key in the relevant pane.

- **Playlist editing** — `playlist-modify-public`, `playlist-modify-private`
  Add tracks to a playlist, remove tracks from a playlist, and reorder tracks
  within a playlist. Only playlists you own can be modified — Spotify rejects
  edits to playlists owned by other users regardless of scope.

## What spotnik does *not* do

- **No telemetry** — there is no usage reporting, crash reporting, or analytics
  of any kind. Spotnik makes no network calls beyond the ones listed below.
- **No third-party endpoints** — the only outbound hosts spotnik contacts are:
  - `https://accounts.spotify.com` — OAuth authorisation and token exchange.
  - `https://api.spotify.com` — Spotify Web API for all reads and writes above.
  - `http://127.0.0.1:8888` — local-only OAuth callback listener, bound to
    loopback, active for ~120 seconds during the initial authorisation
    handshake and shut down immediately after.
- **No background activity** — spotnik only makes API calls while the TUI is
  running and you are interacting with it. There is no daemon, no scheduled
  task, and no startup hook.
- **No data leaves your machine** — track metadata, playlists, and listening
  history are read from Spotify, rendered, and discarded when the TUI exits.
  Nothing is persisted to disk other than the OAuth tokens (in your OS
  keychain) and your config file at `~/.config/spotnik/config.toml`.

## How to revoke

To revoke spotnik's access at any time:

1. Visit [https://www.spotify.com/account/apps](https://www.spotify.com/account/apps).
2. Find **Spotnik** in the list and click **Remove Access**.

Spotify immediately invalidates the refresh token. The next time you launch
spotnik it will re-prompt for authorisation. To also remove the locally cached
tokens before re-authenticating, run `spotnik auth forget`.

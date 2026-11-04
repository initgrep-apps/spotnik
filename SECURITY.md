# Security policy

Spotnik is a local terminal client. It talks to the Spotify Web API, stores OAuth
tokens in the OS keychain, and runs its OAuth callback on `127.0.0.1:8888`. The
main security surface is authentication, local token handling, and the install
scripts.

## Supported versions

| Version | Supported |
|---|---|
| 0.1.x | Yes |
| Older versions | No — upgrade to 0.1.x |

Spotnik is pre-1.0. Security fixes ship in the latest release; older tags are not
patched separately.

## Reporting a vulnerability

**Do not open a public issue.**

Use GitHub private vulnerability reporting:

<https://github.com/initgrep-apps/spotnik/security/advisories/new>

Include:

- the affected version (`spotnik --version`) and platform,
- steps to reproduce or a proof of concept,
- the impact you believe it has,
- any suggested fix or mitigation.

For issues in the Spotify Web API or Spotify account itself, report them to
Spotify. Report here if Spotnik's handling makes the issue reachable.

## What to expect

- **Acknowledgement** within 3 business days.
- **Initial assessment** within 10 business days.
- **Fix** targeted within 30 days for high-severity issues and 90 days for
  lower-severity issues, depending on complexity.
- **Coordinated disclosure**: we agree on a public date with you, normally after
  a fixed release is available. We credit reporters unless asked not to.

This is a volunteer-maintained project. There is no bug bounty.

## Scope

In scope:

- the OAuth PKCE flow, callback handler, and token storage,
- API request handling, rate limiting, and error paths,
- the install and uninstall scripts,
- the release artifacts and build workflow.

Out of scope:

- vulnerabilities in Spotify's service or the user's Spotify account,
- a compromised local machine, shell, terminal, or OS keychain,
- social engineering and physical access attacks,
- the fact that Spotify Premium is required.

## Token handling

Tokens are stored in the OS keychain through `github.com/zalando/go-keyring`.
`spotnik auth forget` removes them. The Client ID lives in
`~/.config/spotnik/config.toml`; it is not a secret. Spotnik never stores a
Client Secret and never sends tokens anywhere except the Spotify API.

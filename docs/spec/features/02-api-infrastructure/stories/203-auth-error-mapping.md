---
title: "Auth Error Mapping"
feature: 02-api-infrastructure
status: done
---

## Background

The onboarding `stepError` view renders `m.err.Error()` directly. Users see raw Go error strings
like `"invalid_grant: The authorization code expired"` or
`"invalid_client: bad client secret"`. These strings are Spotify OAuth error codes — internal
identifiers that are meaningless to end users.

A lookup table maps known OAuth error codes to user-friendly messages. Unknown codes fall back to
a generic message that tells the user how to recover. The lookup is case-insensitive substring
matching on the raw error string (Spotify may include additional context after the code).

## Design

### `internal/app/auth.go` — `authErrorMessages` + `friendlyAuthError`

Add `"strings"` to imports.

After the existing type declarations in `auth.go`, add:

```go
// authErrorMessages maps Spotify OAuth error codes to user-friendly strings.
var authErrorMessages = map[string]string{
    "invalid_grant":          "Authorization expired. Please sign in again.",
    "invalid_client":         "Client ID is invalid. Check your config file.",
    "invalid_request":        "Sign-in failed. Please try again.",
    "access_denied":          "Authorization denied. Please allow access when prompted.",
    "unsupported_grant_type": "Sign-in configuration error. Please file a bug report.",
}

// friendlyAuthError maps a raw OAuth error to a user-friendly string.
// Returns a generic fallback for unknown codes or nil errors.
func friendlyAuthError(err error) string {
    if err == nil {
        return "Sign-in failed. Please run 'spotnik auth' to try again."
    }
    msg := err.Error()
    for code, friendly := range authErrorMessages {
        if strings.Contains(msg, code) {
            return friendly
        }
    }
    return "Sign-in failed. Please run 'spotnik auth' to try again."
}

// FriendlyAuthError is an exported wrapper around friendlyAuthError for testing.
func FriendlyAuthError(err error) string { return friendlyAuthError(err) }
```

### `internal/app/handlers.go` — wire into onboarding error handler

In the `authErrorMsg` handler, change:

```go
// OLD:
a.onboardingError = m.err.Error()

// NEW:
a.onboardingError = friendlyAuthError(m.err)
```

## Acceptance Criteria

- [ ] `friendlyAuthError(errors.New("invalid_grant: ..."))` returns `"Authorization expired. Please sign in again."`
- [ ] `friendlyAuthError(errors.New("invalid_client: ..."))` returns `"Client ID is invalid. Check your config file."`
- [ ] `friendlyAuthError(errors.New("access_denied"))` returns `"Authorization denied. Please allow access when prompted."`
- [ ] `friendlyAuthError(errors.New("something_unexpected"))` returns the generic fallback
- [ ] `friendlyAuthError(nil)` returns the generic fallback
- [ ] Onboarding `stepError` view shows mapped message — no raw Go error string
- [ ] `make ci` passes

## Tasks

- [ ] Write failing tests in `internal/app/auth_test.go`:
      `TestFriendlyAuthError_InvalidGrant`, `_InvalidClient`, `_AccessDenied`,
      `_Unknown`, `_Nil`
      - test: `go test ./internal/app/ -run "TestFriendlyAuthError" -v` → FAIL (symbol not defined)

- [ ] Add `authErrorMessages`, `friendlyAuthError()`, and exported `FriendlyAuthError()`
      to `internal/app/auth.go`; add `"strings"` import
      - test: `go test ./internal/app/ -run "TestFriendlyAuthError" -v` → all PASS

- [ ] Update `authErrorMsg` handler in `internal/app/handlers.go`:
      `a.onboardingError = friendlyAuthError(m.err)`
      - test: `go build ./internal/app/...` compiles cleanly

- [ ] `make ci` passes

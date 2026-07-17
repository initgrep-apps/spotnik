---
title: "Fix reflected XSS in OAuth callback error response"
feature: 09-auth-and-profile
status: open
---

## Background

The local OAuth callback server in `internal/api/auth.go` echoes the user-provided
`error` query parameter directly into the HTTP response body when Spotify returns
an authorization error:

```go
_, _ = fmt.Fprintf(w, "Authorization failed: %s", errParam)
```

Because the value is not escaped, an attacker who can direct the user's browser
to `http://127.0.0.1:{port}/callback?error=<script>...</script>` can execute
arbitrary JavaScript in the browser context. The callback server is bound to
localhost, but this still matters for malicious links, browser extensions, or
other applications on the same machine that can craft a request to the loopback
address.

The same value is also reflected into `CallbackResult.Err`, but that string is
passed through a Go channel and never rendered as HTML, so it is not an XSS sink.

## Design

### Escape output in callback handler

In `internal/api/auth.go` import `"html"` and change the error response write to:

```go
_, _ = fmt.Fprintf(w, "Authorization failed: %s", html.EscapeString(errParam))
```

Keep the same HTTP status (`http.StatusBadRequest`) and response format.

### Add regression test

In `internal/api/auth_test.go` add `TestCallbackServer_ErrorResponse_EscapesHTML`.
Use the existing callback server test pattern to send a request with
`error=<script>alert(1)</script>`. Assert that:

- Response status is 400 Bad Request.
- Response body contains the escaped substring `&lt;script&gt;`.
- Response body does not contain the raw substring `<script>`.

## Files

### Modify

- `internal/api/auth.go` — HTML-escape `errParam` in `/callback` error response.
- `internal/api/auth_test.go` — add reflected-XSS regression test.

## Acceptance Criteria

- [ ] `error=<script>alert(1)</script>` in callback request returns 400 with escaped body.
- [ ] Response body does not contain raw `<script>` tag.
- [ ] Existing callback server tests still pass.
- [ ] `make ci` passes (lint, tests, coverage).

## Tasks

- [ ] Escape `errParam` with `html.EscapeString` in `internal/api/auth.go`
      - test: existing `TestCallbackServer_HandlesError` still passes
- [ ] Add XSS regression test `TestCallbackServer_ErrorResponse_EscapesHTML`
      - test: `TestCallbackServer_ErrorResponse_EscapesHTML`
- [ ] Run `make ci` and fix any failures

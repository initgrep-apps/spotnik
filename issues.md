# Spotnik — Non-Critical Issues Log

> Issues logged during PR reviews. Not merge-blocking but should be addressed.

---

## Feature 61: Fix Request Flow Gateway Visualization

*All issues resolved in PR #75.*

### ~~I61-1~~ RESOLVED (PR #75)
### ~~I61-2~~ RESOLVED (PR #75)
### ~~I61-3~~ DISCARDED — acceptable for 1s refresh diagnostic pane
### ~~I61-4~~ RESOLVED (PR #75) — documented caller requirement with IMPORTANT comment
### ~~I61-5~~ RESOLVED (PR #75)
### ~~I61-6~~ RESOLVED (PR #75)
### ~~I61-7~~ RESOLVED (PR #75)
### ~~I61-8~~ RESOLVED (PR #75)

---

## Feature 62: Request Flow Boxed Layout

*All issues resolved in PR #77 (Feature 63).*

### ~~I62-1~~ RESOLVED (PR #77) — post-clamp overflow guard added
### ~~I62-2~~ RESOLVED (PR #77) — doc comment added documenting caller precondition
### ~~I62-3~~ RESOLVED (PR #77) — maxRows guard added to both arrow builders
### ~~I62-4~~ RESOLVED (PR #77) — height fallback to viewFlat() when boxAreaHeight < 3

---

## Feature 64: Gateway Liveness & Watermarks

### ~~I64-1~~ SUPERSEDED by Feature 65 — UI-side watermark fields removed entirely
### ~~I64-2~~ SUPERSEDED by Feature 68 — watermark annotations removed; boxed layout now renders from replay state

---

## Feature 65: Gateway-Internal Watermarks

### ~~I65-1~~ SUPERSEDED by Feature 68 — ResetWatermarks() removed entirely
### ~~I65-2~~ SUPERSEDED by Feature 68 — old Snapshot() removed, replaced by captureSnapshot()

---

## Feature 67: Gateway Event Instrumentation

### ~~I67-1~~ RESOLVED by Feature 69 — ARCHITECTURE.md updated, stale GatewayRecorder refs removed
### ~~I67-2~~ RESOLVED by Feature 69 — auth.go comment updated to reference GatewayEventRecorder

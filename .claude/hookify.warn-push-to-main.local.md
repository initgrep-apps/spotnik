---
name: warn-push-to-main
enabled: true
event: bash
pattern: git\s+push\s+(origin\s+)?main
action: warn
---

⚠️ **Pushing directly to main detected**

CLAUDE.md rule: Never work directly on `main`. All work requires a feature branch.

Before proceeding, verify:
- You are on a feature branch (`git branch --show-current`)
- This is intentional (e.g. post-merge pull, not committing work)

If pushing feature work: stop, create `feat/NN-feature-name` branch first.

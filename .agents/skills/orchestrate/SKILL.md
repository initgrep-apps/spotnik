---
name: orchestrate
description: |
  Autonomous end-to-end implementation pipeline. Validates a spec, launches the
  feature-implementer agent, runs external PR review with fix cycles, finalizes
  docs, merges, and reports. Use when: "orchestrate feature 15",
  "orchestrate issue 19", "implement feature 15", "build issue 19", or any
  request to run the full implementation pipeline for a numbered spec.
---

# Orchestrator

You are the orchestrator — an autonomous pipeline that takes a spec from
implementation through to merge. You launch subagents, manage review cycles,
and own the merge decision.

The pipeline has **6 steps** and they must run in order. Do not skip steps.
Do not reorder steps. Each step has an explicit gate that must pass before
moving to the next.

---

## MANDATORY: FETCH DEFERRED TOOLS FIRST

Before doing anything else, fetch the SendMessage tool schema. You will need it
in Steps 3 and 4 to communicate with the feature-implementer agent. If you skip
this, the tool call will fail with "not available" later.

```
ToolSearch("select:SendMessage")
```

Do this as your very first action. Then read `AGENTS.md` and
`docs/spec/00-overview.md` in parallel.

---

## INPUT

The user provides a feature and optionally a story identifier. Examples:
- `feature 15` — implement the next open story in feature 15
- `feature 15 story 57` — implement a specific story
- `feature 15-cicd` — feature by name slug

Parse into:
- **feature**: number or number-name slug
- **story** (optional): specific story number

---

## STEP 1 — VALIDATE

Gate: do not proceed to Step 2 until validation passes.

1. Resolve the feature directory from AGENTS.md spec locations (typically
   `docs/spec/features/NN-name/`).
2. Zero-pad the number to two digits, glob for matching directories.
3. Read the feature's `feature.md` — verify `status: open` or `in-progress`.
4. List stories in the `stories/` subdirectory. Read each story's frontmatter
   to identify which are open.
5. If a specific story was requested, verify it exists and is open.
6. If the feature is not found, already done, or ambiguous: **STOP** and tell
   the user. List available open features.
7. **Dependency check**: if the feature spec declares dependencies on other
   features, verify those features have `status: done` and their PRs are
   merged. If not, **STOP** and report which dependency must be completed first.

---

## STEP 2 — LAUNCH FEATURE-IMPLEMENTER

Gate: do not proceed to Step 3 until a PR URL is in hand.

1. Launch the `feature-implementer` agent via the `Agent` tool.
2. Prompt:
   ```
   Implement story {story_number} from feature {NN-name}.
   Feature spec: {absolute_path_to_feature.md}
   Story spec: {absolute_path_to_story.md}
   ```
3. Await the agent's return. Expect:
   - A summary of what was built
   - A PR URL
4. **Record the agent's name** from the task result. This is the exact string
   you pass as the `to` field in all subsequent `SendMessage` calls. Write it
   down — you cannot recover it later.
5. Extract the PR number from the URL.

**If the agent escalates** (spec ambiguity, persistent CI failure, blocker):
surface the escalation message to the user and **STOP**.

**If no PR URL is returned:** report the agent's output to the user and **STOP**.

---

## STEP 3 — EXTERNAL REVIEW LOOP (up to 3 rounds)

**This step is mandatory. Do not skip it. Do not proceed to Step 4 without
running at least one review round.**

The feature-implementer self-reviews during implementation. This is an
independent second opinion on the final PR.

Initialise `minor_issues = []` before entering the loop.

For round = 1 to 3:

**3a. Run the review:**
Invoke the `pr-review-toolkit:review-pr` skill via the `Skill` tool, passing
the PR number.

**3b. Classify the findings:**
- **CRITICAL / IMPORTANT** — must fix before merge
- **MINOR / SUGGESTION** — append to `minor_issues`, do not block merge

**3c. If clean or minor-only → break the loop.** Review passed. Proceed to Step 4.

**3d. If critical/important issues found → send fixes to the existing agent:**

Use `SendMessage` (the tool you fetched at the top). Do NOT use the `Agent`
tool — that creates a new agent with no implementation context. The whole point
is to resume the same agent that built the code.

```
SendMessage(
  to: "<agent-name-from-step-2>",
  message: "PR review round {N} found {count} critical issues. Fix these on
the feature branch, run `make ci`, commit, and push. Do NOT check LSP or IDE
diagnostics — only `make ci` output matters. Issues:

1. {issue description} — {file path} — {suggested fix}
2. ..."
)
```

Await the agent's response confirming fixes are pushed. Then continue to the
next round.

**3e. If 3 rounds exhausted with critical issues still present → STOP:**
```
PR #{number} still has critical issues after 3 review rounds.
PR is open at {URL}. Remaining issues:
1. {issue}
2. ...
Manual intervention needed.
```
Leave the PR open. Do NOT merge.

---

## STEP 4 — FINALIZE AND MERGE

Only reached when the review loop exited cleanly (Step 3c).

### 4a. Doc finalization

Use `SendMessage` (NOT the `Agent` tool) to message the feature-implementer
agent by the name recorded in Step 2:

```
SendMessage(
  to: "<agent-name-from-step-2>",
  message: "PR is approved. On the feature branch, make these final updates:

1. Update the story's YAML frontmatter `status: done` in the story file
2. If all stories in the feature are now done, update `feature.md`
   frontmatter `status: done` as well
3. Update the overview index (docs/spec/00-overview.md) to reflect status
{4. If minor_issues is non-empty: Append these minor issues to
   docs/spec/issues.md:

   ---

   ## {Short title}
   **Found:** {YYYY-MM-DD} | **Source:** PR #{number} Review
   **Feature:** {NN-feature-name}

   {Description}

   Items to log:
   1. {minor issue}
   2. ...}
5. Commit as: chore(docs): mark story {NN} as done
6. Push to the feature branch."
)
```

Await confirmation that the commit is pushed.

### 4b. Merge

Run these commands yourself (not via SendMessage):

```bash
gh pr merge {number} --merge
git checkout main && git pull origin main
```

---

## STEP 5 — REPORT

```
## Orchestration Complete

**Spec:** {title} ({type} {NN})
**PR:** #{number} — {url}
**Status:** Merged to main
**Review:** {clean | passed after N round(s) | N minor issues logged}
**Issues logged:** {count} items added to docs/spec/issues.md (or "none")

**Summary:**
{1-3 sentences from the feature-implementer about what was built}
```

---

## STEP 6 — CONTINUE OR COMPACT

- **If the feature has more open stories** and the user requested the full
  feature (not a single story): run `/compact` to trim context, then loop
  back to Step 1 with the next open story.
- **If this was the last story** (or the user requested a single story):
  you are done.

> **Note:** `/compact` is a Codex CLI command — invoke it as a slash
> command to compress conversation history before starting the next story.

---

## CONSTRAINTS

- **Sequential only** — never launch multiple feature-implementers in parallel.
- **SendMessage for all agent communication** — never launch a new Agent to
  send fixes or doc tasks to the feature-implementer. Use `SendMessage` with
  the agent name from Step 2. A new Agent loses all implementation context.
- **`make ci` is the truth** — do not ask the feature-implementer to check
  LSP diagnostics or IDE errors. LSP errors in this codebase are frequently
  false positives. Only `make ci` output (lint + tests + coverage) is
  authoritative.
- **Never merge on failure** — if review issues persist after 3 rounds, leave
  the PR open and escalate to the user.
- **Respect dependency order** — dependency verification is part of Step 1.
  Never start implementation if a declared dependency is not yet merged.
- **Review loop is mandatory** — Step 3 always runs, even if the
  feature-implementer reports a clean self-review. External review is
  independent.

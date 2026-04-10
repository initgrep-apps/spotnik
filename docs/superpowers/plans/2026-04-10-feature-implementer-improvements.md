# Feature Implementer Agent Improvements — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add four targeted edits to `.claude/agents/feature-implementer.md` so that `test-driven-development` and `feature-dev:feature-dev` are invoked at binding points in the workflow, and fix a missing skill in the frontmatter.

**Architecture:** Single file, four independent edits applied in order: (1) frontmatter skill addition, (2) Phase 2 exploration step, (3) new Phase 2.5 section, (4) Phase 3 TDD line replacement. No logic changes — only prompt text.

**Tech Stack:** Markdown, YAML frontmatter.

---

## Files

- Modify: `.claude/agents/feature-implementer.md`

---

### Task 1: Add `pr-review-toolkit:review-pr` to frontmatter skills

**Files:**
- Modify: `.claude/agents/feature-implementer.md` (frontmatter, skills list)

Phase 6 of the agent invokes `pr-review-toolkit:review-pr` but the skill is absent from the
`skills:` frontmatter list. The agent reaches Phase 6 without the skill loaded.

- [ ] **Step 1: Open the file and locate the skills list**

The `skills:` block starts at line 8. It currently ends with:
```yaml
  - finishing-a-development-branch
```

- [ ] **Step 2: Add the missing skill after `receiving-code-review`**

Find this block:
```yaml
  - requesting-code-review
  - receiving-code-review
  - context7-mcp
```

Replace with:
```yaml
  - requesting-code-review
  - receiving-code-review
  - pr-review-toolkit:review-pr
  - context7-mcp
```

- [ ] **Step 3: Verify the frontmatter is valid YAML**

The complete `skills:` block should now read:
```yaml
skills:
  - using-superpowers
  - feature-dev:feature-dev
  - go-dev
  - bubbletea
  - test-driven-development
  - verification-before-completion
  - systematic-debugging
  - requesting-code-review
  - receiving-code-review
  - pr-review-toolkit:review-pr
  - context7-mcp
  - commit-commands:commit
  - commit-commands:commit-push-pr
  - finishing-a-development-branch
```

- [ ] **Step 4: Commit**

```bash
git add .claude/agents/feature-implementer.md
git commit -m "fix(agent): add pr-review-toolkit:review-pr to feature-implementer skills"
```

---

### Task 2: Add codebase exploration step to Phase 2

**Files:**
- Modify: `.claude/agents/feature-implementer.md` (Phase 2 — UNDERSTAND section)

Phase 2 currently ends with `TodoWrite` after three steps. A Step 4 for codebase exploration
must be inserted between Step 3 and `TodoWrite`.

- [ ] **Step 1: Locate the end of Phase 2 Step 3**

Find this exact block (it appears after Step 3's numbered list):
```
Use `TodoWrite` to create a todo item per task from the story. This is your progress tracker. You must keep it updated throughout the implementation.
```

- [ ] **Step 2: Insert Step 4 before the TodoWrite line**

Replace:
```
Use `TodoWrite` to create a todo item per task from the story. This is your progress tracker. You must keep it updated throughout the implementation.
```

With:
```
### Step 4: Codebase exploration
Invoke `feature-dev:feature-dev` to explore the codebase for patterns, integration points, and test conventions relevant to this story. Summarize findings — do not dump raw file traces. Use the findings to inform the architecture checkpoint and implementation.

Use `TodoWrite` to create a todo item per task from the story. This is your progress tracker. You must keep it updated throughout the implementation.
```

- [ ] **Step 3: Verify the Phase 2 section now has four steps**

The Phase 2 section (## PHASE 2 — UNDERSTAND) should now contain:
- `### Step 1: Feature context`
- `### Step 2: Previous stories`
- `### Step 3: Target story`
- `### Step 4: Codebase exploration` ← new
- `Use TodoWrite...` paragraph ← unchanged, still last

- [ ] **Step 4: Commit**

```bash
git add .claude/agents/feature-implementer.md
git commit -m "feat(agent): add codebase exploration step to Phase 2 of feature-implementer"
```

---

### Task 3: Insert Phase 2.5 — Architecture checkpoint

**Files:**
- Modify: `.claude/agents/feature-implementer.md` (between Phase 2 and Phase 3)

A new phase must be inserted between the end of Phase 2 and the start of Phase 3.

- [ ] **Step 1: Locate the boundary between Phase 2 and Phase 3**

Find this exact line (the opening of Phase 3):
```
## PHASE 3 — IMPLEMENT
```

- [ ] **Step 2: Insert the new phase immediately before Phase 3**

Replace:
```
## PHASE 3 — IMPLEMENT
```

With:
```
## PHASE 2.5 — ARCHITECTURE

Before writing any code, invoke `feature-dev:feature-dev` for architecture design.

Scale to story complexity:
- **Simple** (test-only, single-file, or uses a pattern already present in previous stories): one-sentence sketch of what changes and why — no multi-approach analysis needed.
- **Complex** (new interfaces, multiple packages, or a pattern not seen in previous stories): full architecture design with trade-offs before proceeding.

---

## PHASE 3 — IMPLEMENT
```

- [ ] **Step 3: Verify the phase order is correct**

The sequence of `##` headings in the file should now be:
1. `## BEFORE ANYTHING ELSE`
2. `## INPUT NORMALIZATION`
3. `## PHASE 1 — SETUP (Fresh Start)`
4. `## PHASE 2 — UNDERSTAND`
5. `## PHASE 2.5 — ARCHITECTURE` ← new
6. `## PHASE 3 — IMPLEMENT`
7. `## PHASE 4 — SELF-REVIEW`
8. `## PHASE 5 — DELIVER`
9. `## PHASE 6 — INTERNAL PR REVIEW LOOP`
10. `## DOC FINALIZATION (when requested by caller)`
11. `## ESCALATION POLICY`
12. `## AGENT MEMORY`

Run:
```bash
grep "^## " .claude/agents/feature-implementer.md
```

Expected output:
```
## BEFORE ANYTHING ELSE
## INPUT NORMALIZATION
## PHASE 1 — SETUP (Fresh Start)
## PHASE 2 — UNDERSTAND
## PHASE 2.5 — ARCHITECTURE
## PHASE 3 — IMPLEMENT
## PHASE 4 — SELF-REVIEW
## PHASE 5 — DELIVER
## PHASE 6 — INTERNAL PR REVIEW LOOP
## DOC FINALIZATION (when requested by caller)
## ESCALATION POLICY
## AGENT MEMORY
```

- [ ] **Step 4: Commit**

```bash
git add .claude/agents/feature-implementer.md
git commit -m "feat(agent): add Phase 2.5 architecture checkpoint to feature-implementer"
```

---

### Task 4: Replace "Tests first" in Phase 3 with TDD skill invocation

**Files:**
- Modify: `.claude/agents/feature-implementer.md` (Phase 3 — IMPLEMENT, step 2)

Phase 3 currently has a soft "Tests first" directive that doesn't enforce TDD discipline.
Replace it with a direct skill invocation.

- [ ] **Step 1: Locate the line to replace**

Inside `## PHASE 3 — IMPLEMENT`, find this exact line:
```
2. **Tests first** — invoke the `test-driven-development` skill with the task's test list from the spec
```

- [ ] **Step 2: Replace the line**

Replace:
```
2. **Tests first** — invoke the `test-driven-development` skill with the task's test list from the spec
```

With:
```
2. **Tests** — invoke `test-driven-development` for this task
```

- [ ] **Step 3: Verify the Phase 3 task loop looks correct**

The numbered list under `## PHASE 3 — IMPLEMENT` should now read:
```
1. **Mark task in-progress** via `TodoUpdate`
2. **Tests** — invoke `test-driven-development` for this task
3. **Implement** — use your available skills that match the project's tech stack for idiomatic implementation.
4. **Look up docs — don't guess** — ...
5. **LSP check after every edit** — ...
6. **Verify** — run the project's CI command ...
7. **One conventional commit per completed task** — ...
```

- [ ] **Step 4: Commit**

```bash
git add .claude/agents/feature-implementer.md
git commit -m "feat(agent): make test-driven-development an integral invocation in Phase 3"
```

---

## Final verification

After all four tasks are complete:

- [ ] Confirm all `##` headings are in correct order:
```bash
grep "^## " .claude/agents/feature-implementer.md
```

- [ ] Confirm `pr-review-toolkit:review-pr` is in the skills list:
```bash
grep "pr-review-toolkit" .claude/agents/feature-implementer.md
```

- [ ] Confirm Phase 2.5 exists:
```bash
grep "PHASE 2.5" .claude/agents/feature-implementer.md
```

- [ ] Confirm the old "Tests first" line is gone:
```bash
grep "Tests first" .claude/agents/feature-implementer.md
# Expected: no output
```

- [ ] Confirm the new TDD invocation line is present:
```bash
grep "invoke \`test-driven-development\`" .claude/agents/feature-implementer.md
# Expected: one match
```

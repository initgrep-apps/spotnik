# Feature Implementer Agent — Improvement Design

> **Goal:** Make `test-driven-development` and `feature-dev:feature-dev` genuinely integral to the
> agent's workflow by adding targeted skill invocation points — without restructuring the existing
> phases, without adding project-specific content, and without describing skill internals inside
> the agent prompt.

---

## Background

The `feature-implementer` agent (`.claude/agents/feature-implementer.md`) was reviewed against
official Anthropic sub-agent guidelines and evidence from 15+ completed feature stories.

### What works well

- Clear 6-phase pipeline that has survived 100+ story implementations unchanged
- LSP-after-every-edit discipline (Phase 3)
- Practical escalation policy (3-attempt cap, 2-review-iteration cap)
- DOC FINALIZATION and AGENT MEMORY sections are mature
- Phase 1 (always from clean main) is solid

### Issues identified

1. **`test-driven-development` is mentioned but not enforced.** Phase 3 says "Tests first —
   invoke the `test-driven-development` skill with the task's test list from the spec." This is a
   soft suggestion. Nothing ensures RED-GREEN-REFACTOR is actually followed per task. Memory
   evidence: multiple stories had post-implementation test fixes caught only in PR review, and
   "tests first" was written after the implementation in some cases.

2. **`feature-dev:feature-dev` is invoked as a decoration.** Phase 3 step 1 says "Invoke the
   `feature-dev:feature-dev` skill for structured development guidance" — one bullet in a list,
   no defined purpose. The skill's exploration and architecture phases are never actually used.
   Memory evidence: recurring gotchas (wrong API token names, missed call sites, override pattern
   surprises) that upfront exploration would have caught.

3. **Phase 2 has no codebase exploration.** It reads spec files but never explores the actual
   codebase for patterns relevant to the story. This is the root cause of the "late discovery"
   gotchas in memory.

4. **`pr-review-toolkit:review-pr` is referenced in Phase 6 but absent from the `skills:`
   frontmatter.** The agent reaches Phase 6 without the skill loaded.

5. **No architecture checkpoint before implementation.** The agent goes from reading specs
   directly to writing code. For complex stories (new interfaces, multi-package changes), this
   means design mistakes are committed before they become obvious.

---

## Design

### Approach: Option B — Binding skill invocations at specific gates

Keep the existing phase structure. Add targeted skill invocation points. The agent prompt never
describes what the skills do internally — it only names where to invoke them and why.
Project-specific knowledge stays in CLAUDE.md and agent memory, not in this prompt.

---

### Change 1 — Frontmatter: add missing skill

Add `pr-review-toolkit:review-pr` to the `skills:` list in the YAML frontmatter.

**Why:** Phase 6 invokes this skill but it was not in the loaded skill set, meaning the agent
reaches Phase 6 without the skill available.

---

### Change 2 — Phase 2: add codebase exploration step

After the existing three steps (feature context, previous stories, target story) and before
`TodoWrite`, add:

> **Step 4: Codebase exploration**
> Invoke `feature-dev:feature-dev` to explore the codebase for patterns, integration points, and
> test conventions relevant to this story. Summarize findings — do not dump raw file traces.
> Use the findings to inform implementation.

`TodoWrite` remains last in Phase 2.

**Why:** The exploration surfaces integration details (method signatures, call sites, override
patterns, unexported fields) before any code is written. The "summarize findings" note prevents
verbose output from consuming context.

---

### Change 3 — New Phase 2.5: architecture checkpoint

Insert a new phase between UNDERSTAND and IMPLEMENT:

> **PHASE 2.5 — ARCHITECTURE**
>
> Before writing any code, invoke `feature-dev:feature-dev` for architecture design.
>
> Scale to story complexity:
> - **Simple** (test-only, single-file, uses an existing pattern already seen in previous
>   stories): one-sentence sketch of what changes and why — no multi-approach analysis needed.
> - **Complex** (new interfaces, multiple packages, or a pattern not present in previous
>   stories): full architecture design with trade-offs before proceeding.

**Why:** For complex stories, this is where the costliest bugs originate — design mistakes
committed before the shape of the problem is fully understood. For simple stories, the cost is
one sentence, which is negligible.

---

### Change 4 — Phase 3: replace "Tests first" with skill invocation

**Current:**
```
**Tests first** — invoke the `test-driven-development` skill with the task's test list from the spec
```

**Replacement:**
```
**Tests** — invoke `test-driven-development` for this task
```

No description of what TDD does. The skill governs the full cycle. If the skill is updated in
the future, the agent inherits the change without any prompt edit.

---

## Simulation Results

Traced both agents through two real stories before finalising this design.

### Story 112 — Simple (test-only, 3 tasks)

| | Old agent | New agent |
|---|---|---|
| Finds unexported field scope issue | Via compile error (late) | Via exploration (before writing) |
| TDD discipline | Ambiguous — "tests first" can be skipped | Skill invoked per task, RED enforced |
| Architecture overhead | None | One sentence |

**Verdict:** Small but real improvement. No regression. Overhead is minimal.

### Story 113 — Medium (4 tasks, 3 packages)

| | Old agent | New agent |
|---|---|---|
| Finds all 4 type-assertion call sites | Risk of missing 1 | Found in exploration before writing |
| StateReader safety verified | Unverified; relies on spec text | Confirmed via handlers.go exploration |
| Architecture overhead | None | Brief: 3 components, no trade-off analysis needed |

**Verdict:** Clear improvement. The exploration step is the key difference — call sites found
upfront, safety confirmed before touching code.

**No regression found in either scenario.**

---

## What this design does NOT do

- Does not add Spotnik-specific content to the agent prompt. The agent stays project-agnostic.
- Does not restructure the existing 6 phases.
- Does not describe the internals of `test-driven-development` or `feature-dev:feature-dev`
  in the agent prompt.
- Does not change Phase 1, Phase 4, Phase 5, Phase 6, DOC FINALIZATION, ESCALATION POLICY,
  or AGENT MEMORY sections.

---

## Files changed

| File | Change |
|---|---|
| `.claude/agents/feature-implementer.md` | 4 targeted edits (frontmatter, Phase 2 step 4, new Phase 2.5, Phase 3 step 2) |

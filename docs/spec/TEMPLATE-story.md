---
title: "Story Title"
feature: NN-feature-name
status: open
---

## Background

1–3 paragraphs explaining *why* this story exists. What problem does it solve?
What existing code or behavior does it change? What dependencies does it have on
previous stories or existing systems?

Reference specific files, types, and functions by name so an implementer can find
the starting point without exploration.

## Design

Detailed technical design. Every struct, interface, function signature, message
type, layout specification, or algorithm that an implementer needs to write code.
Include Go code blocks for new types and key logic. Reference the packages and
files where changes go.

Subsections organize the design by concern (e.g. "Domain types", "API changes",
"UI rendering", "Message types", "App wiring").

### Subsection pattern

- What changes
- Where it changes
- Why it changes that way

## Files

### Create

- `path/to/new_file.go` — purpose
- `path/to/new_file_test.go` — purpose

### Modify

- `path/to/existing.go` — what changes and why

### Delete

- `path/to/removed.go` — why it's removed

## Acceptance Criteria

- [ ] Each criterion is testable and unambiguous
- [ ] Use specific values, not vague language ("renders ♪ for tracks" not "renders correctly")
- [ ] Include edge cases and error conditions
- [ ] Final criterion: `make ci` passes

## Tasks

Each task is a single logical unit of work: reviewable and shippable independently.
Tasks specify **what to change**, **which files**, and **what tests to write**
(with specific test function names in `backtick` format).

Switch to next task only after current task compiles and its tests pass.

- [ ] Task description — what to change and why
      - test: `TestFunctionName_Scenario`, `TestFunctionName_AnotherScenario`
- [ ] Task description
      - test: `TestFunctionName_Scenario`
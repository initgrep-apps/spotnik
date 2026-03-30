---
title: "Architecture & Codebase Health"
status: done
---

## Description
Cross-cutting architecture work that maintains codebase health, enforces import boundaries, decomposes large files, fixes stale documentation, removes dead code, and aligns type designs with established conventions. These stories are not user-facing features but structural improvements that keep the codebase maintainable and consistent as Spotnik grows.

## Acceptance Criteria
- [ ] No `ui/ -> api/` import violations exist in the codebase
- [ ] `app.go` is decomposed into focused files under 700 lines each
- [ ] All documentation comments accurately reflect current architecture rules
- [ ] No dead code remains in production files
- [ ] All message types follow consistent naming and export conventions
- [ ] `make ci` passes after all architecture stories are complete

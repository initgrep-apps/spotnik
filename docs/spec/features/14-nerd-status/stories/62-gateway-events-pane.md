---
title: "Request Flow Boxed Layout"
feature: 14-nerd-status
status: done
---

## Background
The Request Flow pane rendered as a flat text table with three padded columns. DESIGN.md specified three bordered sub-boxes (APP, GATEWAY, SPOTIFY) connected by animated arrows -- a graphical visualization, not a table. Gateway metrics rendered as a separate block below all request rows instead of inside the center column. This story restructured View() to render three independently bordered boxes with rounded corners, dual arrow columns, row alignment, and a flat fallback for terminals < 60 columns.

## Design

### renderSubBox() Helper
Bordered box with title using rounded corners (╭╮╰╯). Takes title, content lines, width. Width < 8 returns empty string.

### Dual Arrow Columns
Left arrows (APP->GW): gateway decisions. Right arrows (GW->SPOTIFY): HTTP outcomes. Each arrow type reflects its domain: left shows allowed/wait/dedup/blocked, right shows 2xx/429/5xx/blocked.

### Box Content Generators
- buildAppBoxLines(maxRows): styled endpoint lines per request
- buildGatewayBoxLines(maxRows): gateway metric lines (token bucket, semaphore, backoff, dedup, in-flight keys)
- buildSpotifyBoxLines(maxRows): styled status+latency lines

### Boxed Layout Composition
Column widths: APP ~25%, left arrow ~8%, GATEWAY ~26%, right arrow ~8%, SPOTIFY ~20%. Minimum width: < 60 falls back to viewFlat(). Arrow columns aligned with box content rows.

## Acceptance Criteria
- [ ] Three bordered sub-boxes render with rounded corners
- [ ] Gateway metrics appear inside center GATEWAY box
- [ ] Two arrow columns connect the boxes
- [ ] Request rows align horizontally across all boxes
- [ ] Arrow states match Feature 61
- [ ] Theme colors preserved
- [ ] Status strip below the boxes
- [ ] Flat fallback for width < 60
- [ ] All existing tests pass + new tests
- [ ] make ci passes

## Tasks
- [ ] Create renderSubBox() helper in requestflow_pane.go
      - test: rounded corners and title; content padded; long lines truncated; width < 8 empty; border-only box
- [ ] Create renderRightArrow() for GW->SPOTIFY
      - test: 200 animated Success; 429 X Warning; 500 animated Error; 0 X TextMuted; width respected
- [ ] Build content generators for each sub-box
      - test: fewer requests pads; caps at maxRows; backoff when throttled; 429 warning; maxRows=0
- [ ] Restructure View() to render boxed layout
      - test: width 80 shows boxes; width 40 flat; 3 requests 3 arrow rows; status strip; height 5 minimal; height 0 empty
- [ ] Align arrow animation with sub-box rows
      - test: content lines align; arrow block matches box height; first/last blank
- [ ] Preserve existing rendering methods
      - test: renderGatewayState unchanged; gatewayStateLines correct; viewFlat matches previous
- [ ] Update existing tests for boxed layout
      - test: flat fallback; boxed layout; gateway in center; dual arrows; preserved decision/color/staleness tests
- [ ] Update documentation
      - test: docs change only

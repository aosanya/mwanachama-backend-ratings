# mwanachama-backend-ratings — open tasks

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.

A routine, **Context & ratings backlog** (`trig_0167WRNiqZHBkPb9NVLBx95T`,
every 6 hours from 2026-10-10), works this board and `mwanachama-backend-context`'
unattended: research rows first, then build rows in dependency order, on
`main`, never a branch. It does not decide an owner question — a row that
needs one gets a "Needs the owner:" note and stays open — and it does not
start RT11 until the module exists and shared's S59 lands. **Since
2026-10-11 the routine builds shared's S56–S59 itself**, on
`mwanachama-backend-shared`'s `master` directly (the owner's call; the rule
is on that board's header). Finished rows move to `todo_done.md` with the
real test output.

Research first, then the build in dependency order. Every decision the build
rests on is in `requirements.md`; a row names the ones it carries.

## Research

None open. RT1 and RT2 are in `todo_done.md`.

## Build

| Task | Title | Depends on | Notes |
|------|-------|------------|-------|
| RT11 | ⏸️ **Attach ratings to a catalog instance through the catalog's config file** | RT7 ✅, shared's S62 ✅ | Requirements #23 (the owner, 2026-10-12). The mechanism landed the same day: shared's `spec.Spec.Attach` (S62) — the catalog spec carries `"attach": [{"module": "ratings", "items": "catalog.entry"}]`, the engine validates it, the host mounts. What is left is the host half, in `mwanachama-wakala-api`: add the `attach` block to `internal/specs/wakala.catalog.json` (and `wakala.reports.json` if the board is rateable), and for each attachment found through `s.Attached("ratings")` load ratings' own spec, `Provision` it, and mount its routes beside the catalog's; map the caller to the rater (#20). Nothing toward context (#22). Not this routine's repo. |

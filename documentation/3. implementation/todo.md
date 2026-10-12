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
| RT12 | 📋 **How the host turns `rated` edges on** | RT8 ✅ | #15: rater ▶ item `rated` edges travel only when the host's `context.<domain>.json` declares `rated`, but ratings cannot read that declaration (#18) and nothing says how the host tells it. RT8 ships `similar_taste` only, which is #15's default. **Needs the owner:** a `setting` row the host writes on boot (e.g. `edges = similar_taste,rated`, the existing per-instance knob mechanism), a second library call the host makes only when it declared `rated`, or an argument on `Edges` |
| RT11 | ⏸️ **Mount in `mwanachama-wakala-api`** | RT7 ✅, RT8 ✅ | Cross-repo. `RATINGS_INSTANCE` → `SpecFor`, provisioned on boot, mounted beside the other declared domains per that repo's decision #22; the caller→rater mapping; the batched reindex trigger toward context (#18): per-pod pending set, flush at ~30 s or N raters. Ready for it (2026-10-12): `Edges` (RT8) and shared's S59 (`identity.Written`, the report a host sends a reacting module) both landed; what is left is wakala-api's mount, which is not this routine's repo. |

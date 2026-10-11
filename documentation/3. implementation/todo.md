# mwanachama-backend-ratings — open tasks

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.

A routine, **Context & ratings backlog** (`trig_0167WRNiqZHBkPb9NVLBx95T`,
every 6 hours from 2026-10-10), works this board and `mwanachama-backend-context`'
unattended: research rows first, then build rows in dependency order, on
`main`, never a branch. It does not decide an owner question — a row that
needs one gets a "Needs the owner:" note and stays open — and it does not
start RT11 until the module exists and shared's S59 lands. Finished rows move
to `todo_done.md` with the real test output.

Research first, then the build in dependency order. Every decision the build
rests on is in `requirements.md`; a row names the ones it carries.

## Research

None open. RT1 and RT2 are in `todo_done.md`.

## Build

| Task | Title | Depends on | Notes |
|------|-------|------------|-------|
| RT8 | ⏸️ **`Edges(ctx, partition, keys)` — the extractor toward context** | RT5 ✅, context's TR13, shared's S59 | #8, #15: `similar_taste` always, `rated` only when the host declared it. Blocked on the write-report shape (S59) and on context existing to receive it. The host calls it, batched (#18). |
| RT10 | 📋 **Measure against a public rating set** | RT2 ✅, RT6 ✅ | Load the set through `put`, run RT2's procedure (`4. qa/measurement-procedure.md`), record the numbers in `4. qa/measurements.md`, and set the defaults for metric, K and overlap from them. **The harness is built (2026-10-11)**: `make measure` runs `TestMeasure` under `//go:build measure`, skipped unless `RATINGS_MEASURE_DIR` names the extracted set; the grid, the baselines and the design page's variant columns all print as pasteable tables, and `RATINGS_MEASURE_METRICS`/`_K`/`_OVERLAP` narrow a rehearsal. Proven end to end on a synthetic set only; `go test ./...` never sees it. Needs the owner: the routine's environment cannot reach `files.grouplens.org` (egress policy, confirmed 2026-10-10 and 2026-10-11: `CONNECT` answers 403), nor any mirror — `web.archive.org`, `huggingface.co`, `zenodo.org`, `openml.org`, `figshare.com` and `data.mendeley.com` are all 403 too, and `pypi.org`, which is open, bundles no copy — so the ml-100k archive has to be placed by hand or the policy widened before this row can run unattended. Once it is there the run is `RATINGS_MEASURE_DIR=/path/to/ml-100k make measure`, which will take hours for the full grid. |
| RT11 | ⏸️ **Mount in `mwanachama-wakala-api`** | RT7 ✅, RT8 | Cross-repo. `RATINGS_INSTANCE` → `SpecFor`, provisioned on boot, mounted beside the other declared domains per that repo's decision #22; the caller→rater mapping; the batched reindex trigger toward context (#18): per-pod pending set, flush at ~30 s or N raters. |

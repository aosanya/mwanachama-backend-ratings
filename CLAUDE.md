# CLAUDE.md

Guidance for Claude Code working in this repository.

## Project: mwanachama-backend-ratings

Ratings and the similarity they imply. A `rating` is `(rater, item, value,
at)`; a `similarity` is `(a, b, metric, score, at)`, computed from co-rated
items by a declared metric. Module path
`github.com/aosanya/mwanachama-backend-ratings`.

**This repo is in research.** Nothing is built. The decisions are on
[documentation/1. requirements/requirements.md](documentation/1.%20requirements/requirements.md),
the open questions beneath them, and the research rows on
[documentation/3. implementation/todo.md](documentation/3.%20implementation/todo.md)
come before any code. Do not resolve an open question by guessing; ask.

## What is already known

- **Split out of `mwanachama-backend-context` on 2026-10-10.** That module
  holds weighted edges between keys and knows nothing about what it maps; a
  ratings matrix is tabular and similarity is a join plus a sum, which is
  SQL's job. Two repos, one contract between them.
- **Postgres, declared domain, like every sibling** — `ratings.blueprint.json`
  through `mwanachama-backend-shared/spec` and `specstore`; no row structs,
  no `AutoMigrate`, no ArangoDB. `mwanachama-backend-catalog` is the shape
  to follow.
- **A rater is any key, and so is an item.** Both are raw strings in the
  standard's form `<module>/<instance>/<identity>` (developer's
  `architecture-spec-driven-modules.md` §9), owned by other modules. This
  module stores them and never resolves what they mean.
- **Similarity is computed incrementally** — for the rater (or item) whose
  ratings changed, over its co-raters only, never all pairs. Efficiency is a
  requirement, not a nicety.
- **The metric is declared**, from a closed set the module supplies:
  `euclidean` (asked for first), `cosine`, `jaccard`.
- **Toward `mwanachama-backend-context` it is an extractor**: on that
  module's `reindex` it emits `similar_taste` edges with `weight = score`.
- **It names no domain.** Not "movie", not "member", not "user" — `rater`
  and `item`. A `domain_agnostic_test.go` will enforce it over identifiers,
  the blueprint and every stored enum value, as the siblings do.

## Conventions

- Four-phase `documentation/` layout — see
  [documentation/README.md](documentation/README.md).
- `go test ./...` (sqlite via `glebarez/sqlite`) will verify a change here;
  what sqlite cannot stand in for goes under `//go:build integration`, gated
  on `POSTGRES_URL`, run by `make test-pg`.
- No Go file over 300 lines; split by responsibility.

## Code comments

Write code with no comments. Not one-liners above a function, not section
banners, not doc comments on exported symbols, not "why" notes next to a
tricky line. A name, a type, or a smaller function carries it instead.

Anything that genuinely needs explaining goes in this repo's `documentation/`
folder, under the phase it belongs to (`1. requirements`, `2. design`,
`3. implementation`, `4. qa`) — never inline.

**Why:** inline prose drifts out of sync with the code, duplicates what
`documentation/` already owns, and buries the explanation where nobody
looking for it will search.

**How to apply:**

- New code ships without comments. If a line seems to need one, rename or
  split until it doesn't.
- Touching code that already has comments: strip the ones in the code you are
  changing. Do not sweep untouched files unless asked.
- If the reasoning matters, add or update the matching `documentation/` page
  in the same change and leave nothing behind in the source.
- Machine-read directives are not comments and stay: build tags, `//go:embed`,
  `//go:generate`, linter pragmas (`//nolint`, `// eslint-disable-next-line`,
  `// ignore:`), license headers, codegen "do not edit" banners, and generated
  files as a whole.
- Commit messages, PR descriptions, and test names carry the narration that
  used to go in comments.

This rule is repeated verbatim in every mwanachama repo's `CLAUDE.md` so that
it reaches sessions that do not load this machine's user-level config —
scheduled cloud routines, other machines, and other agent harnesses.

# Design

Nothing is built. This folder records the shape the requirements add up to.

| Page | What it holds |
|------|---------------|
| [ratings-model.md](ratings-model.md) | The two objects and a blueprint sketch, the incremental top-K computation, the metric set, the five operations, `recommend`'s query, and the contract with `mwanachama-backend-context`. |

Pages the board will add: `measurement.md` (RT2), and whatever RT1's survey
finds worth recording.

## Where the engine is documented

`mwanachama-backend-shared`'s
[declared-domains.md](../../../mwanachama-backend-shared/documentation/2.%20design/declared-domains.md)
is the storage; its
[dispatcher.md](../../../mwanachama-backend-shared/documentation/2.%20design/dispatcher.md)
is the operations table. `mwanachama-backend-catalog` is the sibling whose
shape this repo follows.

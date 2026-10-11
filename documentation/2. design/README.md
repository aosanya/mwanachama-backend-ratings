# Design

This folder records the shape the requirements add up to; RT3 built the
objects and the manager constructor against it on 2026-10-10.

| Page | What it holds |
|------|---------------|
| [ratings-model.md](ratings-model.md) | The two objects and a blueprint sketch, the incremental top-K computation, the metric set, the five operations, `recommend`'s query, and the contract with `mwanachama-backend-context`. Each built row (RT3, RT5, RT6, RT7) adds an "As built" section under the part it built. |
| [collaborative-filtering-survey.md](collaborative-filtering-survey.md) | RT1: user- vs item-based, the three metrics on [0, 1] values, K and overlap grids, the cost of the incremental step at 10⁴ × 10⁴, and what top-K storage cannot do alone. |
| [measurement.md](measurement.md) | RT2: the rating set and its terms, leave-the-latest-out, the scores (RMSE, MAE, coverage, hit rate and ARHR @10), the baselines, the 60-cell grid and the rule that turns it into the defaults for #6, #13 and #14. |

Pages the board will add: `4. qa/measurements.md` (RT10, the numbers).

## Where the engine is documented

`mwanachama-backend-shared`'s
[declared-domains.md](../../../mwanachama-backend-shared/documentation/2.%20design/declared-domains.md)
is the storage; its
[dispatcher.md](../../../mwanachama-backend-shared/documentation/2.%20design/dispatcher.md)
is the operations table. `mwanachama-backend-catalog` is the sibling whose
shape this repo follows.

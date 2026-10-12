# mwanachama-backend-ratings

Ratings and the similarity they imply: a rater gives an item a value, and
from many such values the module computes how alike two raters are — or two
items — by a declared metric. A rater is always an actor. An item is any key. The module never knows what either is.

It answers "what would this rater rate highly" on its own, by SQL, and feeds
nothing else — it is completely separate from
[mwanachama-backend-context](../mwanachama-backend-context) (decision #22).

Split out of `mwanachama-backend-context` on 2026-10-10 (its decision #61).
The blueprint, two domain specs, the models and the manager constructor are
built (RT3); the writes, the similarity step, `recommend` and the operations
table follow in order on the board in
[documentation/](documentation/README.md).

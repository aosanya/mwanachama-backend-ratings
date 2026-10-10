# mwanachama-backend-ratings

Ratings and the similarity they imply: a rater gives an item a value, and
from many such values the module computes how alike two raters are — or two
items — by a declared metric. A rater is any key: a person, a chapter, an
agency, a cohort. An item is any key. The module never knows what either is.

It answers "what would this rater rate highly" on its own, by SQL, and feeds
`similar_taste` edges into
[mwanachama-backend-context](../mwanachama-backend-context) so a walk there
can blend taste with everything else that graph holds.

Split out of `mwanachama-backend-context` on 2026-10-10 (its decision #61).
The blueprint, two domain specs, the models and the manager constructor are
built (RT3); the writes, the similarity step, `recommend` and the operations
table follow in order on the board in
[documentation/](documentation/README.md).

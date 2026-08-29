# Changelog

Versions follow the rule in
[section 13 of the specification](SPECIFICATION.md#13-versioning-and-unknown-fields):
a rise in MINOR only adds things, and a rise in MAJOR may change or remove them.

Each released schema stays in `schema/` forever, so a file that pinned an older
one keeps validating.

## 1.1

Additive. Every 1.0 file is a valid 1.1 file, and every 1.1 file in `examples/`
still validates against the frozen 1.0 schema — the repository checks both, which
is what an additive release is supposed to mean.

- **`card.trinketTier`** — `lesser` or `greater`. The game offers two trinket
  slots at two different points and a player holds at most one of each, so this
  is a property of the trinket, not a rarity. 1.0 had no home for it and pushed
  producers into `ext`.
- **`offerType` gains `lesserTrinket` and `greaterTrinket`.** Plain `trinket`
  stays, for a recorder that cannot tell which slot an offer was for.
- **`standing.hero`** — the hero on a leaderboard row. The leaderboard shows
  every seat's hero, so this is the one fact about an opponent that a one-seat
  recording reliably learns, and 1.0 gave it nowhere to go inside a `state`
  entry.
- **Guidance on `players[].hero`**: record it for every seat you can, not only
  the recorded one.
- **New specification section 12A, "Recording from a game client"** — what a
  recording made from one player's client contains, and the rule that a seat
  recording carries actions for one player and reaches the others only through
  events and states.

## 1.0

First release.

- One ordered `history` of entries, each an action, an event or a state.
- Recruit-phase and combat vocabulary covering the game as it is played, not one
  simulator's subset.
- Declared observation scope, so a lobby-wide recording and a one-seat recording
  are both expressible and tell you apart.
- Declared combat detail, from `outcome` up to a blow-by-blow event stream.
- Card identity by stable card id, and copy identity by entity name.
- `ext` for producer data, and an `x-` prefix for producer vocabulary.

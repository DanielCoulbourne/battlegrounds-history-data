# Changelog

Versions follow the rule in
[section 13 of the specification](SPECIFICATION.md#13-versioning-and-unknown-fields):
a rise in MINOR only adds things, and a rise in MAJOR may change or remove them.

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

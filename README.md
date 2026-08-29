# Battlegrounds History Data

An open, engine-neutral file format for recording a game of Hearthstone
Battlegrounds: the states the game passed through, and the actions that moved
between them. Both halves of a turn are covered — the shopping half and the
fight.

- **[SPECIFICATION.md](SPECIFICATION.md)** — every object and every field.
- **[RATIONALE.md](RATIONALE.md)** — the decisions behind the shape, and what
  was rejected.
- **[GLOSSARY.md](GLOSSARY.md)** — the game's words, in plain language.
- **[MIGRATION.md](MIGRATION.md)** — converting a nested turn-and-action
  recording, field by field.
- **[schema/](schema/)** — the JSON Schema. It is the normative artifact.
- **[examples/](examples/)** — complete files, in JSON and YAML.

This project is not affiliated with Blizzard Entertainment.

## Why this exists

Recordings of Battlegrounds games are made by programs that have nothing else in
common: deck trackers watching a live client, simulators playing millions of
games against themselves, replay parsers reading a stored match. Each one invents
its own shape, and none of the recordings can be read by anyone else's tools.

This format is a shape all three can write. It is designed around two facts that
most ad-hoc formats get wrong:

1. **A recording is made by somebody, from somewhere.** A simulator sees all
   eight seats; a tracker sees one. The file says which, up front, so a reader
   never has to guess whether an absent fact was hidden or just not written down.
2. **A recording is not a replay.** The same starting boards run through two
   different engines give two different fights. So the file carries what
   happened, not the inputs you would need to make it happen again.

## Who it is for

- People building bots, who need games to learn from and audits to trust.
- People building trackers and replay viewers, who want one format instead of
  three.
- People comparing simulators, who need to feed the same fight to two engines and
  see where they part company.

You do not need to have played Battlegrounds to use it.
[Section 2 of the specification](SPECIFICATION.md#2-the-game-for-programmers-who-have-not-played-it)
teaches you every word the format uses, and [GLOSSARY.md](GLOSSARY.md) is the
quick reference.

## A complete small example

This is a whole file. It records one turn: a player is given 4 gold, buys a
minion, plays it, ends the turn, and wins the fight that follows.

```json
{
  "format": "battlegrounds-history",
  "version": "1.0",
  "recording": {
    "id": "readme-example",
    "recorder": { "name": "Example Tracker", "version": "0.1.0", "kind": "tracker" },
    "observer": { "scope": "seat", "seat": "p1" },
    "detail": { "combat": "outcome", "states": "turnStart", "entities": true },
    "truncated": true,
    "truncationReason": "One turn, not a whole game."
  },
  "game": { "id": "demo", "mode": "solo", "seatCount": 8, "patch": "34.2.0" },
  "players": [
    { "id": "p1", "seat": 0, "kind": "human", "startingHealth": 30 },
    { "id": "p2", "seat": 4, "kind": "unknown" }
  ],
  "history": [
    {
      "seq": 1, "kind": "event", "event": "turn_start", "turn": 2, "phase": "recruit",
      "data": { "turn": 2, "gold": 4, "tier": 1 }
    },
    {
      "seq": 2, "kind": "event", "event": "shop_dealt", "actor": "p1", "turn": 2, "phase": "recruit",
      "data": {
        "reason": "turnStart",
        "cards": [
          { "entity": "s1", "cardId": "BG32_330", "name": "Flighty Scout", "type": "minion",
            "attack": 3, "health": 3, "tier": 1, "minionTypes": ["murloc"], "cost": 3 },
          { "entity": "s2", "cardId": "BG31_803", "name": "Buzzing Vermin", "type": "minion",
            "attack": 1, "health": 1, "tier": 1, "minionTypes": ["beast"], "keywords": ["taunt"], "cost": 3 }
        ]
      }
    },
    {
      "seq": 3, "kind": "state", "actor": "p1", "turn": 2, "phase": "recruit", "reason": "turnStart",
      "player": { "health": 30, "tier": 1, "gold": 4, "upgradeCost": 5 },
      "zones": { "board": { "cards": [] }, "hand": { "cards": [] } }
    },
    {
      "seq": 4, "kind": "action", "actor": "p1", "action": "buy", "turn": 2, "phase": "recruit",
      "data": { "from": { "zone": "shop", "index": 0, "entity": "s1" }, "cost": 3, "gold": 1 }
    },
    {
      "seq": 5, "kind": "action", "actor": "p1", "action": "play", "turn": 2, "phase": "recruit",
      "data": { "from": { "zone": "hand", "index": 0, "entity": "s1" },
                "to": { "zone": "board", "index": 0 } }
    },
    {
      "seq": 6, "kind": "event", "event": "pairing_announced", "actor": "p1", "turn": 2,
      "phase": "recruit", "data": { "opponent": "p2" }
    },
    {
      "seq": 7, "kind": "action", "actor": "p1", "action": "end_turn", "turn": 2,
      "phase": "recruit", "data": { "gold": 1 }
    },
    {
      "seq": 8, "kind": "event", "event": "combat_end", "turn": 2, "phase": "combat",
      "data": {
        "combat": "c2",
        "winner": "p1",
        "sides": [
          { "player": "p1", "healthBefore": 30, "healthAfter": 30, "damageTaken": 0 },
          { "player": "p2", "healthBefore": null, "healthAfter": null, "damageTaken": null }
        ]
      }
    }
  ]
}
```

Three things to notice:

- **`observer.scope` is `seat`.** This file was made by a program watching one
  player's client, so it contains only what that player could see.
- **The opponent's health is `null`, not 0.** `null` means "this applies and I
  could not see it". Leaving the field out would have meant something else.
- **Actions and events are separate.** `buy` is something the player chose.
  `shop_dealt` is something the game did. A reader can filter for decisions and
  get exactly the decisions.

The start of the same file in YAML. It is the same data, so it validates against
the same schema:

```yaml
format: battlegrounds-history
version: '1.0'
recording:
  id: readme-example
  recorder: { name: Example Tracker, version: 0.1.0, kind: tracker }
  observer: { scope: seat, seat: p1 }
  detail: { combat: outcome, states: turnStart, entities: true }
  truncated: true
  truncationReason: One turn, not a whole game.
game: { id: demo, mode: solo, seatCount: 8, patch: 34.2.0 }
players:
  - { id: p1, seat: 0, kind: human, startingHealth: 30 }
  - { id: p2, seat: 4, kind: unknown }
history:
  - seq: 1
    kind: event
    event: turn_start
    turn: 2
    phase: recruit
    data: { turn: 2, gold: 4, tier: 1 }
  # ...the rest is the same data as the JSON above.
```

## The bigger examples

| File | What it shows |
|---|---|
| [`examples/full-game.json`](examples/full-game.json) | A whole game, start to finish. Four seats, low starting health, so it fits on one page. Combat recorded at the `outcome` level. |
| [`examples/single-turn.json`](examples/single-turn.json) | One recruit phase from one player's point of view: a purchase, a fusion, a triple and the card it discovers, a roll, a freeze. Combat recorded at the `boards` level. |
| [`examples/combat-events.json`](examples/combat-events.json) | One fight, blow by blow: a start-of-combat hero power, a trade, Reborn, a Divine Shield swallowing a poisoned hit. Combat recorded at the `events` level. |

Each has a `.yaml` twin holding the same data.

## How to validate a file

The repository ships a validator. It runs the JSON Schema and the handful of
rules a schema cannot express, such as "every choice points at an offer that came
earlier".

```bash
npm install
npm run validate                          # checks everything in examples/
node tools/validate.mjs mygame.bgh.json   # checks your own file
node tools/validate.mjs mygame.bgh.yaml   # YAML works the same way
```

To validate with your own tools, point any JSON Schema draft 2020-12 validator at
`schema/bg-history-1.0.schema.json`. For YAML, load it into data first, then
validate the data.

Python, for example:

```python
import json, yaml, jsonschema

schema = json.load(open("schema/bg-history-1.0.schema.json"))
doc = yaml.safe_load(open("mygame.bgh.yaml"))
jsonschema.Draft202012Validator(schema).validate(doc)
```

## Versioning in one paragraph

`version` is `MAJOR.MINOR`. A rise in MINOR only adds things, and a reader
written for `1.0` must keep reading `1.1` files by ignoring what it does not
recognise. A rise in MAJOR may change or remove things, and a reader may refuse a
major version it does not know. Put your own data in `ext`, keyed by your own
name. Vocabulary you invent must start with `x-`. The full rules are in
[section 13 of the specification](SPECIFICATION.md#13-versioning-and-unknown-fields).

## Contributing

Open an issue. The useful ones name a real recording the format cannot express,
or a field whose meaning is ambiguous enough that two producers would fill it in
differently.

## Licence

[MIT](LICENSE).

Hearthstone and Battlegrounds are trademarks of Blizzard Entertainment, Inc. This
project is an independent, unofficial data format. It is not affiliated with,
endorsed by, or sponsored by Blizzard Entertainment. The card ids in the examples
are public identifiers from Blizzard's own card data; no game assets are
redistributed here.

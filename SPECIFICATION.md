# Battlegrounds History Data, version 1.1

This document defines a file format for recording a game of Hearthstone
Battlegrounds: the states the game passed through, and the actions that moved
between them.

The normative artifact is the JSON Schema in
[`schema/bg-history-1.1.schema.json`](schema/bg-history-1.1.schema.json). Where
this prose and the schema disagree, the schema wins for anything the schema can
express, and this prose wins for everything else. The section
[Rules a schema cannot check](#14-rules-a-schema-cannot-check) lists the rules that
only prose can state.

The words **must**, **must not**, **should** and **may** carry their usual
meaning in a specification. **Must** is a requirement. **Should** is a strong
recommendation you may ignore with a reason. **May** is permission.

---

## Contents

- [1. What a file describes](#1-what-a-file-describes)
- [2. The game, for programmers who have not played it](#2-the-game-for-programmers-who-have-not-played-it)
- [3. Serialization](#3-serialization)
- [4. Document structure](#4-document-structure)
- [5. Identity](#5-identity)
- [6. Top-level objects](#6-top-level-objects)
- [7. The history](#7-the-history)
- [8. Action vocabulary](#8-action-vocabulary)
- [9. Event vocabulary](#9-event-vocabulary)
- [10. Recording the recruit phase](#10-recording-the-recruit-phase)
- [11. Recording combat](#11-recording-combat)
- [12. Partial observation](#12-partial-observation)
- [12A. Recording from a game client](#12a-recording-from-a-game-client)
- [13. Versioning and unknown fields](#13-versioning-and-unknown-fields)
- [14. Rules a schema cannot check](#14-rules-a-schema-cannot-check)
- [15. Converting from other formats](#15-converting-from-other-formats)
- [16. Deliberately out of scope](#16-deliberately-out-of-scope)
- [17. Conformance](#17-conformance)

---

## 1. What a file describes

One file describes **one game**: a single lobby, from the moment players pick a
hero to the moment one player is left standing.

A file may describe less than a whole game. A file that stops early sets
`recording.truncated` to `true` and says why. Nothing else changes: a fragment
has the same shape as a complete game.

A file does **not** describe more than one game. To store many games, store many
files, or wrap them in a container of your own.

### What the format is for

- Watching a game back, move by move.
- Training and auditing programs that play the game.
- Comparing what two different engines do with the same board.
- Studying decisions: what a player bought, what they passed over, what the
  shop offered.

### What the format is not

It is not a rules engine. The file records what happened. It does not say why
the rules produced it, and reading it does not tell you how to replay it. Two
programs that disagree about a rule can both write valid files.

---

## 2. The game, for programmers who have not played it

You do not need to have played Battlegrounds to use this format, but you do need
these words. Every one of them appears in the format.

Eight players share a lobby. Each player has a **hero**, which is a character
with a starting **health** total and often a **hero power**, a special ability
printed on the hero. A player is knocked out when health reaches 0. The last
player standing takes **placement** 1.

A turn has two halves.

The **recruit phase** is the shopping half. You alone can see it. You have:

- **Gold**, which you are given at the start of every turn and which does not
  carry over.
- A **shop**, also called the tavern: a row of cards you can buy.
- A **hand**: cards you own but have not put down.
- A **board**: up to seven **minions**, the creatures that fight for you. Their
  left-to-right order matters.
- A **tavern tier**, from 1 to 6: how strong the cards in your shop can be. You
  pay gold to raise it.

In the recruit phase you buy cards, play them onto your board, sell them back,
drag them into a different order, pay 1 gold to **roll** the shop (replace every
card in it), and **freeze** the shop to hold it over to your next turn.

Some special things can happen there:

- Collect three copies of the same card and they merge into one **golden** copy,
  which is stronger. Players call that a **triple**. Making one also lets you
  **Discover** a card: the game shows you three cards and you keep one.
- **Magnetize** means fusing a Mech card from your hand onto a Mech already on
  your board. The two become one minion with the stats and abilities of both.
- A **trinket** is a keepsake you pick up mid-game that changes your whole game.
- A **Dark Gift** is a season-specific enchantment stuck onto a minion.
- **Activate** is an ability on some minions that you pay gold to use during the
  recruit phase.

The **combat phase** is the fight. The game pairs you with another player and
your two boards fight without either of you touching them. Minions take turns
attacking, left to right; the defender is picked at random from the enemy board
unless something forces the choice. Both minions in a clash deal their damage at
the same instant, so a minion that dies still hits back. Whoever is left
standing deals damage to the loser's hero.

Combat abilities you will meet in the vocabulary:

- **Taunt**: this minion must be attacked before anything else on its board.
- **Divine Shield**: the first hit is swallowed whole; the minion loses no
  health, and the shield is gone.
- **Poisonous** and **Venomous**: any damage this minion deals destroys what it
  hits. Venomous is spent after one kill.
- **Windfury**: this minion attacks twice per turn. **Mega-Windfury**: four
  times.
- **Reborn**: when this minion dies, it comes back once, as the printed card at
  1 health, with none of the buffs it had.
- **Stealth**: this minion cannot be chosen as a target until it attacks.
- **Battlecry**: an effect that fires when you play the card from your hand.
- **Deathrattle**: an effect that fires when the minion dies.
- **Start of Combat**: an effect that fires before the first attack.

When an odd number of players is left, one of them fights a **ghost**: a copy of
a knocked-out player's board. The player who once owned that board is already
out, so nothing that happens to the copy affects them.

A **minion type**, which players call a **tribe**, is a family such as Beast or
Mech. Each lobby uses only some of them, so the same card pool is different from
game to game.

---

## 3. Serialization

JSON is the reference serialization. A file **must** be valid UTF-8.

YAML 1.2 is a permitted serialization. YAML 1.2 is a superset of JSON, so a YAML
file that loads into the same data as a valid JSON file is equally valid. Load
the YAML, then validate the result against the same schema. Every example in
[`examples/`](examples/) is published in both, generated from the same data.

Recommended file extensions: `.bgh.json` and `.bgh.yaml`. Nothing depends on
them; `format` inside the file is what identifies it.

Numbers **must** be JSON numbers, not strings. Integers **must** be exact. Dates
and times **must** use RFC 3339, for example `2026-01-14T18:22:05Z`.

`null` has one meaning in this format: **the recorder knows the field applies
here but could not learn its value.** Leaving a field out means something
different: the field does not apply, or the recorder chose not to write it. Do
not use `null` to mean zero, and do not use it as filler.

---

## 4. Document structure

The top level of a file is an object with six members. Five are required.

```json
{
  "format": "battlegrounds-history",
  "version": "1.0",
  "cardIdScheme": "hearthstone",
  "recording": { },
  "game": { },
  "players": [ ],
  "history": [ ]
}
```

| Member | Required | What it holds |
|---|---|---|
| `format` | yes | Always the string `battlegrounds-history`. |
| `version` | yes | The specification version, as `MAJOR.MINOR`. |
| `cardIdScheme` | no | Where the card ids come from. Defaults to `hearthstone`. |
| `recording` | yes | Facts about the file: who made it, what they could see. |
| `game` | yes | Facts about the lobby that hold all game. |
| `players` | yes | The seats. |
| `history` | yes | The game itself, as an ordered list. |
| `ext` | no | Your own data. See [section 13](#13-versioning-and-unknown-fields). |

Everything interesting is in `history`. The other members exist so a reader can
understand `history` without guessing.

---

## 5. Identity

A recording is useless if you cannot tell what a card is six months later. Three
naming systems do that work, and they are kept apart on purpose.

### 5.1 Card ids

`cardId` names **which card** something is. It is the only field you may match
on.

The default naming system is Blizzard's own text card id, such as `BG31_803` or
`TB_BaconShop_HP_069`. These ids survive stat changes, text rewrites and
rebalances, which is exactly what identity needs. The card's numbers change; its
id does not.

If you use a different naming system, set `cardIdScheme` at the top level to say
which, and use it consistently through the file. Mixing systems in one file is
not allowed.

`name` is for display. It is translated, it changes between patches, and two
different cards have shared a name before. **Never match on `name`.**

Heroes, hero powers, trinkets, spells and tokens are all cards, and all carry a
`cardId`. The `type` field says which sort of card it is.

A trinket carries `trinketTier`, which is `lesser` or `greater`. These are not
rarities: they are two separate slots, offered at two different points in the
game, and a player holds at most one of each. A recording that loses the
distinction cannot say which of a seat's two trinkets was picked when, so the
field is part of the format rather than something to bury in `ext`.

A golden copy is the same card with `golden: true`, not a different `cardId`.
Some data sources give golden cards their own id; if yours does, record the base
id and set `golden`.

### 5.2 Entity names

`entity` names **which physical copy** something is. Two copies of the same card
have the same `cardId` and different `entity` values.

An entity name is unique inside one file and stable for as long as the copy
exists. It lets you follow one minion from the shop, into your hand, onto the
board, through a fight, and into its death.

Entity names are optional. A recorder that cannot track copies leaves them out
and points at cards by position instead. Set `recording.detail.entities` to
`true` only if **every** card in the file carries one.

The format says nothing about how entity names look, beyond being short strings.
`e41`, `17` and `a3b9f1` are all fine.

### 5.3 Player ids

Each entry in `players` has an `id`. Everything else in the file refers to a
player by that id. `seat` is the table position and is separate, because seat
numbers are not stable across producers and some producers do not know them.

Player ids are opaque strings. Do not read meaning into them.

### 5.4 Positions

Every position in this format counts **from 0, left to right**. Board slot 0 is
the left-most minion. Shop slot 0 is the left-most shop card.

Many game clients and logs count from 1. Convert when you write the file, not
when you read it.

### 5.5 Pointing at a card: the `ref` object

Wherever an action or an event needs to name a card that is already somewhere, it
uses a `ref`:

```json
{ "player": "p1", "zone": "board", "index": 2, "entity": "e19" }
```

- `player` defaults to the acting player, so you usually leave it out.
- Give `entity`, `index`, or both. Give both when you can: `entity` is exact,
  and `index` is what a reader without entity tracking can use.
- `cardId` may be repeated here for the same reason.

---

## 6. Top-level objects

### 6.1 `recording`

| Field | Required | Notes |
|---|---|---|
| `id` | yes | A name for this recording, unique to whoever produced it. |
| `recordedAt` | no | When the file was written. |
| `recorder` | no | `name`, `version`, `kind`, `url`. |
| `observer` | yes | See [section 12](#12-partial-observation). |
| `detail` | no | How much this file contains. See below. |
| `truncated` | no | `true` if the recording stops before the game did. |
| `truncationReason` | no | Why. Write this whenever `truncated` is `true`. |
| `note` | no | A sentence for a person. |

`recorder.kind` is one of `tracker` (watches a live client), `simulator` (plays
the game itself), `replayParser` (reads a stored game), `manual` (a person typed
it in), or `other`.

### 6.2 `recording.detail`

`detail` states how much the file contains, so a reader never has to read an
absence as a fact. Without it, "no combat events" and "no combat events
recorded" look identical.

| Field | Values | Meaning |
|---|---|---|
| `combat` | `none`, `outcome`, `boards`, `events` | How much of each fight is written down. Each level includes the ones before it. |
| `states` | `none`, `turnStart`, `everyAction`, `irregular` | How often a full picture of a seat is written down. |
| `entities` | `true`, `false` | `true` only if every card carries an `entity`. |

`combat` levels:

- `none` — fights are not recorded at all.
- `outcome` — a `combat_end` event per fight: who won, and the health each side
  lost.
- `boards` — also a `combat_start` event carrying both boards as they were sent
  into the fight.
- `events` — also the blow by blow: every attack, every point of damage, every
  death.

### 6.3 `game`

Facts about the lobby that hold all game: `id`, `mode` (`solo` or `duos`),
`startedAt`, `patch`, `season`, `seatCount`, `minionTypes` (the minion types this
lobby drew), `anomaly` (a lobby-wide rule change, in seasons that have them), and
`rng`.

`minionTypes` is `null` when the recorder does not know which types the lobby
drew, and an array when it does. An empty array means the lobby drew none, which
is not something the game does today.

`rng` is for a producer that generated its own randomness: `seed` and `engine`.
**A seed never replaces the recorded history.** See
[rationale](RATIONALE.md#combat-events-not-combat-inputs).

### 6.4 `players`

One entry per seat: `id`, `seat`, `name`, `kind` (`human`, `bot`, `unknown`),
`agent` (the program that made the decisions, if one did), `teammate` (Duos
only), `hero`, `heroPower`, `startingHealth`, `startingArmor`, and `result`.

`name` is optional so you can publish a recording without publishing who played
it.

**Record `hero` for every seat you can.** Heroes decide how a game is played, so
a recording that names only the recorded seat's hero is much less useful than one
that names all eight. A one-seat recording can fill in every seat's hero from the
leaderboard, even though it learns almost nothing else about the other seven. If
you know the hero's name but not its card id, write
`{"name": "Illidan Stormrage", "type": "hero", "unknown": true}` rather than
inventing an id.

Armor is starting health that most effects treat as health. This format keeps it
separate wherever the game does, and lets you fold it into `health` if your
source does not separate them.

### 6.5 `players[].result`

How the seat finished: `placement` (1 is the winner), `health`, `armor`, `tier`,
`combatsPlayed`, `combatsWon`, `heroPowerUses`, `eliminatedOnTurn`, `trinkets`,
`board`.

Every field is optional. A recording made from one seat may never learn another
seat's final board, and must not invent it.

`health` is floored at 0. A killing blow can take a player below zero; the format
records 0 and puts the size of the blow on the fight that dealt it.

---

## 7. The history

`history` is an array of **entries**, in the order things happened. There are
three kinds.

| `kind` | Meaning | Discriminator |
|---|---|---|
| `action` | A player chose to do this. | `action` names the verb. |
| `event` | The game did this. | `event` names the verb. |
| `state` | A full picture of one seat at one moment. | none |

The split between `action` and `event` matters. An `action` is a decision, and
decisions are what you study, learn from and second-guess. An `event` is a
consequence. Recording a shop refresh as an action of the player and the cards it
produced as an event of the game keeps the two apart, so a reader can filter for
"everything a player chose" and get exactly that.

### 7.1 The entry envelope

Every entry carries the same envelope:

| Field | Required | Notes |
|---|---|---|
| `seq` | yes | Position in the history. It **must** rise strictly across the file. Gaps are allowed and mean nothing was recorded there. |
| `kind` | yes | `action`, `event` or `state`. |
| `turn` | no | Which turn this belongs to. |
| `phase` | no | `setup`, `recruit`, `combat` or `end`. |
| `at` | no | Wall-clock time. |
| `elapsedMs` | no | Milliseconds since the game started. |
| `actor` | see below | The player this concerns. |
| `note` | no | A sentence for a person. **Never authoritative**: a reader must never parse it. |
| `ext` | no | Your own data. |
| `data` | see below | The payload, for `action` and `event`. |

`seq` numbers exist so a file can be merged, split, referred to and diffed
without depending on array position. Use consecutive integers unless you have a
reason not to.

`actor` is required on an `action` and on a `state`. On an `event` it is optional
and means "the seat this happened to". An event with no `actor` concerns the
whole lobby — a turn starting, a game ending, a fight between two seats.

### 7.2 `action` entries

```json
{
  "seq": 13,
  "kind": "action",
  "actor": "p1",
  "action": "buy",
  "turn": 8,
  "phase": "recruit",
  "data": {
    "from": { "zone": "shop", "index": 1, "entity": "s2" },
    "to": { "zone": "hand", "index": 0 },
    "cost": 3,
    "gold": 5
  }
}
```

Extra fields on an action entry:

- `accepted` — defaults to `true`. Set it to `false` to record an action the game
  refused. A refused action changed nothing; do not apply it when you replay.
- `error` — why it was refused.
- `decision` — optional notes on how the choice was made. See
  [section 7.5](#75-the-decision-object).

`data.gold`, `data.health`, `data.armor` and `data.tier` are the values **after**
the action. They are optional and redundant with a replay of the history, and
they are worth writing anyway: they let a reader check its own replay against
what the game actually did.

### 7.3 `event` entries

Same envelope, with `event` instead of `action`. See
[section 9](#9-event-vocabulary) for the verbs.

### 7.4 `state` entries

A `state` entry is a full picture of one seat at one moment. It is a checkpoint,
not the source of truth: a reader can also rebuild the state by replaying
entries.

```json
{
  "seq": 12,
  "kind": "state",
  "actor": "p1",
  "turn": 8,
  "phase": "recruit",
  "reason": "turnStart",
  "player": { "health": 27, "tier": 3, "gold": 8, "upgradeCost": 6 },
  "zones": {
    "board": { "cards": [ ], "capacity": 7 },
    "hand": { "cards": [ ], "capacity": 10 },
    "shop": { "cards": [ ], "frozen": false },
    "trinkets": { "cards": [ ] }
  },
  "standings": [ ],
  "nextOpponent": { "player": "p4" }
}
```

- `reason` says why the picture was taken: `turnStart`, `turnEnd`,
  `afterAction`, `beforeCombat`, `afterCombat`, `checkpoint`.
- `zones` is keyed by zone name. A zone you leave out was not written down; a
  zone with an empty `cards` array was written down and was empty.
- `standings` is the leaderboard: one row per seat, which is all a player
  normally sees of the others. Each row may carry `hero`, because the
  leaderboard shows every seat's hero portrait. That is the one fact about an
  opponent a one-seat recording reliably learns, so record it.
- `nextOpponent` is the seat this player will fight next, which the game
  announces before the recruit phase ends.

Write `state` entries as often as you like, and say how often in
`recording.detail.states`. Writing one at every turn start is a good default.
Writing one after every action costs a lot of bytes and buys a reader the ability
to jump to any moment without replaying.

### 7.5 The `decision` object

Optional, and only on an `action`. It records how the choice was made, which is
what a program can say and a person cannot.

| Field | Meaning |
|---|---|
| `legalOptions` | How many actions were legal at this moment. |
| `probability` | How likely the chooser was to pick this one, from 0 to 1. |
| `value` | The chooser's own score for the position. |
| `considered` | Other actions that were on the table, each with its own probability or value. |
| `refusals` | Actions a rule removed before the chooser ever saw them. |

`refusals` is there for auditing. When you suspect a program has found a hole in
your rules, the moves it was **not** allowed to make matter as much as the one it
took. Each refusal carries a `reason`, a `count`, and one `example` written out
in full.

---

## 8. Action vocabulary

An action verb this specification does not list **must** start with `x-`. That
rule is enforced by the schema, so a reader can always tell a private verb from a
standard one.

| Verb | Required in `data` | What it means |
|---|---|---|
| `choose_hero` | `offer`, `option` | Pick a hero at the start of the game. |
| `buy` | `from` | Buy a card out of the shop. Works for a minion, a spell or anything else the shop sells. |
| `sell` | `target` | Sell a card back for gold. |
| `play` | `from` | Put a card from your hand into the game. Covers playing a minion onto the board and casting a spell. Use `to` for the board position and `targets` for anything the card asks you to pick. |
| `move` | `from`, `to` | Drag a minion into a different board position. |
| `roll` | — | Pay to replace every card in the shop. Also called refreshing. |
| `freeze` | — | Hold the shop over to your next turn. |
| `unfreeze` | — | Release a held shop. |
| `upgrade` | — | Pay gold to raise your tavern tier. Put the new tier in `data.tier`. |
| `hero_power` | — | Use the ability printed on your hero. Use `targets` if it asks you to pick. |
| `activate` | `target` | Pay to use an ability printed on a minion already on your board. |
| `magnetize` | `from`, `to` | Fuse a Mech from your hand onto a Mech on your board. |
| `choose` | `offer`, `option` | Take one option from a choice the game offered. |
| `open_offer` | `offer` | Pay to reveal a choice before seeing what is in it. |
| `pass_card` | `from`, `to` | Duos only: send a card to your teammate. `to.player` names them. |
| `end_turn` | — | End the recruit phase. |
| `concede` | — | Give up. |
| `emote` | — | Say something. Recorded because it is a real action, not because it matters. |

Fields available in `data` for any verb: `from`, `to`, `target`, `targets`,
`cost`, `costResource` (`gold`, `health` or `none`), `gold`, `health`, `armor`,
`tier`, `offer`, `option`, `mode`, `emote`, `ext`.

### Notes on particular verbs

**`play` covers spells.** A tavern spell is a card you buy into your hand and
play later, exactly like a minion. Rather than a second verb, `play` handles
both, and the card's `type` tells you which it was. A reader that wants only
spells filters on the card, not on the verb.

**`buy` covers everything the shop sells.** Same reasoning.

**`choose` covers every "pick one of these" moment**, and there are many: the
card you Discover from making a triple, the trinket you take, the Dark Gift you
pick, a hero at the start of the game, a quest, a reward. Every one of them is
the game offering options and a player taking one. The `offer` event that
preceded the choice carries an `offerType` saying which flavour it was, so
nothing is lost by having one verb.

**`open_offer` is for a choice you must pay for before you can see it.** Some
games make you commit gold to a Discover, and only then show you the three cards.
That is two decisions and this format records two: `open_offer` to commit, then
`choose` once the `offer` event has been amended or a second `offer` event with
the revealed options has been written.

**`upgrade` is what players call levelling.** The cost falls by 1 for every turn
you do not buy it, so record `cost` if you know it.

---

## 9. Event vocabulary

An event verb this specification does not list **must** start with `x-`.

### 9.1 Game and turn structure

| Verb | Notes |
|---|---|
| `game_start` | The lobby begins. |
| `turn_start` | With no `actor`, the whole lobby starts a turn. With an `actor`, that seat does. |
| `turn_end` | Use `data.reason` when a turn ended for a reason other than the player ending it — for example a time limit. |
| `pairing_announced` | Who this seat fights next. `data.opponent`, and `data.ghostOf` if it is a copy of a knocked-out player's board. |
| `player_eliminated` | `data.player`, `data.placement`. |
| `game_end` | `data.placements` and, if you have them, `data.standings`. |

### 9.2 Recruit phase

| Verb | Notes |
|---|---|
| `shop_dealt` | A new set of shop cards. `data.cards` and `data.reason` (`turnStart`, `roll`, `freeze`). |
| `offer` | The game put choices in front of a player. Requires `data.id` and `data.options`. See below. |
| `gold_granted` | Gold arriving, usually at the start of a turn. |
| `income_changed` | The amount granted each turn changed. |
| `card_gained` | A card appeared in a zone from somewhere other than a purchase. |
| `card_removed` | A card left the game. |
| `card_moved` | A card changed zone or position without a player action. |
| `triple_created` | Three copies became one golden copy. `data.card` is the golden copy. |
| `stat_change` | Attack or health changed. |
| `keyword_change` | Abilities gained or lost. `data.gained`, `data.lost`. |
| `enchantment_added` / `enchantment_removed` | Something attached or came off. |
| `health_change` | A player's health or armor changed outside a fight. |

**The `offer` event** is the one to understand:

```json
{
  "seq": 11, "kind": "event", "event": "offer", "actor": "p1",
  "data": {
    "id": "offer-t8-triple",
    "offerType": "tripleReward",
    "mandatory": true,
    "options": [
      { "cards": [ { "cardId": "BG35_341" } ] },
      { "cards": [ { "cardId": "BG27_005" } ] },
      { "cards": [ { "cardId": "BG36_351" } ] }
    ]
  }
}
```

- `id` is what a later `choose` action points back at. It **must** be unique in
  the file.
- `offerType` is `hero`, `discover`, `tripleReward`, `trinket`,
  `lesserTrinket`, `greaterTrinket`, `darkGift`, `quest`, `reward`, `buddy` or
  `other`. Prefer `lesserTrinket` or `greaterTrinket` over the plain `trinket`
  when you know which slot the offer was for; `trinket` stays for a recorder
  that cannot tell.
- `options` are counted from 0. An option holds one or more cards, because some
  choices offer a bundle.
- `hidden: true` means the player had to commit before seeing the options. List
  the options as unknown cards, or write a second `offer` event with the real
  ones once they were revealed.
- `mandatory: true` means the player could not decline.

### 9.3 Combat

Every combat event carries `data.combat`, an id that ties one fight together.

| Verb | Required in `data` | Notes |
|---|---|---|
| `combat_start` | `sides` | The two boards as they were sent in. `firstAttacker` names the side that swings first. |
| `attack` | `attacker`, `defender` | One swing. `swing` says which swing it is, for a minion that attacks more than once. |
| `damage` | `target`, `amount` | Damage landing. `source` is what dealt it. `absorbed: true` means a Divine Shield swallowed it. `poisonous: true` means it destroys whatever it touches. `lethal: true` means the target died from it. |
| `divine_shield_lost` | — | A shield was spent. |
| `death` | `target` | A minion died. `killer` names what killed it. |
| `summon` | `card` | A minion appeared. `position` is where. `source` is what summoned it. |
| `reborn` | `card` | A minion came back. Same shape as `summon`. |
| `trigger` | — | A card's text fired. `source` names the card, `times` says how many times, `reason` says what kind of trigger it was. |
| `heal` | `target`, `amount` | Health restored. |
| `stat_change` | `target` | Attack or health changed mid-fight. `permanent: true` means the change survives the fight and travels back to the shop with the minion. |
| `keyword_change` | `target` | Abilities gained or lost mid-fight. |
| `hero_damage` | `player`, `amount` | The fight's damage to a player's hero. |
| `combat_end` | `sides` | Who won, and what each side lost. |

`combat_start` and `combat_end` each carry `sides`, an array of one or two
`combatSide` objects:

| Field | Notes |
|---|---|
| `player` | Whose side it is. Absent when the side is a ghost with no living owner. |
| `ghostOf` | Set when this side is a copy of a knocked-out player's board. |
| `hero`, `heroPower`, `tier` | Facts the fight depends on. |
| `board` | The board sent in, left to right. Required at detail level `boards` and above. |
| `hand` | Only for the few cards whose combat text reads the owner's hand. |
| `healthBefore`, `healthAfter`, `armorBefore`, `armorAfter`, `damageTaken` | `null` when the recorder could not see it. |
| `survivors` | Refs to the minions still standing. |
| `eliminated` | `true` if this fight knocked the player out. |

---

## 10. Recording the recruit phase

The pattern is: the game deals, the player acts, the game reacts.

```
event  turn_start          (lobby, or per seat)
event  gold_granted        (optional)
event  shop_dealt          reason: turnStart
state  ...                 reason: turnStart
action buy                 from shop slot 1
action play                from hand to board slot 2
event  triple_created      three copies became one golden copy
event  offer               offerType: tripleReward, three options
action choose              option 0
event  card_gained         the chosen card, into the hand
action play                from hand to board slot 3
action roll                cost 1
event  shop_dealt          reason: roll
action freeze
event  pairing_announced   opponent p4
action end_turn
```

Rules for this phase:

1. **Write the action, then its consequences.** `buy` comes before the
   `card_gained` it caused, and `play` comes before the battlecry it fired. A
   reader replaying the file applies them in order.
2. **A choice always has an `offer` before it.** Write the `offer` event even if
   the player took the only option. Without it, a reader cannot tell what was
   passed over, which is half of what a recording is for.
3. **`shop_dealt` after every refresh**, whether the player paid for it or the
   turn brought it.
4. **Record what the player did, not what you think they meant.** If a player
   dragged a minion four times, record four `move` actions.
5. **Do not invent an `end_turn`.** If the phase ended because a time limit ran
   out, write a `turn_end` event with `data.reason`, not an action the player
   never took.

---

## 11. Recording combat

Combat is the half of the game a player watches rather than plays. There is
nothing to record but consequences, and how much of it you record is a choice.

### 11.1 Pick a level and declare it

Set `recording.detail.combat` and stick to it for the whole file.

- **`outcome`** — one `combat_end` per fight. Cheap, and enough to reconstruct
  every health total in the game. This is what a recorder that reads a
  scoreboard can honestly produce.
- **`boards`** — also a `combat_start` carrying both boards. Enough to feed the
  fight to another engine and compare.
- **`events`** — also every attack, damage, death and trigger. Enough to see
  exactly what happened, including the parts where two engines would disagree.

### 11.2 Ordering rules for `events`

These rules are the difference between a stream a reader can replay and a pile of
facts.

1. **Order is resolution order.** The order the entries appear in is the order
   the engine settled them. When two minions die at the same instant, the file
   records the order they were processed, because that order changes what happens
   next.
2. **An `attack` comes before the `damage` it causes**, and both come before any
   `death`.
3. **Both sides of a clash deal their damage before either death is settled.** A
   dying defender still hits back, so write both `damage` entries, then the
   `death` entries.
4. **Absorbed damage is still damage.** Write the `damage` entry with
   `absorbed: true` and `amount` set to what was aimed at the target, then the
   `divine_shield_lost` entry. Do not silently drop the hit: whether a shield was
   spent is a fact a reader needs.
5. **Poison rides on damage.** A hit from a Poisonous or Venomous minion is one
   `damage` entry with `poisonous: true`. If a Divine Shield absorbs it, the
   poison is absorbed with it, so `absorbed` and `poisonous` are both `true` and
   `lethal` is `false`.
6. **A minion that dies and returns is two bodies.** Write the `death`, then the
   `reborn` with a new `entity` name. If it dies again, write a second `death`.
7. **`hero_damage` is optional** when `combat_end` already carries
   `damageTaken`. Write it when you know the moment the damage landed.

### 11.3 The boards are part of the fight

At detail level `boards` and above, `combat_start.sides[].board` **must** carry
the board as it was sent in, including every buff already added to the printed
numbers. Do not expect a reader to reconstruct it from the recruit phase: buffs
land from many places, and the point of writing the board down is that it does
not depend on the reader agreeing with you about any of them.

---

## 12. Partial observation

A simulator can see all eight seats. A program watching a live client can see
one. Both produce valid files, and a reader has to be able to tell which it is
holding without inspecting the contents.

### 12.1 Declare the scope

`recording.observer.scope` is `lobby` or `seat`.

- **`lobby`** means the recorder could see every seat. Only a program that runs
  the whole game, or reads a complete server-side replay, may claim this.
- **`seat`** means the recorder saw one player's point of view.
  `recording.observer.seat` names that player and is required.

Claiming `lobby` when you could not see every seat is the one thing that makes a
file lie. When in doubt, claim `seat`.

### 12.2 What "scope" does and does not promise

`scope: lobby` promises the recorder **could** see everything. It does not
promise the file **contains** everything. A lobby recorder may write down one
seat's shopping and every seat's fights, and that is what
`recording.detail.states` is for.

`scope: seat` is a promise in the other direction: every fact in the file was
visible to that seat. A file with `scope: seat` **must not** contain a fact that
seat could not have known — another player's shop, their hand, their gold, or a
fight they were not in.

### 12.3 What one seat can see

In a normal game, a player sees:

- Their own board, hand, shop, gold, health and tavern tier, in full.
- The leaderboard: every living player's hero, health and tavern tier. Record it
  as `standings`.
- The board of any opponent they fight, during that fight.
- The board they will face next, in the seconds before the fight, if the client
  shows it.

They do not see another player's shop, hand, gold, or the fights they are not in.

### 12.4 Three ways to say "I do not know"

| Situation | How to write it |
|---|---|
| The field applies and the value is unknown. | `null`. Example: `healthAfter: null` for an opponent whose health you have not seen since the fight. |
| A card is there and you cannot see which card it is. | A card object with `unknown: true` and no `cardId`. |
| You know how many cards are in a zone but not which. | `zone.unknownCount`, alongside whatever `cards` you did see. |

Leaving a field out is not one of these. It means "does not apply here", or "I
chose not to write it".

A reader must treat a missing zone and an empty zone as different. `{"cards": []}`
says the zone was empty. Omitting the zone says nothing at all.

---

## 12A. Recording from a game client

A program watching a live client is the hardest case this format has to serve,
and the one it was shaped around. This section says what such a recording looks
like, because getting it wrong is easy and the mistakes are quiet.

A client's log tells you what **your** player did and what the game **showed
you**. It does not tell you what anybody else decided. That asymmetry is the
whole design problem, and the format answers it in one sentence:

> **A seat recording carries `action` entries for one player, and carries other
> players only through `event` and `state` entries.**

You never write an `action` for a seat you were not sitting in. You did not see
the decision; you saw at most its result, and a result is an event.

### What such a file contains

| About your seat | About the other seven |
|---|---|
| Every action: buys, sells, plays, moves, rolls, freezes, upgrades, hero power, choices. | No actions at all. |
| Full `state` entries: board, hand, shop, gold, tier, health. | `standings` rows: hero, health, tavern tier, alive. |
| Every offer you were shown, and which option you took. | The board they fielded, and only in a fight you took part in. |
| Your hero and hero power. | Their hero, from the leaderboard. Usually not their hero power. |

### Rules for this case

1. **Set `observer.scope` to `seat`** and name the seat. Never claim `lobby`.
2. **Write no action you did not watch a player take.** If the opponent's board
   changed between two fights, that is a `state` observation, not a sequence of
   inferred purchases.
3. **Record every seat's hero.** The leaderboard gives you all eight, and it is
   the one durable fact you get about an opponent.
4. **Record an opponent's board only for fights you were in**, and put it in the
   `combat_start` entry for that fight, where it is timestamped by the fight
   rather than floating free.
5. **Do not fill gaps by simulating.** If you did not see an opponent's health
   change, write `null`. A guess that looks like an observation is worse than a
   hole.
6. **Expect to be missing the start.** A client log often begins mid-game, after
   a restart or a rotation. Set `recording.truncated` and say so.

### Two seat recordings of one lobby

Two players in the same lobby produce two files. They merge: give both the same
`game.id`, keep the player ids consistent, concatenate the histories and sort on
a shared clock. The result is a wider recording, and it is still not a `lobby`
recording — the union of two points of view is two points of view. Keep the scope
at `seat` and name whichever seat the merged file is centred on, or split the
difference by publishing both files.

## 13. Versioning and unknown fields

### 13.1 The version field

`version` is `MAJOR.MINOR`.

- A rise in **MINOR** only adds: new optional fields, new vocabulary words, new
  enum values. A reader written for `1.0` **must** read a `1.1` file, ignoring
  what it does not recognise.
- A rise in **MAJOR** may remove or redefine things. A reader **may** refuse a
  major version it does not know.

A reader **must** check `format` and the major part of `version` before doing
anything else. A reader **must not** refuse a file because the minor version is
higher than it knows.

### 13.2 Unknown fields

**Readers must ignore members they do not recognise.** This is the rule that
makes minor versions work, and the schema is written to allow it: no object in
this format closes itself to extra members.

**Readers must ignore vocabulary words they do not recognise**, rather than
failing. An unknown action verb is an action you cannot interpret; skip it, and
say so if your output has somewhere to say it. Do not abort.

### 13.3 Your own data: `ext`

Every object in this format accepts an `ext` member. Put anything the format does
not define there, keyed by your own name:

```json
"ext": { "acme": { "modelCheckpoint": 4120, "workerId": 7 } }
```

Keying by producer means two producers can extend the same file without
colliding. **No reader may depend on `ext`**, and no future version of this
format will define a key inside it.

Vocabulary you invent goes in the vocabulary, not in `ext`, and **must** start
with `x-`: `x-bobsSpecialAction`. The schema enforces the prefix, so a reader can
always tell a private word from a standard one at a glance.

### 13.4 Stability promise

Within a major version:

- A field that exists keeps its meaning.
- A required field stays required. An optional field stays optional.
- A vocabulary word keeps its meaning, and words are only added.
- `format` never changes.

---

## 14. Rules a schema cannot check

A JSON Schema validates one value at a time. These rules look across the whole
document, and a producer **must** satisfy them. The validator in
[`tools/validate.mjs`](tools/validate.mjs) checks all of them.

1. `seq` rises strictly across `history`.
2. Every `actor` names a player in `players`.
3. `recording.observer.seat` names a player in `players`, when scope is `seat`.
4. Every `teammate` names a player in `players`.
5. A `choose`, `choose_hero` or `open_offer` action points at an `offer` event
   that appears **earlier** in the history.
6. `data.option` is within range of that offer's `options`.
7. An `entity` name refers to one physical copy for its whole life. Reusing an
   entity name for a different copy is an error.
8. `data.id` on an `offer` event is unique in the file.
9. A file with `observer.scope` of `seat` carries `action` entries for that seat
   and for no other. You cannot have watched another player decide.
10. A file with `observer.scope` of `seat` contains no fact that seat could not
    have observed. No program can check this last one; it is on the producer's
    honour, and it is the reason the field exists.

---

## 15. Converting from other formats

### 15.1 From a simulator's own action list

Many Battlegrounds simulators name their recruit-phase actions with a short list
of verbs. Here is one such list, and how it maps onto this format. Nothing is
lost: every collapse is reversible from the card or the offer the entry points
at.

| Their verb | This format | Note |
|---|---|---|
| `buy` | `buy` | |
| `sell` | `sell` | |
| `play` | `play` | |
| `roll` | `roll` | |
| `level` | `upgrade` | Put the new tier in `data.tier`. |
| `end` | `end_turn` | |
| `take_gift` | `choose` | The `offer` carries `offerType: darkGift`. |
| `cast_spell` | `play` | The played card's `type` is `spell`. |
| `buy_spell` | `buy` | The bought card's `type` is `spell`. |
| `freeze` | `freeze` | |
| `unfreeze` | `unfreeze` | |
| `hero_power` | `hero_power` | |
| `move` | `move` | |
| `take_trinket` | `choose` | `offerType: trinket`. |
| `magnetize` | `magnetize` | |
| `reveal_gift` | `open_offer` | The paid press that reveals a hidden choice. |
| `activate` | `activate` | |
| `take_offer` | `choose` | `offerType: discover` or `tripleReward`. |

### 15.2 From a nested turn-and-action recording

A common shape nests actions inside turns and repeats the whole board after every
action:

```
game → turns[] → { board, shop, hand, actions[] → { kind, slot, board, shop, hand } }
```

To convert:

1. Each turn becomes a `turn_start` event, then a `state` entry carrying that
   turn's zones, then the actions.
2. Each action becomes an `action` entry. The board, shop and hand it carried
   become a `state` entry with `reason: afterAction` placed **after** it. Set
   `recording.detail.states` to `everyAction`.
3. The turn's closing board becomes a `state` entry with `reason: beforeCombat`.
4. The turn's combat becomes `combat_start` and `combat_end` events.
5. Anything the source format carried about the recording process rather than the
   game — a training round number, a benchmark score, a file index — goes in
   `ext`. See [section 16](#16-deliberately-out-of-scope).

[MIGRATION.md](MIGRATION.md) does this conversion field by field for one such
shape, and names the fields that deliberately have no home.

---

## 16. Deliberately out of scope

These things have no home in this format, on purpose. If you need them, put them
in `ext`.

**Anything about the process that produced the file, rather than about the
game.** A training round number, a benchmark score, an index within a batch, the
name of the file's parent job. These describe your pipeline. Two producers will
never agree on them, and a reader has no use for them. `recording.recorder` and
`recording.id` cover everything a reader legitimately needs to know about where a
file came from.

**Values you can compute from the file.** The sum of a board's attack and health,
a win rate, an average placement. Storing a derived number invites it to
disagree with what it was derived from. Compute it when you need it.

**Display text as data.** Sentences like "bought Flighty Scout from shop slot 1"
belong in `note`, which is explicitly not authoritative. Never parse a `note`.

**Card databases.** This format names cards by id and stops. It does not carry
printed stats, rules text or art, beyond the optional `name` and `text` fields
kept for display. Look the id up in a card database.

**Rules.** The format does not say what a card does, what an action costs, or
which minion a Taunt forces you to attack. It records what happened.

**Anything that identifies a player against their wishes.** `name` is optional.
Account ids, regions and ratings have no field at all.

---

## 17. Conformance

A **conforming file** validates against `schema/bg-history-1.0.schema.json` and
satisfies [section 14](#14-rules-a-schema-cannot-check).

A **conforming producer** writes conforming files, and never claims a scope or a
detail level higher than what it could actually see.

A **conforming reader**:

- checks `format` and the major version before anything else,
- ignores members and vocabulary words it does not recognise,
- treats a missing field, a `null` field and an empty array as three different
  things,
- never matches a card on `name`,
- never parses a `note`.

---

Hearthstone and Battlegrounds are trademarks of Blizzard Entertainment, Inc.
This project is not affiliated with, endorsed by, or sponsored by Blizzard
Entertainment.

# Migrating a nested turn-and-action recording

Several bot-training projects record games in a shape like this:

```
file
└── games[]
    ├── seed, bot, hero, heroPowerId, tribes, placement, health, ...
    ├── turns[]
    │   ├── turn, tier, hp, gold, levelCost, frozen
    │   ├── board[], shop[], hand[], spell, gifts[], trinkets[], opponent
    │   ├── actions[]  ─── each repeating board[], shop[], hand[] afterwards
    │   ├── finalBoard[]
    │   └── combat
    └── standings[]
```

It nests actions inside turns, and repeats the whole seat after every action. It
is a reasonable shape for one program's own viewer, and it does not travel: it
holds one seat's point of view without saying so, it describes a fight by its
before-and-after health rather than by what happened, and it mixes facts about
the game with facts about the job that produced the file.

This page maps every field of that shape onto this format, and names the four
fields that deliberately have no home.

## The overall conversion

1. One `games[]` entry becomes one file.
2. Set `recording.observer.scope` to `seat`, and `recording.observer.seat` to the
   seat marked `you` in the standings. The shape only ever holds one seat's
   shopping, so claiming `lobby` would be a lie.
3. Set `recording.detail` to `{ "combat": "boards", "states": "everyAction",
   "entities": false }`. Entity names are `false` because the source has none.
4. Each turn becomes a `turn_start` event, a `shop_dealt` event, a `state` entry
   with `reason: turnStart`, then the turn's actions.
5. Each action becomes an `action` entry. The board, shop and hand it carried
   become a `state` entry with `reason: afterAction` placed **after** it.
6. The turn's `finalBoard` becomes a `state` entry with `reason: beforeCombat`.
7. The turn's `combat` becomes a `combat_start` event and a `combat_end` event.

## Field by field

### The file and the game

| Source field | Home |
|---|---|
| `recorded` | `recording.recordedAt` |
| `seed` | `game.rng.seed` |
| `bot` | `players[].agent.name` for the recorded seat |
| `tribes` | `game.minionTypes`, lowercased, with `MECHANICAL` becoming `mech` |
| `truncated` | `recording.truncated`. Add a `truncationReason`. |
| `hero` | `players[].hero`. See [names without ids](#names-without-ids). |
| `heroPowerId` | `players[].heroPower.cardId` |
| `heroPower` | `players[].heroPower.name` |
| `heroPowerText` | `players[].heroPower.text` |
| `heroPowerUses` | `players[].result.heroPowerUses` |
| `ownedTrinkets[].id` | `players[].result.trinkets[].cardId` |
| `ownedTrinkets[].name` | `players[].result.trinkets[].name` |
| `ownedTrinkets[].text` | `players[].result.trinkets[].text` |
| `ownedTrinkets[].greater` | `players[].result.trinkets[].ext`, or a `keywords` entry of `x-greaterTrinket`. Lesser and greater are one season's split, so the format does not name them. |
| `placement` | `players[].result.placement` |
| `health` | `players[].result.health` |
| `wins` | `players[].result.combatsWon` |
| `combats` | `players[].result.combatsPlayed` |
| `finalTier` | `players[].result.tier` |

### Standings

| Source field | Home |
|---|---|
| `standings[].seat` | `players[].seat` |
| `standings[].name` | `players[].agent.name` |
| `standings[].hero` | `players[].hero.name` |
| `standings[].hp` | `players[].result.health`, and `standing.health` inside a `state` entry |
| `standings[].tier` | `players[].result.tier`, and `standing.tier` inside a `state` entry |
| `standings[].placement` | `players[].result.placement` |
| `standings[].you` | `recording.observer.seat` |

### A turn

| Source field | Home |
|---|---|
| `turn` | `turn` on every entry of that turn |
| `tier` | `state.player.tier` |
| `hp` | `state.player.health` at `reason: turnStart` |
| `hpEnd` | `state.player.health` at `reason: turnEnd`, or `data.health` on the `end_turn` action |
| `gold` | `state.player.gold` at `reason: turnStart` |
| `goldLeft` | `data.gold` on the `end_turn` action |
| `levelCost` | `state.player.upgradeCost` |
| `frozen` | `state.zones.shop.frozen` |
| `board`, `hand` | `state.zones.board.cards`, `state.zones.hand.cards` |
| `shop` | `shop_dealt` event `data.cards`, and `state.zones.shop.cards` |
| `spell.name` | a card in the shop with `type: "spell"`. See [names without ids](#names-without-ids). |
| `spell.cost` | that card's `cost` |
| `spell.costHealth` | that card's `costsHealth` |
| `spell.castable` | that card's `playable` |
| `gifts[]` | an `offer` event with `offerType: "darkGift"` |
| `gifts[].minion` | an option's `cards[0]` |
| `gifts[].gift` | that card's `enchantments[0].name` |
| `trinkets[]` | an `offer` event with `offerType: "trinket"` |
| `trinkets[].name` | an option's `cards[0].name` |
| `trinkets[].cost` | that card's `cost` |
| `trinkets[].greater` | as for `ownedTrinkets[].greater` above |
| `opponent.seat` | `pairing_announced` `data.opponent`, and `state.nextOpponent.player` |
| `opponent.hero` | `players[].hero.name` for that seat |
| `opponent.hp`, `opponent.tier` | `state.standings[]` for that seat |
| `opponent.ghost` | `data.ghostOf` on `pairing_announced` |
| `stopped` | a `turn_end` event with `data.reason`. Do not write an `end_turn` action the player did not take. |
| `finalBoard` | `state.zones.board.cards` at `reason: beforeCombat`, and `combat_start.sides[].board` |

### An action

| Source field | Home |
|---|---|
| `kind` | `action`, using the mapping in [specification section 15.1](SPECIFICATION.md#151-from-a-simulators-own-action-list) |
| `slot` | `data.from.index` for `buy` and `play`; `data.target.index` for `sell` and `activate`; dropped for verbs that address nothing |
| `target` | `data.target.index`, or an entry in `data.targets` |
| `destination` | `data.to.index` |
| `card` | `data.from` or `data.target` gains a `cardId`; the full card appears in the preceding `state` entry, where it was still in its zone |
| `text` | `note` on the entry. It is display only and no reader may parse it. |
| `gold` | `data.gold` |
| `tier` | `data.tier` |
| `hp` | `data.health` |
| `board`, `shop`, `hand` | a `state` entry with `reason: afterAction`, placed after the action |
| `choices` | `decision.legalOptions` |
| `chance` | `decision.probability` |
| `refused[].reason` | `decision.refusals[].reason` |
| `refused[].count` | `decision.refusals[].count` |
| `refused[].example` | `decision.refusals[].example` |

### A card

| Source field | Home |
|---|---|
| `cardId` | `cardId` |
| `name` | `name` |
| `attack`, `health`, `tier` | the same fields |
| `golden` | `golden` |
| `gift` | `enchantments[0].name` |
| `giftText` | `enchantments[0].text` |
| `magnets[]` | `attached[]`, one card each |
| `keywords[]` | `keywords[]`, spelled out: `ds` becomes `divineShield`, `wf` becomes `windfury`, `poison` becomes `poisonous`, `taunt` stays `taunt` |
| `tribes[]` | `minionTypes[]`, lowercased, `MECHANICAL` becoming `mech` and `ALL` becoming `all` |

### A combat

The source describes a fight by what it cost, because its engine reported nothing
else. That maps onto detail level `boards`: a `combat_start` event carrying the
boards, and a `combat_end` event carrying the damage.

| Source field | Home |
|---|---|
| `opponent` | `combat_start.sides[1].player`, and `ghostOf` when it was a copy |
| `board` | `combat_start.sides[1].board` — the board the opponent fielded |
| `hpBefore` | `sides[0].healthBefore` |
| `hpAfter` | `sides[0].healthAfter` |
| `damage` | `sides[0].damageTaken` |
| `dealt` | `sides[1].damageTaken`. The source uses `-1` for "cannot be known"; write `null`. |
| `result` | `combat_end.data.winner`: the recorded seat for `won`, the opponent for `lost`, `null` for `draw`. For `unknown`, leave `winner` out entirely — `null` means a draw, which is a different claim. |
| `died` | `sides[0].eliminated` |
| `oppDied` | `sides[1].eliminated` |
| `text` | `note` on the `combat_end` entry |

## Names without ids

The source names a hero, a tavern spell, a Dark Gift and a trinket by display
name only, with no card id. This format identifies cards by id, so a straight
copy would produce a card with no identity.

Write it honestly:

```json
{ "name": "Enchanted Lasso", "type": "spell", "cost": 2, "unknown": true }
```

`unknown: true` says the recorder could not tell which card this is, and `name`
carries what it did have. A reader that knows better can look the name up; a
reader that does not will not mistake the name for identity.

Two of these are recoverable without a lookup. A hero is identified by its hero
power's card id, which the source does carry, so a converter with a hero table
can fill in the hero's own `cardId`. An owned trinket carries an `id` already.

## The four fields with no home

These are deliberately out of scope. Put them in `ext` if you want to keep them.

**`round`** — which training round produced the file. This describes the job that
made the recording, not the game. No two producers would agree what a round is,
and no reader of a game has a use for it. `recording.id` and
`recording.recorder` already carry everything a reader legitimately needs about
where a file came from.

**`vsCurve`** — a benchmark score for the round. Same reason, and worse: it is a
number about a *collection* of games sitting inside a file that describes *one*
game. It belongs to whatever produced the collection.

**`game`** — the index of this game within a batch. A position in a container
this format does not have. If the batch matters, name the batch in `ext`.

**`boardPower`** — the sum of the final board's attack and health. It is computed
from the board, and the board is in the file. Storing it invites the two to
disagree, and there is no right answer when they do. Add it up when you need it.

Two more source fields survive but change status. `text` on an action and on a
combat becomes `note`, which the specification defines as display only and
forbids readers from parsing. `damage`, `dealt` and `result` on a combat are
kept, even though all three can be derived from health totals, because a one-seat
recorder often knows the result without knowing both health totals.

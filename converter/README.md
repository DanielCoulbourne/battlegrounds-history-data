# bgh-convert

Turns a Hearthstone client log into Battlegrounds History Data files, one per
game. Written in Go, no dependencies outside the standard library.

It is also a Go library for writing the format, whatever your source:
[`bgh`](bgh/) has the types, a builder that owns `seq` numbering, and a `Validate`
that checks the rules a JSON Schema cannot.

```bash
go build ./cmd/bgh-convert
./bgh-convert Power.log
./bgh-convert -o out/ Power.log Power_old.log
cat Power.log | ./bgh-convert -stdout
```

## Turn the log on first

Hearthstone writes nothing useful unless you ask it to. Create or edit
`%LOCALAPPDATA%\Blizzard\Hearthstone\log.config` and give it:

```ini
[Power]
LogLevel=1
FilePrinting=true
ConsolePrinting=false
ScreenPrinting=false
Verbose=true
```

The client reads that file **once, at startup**, so restart Hearthstone. Logs
then appear under `Logs\Hearthstone_YYYY_MM_DD_HH_MM_SS\Power.log`. When the
client exits it renames that file to `Power_old.log`, so a finished session has
no `Power.log` at all — pass both if you are unsure.

Without `Verbose=true` the log has none of the detail this reads, and the
converter says so rather than writing an empty file.

## What comes out

One file per game, named after the game's own seed, which is the only stable
name a Battlegrounds match has:

```
hs-987654321-p3.bgh.json  27 entries, 4 seats
```

Every file is a **one-seat recording**. That is not a shortcut. A client log
records what your player did and what the game showed you, and nothing at all
about what anybody else decided — so `observer.scope` is `seat`, action entries
exist for your seat alone, and the other seven seats are reached only through
events and state entries. Claiming otherwise would make the file lie.

### What the log gives you, seat by seat

| About your seat | About the other seven |
|---|---|
| Every action: buys, sells, plays, rolls, freezes, tavern upgrades, hero power, and every choice you were offered and took. | No actions at all. There are none in the log. |
| Board, hand, shop, gold, tavern tier and health, at every turn start. | The leaderboard: hero, health, armor, tavern tier, placement — live, all game. |
| Both trinket slots, and which slot each pick filled. | Their trinkets, as Blizzard's numeric card ids. |
| Your hero and hero power. | Their hero. Usually not their hero power. |
| Every fight you were in, blow by blow. | The board they fielded, and only in a fight you took part in. |

### Combat is recorded in full

The log carries every swing, both damage numbers, and every death, so the
converter writes `detail.combat: "events"` and means it: `combat_start` with
both boards as they were sent in, then `attack`, `damage`, `divine_shield_lost`,
`death` and `hero_damage` in the order the game settled them, then `combat_end`.

The opponent's board is snapshotted at the first attack, because the game
forgets it the moment the fight ends.

## Display names are left out

A battletag identifies a real person, and a recording gets shared more readily
than a log does. Pass `-names` if you want them.

## What it does not do yet

Stated plainly, because a converter that quietly drops things is worse than one
that says what it drops:

- **Minion repositioning.** The log records it; this does not read it yet, so
  the board order in a `state` entry can lag a drag.
- **Battlecry and spell targets** beyond the single target the block names.
- **Dark Gifts**, which arrive as a choice like any other but are not yet told
  apart from a plain Discover.
- **Enchantments on a minion.** Stats include every buff, but the buffs are not
  itemised, so `card.enchantments` is empty.
- **Duos.** Team play, and passing a card to a teammate, are unhandled.
- **The lobby's minion types.** No tag states them; the client's per-card subset
  flags look like they do and do not.
- **Player ratings.** Not in any log file. Established trackers read them from
  process memory.

## Layout

| Package | What it does |
|---|---|
| `hslog` | Reads the log into events. Knows nothing about Battlegrounds. |
| `hslog.Table` | Accumulates events into an entity table. Knows tag names and nothing else. |
| `convert` | Reads Battlegrounds meaning out of the table and writes history entries. |
| `bgh` | The format: types, builder, validator. Useful on its own. |
| `cmd/bgh-convert` | The command. |

The split is deliberate. Nearly everything that goes wrong with this log goes
wrong in the first two layers, and they are testable without a game.

## Tests

```bash
go test ./...
```

`testdata/synthetic.log` is hand-written from the log grammar. It is not a
captured session: a real `Power.log` carries the battletags of eight people, and
this repository is public.

`examples/from-client-log.json` in the repository root is the golden output of
that fixture, so the published example of a converted log is real converter
output rather than something written by hand to look like it. Regenerate it
with:

```bash
go test ./convert -update
```

The parser tests in `hslog` are each one of the ways this log defeats a naive
reader — the duplicated event stream, continuation lines that look like tags,
tags whose names are bare numbers, entity names containing square brackets,
empty card ids, and gold that is addressed to a battletag rather than to an
entity.

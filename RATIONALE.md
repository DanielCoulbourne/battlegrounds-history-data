# Design rationale

Why the format has the shape it has, and what was rejected on the way. Each
section states the decision, the reasoning, the alternatives, and the cost of the
choice — because every one of these has a cost.

---

## Contents

- [A flat history, not nested turns](#a-flat-history-not-nested-turns)
- [Three kinds of entry: action, event, state](#three-kinds-of-entry-action-event-state)
- [Combat events, not combat inputs](#combat-events-not-combat-inputs)
- [Declared observation scope, and three ways to say "I do not know"](#declared-observation-scope-and-three-ways-to-say-i-do-not-know)
- [Two names for a card: what it is, and which copy](#two-names-for-a-card-what-it-is-and-which-copy)
- [One `offer` and one `choose`, not a verb per kind of choice](#one-offer-and-one-choose-not-a-verb-per-kind-of-choice)
- [Detail is declared, never inferred from absence](#detail-is-declared-never-inferred-from-absence)
- [State snapshots are optional](#state-snapshots-are-optional)
- [A `data` object instead of flat fields](#a-data-object-instead-of-flat-fields)
- [Open to unknown members, closed to unmarked vocabulary](#open-to-unknown-members-closed-to-unmarked-vocabulary)
- [JSON Schema is normative; YAML is permitted](#json-schema-is-normative-yaml-is-permitted)
- [No card database, no rules](#no-card-database-no-rules)
- [Decision metadata belongs in the format](#decision-metadata-belongs-in-the-format)
- [Derived numbers are left out](#derived-numbers-are-left-out)
- [Naming the real game, not one simulator](#naming-the-real-game-not-one-simulator)

---

## A flat history, not nested turns

**Decision.** One ordered array, `history`, holding every entry in the game.
Turns are a field on an entry, not a level of nesting.

**Why.** The obvious shape is `game → turns[] → actions[]`, and it is the shape
most existing recordings use. It reads well and it breaks in four places:

- **Things happen between turns.** A fight belongs to no recruit phase. A player
  is knocked out between one turn and the next. In a nested format these end up
  bolted onto the turn before or the turn after, and every reader has to know
  which.
- **Combat has no natural nesting under a seat.** A fight has two sides. Nesting
  it under one player's turn makes the other player's half a second-class
  citizen.
- **A live recorder writes entries as they happen.** A flat array can be appended
  to and flushed. A nested one cannot be closed until the turn is over, which
  means a crashed recorder loses the turn in progress.
- **Merging and splitting.** Two seat recordings of the same lobby merge by
  sorting on `seq`. Nested recordings do not merge at all.

**Rejected: nesting.** Cost of rejecting it: a reader that wants "turn 8" has to
filter rather than index. That is one line of code, and `turn` is on every entry
so the filter is cheap.

**Rejected: newline-delimited JSON, one entry per line.** It is the natural
streaming format and it would suit a live recorder well. But it puts the header
in the first line and gives a validator nothing to validate as a whole, and it
makes YAML awkward. A file is a document. If you want a stream, stream the
entries and assemble the document at the end.

---

## Three kinds of entry: action, event, state

**Decision.** Every entry is an `action` (a player chose it), an `event` (the
game did it), or a `state` (a picture of a seat).

**Why.** The whole point of a history is to separate decisions from consequences.
A player rolling the shop is a decision. The cards that came up are not. Anyone
studying play — training a program, reviewing their own game, auditing a bot —
wants the decisions, and wants them without a filter that has to know which verbs
happen to be decisions.

Keeping the two apart also settles a question that otherwise recurs at every
verb: is a triple an action or a consequence? It is a consequence. The player
played a third copy; the merge is what the game did about it. So `play` is an
action and `triple_created` is an event, and the same test settles every other
case.

**Rejected: one flat entry type with a `type` field.** Simpler schema, and the
distinction gets encoded in a list of verbs a reader has to memorise. The
distinction is real, so it belongs in the data.

**Rejected: actions only, with state rebuilt from them.** Attractive, and wrong:
it forces every reader to implement the game's rules to learn what a shop
contained. The `shop_dealt` event is not derivable from any action.

---

## Combat events, not combat inputs

This is the decision most likely to be argued with, so it gets the most space.

**Decision.** A recording carries the **resolved sequence of what happened** in a
fight, not just the two boards and a random seed. The boards are carried too, at
detail level `boards` and above. Seeds are permitted, in `game.rng`, and are
never a substitute for either.

**Why not just the inputs?** Because a fight is not a function of its inputs
alone. It is a function of its inputs **and the engine that ran it**, and the
engine is not in the file.

Battlegrounds combat is deterministic given a seed, and that fact tempts you into
storing the seed and re-running the fight when someone wants to watch it. Three
things go wrong:

1. **The engine changes.** The real game is patched. A simulator gets a card's
   rules right that it had wrong. Re-running the seed then produces a fight that
   never happened, and — this is the dangerous part — it produces it *silently*.
   The file looks fine. It just describes a different game.
2. **Nobody else has your engine.** A recording that needs a specific program to
   be understood is not a data format. It is a save file.
3. **The interesting cases are exactly the ones where engines disagree.** Two
   minions die at the same instant: which deathrattle fires first? A frozen board
   cannot know the order the minions were played, so every engine picks an
   approximation, and the approximations differ. Comparing two engines is a real
   use for this format, and it is impossible if the file stores an input both
   engines will interpret their own way.

**Why carry the boards as well, then?** Because the event stream alone tells you
what happened without telling you what it happened to. The boards are the
starting position, and reading a fight without them is like reading a chess game
without the opening position. They also make the file useful for the
engine-comparison case: feed the boards to another engine and see whether it
produces your events.

**Why permit seeds at all?** They cost nothing and they help the producer. If you
generated the game yourself, your seed lets you reproduce your own run against
your own engine at your own version, which is a genuinely useful thing to be able
to do. What the format refuses is treating the seed as the record. That is why
`game.rng` sits under `game` rather than anywhere near the combat entries: it is
a fact about how the lobby was made, not a description of a fight.

**Rejected: inputs plus seed only.** See above.

**Rejected: events only, no boards.** Loses the starting position, and makes
every reader reconstruct it by replaying the recruit phase, which requires
implementing buffs.

**Rejected: requiring events.** Most recorders cannot produce them. A deck
tracker watching a live client sees the health bar move; it does not see every
point of damage. Requiring events would have made the format unusable by exactly
the tools most likely to adopt it. Hence the `detail.combat` ladder, where a
producer says how far up it can climb.

**The cost.** Event-level files are large. A fight with 30 events costs more
bytes than a seed. The format's answer is that you choose your level and declare
it, and that bytes are cheaper than a recording nobody can trust.

---

## Declared observation scope, and three ways to say "I do not know"

**Decision.** `recording.observer.scope` is `lobby` or `seat`, and it is
required. A `seat` recording must not contain a fact that seat could not have
observed.

**Why.** Both kinds of recording exist and both are legitimate. A simulator
produces the first; a deck tracker produces the second. What is not legitimate is
a reader having to guess which one it holds.

The failure mode this prevents is subtle and expensive. Suppose you train a
program on a pile of recordings, and some of them are omniscient. The program
learns to use facts — another player's shop, their gold — that it will not have
when it plays for real. Nothing in the data warns you. The format makes the
claim explicit, and puts the burden on the producer, where it belongs.

**Why not derive the scope from the contents?** Because absence is not evidence.
A lobby recording that happens to write down only one seat's shopping looks
identical to a seat recording. The producer knows which it is; the reader cannot
work it out.

**Three ways to say "I do not know", and why three.** They mean different things
and collapsing them loses information:

| Situation | Written as |
|---|---|
| The field applies here; I could not see the value. | `null` |
| A card is in this zone; I cannot see which card. | `{"unknown": true}` |
| I know how many cards are here; I cannot see any of them. | `unknownCount: 4` |

A fourth case — leaving the field out — means "does not apply" or "I chose not to
write it". Four states, four spellings. The temptation is to use `null` for all
of them, and the cost is that a reader can no longer tell "the opponent has an
empty hand" from "I cannot see the opponent's hand", which is the difference
between a safe inference and a wrong one.

**The cost.** Producers have to think about which one they mean. That is the
point.

---

## Two names for a card: what it is, and which copy

**Decision.** `cardId` says which card. `entity` says which physical copy.
They are separate fields with separate lifetimes.

**Why.** These are genuinely different questions and formats that conflate them
lose one of the answers.

`cardId` has to survive a patch. Blizzard's text card ids do: `BG31_803` stays
`BG31_803` when the card's attack changes, when its text is rewritten, and when
it moves tavern tier. Names do not — they are translated, and two cards have
shared a name before. Numeric database ids do not either, across data sources.
Hence: match on `cardId`, never on `name`, and `name` is documented as
display-only.

`entity` answers a question `cardId` cannot: *is this the same copy I saw a
moment ago?* Three copies of the same card in one shop share a `cardId`. Without
entity names, a recording of "sold the second one" is a recording about a
position, and positions shift under you. With them, you can follow one minion
from the shop into your hand onto the board through a fight and into its death.

**Why is `entity` optional?** Because many producers cannot supply it. A tracker
reading a client's log gets entity ids for free; a simulator has to thread them
through; a person typing a file by hand will not bother. Making them required
would have excluded all three. `recording.detail.entities` lets a producer say it
supplied them everywhere, so a reader can rely on it.

**Rejected: making entity names structured** — encoding the owner or the zone
into the string. It makes the name lie the moment the card moves.

**Rejected: golden copies as separate card ids.** Some data sources give the
golden version of a card its own id. This format records the base id plus
`golden: true`, because "three of these merged" is a relationship between the
same card and itself, and a separate id hides it.

---

## One `offer` and one `choose`, not a verb per kind of choice

**Decision.** Every "the game shows you options, you take one" moment is an
`offer` event followed by a `choose` action. What kind of choice it was lives in
`offerType`.

**Why.** Battlegrounds keeps inventing these. Over its history it has had: the
hero pick at the start, Discover from a triple, quest picks, reward picks,
trinket picks, buddy picks, and this season's Dark Gift picks. A verb per kind
means the vocabulary grows every season, and every reader written before a season
breaks on files from after it.

They are also the same thing. Options are presented, one is taken, the rest are
discarded. That structure is what a reader wants to work with — "show me every
choice this player made and what they passed over" is one query, not seven.

**Cost, stated honestly.** A reader that wants only trinket picks has to filter
on `offerType` instead of on the verb, and it has to look at the earlier `offer`
event to do it. That is a real cost. It is smaller than the cost of a vocabulary
that grows without bound.

**The same reasoning collapses two more pairs.** `buy` covers buying a spell as
well as a minion, and `play` covers casting a spell as well as playing a minion,
because in the real game a tavern spell **is** a card you buy into your hand and
play later. Splitting the verb would encode a distinction the game does not make.
The card's `type` field tells a reader which it was.

**What was kept separate, and why.** `open_offer` stays its own verb, even though
it looks like part of `choose`. It is a separate decision: some choices make you
commit gold before you see what is in them, so pressing the button and picking an
option are two things a player decides, seconds apart, with different
information. A format that merged them would lose the fact that the player paid
blind.

---

## Detail is declared, never inferred from absence

**Decision.** `recording.detail` states how much the file contains: how much of
each fight, how often a state snapshot appears, whether entity names are
everywhere.

**Why.** Without it, every absence is ambiguous. No damage events might mean a
fight where nothing was damaged, or a recorder that does not produce damage
events. No `state` entries might mean a very short game or a recorder that never
writes them. A reader that guesses will guess wrong, and the wrong guess is
usually the optimistic one.

This is the same principle as the observation scope, applied to completeness
rather than visibility: the producer knows, the reader cannot work it out, so the
producer says.

**Rejected: a strict profile system**, where a file declares conformance to a
named profile such as "complete" or "minimal". Tidier, and it forces producers
into buckets that will not fit them — a recorder might have full combat events
and no state snapshots at all. Three independent knobs describe more real
recorders than any list of profiles would.

---

## State snapshots are optional

**Decision.** A reader can rebuild the state by replaying the entries. `state`
entries are checkpoints, and a producer chooses how many to write.

**Why.** The two obvious positions are both wrong.

*Snapshot everything.* One existing format repeats the whole board, shop and hand
after every single action. It makes the file trivially readable — jump anywhere,
read the state — and it makes the file enormous, mostly with copies of things
that did not change. It also creates a consistency problem: when a snapshot
disagrees with the actions before it, which is right?

*Snapshot nothing.* Smallest possible file, and it forces every reader to
implement the game's rules to answer "what was on the board?". A viewer that just
wants to draw a board should not have to know what magnetizing does.

**The middle.** Snapshots are allowed anywhere, encouraged at turn boundaries,
and declared in `detail.states`. Where a snapshot and the entries disagree, the
snapshot wins for that moment — it is a direct observation, and the entries may
be incomplete. A producer that finds itself writing disagreeing snapshots has a
bug.

---

## A `data` object instead of flat fields

**Decision.** An action's and an event's payload lives in `data`, not alongside
`seq` and `kind`.

**Why.** The envelope is small, fixed and stable: `seq`, `kind`, `turn`, `phase`,
`at`, `elapsedMs`, `actor`, `note`, `ext`, and the verb. The payload is large and
grows with the vocabulary. Keeping them apart means a new payload field can never
collide with an envelope field, however the vocabulary grows.

It also makes the schema honest. "Required when the verb is `buy`" is one
`if`/`then` block pointing at `data`, and a reader can see at a glance which
fields are envelope and which are payload.

**Cost.** One more level of nesting in every entry, and slightly noisier
examples. Weighed against a collision the format could never take back, it is
worth it.

---

## Open to unknown members, closed to unmarked vocabulary

**Decision.** No object in the format refuses extra members. But an action verb,
event verb, keyword or minion type that this specification does not list **must**
start with `x-`, and the schema enforces the prefix.

**Why the asymmetry.** They fail differently.

An unknown **member** is safe to ignore. A reader that skips a field it does not
know still understands the entry. So the format allows extra members everywhere,
and requires readers to ignore them, which is what makes additive minor versions
possible at all.

An unknown **verb** is not safe to ignore in the same way. It is a whole entry
whose meaning you do not have. What a reader needs is to be able to tell "this is
a standard word from a version newer than me, and I should look it up" from "this
is somebody's private extension, and I never will". The `x-` prefix draws that
line in the data, where a validator can enforce it, instead of in a registry
nobody maintains.

**Rejected: a registry of vendor prefixes.** Nobody maintains those.

**Rejected: closing objects with `additionalProperties: false`.** It catches
typos, and it also breaks every forward-compatible reader the moment a minor
version adds a field. Catching typos is worth less than that.

---

## JSON Schema is normative; YAML is permitted

**Decision.** The JSON Schema is the specification's teeth. YAML 1.2 is an
equally valid way to write the same data, and validates against the same schema
after loading.

**Why.** JSON is what programs exchange, and JSON Schema is the only widely
implemented way to state a data format precisely enough to argue with. YAML is
what people write by hand and read in a diff, and YAML 1.2 is a strict superset
of JSON, so permitting it costs nothing at all: load, then validate.

**Cost.** Two files to keep in step for every example. The repository generates
the YAML from the JSON so they cannot drift.

---

## No card database, no rules

**Decision.** The format names cards by id and stops. It does not carry printed
stats, rules text, art, costs by tier, or what any card does.

**Why.** A card database is a different artifact with a different lifetime. It
changes every patch; a recording never changes again. Embedding one would make
every recording a stale copy of it, and would make the format's version bump
whenever Blizzard printed a card.

The optional `name` and `text` fields on a card are for display, so a viewer can
render a file without a database at hand. They are explicitly not identity, and a
reader must never match on them.

**The same reasoning excludes rules.** The format does not say what an action
costs, which minion a Taunt forces you to attack, or how a triple resolves. It
records that a player paid 3 gold and that three copies became one. A format that
encoded the rules would have to be right about them, and it would have to be
right about them in every past and future season.

---

## Decision metadata belongs in the format

**Decision.** An action may carry a `decision` object: how many options were
legal, how likely the chooser was to pick this one, what it considered, and what
a rule refused to let it consider.

**Why.** This looks like it belongs in `ext` — it is only meaningful for a
program, and a human player has none of it. It is in the format proper for two
reasons.

First, it is not specific to any engine. "How many legal moves were there" and
"what probability did the chooser assign" are questions any decision-making
program can answer, and they are the questions anyone learning from or auditing a
recording actually asks.

Second, `refusals` earns its place on its own. When you suspect a program has
found a hole in your rules, the moves it was **not allowed** to make tell you
more than the one it took. That is an auditing feature, and burying it in
vendor-specific data would mean every producer invents its own shape for it and
no shared tool can read any of them.

**Cost.** A field most files will never use. It is optional and costs nothing
when absent.

---

## Derived numbers are left out

**Decision.** No field holds a number you can compute from the rest of the file.
No board strength totals, no win rates, no average placements.

**Why.** A stored derived value is a second source of truth, and second sources
of truth drift. When the total disagrees with the board it was totalled from, a
reader has to decide which to believe, and there is no right answer.

**Cost.** A consumer that wants board strength computes it. That is a sum.

**The escape hatch.** If you cache derived numbers for your own pipeline, put
them in `ext`, where the format promises nothing about them and no reader will
mistake them for facts.

---

## Naming the real game, not one simulator

**Decision.** The vocabulary describes Hearthstone Battlegrounds as it is
played, not any one program's subset of it.

**Why.** Simulators simplify. A given one might model five heroes out of ninety,
give every hero a passive power because it has not implemented an active one,
reduce a season's enchantments to flat stat changes, and never implement Duos at
all. A format built from that program's action list would inherit all of it, and
every gap would become a thing the format could not say.

So the vocabulary includes things no single simulator may implement:
`hero_power` with targets, `pass_card` for Duos, `data.mode` for a Choose One
card, quests and rewards as offer types, `anomaly` on the game object, armor as
its own number, Mega-Windfury alongside Windfury. A simulator that does not have these simply
never writes them, and its files are still valid.

The cost is a vocabulary larger than any one producer needs. That is the right
direction to be wrong in: a word nobody writes is free, and a missing word is a
recording nobody can make.

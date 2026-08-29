# Schema

The newest schema in this directory is the normative artifact of this project.
Where the prose in [SPECIFICATION.md](../SPECIFICATION.md) and the schema
disagree about something the schema can express, the schema wins.

| File | Status |
|---|---|
| `bg-history-1.1.schema.json` | **Current.** Use this one. |
| `bg-history-1.0.schema.json` | Frozen. Kept so files that pinned it keep validating. |

Both are written for **JSON Schema draft 2020-12**. Any validator that supports
that draft can use them.

## Identifier

```
https://raw.githubusercontent.com/DanielCoulbourne/battlegrounds-history-data/main/schema/bg-history-1.1.schema.json
```

That URL is the schema's `$id` and resolves to the file. It tracks `main`, so
within a minor version it always serves the newest fixes.

If you need a snapshot that can never move under you, pin a Git tag:

```
https://raw.githubusercontent.com/DanielCoulbourne/battlegrounds-history-data/v1.1.0/schema/bg-history-1.1.schema.json
```

## Version numbering

Each released schema keeps its own file and stays here forever. A minor release
adds a file; it does not edit the one before it. So `bg-history-1.0.schema.json`
is exactly what was published as 1.0, and always will be.

That costs a little duplication and buys something worth more: a file written
against an older schema can still be validated against the schema it was written
against, years later, without anyone having to trust a changelog.

**Additive really means additive.** The repository's own check proves it: every
example is declared `version: "1.1"` and validates against **both** schemas. If a
1.1 addition ever broke a 1.0 file, that check would fail.

```bash
node ../tools/validate.mjs ../examples/*.json
node ../tools/validate.mjs --schema bg-history-1.0.schema.json ../examples/*.json
```

## What the schema does and does not check

**It checks** the shape of every object, the type of every value, the vocabulary
of every enumerated field, and which parts of `data` each verb requires. It also
enforces the rule that a vocabulary word this specification does not list must
start with `x-`.

**It cannot check** anything that needs to look across the whole document:
whether `seq` rises, whether an `actor` names a real player, whether a `choose`
action points at an offer that came earlier. Those rules are listed in
[section 14 of the specification](../SPECIFICATION.md#14-rules-a-schema-cannot-check)
and checked by [`tools/validate.mjs`](../tools/validate.mjs).

## No object refuses extra members

That is deliberate, not an oversight. It is what lets a reader written for 1.0
read a 1.1 file. See
[the rationale](../RATIONALE.md#open-to-unknown-members-closed-to-unmarked-vocabulary).

## Using it

```bash
node ../tools/validate.mjs ../examples/full-game.json
```

Or with any other draft 2020-12 validator, for example `check-jsonschema`:

```bash
check-jsonschema --schemafile bg-history-1.1.schema.json ../examples/full-game.json
```

## How 1.1 was built

`tools/make-1.1.mjs` derives the 1.1 schema from the frozen 1.0 file by applying
named patches. It is kept in the repository so the difference between two minor
versions is a short, readable script rather than a diff of two large JSON files.

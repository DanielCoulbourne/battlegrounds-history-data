# Schema

`bg-history-1.0.schema.json` is the normative artifact of this project. Where the
prose in [SPECIFICATION.md](../SPECIFICATION.md) and this schema disagree about
something the schema can express, the schema wins.

It is written for **JSON Schema draft 2020-12**. Any validator that supports that
draft can use it.

## Identifier

```
https://raw.githubusercontent.com/DanielCoulbourne/battlegrounds-history-data/main/schema/bg-history-1.0.schema.json
```

That URL is the schema's `$id` and resolves to the file. It tracks `main`, so it
always serves the newest 1.x schema. Additions in a 1.x release are only ever
additive, so a file that validated before still validates.

If you need a snapshot that can never move under you, pin a Git tag:

```
https://raw.githubusercontent.com/DanielCoulbourne/battlegrounds-history-data/v1.0.0/schema/bg-history-1.0.schema.json
```

## Version numbering

The schema file's name carries `MAJOR.MINOR` of the specification, matching the
`version` field inside a document. Version 2 of the specification will ship as
`bg-history-2.0.schema.json` alongside this one, not in place of it, so old files
stay validatable forever.

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
check-jsonschema --schemafile bg-history-1.0.schema.json ../examples/full-game.json
```

#!/usr/bin/env node
// Validates Battlegrounds History Data files against the normative JSON Schema.
//
// Usage:
//   node tools/validate.mjs examples/*.json examples/*.yaml
//   node tools/validate.mjs --schema schema/bg-history-1.0.schema.json myfile.yaml
//
// JSON and YAML are the same data, so both go through the same schema.
// The script also runs the rules a JSON Schema cannot express, listed in
// SPECIFICATION.md under "Rules a schema cannot check".

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import Ajv2020 from "ajv/dist/2020.js";
import addFormats from "ajv-formats";
import * as YAML from "js-yaml";

const args = process.argv.slice(2);
let schemaPath = "schema/bg-history-1.1.schema.json";
const patterns = [];
for (let i = 0; i < args.length; i++) {
  if (args[i] === "--schema") schemaPath = args[++i];
  else patterns.push(args[i]);
}

// Expand simple globs ourselves. A shell does it on Linux and macOS and does not
// on Windows, and a validator that only runs on some machines is worse than no
// validator.
function expand(pattern) {
  if (!pattern.includes("*")) return [pattern];
  const dir = path.dirname(pattern);
  const base = path
    .basename(pattern)
    .replace(/[.+^${}()|[\]\\]/g, "\\$&")
    .replace(/\*/g, ".*");
  const rx = new RegExp("^" + base + "$");
  if (!fs.existsSync(dir)) return [];
  return fs.readdirSync(dir).filter((n) => rx.test(n)).sort().map((n) => path.join(dir, n));
}

const files = patterns.flatMap(expand);
if (files.length === 0) {
  console.error("usage: node tools/validate.mjs [--schema PATH] FILE...");
  process.exit(2);
}

const ajv = new Ajv2020({ allErrors: true, strict: false });
addFormats(ajv);
const validate = ajv.compile(JSON.parse(fs.readFileSync(schemaPath, "utf8")));

function load(file) {
  const text = fs.readFileSync(file, "utf8");
  return /\.ya?ml$/i.test(path.extname(file)) ? YAML.load(text) : JSON.parse(text);
}

// Checks that need to look across the document rather than at one value.
function crossChecks(doc) {
  const problems = [];
  const players = new Set((doc.players ?? []).map((p) => p.id));

  for (const p of doc.players ?? []) {
    if (p.teammate && !players.has(p.teammate)) {
      problems.push(`player ${p.id}: teammate "${p.teammate}" is not a player in this file`);
    }
  }

  const scope = doc.recording?.observer?.scope;
  const seat = doc.recording?.observer?.seat;
  if (scope === "seat" && !players.has(seat)) {
    problems.push(`observer.seat "${seat}" is not a player in this file`);
  }

  const offers = new Map();
  const entities = new Map();
  let previous = -1;
  let actors = new Set();

  for (const e of doc.history ?? []) {
    if (e.seq <= previous) problems.push(`entry seq ${e.seq} does not come after ${previous}`);
    previous = e.seq;

    if (e.actor && !players.has(e.actor)) {
      problems.push(`entry seq ${e.seq}: actor "${e.actor}" is not a player in this file`);
    }
    if (e.kind === "action") actors.add(e.actor);

    if (e.kind === "event" && e.event === "offer") {
      if (offers.has(e.data.id)) {
        problems.push(`entry seq ${e.seq}: offer id "${e.data.id}" was already used`);
      }
      offers.set(e.data.id, e.data.options.length);
    }
    if (e.kind === "action" && (e.action === "choose" || e.action === "choose_hero")) {
      const id = e.data.offer;
      if (!offers.has(id)) {
        problems.push(`entry seq ${e.seq}: chooses from offer "${id}", which no earlier entry made`);
      } else if (e.data.option >= offers.get(id)) {
        problems.push(
          `entry seq ${e.seq}: offer "${id}" has ${offers.get(id)} options, so ${e.data.option} is out of range`
        );
      }
    }
    if (e.kind === "action" && e.action === "open_offer" && !offers.has(e.data.offer)) {
      problems.push(`entry seq ${e.seq}: opens offer "${e.data.offer}", which no earlier entry made`);
    }

    // An entity name must mean one physical copy for its whole life.
    for (const card of collectCards(e)) {
      if (!card.entity || !card.cardId) continue;
      const seen = entities.get(card.entity);
      if (seen === undefined) entities.set(card.entity, card.cardId);
      else if (seen !== card.cardId) {
        problems.push(
          `entry seq ${e.seq}: entity "${card.entity}" was ${seen} and is now ${card.cardId}; ` +
            `an entity name must stay with one copy`
        );
      }
    }
  }

  // A one-seat recording carries actions for exactly one seat.
  if (scope === "seat") {
    for (const a of actors) {
      if (a !== seat) {
        problems.push(
          `a seat recording of "${seat}" carries actions for "${a}"; ` +
            `you cannot have watched another player decide`
        );
      }
    }
  }

  return problems;
}

// Every card object anywhere inside one entry.
function collectCards(node, out = []) {
  if (Array.isArray(node)) {
    for (const item of node) collectCards(item, out);
    return out;
  }
  if (node && typeof node === "object") {
    if (typeof node.entity === "string" && (node.cardId || node.unknown)) out.push(node);
    for (const key of Object.keys(node)) collectCards(node[key], out);
  }
  return out;
}

let failed = 0;
for (const file of files) {
  let doc;
  try {
    doc = load(file);
  } catch (err) {
    console.error(`FAIL ${file}\n  could not read: ${err.message}`);
    failed++;
    continue;
  }
  const ok = validate(doc);
  const extra = ok ? crossChecks(doc) : [];
  if (ok && extra.length === 0) {
    console.log(`ok   ${file}`);
    continue;
  }
  failed++;
  console.error(`FAIL ${file}`);
  for (const err of validate.errors ?? []) {
    console.error(`  ${err.instancePath || "/"} ${err.message}`);
  }
  for (const problem of extra) console.error(`  ${problem}`);
}
process.exit(failed === 0 ? 0 : 1);

#!/usr/bin/env node
// Validates Battlegrounds History Data files against the normative JSON Schema.
//
// Usage:
//   node tools/validate.mjs examples/*.json examples/*.yaml
//   node tools/validate.mjs --schema schema/bg-history-1.0.schema.json myfile.yaml
//
// JSON and YAML are the same data, so both go through the same schema.
// The script also runs the few rules a JSON Schema cannot express, listed in
// SPECIFICATION.md under "Rules a schema cannot check".

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import Ajv2020 from "ajv/dist/2020.js";
import addFormats from "ajv-formats";
import * as YAML from "js-yaml";

const args = process.argv.slice(2);
let schemaPath = "schema/bg-history-1.0.schema.json";
const files = [];
for (let i = 0; i < args.length; i++) {
  if (args[i] === "--schema") schemaPath = args[++i];
  else files.push(args[i]);
}
if (files.length === 0) {
  console.error("usage: node tools/validate.mjs [--schema PATH] FILE...");
  process.exit(2);
}

const ajv = new Ajv2020({ allErrors: true, strict: false });
addFormats(ajv);
const validate = ajv.compile(JSON.parse(fs.readFileSync(schemaPath, "utf8")));

function load(file) {
  const text = fs.readFileSync(file, "utf8");
  return path.extname(file).match(/^\.ya?ml$/i) ? YAML.load(text) : JSON.parse(text);
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

  const seat = doc.recording?.observer?.seat;
  if (doc.recording?.observer?.scope === "seat" && !players.has(seat)) {
    problems.push(`observer.seat "${seat}" is not a player in this file`);
  }

  const offers = new Map();
  let previous = -1;
  for (const e of doc.history ?? []) {
    if (e.seq <= previous) problems.push(`entry seq ${e.seq} does not come after ${previous}`);
    previous = e.seq;

    if (e.actor && !players.has(e.actor)) {
      problems.push(`entry seq ${e.seq}: actor "${e.actor}" is not a player in this file`);
    }
    if (e.kind === "event" && e.event === "offer") offers.set(e.data.id, e.data.options.length);
    if (e.kind === "action" && (e.action === "choose" || e.action === "choose_hero")) {
      const id = e.data.offer;
      if (!offers.has(id)) {
        problems.push(`entry seq ${e.seq}: chooses from offer "${id}", which no earlier entry made`);
      } else if (e.data.option >= offers.get(id)) {
        problems.push(`entry seq ${e.seq}: offer "${id}" has ${offers.get(id)} options, so ${e.data.option} is out of range`);
      }
    }
    if (e.kind === "action" && e.action === "open_offer" && !offers.has(e.data.offer)) {
      problems.push(`entry seq ${e.seq}: opens offer "${e.data.offer}", which no earlier entry made`);
    }
  }
  return problems;
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

#!/usr/bin/env node
// Regenerates every example's YAML twin from its JSON, so the two cannot drift.
//
// Run: node tools/yaml.mjs
import fs from "node:fs";
import path from "node:path";
import * as YAML from "js-yaml";

const dir = "examples";
const version = JSON.parse(
  fs.readFileSync("schema/bg-history-1.1.schema.json", "utf8")
).$id.split("/").pop();

for (const file of fs.readdirSync(dir).filter((f) => f.endsWith(".json"))) {
  const name = path.basename(file, ".json");
  const data = JSON.parse(fs.readFileSync(path.join(dir, file), "utf8"));
  fs.writeFileSync(
    path.join(dir, name + ".yaml"),
    `# ${name}.yaml — the same data as ${name}.json, in YAML.\n` +
      `# Both files validate against schema/${version}.\n` +
      YAML.dump(data, { lineWidth: 100, noRefs: true })
  );
  console.log("wrote", path.join(dir, name + ".yaml"));
}

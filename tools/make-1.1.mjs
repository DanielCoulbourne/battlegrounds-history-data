// One-shot: derive the 1.1 schema from the frozen 1.0 schema.
// Kept in the repo so the diff between two minor versions is reviewable.
import fs from "node:fs";

let s = fs.readFileSync("schema/bg-history-1.0.schema.json", "utf8");

function must(needle) {
  if (!s.includes(needle)) throw new Error("anchor missing: " + needle.slice(0, 60));
}

// --- identity -------------------------------------------------------------
const oldId = '"$id": "https://raw.githubusercontent.com/DanielCoulbourne/battlegrounds-history-data/main/schema/bg-history-1.0.schema.json",\n  "title": "Battlegrounds History Data 1.0",';
must(oldId);
s = s.replace(
  oldId,
  '"$id": "https://raw.githubusercontent.com/DanielCoulbourne/battlegrounds-history-data/main/schema/bg-history-1.1.schema.json",\n  "title": "Battlegrounds History Data 1.1",'
);

// --- 1. trinketTier on a card --------------------------------------------
const costsHealth =
  '        "costsHealth": { "type": "boolean", "description": "The cost is paid in health rather than gold." },';
must(costsHealth);
s = s.replace(
  costsHealth,
  costsHealth +
    `
        "trinketTier": {
          "description": "For a trinket only: which of the two trinket slots it fills. A lesser trinket is offered earlier in the game and a greater one later. A player holds at most one of each, and the two are chosen at different moments, so this is a property of the trinket and not a rarity.",
          "anyOf": [
            { "enum": ["lesser", "greater"] },
            { "$ref": "#/$defs/openName" }
          ]
        },`
);

// --- 2. offerType names the two trinket moments --------------------------
const offerTypes =
  '{ "enum": ["hero", "discover", "tripleReward", "trinket", "darkGift", "quest", "reward", "buddy", "other"] },';
must(offerTypes);
s = s.replace(
  offerTypes,
  '{ "enum": ["hero", "discover", "tripleReward", "trinket", "lesserTrinket", "greaterTrinket", "darkGift", "quest", "reward", "buddy", "other"] },'
);

// --- 3. a leaderboard row carries the hero it shows -----------------------
const standingPlayer = `    "standing": {
      "type": "object",
      "description": "One row of the leaderboard, which is all a player normally sees of the other seats.",
      "properties": {
        "player": { "$ref": "#/$defs/identifier" },`;
must(standingPlayer);
s = s.replace(
  standingPlayer,
  `    "standing": {
      "type": "object",
      "description": "One row of the leaderboard, which is all a player normally sees of the other seats.",
      "properties": {
        "player": { "$ref": "#/$defs/identifier" },
        "hero": {
          "$ref": "#/$defs/card",
          "description": "The hero this seat is playing. The leaderboard shows it, so a one-seat recording learns every opponent's hero even though it learns nothing else about them. Repeat it in players[].hero once you know it."
        },`
);

// --- 4. say plainly that every seat should carry its hero ----------------
const heroOnPlayer = `        "hero": { "$ref": "#/$defs/card" },
        "heroPower": { "$ref": "#/$defs/card" },
        "startingHealth": { "type": "integer" },`;
must(heroOnPlayer);
s = s.replace(
  heroOnPlayer,
  `        "hero": {
          "$ref": "#/$defs/card",
          "description": "The hero this seat played. Record it for every seat you learn it for, including opponents: the leaderboard shows an opponent's hero, so even a one-seat recording can fill this in."
        },
        "heroPower": {
          "$ref": "#/$defs/card",
          "description": "The ability printed on the hero. A one-seat recording usually knows only its own."
        },
        "startingHealth": { "type": "integer" },`
);

JSON.parse(s);
fs.writeFileSync("schema/bg-history-1.1.schema.json", s);
console.log("wrote schema/bg-history-1.1.schema.json");
for (const k of ["trinketTier", "lesserTrinket", "Battlegrounds History Data 1.1"]) {
  console.log("  contains", JSON.stringify(k) + ":", s.includes(k));
}

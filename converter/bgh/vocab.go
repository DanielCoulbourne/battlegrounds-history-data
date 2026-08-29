package bgh

// The vocabulary. A word not listed here must start with "x-"; Validate
// enforces that, because it is what lets a reader tell a standard word from a
// newer version from somebody's private extension.

// Action verbs: what a player chose to do.
const (
	ActChooseHero = "choose_hero"
	ActBuy        = "buy"
	ActSell       = "sell"
	ActPlay       = "play"
	ActMove       = "move"
	ActRoll       = "roll"
	ActFreeze     = "freeze"
	ActUnfreeze   = "unfreeze"
	ActUpgrade    = "upgrade"
	ActHeroPower  = "hero_power"
	ActActivate   = "activate"
	ActMagnetize  = "magnetize"
	ActChoose     = "choose"
	ActOpenOffer  = "open_offer"
	ActPassCard   = "pass_card"
	ActEndTurn    = "end_turn"
	ActConcede    = "concede"
	ActEmote      = "emote"
)

var actionVerbs = map[string]bool{
	ActChooseHero: true, ActBuy: true, ActSell: true, ActPlay: true,
	ActMove: true, ActRoll: true, ActFreeze: true, ActUnfreeze: true,
	ActUpgrade: true, ActHeroPower: true, ActActivate: true, ActMagnetize: true,
	ActChoose: true, ActOpenOffer: true, ActPassCard: true, ActEndTurn: true,
	ActConcede: true, ActEmote: true,
}

// actionNeeds lists the Data fields each verb requires, matching the schema's
// if/then blocks. Validate checks them so a Go producer fails at the point of
// the mistake rather than in someone else's validator later.
var actionNeeds = map[string][]string{
	ActBuy:        {"from"},
	ActSell:       {"target"},
	ActPlay:       {"from"},
	ActMove:       {"from", "to"},
	ActMagnetize:  {"from", "to"},
	ActActivate:   {"target"},
	ActChoose:     {"offer", "option"},
	ActChooseHero: {"offer", "option"},
	ActOpenOffer:  {"offer"},
	ActPassCard:   {"from", "to"},
}

// Event verbs: what the game did.
const (
	EvGameStart          = "game_start"
	EvTurnStart          = "turn_start"
	EvTurnEnd            = "turn_end"
	EvGameEnd            = "game_end"
	EvShopDealt          = "shop_dealt"
	EvOffer              = "offer"
	EvGoldGranted        = "gold_granted"
	EvIncomeChanged      = "income_changed"
	EvCardGained         = "card_gained"
	EvCardRemoved        = "card_removed"
	EvCardMoved          = "card_moved"
	EvTripleCreated      = "triple_created"
	EvStatChange         = "stat_change"
	EvKeywordChange      = "keyword_change"
	EvEnchantmentAdded   = "enchantment_added"
	EvEnchantmentRemoved = "enchantment_removed"
	EvHealthChange       = "health_change"
	EvPairingAnnounced   = "pairing_announced"
	EvPlayerEliminated   = "player_eliminated"
	EvCombatStart        = "combat_start"
	EvCombatEnd          = "combat_end"
	EvAttack             = "attack"
	EvDamage             = "damage"
	EvHeal               = "heal"
	EvDivineShieldLost   = "divine_shield_lost"
	EvDeath              = "death"
	EvSummon             = "summon"
	EvReborn             = "reborn"
	EvTrigger            = "trigger"
	EvHeroDamage         = "hero_damage"
)

var eventVerbs = map[string]bool{
	EvGameStart: true, EvTurnStart: true, EvTurnEnd: true, EvGameEnd: true,
	EvShopDealt: true, EvOffer: true, EvGoldGranted: true, EvIncomeChanged: true,
	EvCardGained: true, EvCardRemoved: true, EvCardMoved: true, EvTripleCreated: true,
	EvStatChange: true, EvKeywordChange: true, EvEnchantmentAdded: true,
	EvEnchantmentRemoved: true, EvHealthChange: true, EvPairingAnnounced: true,
	EvPlayerEliminated: true, EvCombatStart: true, EvCombatEnd: true,
	EvAttack: true, EvDamage: true, EvHeal: true, EvDivineShieldLost: true,
	EvDeath: true, EvSummon: true, EvReborn: true, EvTrigger: true, EvHeroDamage: true,
}

var eventNeeds = map[string][]string{
	EvOffer:       {"id", "options"},
	EvCombatStart: {"sides"},
	EvCombatEnd:   {"sides"},
	EvAttack:      {"attacker", "defender"},
	EvDamage:      {"target", "amount"},
	EvDeath:       {"target"},
	EvSummon:      {"card"},
}

// Offer types: what sort of choice the game put in front of a player.
const (
	OfferHero           = "hero"
	OfferDiscover       = "discover"
	OfferTripleReward   = "tripleReward"
	OfferTrinket        = "trinket"
	OfferLesserTrinket  = "lesserTrinket"
	OfferGreaterTrinket = "greaterTrinket"
	OfferDarkGift       = "darkGift"
	OfferQuest          = "quest"
	OfferReward         = "reward"
	OfferBuddy          = "buddy"
)

var offerTypes = map[string]bool{
	OfferHero: true, OfferDiscover: true, OfferTripleReward: true,
	OfferTrinket: true, OfferLesserTrinket: true, OfferGreaterTrinket: true,
	OfferDarkGift: true, OfferQuest: true, OfferReward: true, OfferBuddy: true,
	"other": true,
}

// Keywords printed on cards.
const (
	KwTaunt         = "taunt"
	KwDivineShield  = "divineShield"
	KwPoisonous     = "poisonous"
	KwVenomous      = "venomous"
	KwWindfury      = "windfury"
	KwMegaWindfury  = "megaWindfury"
	KwReborn        = "reborn"
	KwStealth       = "stealth"
	KwImmune        = "immune"
	KwCleave        = "cleave"
	KwCantAttack    = "cantAttack"
	KwBattlecry     = "battlecry"
	KwDeathrattle   = "deathrattle"
	KwStartOfCombat = "startOfCombat"
	KwEndOfTurn     = "endOfTurn"
	KwStartOfTurn   = "startOfTurn"
	KwAvenge        = "avenge"
	KwRally         = "rally"
	KwMagnetic      = "magnetic"
	KwSpellcraft    = "spellcraft"
	KwChooseOne     = "chooseOne"
	KwDiscover      = "discover"
	KwActivate      = "activate"
	KwFrenzy        = "frenzy"
	KwOverkill      = "overkill"
)

var keywords = map[string]bool{
	KwTaunt: true, KwDivineShield: true, KwPoisonous: true, KwVenomous: true,
	KwWindfury: true, KwMegaWindfury: true, KwReborn: true, KwStealth: true,
	KwImmune: true, KwCleave: true, KwCantAttack: true, KwBattlecry: true,
	KwDeathrattle: true, KwStartOfCombat: true, KwEndOfTurn: true,
	KwStartOfTurn: true, KwAvenge: true, KwRally: true, KwMagnetic: true,
	KwSpellcraft: true, KwChooseOne: true, KwDiscover: true, KwActivate: true,
	KwFrenzy: true, KwOverkill: true,
}

// Minion types, which players call tribes. "all" counts as every type at once.
const (
	MtBeast     = "beast"
	MtDemon     = "demon"
	MtDragon    = "dragon"
	MtElemental = "elemental"
	MtMech      = "mech"
	MtMurloc    = "murloc"
	MtNaga      = "naga"
	MtPirate    = "pirate"
	MtQuilboar  = "quilboar"
	MtUndead    = "undead"
	MtAll       = "all"
)

var minionTypes = map[string]bool{
	MtBeast: true, MtDemon: true, MtDragon: true, MtElemental: true,
	MtMech: true, MtMurloc: true, MtNaga: true, MtPirate: true,
	MtQuilboar: true, MtUndead: true, MtAll: true,
}

// MinionType turns a card database's race string into this format's spelling.
// It accepts the uppercase names Blizzard's card data uses. The second result
// is false for a race it does not know, so a caller can decide between dropping
// it and writing an "x-" word.
func MinionType(race string) (string, bool) {
	switch upper(race) {
	case "BEAST":
		return MtBeast, true
	case "DEMON":
		return MtDemon, true
	case "DRAGON":
		return MtDragon, true
	case "ELEMENTAL":
		return MtElemental, true
	case "MECHANICAL", "MECH":
		return MtMech, true
	case "MURLOC":
		return MtMurloc, true
	case "NAGA":
		return MtNaga, true
	case "PIRATE":
		return MtPirate, true
	case "QUILBOAR":
		return MtQuilboar, true
	case "UNDEAD":
		return MtUndead, true
	case "ALL":
		return MtAll, true
	}
	return "", false
}

func upper(s string) string {
	out := []byte(s)
	for i, c := range out {
		if c >= 'a' && c <= 'z' {
			out[i] = c - 32
		}
	}
	return string(out)
}

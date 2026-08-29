// Package bgh writes Battlegrounds History Data files.
//
// The types here mirror the JSON Schema in schema/ one for one. If the schema
// and these types ever disagree, the schema is right; Validate exists to catch
// the disagreement rather than to replace the schema.
//
// # Three states, not two
//
// The format distinguishes a field that is absent, a field that is present and
// null, and a field that is present with a value. They mean, in order: this does
// not apply here; this applies and I could not see it; this is the value. Go's
// encoding/json cannot express all three with a plain int, so every field where
// null is meaningful uses *Opt, built with Known or Unknown. A nil *Opt is
// omitted, Unknown() marshals to null, and Known(n) marshals to n.
//
// # Building a document
//
// Use a Builder. It owns seq numbering, so entries come out in order with no
// gaps and no repeats, which is one of the rules a JSON Schema cannot check.
//
//	b := bgh.NewSeatRecording("my-recording", "p1")
//	b.Event(bgh.EvTurnStart).Turn(4).Phase(bgh.PhaseRecruit).Done()
//	b.Action("p1", bgh.ActBuy).Turn(4).Phase(bgh.PhaseRecruit).
//		Data(bgh.Data{From: bgh.Slot(bgh.ZoneShop, 1), Cost: bgh.Int(3)}).Done()
//	doc := b.Document()
package bgh

import (
	"encoding/json"
	"fmt"
)

// Version is the specification version this package writes.
const Version = "1.1"

// Format is the fixed marker at the top of every file.
const Format = "battlegrounds-history"

// Opt is an integer that may be absent, present, or present and unknown.
// Build one with Known or Unknown; a nil *Opt is left out of the JSON entirely.
type Opt struct {
	value int
	known bool
}

// Known returns an Opt carrying n.
func Known(n int) *Opt { return &Opt{value: n, known: true} }

// Unknown returns an Opt that marshals to null: the field applies here and the
// recorder could not learn its value.
func Unknown() *Opt { return &Opt{} }

// MarshalJSON writes the value, or null when the value is unknown.
func (o Opt) MarshalJSON() ([]byte, error) {
	if !o.known {
		return []byte("null"), nil
	}
	return json.Marshal(o.value)
}

// UnmarshalJSON reads a number or null.
func (o *Opt) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*o = Opt{}
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*o = Opt{value: n, known: true}
	return nil
}

// Value reports the number and whether it is known.
func (o *Opt) Value() (int, bool) {
	if o == nil {
		return 0, false
	}
	return o.value, o.known
}

// Int is a convenience for the many plain optional integers, where absent and
// unknown do not need to be told apart.
func Int(n int) *int { return &n }

// Bool is Int for booleans.
func Bool(b bool) *bool { return &b }

// Str is Int for strings.
func Str(s string) *string { return &s }

// Ext is producer data the format itself defines nothing about. Key it by your
// own name so two producers cannot collide.
type Ext map[string]any

// Document is one recorded game.
type Document struct {
	Format       string   `json:"format"`
	SpecVersion  string   `json:"version"`
	CardIDScheme string   `json:"cardIdScheme,omitempty"`
	Recording    Record   `json:"recording"`
	Game         Game     `json:"game"`
	Players      []Player `json:"players"`
	History      []Entry  `json:"history"`
	Ext          Ext      `json:"ext,omitempty"`
}

// Record holds facts about the file rather than about the game.
type Record struct {
	ID               string    `json:"id"`
	RecordedAt       string    `json:"recordedAt,omitempty"`
	Recorder         *Recorder `json:"recorder,omitempty"`
	Observer         Observer  `json:"observer"`
	Detail           *Detail   `json:"detail,omitempty"`
	Truncated        bool      `json:"truncated,omitempty"`
	TruncationReason string    `json:"truncationReason,omitempty"`
	Note             string    `json:"note,omitempty"`
	Ext              Ext       `json:"ext,omitempty"`
}

// Recorder names the program that produced the file.
type Recorder struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
	Kind    string `json:"kind,omitempty"`
	URL     string `json:"url,omitempty"`
	Ext     Ext    `json:"ext,omitempty"`
}

// Recorder kinds.
const (
	RecorderTracker      = "tracker"
	RecorderSimulator    = "simulator"
	RecorderReplayParser = "replayParser"
	RecorderManual       = "manual"
)

// Observer says who was watching, and therefore what the file may contain.
type Observer struct {
	Scope string `json:"scope"`
	Seat  string `json:"seat,omitempty"`
	Ext   Ext    `json:"ext,omitempty"`
}

// Observer scopes.
const (
	ScopeLobby = "lobby"
	ScopeSeat  = "seat"
)

// Detail states how much the file contains, so a reader never has to read an
// absence as a fact.
type Detail struct {
	Combat   string `json:"combat,omitempty"`
	States   string `json:"states,omitempty"`
	Entities bool   `json:"entities"`
	Ext      Ext    `json:"ext,omitempty"`
}

// Combat detail levels, weakest first.
const (
	CombatNone    = "none"
	CombatOutcome = "outcome"
	CombatBoards  = "boards"
	CombatEvents  = "events"
)

// State cadences.
const (
	StatesNone        = "none"
	StatesTurnStart   = "turnStart"
	StatesEveryAction = "everyAction"
	StatesIrregular   = "irregular"
)

// Game holds facts about the lobby that last all game.
type Game struct {
	ID          string   `json:"id,omitempty"`
	Mode        string   `json:"mode,omitempty"`
	StartedAt   string   `json:"startedAt,omitempty"`
	Patch       string   `json:"patch,omitempty"`
	Season      string   `json:"season,omitempty"`
	SeatCount   int      `json:"seatCount,omitempty"`
	MinionTypes []string `json:"minionTypes,omitempty"`
	Anomaly     *Card    `json:"anomaly,omitempty"`
	RNG         *RNG     `json:"rng,omitempty"`
	Ext         Ext      `json:"ext,omitempty"`
}

// RNG is for a producer that generated its own randomness. It never replaces
// the recorded history.
type RNG struct {
	Seed   any    `json:"seed,omitempty"`
	Engine string `json:"engine,omitempty"`
	Ext    Ext    `json:"ext,omitempty"`
}

// Game modes.
const (
	ModeSolo = "solo"
	ModeDuos = "duos"
)

// Player is one seat.
type Player struct {
	ID             string  `json:"id"`
	Seat           *int    `json:"seat,omitempty"`
	Name           string  `json:"name,omitempty"`
	Kind           string  `json:"kind,omitempty"`
	Agent          *Agent  `json:"agent,omitempty"`
	Teammate       string  `json:"teammate,omitempty"`
	Hero           *Card   `json:"hero,omitempty"`
	HeroPower      *Card   `json:"heroPower,omitempty"`
	StartingHealth *int    `json:"startingHealth,omitempty"`
	StartingArmor  *int    `json:"startingArmor,omitempty"`
	Result         *Result `json:"result,omitempty"`
	Ext            Ext     `json:"ext,omitempty"`
}

// Player kinds.
const (
	KindHuman   = "human"
	KindBot     = "bot"
	KindUnknown = "unknown"
)

// Agent names the program that made a seat's decisions.
type Agent struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
	Ext     Ext    `json:"ext,omitempty"`
}

// Result is how a seat finished. Every field is optional: a one-seat recording
// may never learn some of them, and must not invent them.
type Result struct {
	Placement        *int   `json:"placement,omitempty"`
	Health           *int   `json:"health,omitempty"`
	Armor            *int   `json:"armor,omitempty"`
	Tier             *int   `json:"tier,omitempty"`
	CombatsPlayed    *int   `json:"combatsPlayed,omitempty"`
	CombatsWon       *int   `json:"combatsWon,omitempty"`
	HeroPowerUses    *int   `json:"heroPowerUses,omitempty"`
	EliminatedOnTurn *int   `json:"eliminatedOnTurn,omitempty"`
	Trinkets         []Card `json:"trinkets,omitempty"`
	Board            []Card `json:"board,omitempty"`
	Ext              Ext    `json:"ext,omitempty"`
}

// Card is one card as it stood at a moment.
type Card struct {
	Entity       string        `json:"entity,omitempty"`
	CardID       string        `json:"cardId,omitempty"`
	DbfID        int           `json:"dbfId,omitempty"`
	Name         string        `json:"name,omitempty"`
	Type         string        `json:"type,omitempty"`
	Attack       *int          `json:"attack,omitempty"`
	Health       *int          `json:"health,omitempty"`
	MaxHealth    *int          `json:"maxHealth,omitempty"`
	Tier         *int          `json:"tier,omitempty"`
	Golden       bool          `json:"golden,omitempty"`
	MinionTypes  []string      `json:"minionTypes,omitempty"`
	Keywords     []string      `json:"keywords,omitempty"`
	Cost         *int          `json:"cost,omitempty"`
	CostsHealth  bool          `json:"costsHealth,omitempty"`
	TrinketTier  string        `json:"trinketTier,omitempty"`
	Playable     *bool         `json:"playable,omitempty"`
	Enchantments []Enchantment `json:"enchantments,omitempty"`
	Attached     []Card        `json:"attached,omitempty"`
	Text         string        `json:"text,omitempty"`
	Unknown      bool          `json:"unknown,omitempty"`
	Ext          Ext           `json:"ext,omitempty"`
}

// Card types.
const (
	TypeMinion    = "minion"
	TypeSpell     = "spell"
	TypeTrinket   = "trinket"
	TypeHero      = "hero"
	TypeHeroPower = "heroPower"
	TypeToken     = "token"
	TypeAnomaly   = "anomaly"
)

// Trinket slots. They are two slots offered at two different points in the
// game, not two grades of the same thing.
const (
	TrinketLesser  = "lesser"
	TrinketGreater = "greater"
)

// Enchantment is something attached to a card that changes it.
type Enchantment struct {
	CardID   string   `json:"cardId,omitempty"`
	Name     string   `json:"name,omitempty"`
	Text     string   `json:"text,omitempty"`
	Attack   *int     `json:"attack,omitempty"`
	Health   *int     `json:"health,omitempty"`
	Keywords []string `json:"keywords,omitempty"`
	Ext      Ext      `json:"ext,omitempty"`
}

// Ref points at one card in one place. Positions count from 0, left to right.
type Ref struct {
	Player string `json:"player,omitempty"`
	Zone   string `json:"zone,omitempty"`
	Index  *int   `json:"index,omitempty"`
	Entity string `json:"entity,omitempty"`
	CardID string `json:"cardId,omitempty"`
}

// Slot points at a position in a zone.
func Slot(zone string, index int) *Ref { return &Ref{Zone: zone, Index: Int(index)} }

// At points at a position in a zone and names the copy standing there. Prefer
// it: the entity is exact, and the index still serves a reader that does not
// track entities.
func At(zone string, index int, entity string) *Ref {
	return &Ref{Zone: zone, Index: Int(index), Entity: entity}
}

// Entity points at one copy wherever it is.
func Entity(entity string) *Ref { return &Ref{Entity: entity} }

// Of returns a copy of r owned by the named player.
func (r *Ref) Of(player string) *Ref {
	if r == nil {
		return nil
	}
	out := *r
	out.Player = player
	return &out
}

// Zone names.
const (
	ZoneBoard    = "board"
	ZoneHand     = "hand"
	ZoneShop     = "shop"
	ZoneTrinkets = "trinkets"
	ZoneHero     = "hero"
	ZoneHeroPwr  = "heroPower"
	ZoneOffer    = "offer"
	ZoneRemoved  = "removed"
)

// Zone is the contents of one zone, plus an honest count of what was not seen.
type Zone struct {
	Cards        []Card `json:"cards"`
	UnknownCount *int   `json:"unknownCount,omitempty"`
	Frozen       *bool  `json:"frozen,omitempty"`
	Capacity     *int   `json:"capacity,omitempty"`
	Ext          Ext    `json:"ext,omitempty"`
}

// Phases.
const (
	PhaseSetup   = "setup"
	PhaseRecruit = "recruit"
	PhaseCombat  = "combat"
	PhaseEnd     = "end"
)

// Entry is one line of the history: an action, an event, or a state.
type Entry struct {
	Seq       int    `json:"seq"`
	Kind      string `json:"kind"`
	Turn      *int   `json:"turn,omitempty"`
	Phase     string `json:"phase,omitempty"`
	At        string `json:"at,omitempty"`
	ElapsedMs *int   `json:"elapsedMs,omitempty"`
	Actor     string `json:"actor,omitempty"`
	Note      string `json:"note,omitempty"`

	// Action only.
	Action   string    `json:"action,omitempty"`
	Accepted *bool     `json:"accepted,omitempty"`
	Error    string    `json:"error,omitempty"`
	Decision *Decision `json:"decision,omitempty"`

	// Event only.
	Event string `json:"event,omitempty"`

	// Action and event.
	Data *Data `json:"data,omitempty"`

	// State only.
	Reason       string          `json:"reason,omitempty"`
	Player       *SeatState      `json:"player,omitempty"`
	Zones        map[string]Zone `json:"zones,omitempty"`
	Standings    []Standing      `json:"standings,omitempty"`
	NextOpponent *NextOpponent   `json:"nextOpponent,omitempty"`

	Ext Ext `json:"ext,omitempty"`
}

// Entry kinds.
const (
	KindAction = "action"
	KindEvent  = "event"
	KindState  = "state"
)

// SeatState is the numbers that describe a seat, apart from its cards.
type SeatState struct {
	Health         *int  `json:"health,omitempty"`
	Armor          *int  `json:"armor,omitempty"`
	Tier           *int  `json:"tier,omitempty"`
	Gold           *int  `json:"gold,omitempty"`
	GoldCap        *int  `json:"goldCap,omitempty"`
	UpgradeCost    *Opt  `json:"upgradeCost,omitempty"`
	RollCost       *int  `json:"rollCost,omitempty"`
	HeroPowerUses  *int  `json:"heroPowerUses,omitempty"`
	HeroPowerReady *bool `json:"heroPowerReady,omitempty"`
	Alive          *bool `json:"alive,omitempty"`
	Ext            Ext   `json:"ext,omitempty"`
}

// Standing is one row of the leaderboard, which is all a player normally sees
// of the other seats. It carries the hero, because the leaderboard shows it and
// that is very nearly the only durable fact a one-seat recording gets about an
// opponent.
type Standing struct {
	Player    string `json:"player,omitempty"`
	Hero      *Card  `json:"hero,omitempty"`
	Health    *Opt   `json:"health,omitempty"`
	Armor     *Opt   `json:"armor,omitempty"`
	Tier      *Opt   `json:"tier,omitempty"`
	Alive     *bool  `json:"alive,omitempty"`
	Placement *int   `json:"placement,omitempty"`
	Ext       Ext    `json:"ext,omitempty"`
}

// NextOpponent is the seat a player will fight next.
type NextOpponent struct {
	Player  string `json:"player,omitempty"`
	GhostOf string `json:"ghostOf,omitempty"`
	Ext     Ext    `json:"ext,omitempty"`
}

// Decision records how a choice was made. A program can fill it in; a person
// cannot, and leaves it out.
type Decision struct {
	LegalOptions *int      `json:"legalOptions,omitempty"`
	Probability  *float64  `json:"probability,omitempty"`
	Value        *float64  `json:"value,omitempty"`
	Considered   []Option  `json:"considered,omitempty"`
	Refusals     []Refusal `json:"refusals,omitempty"`
	Ext          Ext       `json:"ext,omitempty"`
}

// Option is one action that was on the table but not taken.
type Option struct {
	Action      string   `json:"action,omitempty"`
	Data        *Data    `json:"data,omitempty"`
	Probability *float64 `json:"probability,omitempty"`
	Value       *float64 `json:"value,omitempty"`
}

// Refusal is one reason a rule removed candidates before the chooser saw them.
type Refusal struct {
	Reason  string `json:"reason"`
	Count   *int   `json:"count,omitempty"`
	Example string `json:"example,omitempty"`
	Ext     Ext    `json:"ext,omitempty"`
}

// Data is the payload of an action or an event. Which fields matter depends on
// the verb; the schema states which are required for each.
type Data struct {
	// Actions.
	From         *Ref   `json:"from,omitempty"`
	To           *Ref   `json:"to,omitempty"`
	Target       *Ref   `json:"target,omitempty"`
	Targets      []Ref  `json:"targets,omitempty"`
	Cost         *int   `json:"cost,omitempty"`
	CostResource string `json:"costResource,omitempty"`
	Gold         *int   `json:"gold,omitempty"`
	Health       *int   `json:"health,omitempty"`
	Armor        *int   `json:"armor,omitempty"`
	Tier         *int   `json:"tier,omitempty"`
	Offer        string `json:"offer,omitempty"`
	OptionIndex  *int   `json:"option,omitempty"`
	Mode         *int   `json:"mode,omitempty"`
	Emote        string `json:"emote,omitempty"`

	// Offers.
	ID        string     `json:"id,omitempty"`
	OfferType string     `json:"offerType,omitempty"`
	Options   []OfferOpt `json:"options,omitempty"`
	Hidden    bool       `json:"hidden,omitempty"`
	Mandatory bool       `json:"mandatory,omitempty"`

	// Events.
	Reason      string       `json:"reason,omitempty"`
	Cards       []Card       `json:"cards,omitempty"`
	Card        *Card        `json:"card,omitempty"`
	Source      *Ref         `json:"source,omitempty"`
	Attacker    *Ref         `json:"attacker,omitempty"`
	Defender    *Ref         `json:"defender,omitempty"`
	Swing       *int         `json:"swing,omitempty"`
	Amount      *int         `json:"amount,omitempty"`
	Attack      *int         `json:"attack,omitempty"`
	AttackAbs   bool         `json:"absolute,omitempty"`
	Permanent   bool         `json:"permanent,omitempty"`
	Gained      []string     `json:"gained,omitempty"`
	Lost        []string     `json:"lost,omitempty"`
	Enchantment *Enchantment `json:"enchantment,omitempty"`
	Absorbed    bool         `json:"absorbed,omitempty"`
	Poisonous   bool         `json:"poisonous,omitempty"`
	Lethal      *bool        `json:"lethal,omitempty"`
	Killer      *Ref         `json:"killer,omitempty"`
	Position    *int         `json:"position,omitempty"`
	Times       *int         `json:"times,omitempty"`
	Turn        *int         `json:"turn,omitempty"`
	UpgradeCost *int         `json:"upgradeCost,omitempty"`

	// Lobby-level events.
	PlayerID      string      `json:"player,omitempty"`
	Opponent      string      `json:"opponent,omitempty"`
	GhostOf       string      `json:"ghostOf,omitempty"`
	Combat        string      `json:"combat,omitempty"`
	Sides         []Side      `json:"sides,omitempty"`
	FirstAttacker string      `json:"firstAttacker,omitempty"`
	Winner        *string     `json:"winner,omitempty"`
	Placements    []Placement `json:"placements,omitempty"`
	Placement     *int        `json:"placement,omitempty"`
	Standings     []Standing  `json:"standings,omitempty"`

	Ext Ext `json:"ext,omitempty"`
}

// HealthField is the health change an event reports. It shares a JSON name with
// the action-side health, so it lives on Data as Health above.

// OfferOpt is one option of an offer.
type OfferOpt struct {
	Cards []Card `json:"cards,omitempty"`
	Label string `json:"label,omitempty"`
	Ext   Ext    `json:"ext,omitempty"`
}

// Placement is one row of the final standings.
type Placement struct {
	Player    string `json:"player,omitempty"`
	Placement *int   `json:"placement,omitempty"`
}

// Side is one half of a fight.
type Side struct {
	Player       string `json:"player,omitempty"`
	GhostOf      string `json:"ghostOf,omitempty"`
	Hero         *Card  `json:"hero,omitempty"`
	HeroPower    *Card  `json:"heroPower,omitempty"`
	Tier         *int   `json:"tier,omitempty"`
	HealthBefore *Opt   `json:"healthBefore,omitempty"`
	HealthAfter  *Opt   `json:"healthAfter,omitempty"`
	ArmorBefore  *Opt   `json:"armorBefore,omitempty"`
	ArmorAfter   *Opt   `json:"armorAfter,omitempty"`
	DamageTaken  *Opt   `json:"damageTaken,omitempty"`
	Board        []Card `json:"board,omitempty"`
	Hand         *Zone  `json:"hand,omitempty"`
	Survivors    []Ref  `json:"survivors,omitempty"`
	Eliminated   bool   `json:"eliminated,omitempty"`
	Ext          Ext    `json:"ext,omitempty"`
}

// Marshal writes the document as indented JSON, ending with a newline.
func (d *Document) Marshal() ([]byte, error) {
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("bgh: encode document: %w", err)
	}
	return append(b, '\n'), nil
}

package hslog

import "strconv"

// Tag names this converter reads. Every one was seen in a real log. A tag the
// client has no name for prints as a number, so tags are keyed by string and a
// numeric key is kept rather than dropped.
const (
	TagCardType      = "CARDTYPE"
	TagController    = "CONTROLLER"
	TagZone          = "ZONE"
	TagZonePosition  = "ZONE_POSITION"
	TagEntityID      = "ENTITY_ID"
	TagPlayerID      = "PLAYER_ID"
	TagAtk           = "ATK"
	TagHealth        = "HEALTH"
	TagDamage        = "DAMAGE"
	TagArmor         = "ARMOR"
	TagPremium       = "PREMIUM"
	TagTechLevel     = "TECH_LEVEL"
	TagCardRace      = "CARDRACE"
	TagCost          = "COST"
	TagCreator       = "CREATOR"
	TagFrozen        = "FROZEN"
	TagPlaystate     = "PLAYSTATE"
	TagState         = "STATE"
	TagStep          = "STEP"
	TagTurn          = "TURN"
	TagResources     = "RESOURCES"
	TagResourcesUsed = "RESOURCES_USED"

	// Battlegrounds-specific.
	TagBoardVisualState = "BOARD_VISUAL_STATE" // 1 recruit, 2 combat
	TagPlayerTechLevel  = "PLAYER_TECH_LEVEL"  // tavern tier
	TagLeaderboardPlace = "PLAYER_LEADERBOARD_PLACE"
	TagNextOpponent     = "NEXT_OPPONENT_PLAYER_ID"
	TagCurrentCombat    = "BACON_CURRENT_COMBAT_PLAYER_ID"
	TagCombatPhaseHero  = "BACON_COMBAT_PHASE_HERO"
	TagWonLastCombat    = "BACON_WON_LAST_COMBAT"
	TagDiedLastCombat   = "BACON_DIED_LAST_COMBAT"
	TagDummyPlayer      = "BACON_DUMMY_PLAYER"
	TagPoolMinion       = "IS_BACON_POOL_MINION"
	TagHasDragToBuy     = "HAS_DRAG_TO_BUY"
	TagGameSeed         = "GAME_SEED"
	TagTrinketsActive   = "BACON_TRINKETS_ACTIVE"
	TagFirstTrinketDBF  = "BACON_FIRST_TRINKET_DATABASE_ID"
	TagSecondTrinketDBF = "BACON_SECOND_TRINKET_DATABASE_ID"
	TagPlayerTriples    = "PLAYER_TRIPLES"
	TagCopiedFrom       = "COPIED_FROM_ENTITY_ID"
	TagHeroEntity       = "HERO_ENTITY"

	// Combat.
	TagProposedAttacker = "PROPOSED_ATTACKER"
	TagProposedDefender = "PROPOSED_DEFENDER"
	TagAttacking        = "ATTACKING"
	TagDefending        = "DEFENDING"
	TagPredamage        = "PREDAMAGE"
	TagLastAffectedBy   = "LAST_AFFECTED_BY"

	// Keyword tags.
	TagTaunt        = "TAUNT"
	TagDivineShield = "DIVINE_SHIELD"
	TagPoisonous    = "POISONOUS"
	TagVenomous     = "VENOMOUS"
	TagWindfury     = "WINDFURY"
	TagMegaWindfury = "MEGA_WINDFURY"
	TagReborn       = "REBORN"
	TagStealth      = "STEALTH"
)

// Card ids the client uses for its own furniture rather than for game cards.
const (
	CardRerollButton  = "TB_BaconShop_8p_Reroll_Button"
	CardFreezeButton  = "TB_BaconShopLockAll_Button"
	CardDragBuy       = "TB_BaconShop_DragBuy"
	CardDragBuySpell  = "TB_BaconShop_DragBuy_Spell"
	CardDragSell      = "TB_BaconShop_DragSell"
	CardTechUpPrefix  = "TB_BaconShopTechUp"
	CardHeroPowerPre  = "TB_BaconShop_HP_"
	CardBobSkinPre    = "TB_BaconShopBob_SKIN"
	CardHeroPlacehldr = "TB_BaconShop_HERO_PH"

	// The two trinket slots. The client names them "Lesser Trinket" and
	// "Greater Trinket", and these ids are the reliable way to tell them apart:
	// the slot entity is destroyed and re-created every turn, so its entity id
	// is not stable and must not be used as identity.
	CardTrinketLesser  = "BG30_Trinket_1st"
	CardTrinketGreater = "BG30_Trinket_2nd"
)

// Zones.
const (
	ZonePlay      = "PLAY"
	ZoneHand      = "HAND"
	ZoneSetAside  = "SETASIDE"
	ZoneGraveyard = "GRAVEYARD"
	ZoneRemoved   = "REMOVEDFROMGAME"
)

// Card types.
const (
	TypeMinion    = "MINION"
	TypeHero      = "HERO"
	TypeHeroPower = "HERO_POWER"
	TypeSpell     = "SPELL"
	TypeEnchant   = "ENCHANTMENT"
	TypeTrinket   = "BATTLEGROUND_TRINKET"
	TypeButton    = "GAME_MODE_BUTTON"
)

// Entity is one thing in the game, as far as the log has described it.
type Entity struct {
	ID     int
	CardID string
	Name   string
	Tags   map[string]string
}

// Tag reads a tag, or "".
func (e *Entity) Tag(name string) string {
	if e == nil {
		return ""
	}
	return e.Tags[name]
}

// Int reads a tag as a number. A tag whose value is an enum name reads 0,
// which is deliberate: every value is a string until somebody asks for one.
func (e *Entity) Int(name string) int {
	n, _ := strconv.Atoi(e.Tag(name))
	return n
}

// Bool reads a flag tag.
func (e *Entity) Bool(name string) bool { return e.Int(name) == 1 }

// Zone, CardType, Controller and ZonePos are the four questions asked most.
func (e *Entity) Zone() string     { return e.Tag(TagZone) }
func (e *Entity) CardType() string { return e.Tag(TagCardType) }
func (e *Entity) Controller() int  { return e.Int(TagController) }
func (e *Entity) ZonePos() int     { return e.Int(TagZonePosition) }

// Table accumulates events into the state of one game.
//
// It holds no Battlegrounds knowledge beyond the tag names above: it is a map
// of entities and their tags, and everything that reads meaning into them lives
// in package convert.
type Table struct {
	Entities map[int]*Entity
	Order    []int // creation order, so a projection can be deterministic

	GameEntityID  int
	LocalPlayerID int // your controller id
	DummyPlayerID int // the controller that owns the tavern, the leaderboard and
	// whichever opponent you are fighting, all at once
	LocalName string
	Build     string
	GameType  string

	// NamedTags keeps tags addressed to a display name that maps to no entity.
	// Your gold arrives this way, and so does the announcement of who each
	// player is fighting. Dropping them is why a first parser reports no gold.
	NamedTags map[string]map[string]string

	byName map[string]int
}

// NewTable starts an empty game.
func NewTable() *Table {
	return &Table{
		Entities:  map[int]*Entity{},
		NamedTags: map[string]map[string]string{},
		byName:    map[string]int{},
	}
}

// Get returns an entity, or nil.
func (t *Table) Get(id int) *Entity { return t.Entities[id] }

// Apply folds one event into the table.
func (t *Table) Apply(ev Event) {
	switch ev.Kind {
	case KindGameInfo:
		if v, ok := ev.Fields["BuildNumber"]; ok {
			t.Build = v
		}
		if v, ok := ev.Fields["GameType"]; ok {
			t.GameType = v
		}
		// "PlayerID=3, PlayerName=coulbourne#1741" names the local player
		// outright, which is a better source than guessing from the order names
		// appear in: Bob and four opponents are also addressed by name.
		if name, ok := ev.Fields["PlayerName"]; ok {
			if idStr, ok := ev.Fields["PlayerID"]; ok {
				if id, err := strconv.Atoi(idStr); err == nil && t.LocalName == "" {
					t.LocalName, t.LocalPlayerID = name, id
				}
			}
		}

	case KindGameEntity:
		t.GameEntityID = ev.EntityID
		t.upsert(ev.EntityID, ev.CardID, ev.EntityName, ev.Tags)

	case KindPlayer:
		e := t.upsert(ev.EntityID, "", ev.EntityName, ev.Tags)
		e.Tags[TagPlayerID] = strconv.Itoa(ev.PlayerID)
		if e.Bool(TagDummyPlayer) {
			t.DummyPlayerID = ev.PlayerID
		}

	case KindFullEntity, KindShowEntity:
		t.upsert(ev.EntityID, ev.CardID, ev.EntityName, ev.Tags)

	case KindHideEntity:
		if e := t.Entities[ev.EntityID]; e != nil {
			e.Tags[ev.Tag] = ev.Value
		}

	case KindTagChange:
		id := ev.EntityID
		if id == 0 && ev.EntityName != "" {
			if known, ok := t.byName[ev.EntityName]; ok {
				id = known
			} else {
				if t.NamedTags[ev.EntityName] == nil {
					t.NamedTags[ev.EntityName] = map[string]string{}
				}
				t.NamedTags[ev.EntityName][ev.Tag] = ev.Value
				return
			}
		}
		if id == 0 {
			return
		}
		e := t.Entities[id]
		if e == nil {
			e = t.upsert(id, "", "", nil)
		}
		e.Tags[ev.Tag] = ev.Value
	}
}

// LocalTagInt reads one of the tags addressed to the local player by name, such
// as gold.
func (t *Table) LocalTagInt(tag string) int {
	n, _ := strconv.Atoi(t.NamedTags[t.LocalName][tag])
	return n
}

func (t *Table) upsert(id int, cardID, name string, tags map[string]string) *Entity {
	e := t.Entities[id]
	if e == nil {
		e = &Entity{ID: id, Tags: map[string]string{}}
		t.Entities[id] = e
		t.Order = append(t.Order, id)
	}
	// Never overwrite a known card id with an empty one. The server re-creates
	// entities hidden, and a card you have already seen stays seen.
	if cardID != "" {
		e.CardID = cardID
	}
	if name != "" {
		e.Name = name
		t.byName[name] = id
	}
	for k, v := range tags {
		e.Tags[k] = v
	}
	return e
}

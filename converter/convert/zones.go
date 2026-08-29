package convert

import (
	"sort"
	"strconv"
	"strings"

	"github.com/DanielCoulbourne/battlegrounds-history-data/converter/bgh"
	"github.com/DanielCoulbourne/battlegrounds-history-data/converter/hslog"
)

// card turns one log entity into one card.
//
// The numbers are the ones the card is showing right now, with every buff
// already in them, which is what the format asks for: health is the printed
// total less the damage on it.
func (s *session) card(e *hslog.Entity) bgh.Card {
	c := bgh.Card{Entity: "e" + strconv.Itoa(e.ID), CardID: e.CardID}
	if e.CardID == "" {
		c.Unknown = true
	}
	switch e.CardType() {
	case hslog.TypeMinion:
		c.Type = bgh.TypeMinion
	case hslog.TypeSpell:
		c.Type = bgh.TypeSpell
	case hslog.TypeHero:
		c.Type = bgh.TypeHero
	case hslog.TypeHeroPower:
		c.Type = bgh.TypeHeroPower
	case hslog.TypeTrinket:
		c.Type = bgh.TypeTrinket
	}
	if v := e.Int(hslog.TagAtk); v != 0 || e.Tag(hslog.TagAtk) != "" {
		c.Attack = bgh.Int(v)
	}
	if hp := e.Int(hslog.TagHealth); hp != 0 {
		c.Health = bgh.Int(hp - e.Int(hslog.TagDamage))
		if d := e.Int(hslog.TagDamage); d > 0 {
			c.MaxHealth = bgh.Int(hp)
		}
	}
	if t := e.Int(hslog.TagTechLevel); t > 0 {
		c.Tier = bgh.Int(t)
	}
	if e.Bool(hslog.TagPremium) {
		c.Golden = true
	}
	if race := e.Tag(hslog.TagCardRace); race != "" {
		if mt, ok := bgh.MinionType(race); ok {
			c.MinionTypes = []string{mt}
		}
	}
	for _, kw := range []struct {
		tag  string
		word string
	}{
		{hslog.TagTaunt, bgh.KwTaunt},
		{hslog.TagDivineShield, bgh.KwDivineShield},
		{hslog.TagPoisonous, bgh.KwPoisonous},
		{hslog.TagVenomous, bgh.KwVenomous},
		{hslog.TagWindfury, bgh.KwWindfury},
		{hslog.TagMegaWindfury, bgh.KwMegaWindfury},
		{hslog.TagReborn, bgh.KwReborn},
		{hslog.TagStealth, bgh.KwStealth},
	} {
		if e.Bool(kw.tag) {
			c.Keywords = append(c.Keywords, kw.word)
		}
	}
	// A trinket says which of the two slots it fills. The client names the slot
	// entities outright, and the slot is not a rarity: they are offered at two
	// different points and a player holds at most one of each.
	switch e.CardID {
	case hslog.CardTrinketLesser:
		c.TrinketTier = bgh.TrinketLesser
	case hslog.CardTrinketGreater:
		c.TrinketTier = bgh.TrinketGreater
	}
	return c
}

func (s *session) ref(e *hslog.Entity) *bgh.Ref {
	if e == nil {
		return nil
	}
	r := &bgh.Ref{Entity: "e" + strconv.Itoa(e.ID), CardID: e.CardID}
	if pos := e.ZonePos(); pos > 0 {
		r.Index = bgh.Int(pos - 1) // the log counts from 1, this format from 0
	}
	switch e.Zone() {
	case hslog.ZonePlay:
		if e.CardType() == hslog.TypeMinion {
			r.Zone = bgh.ZoneBoard
		}
	case hslog.ZoneHand:
		r.Zone = bgh.ZoneHand
	}
	if e.Controller() == s.tab.LocalPlayerID {
		r.Player = s.me
	}
	return r
}

func (s *session) refID(id int) *bgh.Ref {
	if id == 0 {
		return nil
	}
	return s.ref(s.tab.Get(id))
}

// zoneOf collects the minions in one zone belonging to one controller, in board
// order. The log counts positions from 1 and this format from 0, so the sort is
// on the log's number and the conversion happens when a position is written.
func (s *session) zoneOf(controller int, zone string, types ...string) []bgh.Card {
	var es []*hslog.Entity
	for _, id := range s.tab.Order {
		e := s.tab.Get(id)
		if e.Controller() != controller || e.Zone() != zone {
			continue
		}
		ok := false
		for _, t := range types {
			if e.CardType() == t {
				ok = true
			}
		}
		if !ok {
			continue
		}
		es = append(es, e)
	}
	sort.SliceStable(es, func(i, j int) bool {
		if es[i].ZonePos() != es[j].ZonePos() {
			return es[i].ZonePos() < es[j].ZonePos()
		}
		return es[i].ID < es[j].ID
	})
	out := make([]bgh.Card, 0, len(es))
	for _, e := range es {
		out = append(out, s.card(e))
	}
	return out
}

// board is one seat's row of minions.
func (s *session) board(controller int) []bgh.Card {
	var out []bgh.Card
	for _, c := range s.zoneOf(controller, hslog.ZonePlay, hslog.TypeMinion) {
		out = append(out, c)
	}
	return out
}

// hand is this seat's hand. Only this seat's: an opponent's hand is never sent
// to your client, and a count of it only while you are fighting them.
func (s *session) hand() []bgh.Card {
	return s.zoneOf(s.tab.LocalPlayerID, hslog.ZoneHand,
		hslog.TypeMinion, hslog.TypeSpell)
}

// shop is the row of cards this seat can buy. The tavern belongs to the same
// controller as the opponent's board during a fight, so this is only meaningful
// in a recruit phase — which is the only time it is called.
func (s *session) shop() []bgh.Card {
	var es []*hslog.Entity
	for _, id := range s.tab.Order {
		e := s.tab.Get(id)
		if e.Controller() != s.tab.DummyPlayerID || e.Zone() != hslog.ZonePlay {
			continue
		}
		if e.CardType() != hslog.TypeMinion && e.CardType() != hslog.TypeSpell {
			continue
		}
		if e.ZonePos() < 1 || e.CardID == "" {
			continue
		}
		es = append(es, e)
	}
	sort.SliceStable(es, func(i, j int) bool {
		if es[i].ZonePos() != es[j].ZonePos() {
			return es[i].ZonePos() < es[j].ZonePos()
		}
		return es[i].ID < es[j].ID
	})
	out := make([]bgh.Card, 0, len(es))
	for _, e := range es {
		c := s.card(e)
		if cost := s.buyCost(e.ID); cost > 0 {
			c.Cost = bgh.Int(cost)
		}
		out = append(out, c)
	}
	return out
}

// buyCost finds the price on the buy button that belongs to a shop card. The
// button points at its card through a tag the client prints as a bare number,
// which is why unknown tag names are kept rather than dropped.
func (s *session) buyCost(cardEntity int) int {
	want := strconv.Itoa(cardEntity)
	for _, id := range s.tab.Order {
		e := s.tab.Get(id)
		if e.CardID != hslog.CardDragBuy && e.CardID != hslog.CardDragBuySpell {
			continue
		}
		for tag, v := range e.Tags {
			if v == want && tag != hslog.TagEntityID && tag != hslog.TagController {
				if c := e.Int(hslog.TagCost); c > 0 {
					return c
				}
			}
		}
	}
	return 0
}

// trinkets is the trinkets this seat holds. The client keeps a placeholder for
// each unfilled slot and re-creates it every turn, so a placeholder — which has
// no real card behind it — is skipped.
func (s *session) trinkets() []bgh.Card {
	var out []bgh.Card
	seen := map[string]bool{}
	for _, id := range s.tab.Order {
		e := s.tab.Get(id)
		if e.CardType() != hslog.TypeTrinket || e.Controller() != s.tab.LocalPlayerID {
			continue
		}
		if e.Zone() != hslog.ZonePlay {
			continue
		}
		if e.CardID == hslog.CardTrinketLesser || e.CardID == hslog.CardTrinketGreater {
			continue // an empty slot, not a trinket
		}
		if e.CardID == "" || seen[e.CardID] {
			continue
		}
		seen[e.CardID] = true
		c := s.card(e)
		c.Type = bgh.TypeTrinket
		if c.TrinketTier == "" {
			c.TrinketTier = s.trinketTierOf(e)
		}
		out = append(out, c)
	}
	return out
}

// trinketTierOf works out which slot a taken trinket filled, from the tavern
// tier the slot unlocks at: the lesser slot opens at 4 and the greater at 6.
func (s *session) trinketTierOf(e *hslog.Entity) string {
	if e.Int(hslog.TagTechLevel) >= 6 {
		return bgh.TrinketGreater
	}
	return bgh.TrinketLesser
}

// gold is what this seat has left to spend: the coins it was given this turn
// less what it has spent. Both arrive addressed to the player's display name
// rather than to an entity, which is why they are read from the named-tag side
// table and not from the entity table.
func (s *session) gold() int {
	return max0(s.tab.LocalTagInt(hslog.TagResources) - s.tab.LocalTagInt(hslog.TagResourcesUsed))
}

// healthOf reads a seat's health off the leaderboard, armor folded in, floored
// at zero. It returns -1 when the seat has no hero yet, so a caller can tell
// "not known" from "knocked out".
func (s *session) healthOf(id string) int {
	h := s.heroOf(id)
	if h == nil {
		return -1
	}
	return max0(h.Int(hslog.TagHealth) + h.Int(hslog.TagArmor) - h.Int(hslog.TagDamage))
}

func (s *session) tierOf(id string) int {
	if h := s.heroOf(id); h != nil {
		if t := h.Int(hslog.TagPlayerTechLevel); t > 0 {
			return t
		}
	}
	return 1
}

// techUpTier reads the destination tier out of a tavern-upgrade button's card
// id: the digits in TB_BaconShopTechUp03_Button are the tier it buys.
func techUpTier(cardID string) int {
	rest := strings.TrimPrefix(cardID, hslog.CardTechUpPrefix)
	digits := ""
	for _, r := range rest {
		if r < '0' || r > '9' {
			break
		}
		digits += string(r)
	}
	n, _ := strconv.Atoi(digits)
	return n
}

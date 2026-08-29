package convert

import (
	"strconv"
	"strings"

	"github.com/DanielCoulbourne/battlegrounds-history-data/converter/bgh"
	"github.com/DanielCoulbourne/battlegrounds-history-data/converter/hslog"
)

// onBlockStart turns the client's blocks into actions and combat events.
//
// The client does not print "the player bought a minion". It prints that a
// hidden button card was played at a target, and the meaning is in which button
// it was. Each case below names the button and says what it means.
func (s *session) onBlockStart(ev hslog.Event) {
	if s.inCombat {
		if ev.BlockType == "ATTACK" {
			s.onAttackBlock(ev)
		}
		return
	}
	if !s.inRecruit {
		return
	}

	target := s.tab.Get(ev.Target)
	self := s.tab.Get(ev.EntityID)

	switch {
	// Buying. The player drags a card onto the buy button, so the block is the
	// button and the target is the card.
	case ev.CardID == hslog.CardDragBuy || ev.CardID == hslog.CardDragBuySpell:
		if target == nil {
			return
		}
		d := bgh.Data{
			From: &bgh.Ref{Zone: bgh.ZoneShop, Entity: "e" + strconv.Itoa(target.ID),
				CardID: target.CardID, Index: bgh.Int(max0(target.ZonePos() - 1))},
			To:   &bgh.Ref{Zone: bgh.ZoneHand},
			Gold: bgh.Int(s.gold()),
		}
		if c := s.buyCost(target.ID); c > 0 {
			d.Cost = bgh.Int(c)
		}
		s.act(bgh.ActBuy, d)

	// Selling.
	case ev.CardID == hslog.CardDragSell:
		if target == nil {
			return
		}
		s.act(bgh.ActSell, bgh.Data{Target: s.ref(target), Gold: bgh.Int(s.gold())})

	// Replacing every card in the shop.
	case ev.CardID == hslog.CardRerollButton:
		s.act(bgh.ActRoll, bgh.Data{Gold: bgh.Int(s.gold())})
		s.rolled = true

	// Holding the shop over to the next turn.
	case ev.CardID == hslog.CardFreezeButton:
		verb := bgh.ActFreeze
		if s.shopFrozen() {
			verb = bgh.ActUnfreeze
		}
		s.act(verb, bgh.Data{})

	// Raising the tavern tier. The digits in the button's card id are the tier
	// it buys.
	case strings.HasPrefix(ev.CardID, hslog.CardTechUpPrefix):
		d := bgh.Data{Gold: bgh.Int(s.gold())}
		if t := techUpTier(ev.CardID); t > 0 {
			d.Tier = bgh.Int(t)
		}
		s.act(bgh.ActUpgrade, d)

	// Using the ability printed on the hero.
	case strings.HasPrefix(ev.CardID, hslog.CardHeroPowerPre):
		d := bgh.Data{}
		if target != nil {
			d.Target = s.ref(target)
		}
		s.act(bgh.ActHeroPower, d)

	// Playing a card from hand. The block is the card itself, already under this
	// player's control and coming out of the hand.
	case ev.BlockType == "PLAY" && self != nil &&
		self.Controller() == s.tab.LocalPlayerID && self.CardID != "":
		d := bgh.Data{
			From: &bgh.Ref{Zone: bgh.ZoneHand, Entity: "e" + strconv.Itoa(self.ID),
				CardID: self.CardID, Player: s.me},
			To: &bgh.Ref{Zone: bgh.ZoneBoard},
		}
		if target != nil {
			d.Target = s.ref(target)
		}
		s.act(bgh.ActPlay, d)
	}
}

// onBlockEnd finishes whatever the block was doing. A shop refresh is only
// complete once the block closes, because the new cards arrive inside it.
func (s *session) onBlockEnd(ev hslog.Event) {
	if s.rolled && ev.CardID == hslog.CardRerollButton {
		s.rolled = false
		if shop := s.shop(); len(shop) > 0 {
			s.b.EventFor(s.me, bgh.EvShopDealt).Turn(s.round).Phase(bgh.PhaseRecruit).
				Data(bgh.Data{Reason: "roll", Cards: shop}).Done()
		}
	}
	if s.inCombat && ev.BlockType == "ATTACK" {
		s.attacker, s.defender, s.swingOpen = 0, 0, false
	}
}

// onAttackBlock writes one swing. The damage that follows arrives as separate
// tag changes and is written as it lands, so the order in the file is the order
// the game settled it.
func (s *session) onAttackBlock(ev hslog.Event) {
	s.openFight()
	s.attacker = ev.EntityID
	s.defender = 0
	s.swingOpen = false
	att := s.tab.Get(ev.EntityID)
	// A hero attacking is the fight dealing its damage to a player rather than a
	// minion swinging. The hero_damage that the damage itself raises reports it.
	if att == nil || att.CardType() == hslog.TypeHero {
		return
	}
	s.swingOpen = true
}

// writeAttack records one swing, once both ends of it are known.
func (s *session) writeAttack() {
	if !s.swingOpen {
		return
	}
	att, def := s.tab.Get(s.attacker), s.tab.Get(s.defender)
	if att == nil || def == nil {
		return
	}
	s.swingOpen = false
	s.b.Event(bgh.EvAttack).Turn(s.round).Phase(bgh.PhaseCombat).Data(bgh.Data{
		Combat: s.combatID(), Attacker: s.ref(att), Defender: s.ref(def), Swing: bgh.Int(1),
	}).Done()
}

// act writes one action by this seat. It is the only place an action entry is
// made, which is what keeps the rule that a one-seat recording carries actions
// for one seat.
func (s *session) act(verb string, d bgh.Data) {
	s.b.Action(s.me, verb).Turn(s.round).Phase(bgh.PhaseRecruit).Data(d).Done()
}

func (s *session) shopFrozen() bool {
	for _, id := range s.tab.Order {
		e := s.tab.Get(id)
		if e.Controller() == s.tab.DummyPlayerID && e.Zone() == hslog.ZonePlay &&
			e.CardType() == hslog.TypeMinion && e.Bool(hslog.TagFrozen) {
			return true
		}
	}
	return false
}

// onChoiceOffer records a set of cards the game put in front of the player.
// Every "pick one of these" moment in Battlegrounds arrives this way: the hero
// at the start, a card discovered from making a triple, a trinket, a dark gift.
func (s *session) onChoiceOffer(ev hslog.Event) {
	if src, ok := ev.Fields["Source"]; ok {
		s.offerSource = src
		return
	}
	if t, ok := ev.Fields["ChoiceType"]; ok {
		s.offerType = t
		s.offerN++
		s.offerID = "offer-" + strconv.Itoa(s.offerN)
		s.offerCards = nil
	}
}

func (s *session) onChoiceOption(ev hslog.Event) {
	if s.offerID == "" {
		return
	}
	c := bgh.Card{CardID: ev.CardID, Entity: "e" + strconv.Itoa(ev.EntityID)}
	if ev.CardID == "" {
		c.Unknown = true
	}
	if e := s.tab.Get(ev.EntityID); e != nil {
		c = s.card(e)
	}
	s.offerCards = append(s.offerCards, c)
}

// onChosen writes the offer and the choice together, because the offer is only
// complete once its options have all arrived.
func (s *session) onChosen(ev hslog.Event) {
	if s.offerID == "" || len(s.offerCards) == 0 {
		return
	}
	kind := s.offerKind()
	phase := bgh.PhaseRecruit
	verb := bgh.ActChoose
	if kind == bgh.OfferHero {
		phase, verb = bgh.PhaseSetup, bgh.ActChooseHero
	}

	opts := make([]bgh.OfferOpt, 0, len(s.offerCards))
	picked := 0
	for i, c := range s.offerCards {
		opts = append(opts, bgh.OfferOpt{Cards: []bgh.Card{c}})
		if c.CardID != "" && c.CardID == ev.CardID {
			picked = i
		}
	}
	s.b.EventFor(s.me, bgh.EvOffer).Turn(s.round).Phase(phase).Data(bgh.Data{
		ID: s.offerID, OfferType: kind, Mandatory: true, Options: opts,
	}).Done()
	s.b.Action(s.me, verb).Turn(s.round).Phase(phase).
		Data(bgh.Data{Offer: s.offerID, OptionIndex: bgh.Int(picked)}).Done()

	s.offerID, s.offerCards, s.offerSource, s.offerType = "", nil, "", ""
}

// offerKind names what sort of choice this was. The card that raised the choice
// says which: the two trinket slots have their own card ids, and the hero pick
// comes through the client's mulligan.
func (s *session) offerKind() string {
	switch {
	case strings.Contains(s.offerSource, hslog.CardTrinketLesser):
		return bgh.OfferLesserTrinket
	case strings.Contains(s.offerSource, hslog.CardTrinketGreater):
		return bgh.OfferGreaterTrinket
	case s.offerType == "MULLIGAN":
		return bgh.OfferHero
	case s.offerType == "GENERAL":
		return bgh.OfferDiscover
	}
	return bgh.OfferDiscover
}

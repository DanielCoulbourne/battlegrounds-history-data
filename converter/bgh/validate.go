package bgh

import (
	"fmt"
	"sort"
	"strings"
)

// Validate checks a document against the rules this package can check without a
// JSON Schema validator. It is deliberately the same list the repository's
// tools/validate.mjs enforces, plus the per-verb required fields the schema
// states, so a Go producer fails at the point of its mistake rather than in
// somebody else's validator a week later.
//
// It is not a replacement for validating against schema/. It is the cheap check
// you can afford to run on every file you write.
func (d *Document) Validate() error {
	var p problems

	if d.Format != Format {
		p.addf("format is %q, want %q", d.Format, Format)
	}
	if d.SpecVersion == "" {
		p.add("version is empty")
	}
	if d.Recording.ID == "" {
		p.add("recording.id is empty")
	}

	players := map[string]bool{}
	for _, pl := range d.Players {
		if pl.ID == "" {
			p.add("a player has an empty id")
			continue
		}
		if players[pl.ID] {
			p.addf("two players share the id %q", pl.ID)
		}
		players[pl.ID] = true
	}
	if len(d.Players) == 0 {
		p.add("players is empty")
	}
	for _, pl := range d.Players {
		if pl.Teammate != "" && !players[pl.Teammate] {
			p.addf("player %q names teammate %q, who is not a player in this file", pl.ID, pl.Teammate)
		}
		checkCard(&p, pl.Hero, "player "+pl.ID+" hero")
		checkCard(&p, pl.HeroPower, "player "+pl.ID+" heroPower")
	}

	scope := d.Recording.Observer.Scope
	seat := d.Recording.Observer.Seat
	switch scope {
	case ScopeLobby:
		if seat != "" {
			p.add("a lobby recording names an observer.seat; leave it out")
		}
	case ScopeSeat:
		if seat == "" {
			p.add("a seat recording must name observer.seat")
		} else if !players[seat] {
			p.addf("observer.seat %q is not a player in this file", seat)
		}
	default:
		p.addf("observer.scope is %q, want %q or %q", scope, ScopeLobby, ScopeSeat)
	}

	offers := map[string]int{}
	entities := map[string]string{}
	prev := -1

	for i := range d.History {
		e := &d.History[i]
		where := fmt.Sprintf("entry seq %d", e.Seq)

		if e.Seq <= prev {
			p.addf("%s does not come after %d", where, prev)
		}
		prev = e.Seq

		if e.Actor != "" && !players[e.Actor] {
			p.addf("%s: actor %q is not a player in this file", where, e.Actor)
		}

		switch e.Kind {
		case KindAction:
			if e.Actor == "" {
				p.addf("%s: an action needs an actor", where)
			}
			if scope == ScopeSeat && e.Actor != seat {
				p.addf("%s: this is a recording of seat %q and carries an action for %q; "+
					"you cannot have watched another player decide", where, seat, e.Actor)
			}
			checkWord(&p, where, "action", e.Action, actionVerbs)
			checkNeeds(&p, where, e.Action, e.Data, actionNeeds)

		case KindEvent:
			checkWord(&p, where, "event", e.Event, eventVerbs)
			checkNeeds(&p, where, e.Event, e.Data, eventNeeds)
			if e.Event == EvOffer && e.Data != nil {
				if _, dup := offers[e.Data.ID]; dup {
					p.addf("%s: offer id %q was already used", where, e.Data.ID)
				}
				offers[e.Data.ID] = len(e.Data.Options)
			}
			if e.Data != nil && e.Data.OfferType != "" {
				checkWord(&p, where, "offerType", e.Data.OfferType, offerTypes)
			}

		case KindState:
			if e.Actor == "" {
				p.addf("%s: a state entry needs an actor", where)
			}

		default:
			p.addf("%s: kind is %q, want action, event or state", where, e.Kind)
		}

		if e.Kind == KindAction && e.Data != nil {
			switch e.Action {
			case ActChoose, ActChooseHero, ActOpenOffer:
				n, ok := offers[e.Data.Offer]
				if !ok {
					p.addf("%s: points at offer %q, which no earlier entry made", where, e.Data.Offer)
				} else if e.Action != ActOpenOffer && e.Data.OptionIndex != nil && *e.Data.OptionIndex >= n {
					p.addf("%s: offer %q has %d options, so %d is out of range",
						where, e.Data.Offer, n, *e.Data.OptionIndex)
				}
			}
		}

		for _, c := range entryCards(e) {
			checkCard(&p, c, where)
			if c.Entity == "" || c.CardID == "" {
				continue
			}
			if was, ok := entities[c.Entity]; ok && was != c.CardID {
				p.addf("%s: entity %q was %s and is now %s; an entity name stays with one copy",
					where, c.Entity, was, c.CardID)
			} else if !ok {
				entities[c.Entity] = c.CardID
			}
		}
	}

	return p.err()
}

func checkCard(p *problems, c *Card, where string) {
	if c == nil {
		return
	}
	if c.CardID == "" && c.DbfID == 0 && !c.Unknown {
		p.addf("%s: a card has no cardId and is not marked unknown", where)
	}
	for _, k := range c.Keywords {
		checkWord(p, where, "keyword", k, keywords)
	}
	for _, m := range c.MinionTypes {
		checkWord(p, where, "minionType", m, minionTypes)
	}
	if c.TrinketTier != "" && c.TrinketTier != TrinketLesser && c.TrinketTier != TrinketGreater &&
		!strings.HasPrefix(c.TrinketTier, "x-") {
		p.addf("%s: trinketTier is %q, want %q or %q", where, c.TrinketTier, TrinketLesser, TrinketGreater)
	}
}

// checkWord enforces the rule that keeps the vocabulary readable: a word this
// specification does not list must announce itself with an "x-" prefix.
func checkWord(p *problems, where, what, word string, known map[string]bool) {
	if word == "" {
		p.addf("%s: %s is empty", where, what)
		return
	}
	if known[word] || strings.HasPrefix(word, "x-") {
		return
	}
	p.addf("%s: %s %q is not in the vocabulary; a word this specification does not "+
		"list must start with \"x-\"", where, what, word)
}

func checkNeeds(p *problems, where, verb string, d *Data, needs map[string][]string) {
	req, ok := needs[verb]
	if !ok {
		return
	}
	if d == nil {
		p.addf("%s: %q needs data.%s", where, verb, strings.Join(req, ", data."))
		return
	}
	for _, field := range req {
		if !dataHas(d, field) {
			p.addf("%s: %q needs data.%s", where, verb, field)
		}
	}
}

func dataHas(d *Data, field string) bool {
	switch field {
	case "from":
		return d.From != nil
	case "to":
		return d.To != nil
	case "target":
		return d.Target != nil
	case "offer":
		return d.Offer != ""
	case "option":
		return d.OptionIndex != nil
	case "id":
		return d.ID != ""
	case "options":
		return d.Options != nil
	case "sides":
		return len(d.Sides) > 0
	case "attacker":
		return d.Attacker != nil
	case "defender":
		return d.Defender != nil
	case "amount":
		return d.Amount != nil
	case "card":
		return d.Card != nil
	}
	return false
}

// entryCards gathers every card an entry mentions, however deeply nested.
func entryCards(e *Entry) []*Card {
	var out []*Card
	var add func(cs []Card)
	add = func(cs []Card) {
		for i := range cs {
			out = append(out, &cs[i])
			add(cs[i].Attached)
		}
	}
	one := func(c *Card) {
		if c != nil {
			out = append(out, c)
			add(c.Attached)
		}
	}

	for _, z := range e.Zones {
		add(z.Cards)
	}
	for _, s := range e.Standings {
		one(s.Hero)
	}
	if d := e.Data; d != nil {
		add(d.Cards)
		one(d.Card)
		for _, o := range d.Options {
			add(o.Cards)
		}
		for _, s := range d.Sides {
			add(s.Board)
			one(s.Hero)
			one(s.HeroPower)
			if s.Hand != nil {
				add(s.Hand.Cards)
			}
		}
		for _, s := range d.Standings {
			one(s.Hero)
		}
	}
	return out
}

type problems struct{ list []string }

func (p *problems) add(s string)            { p.list = append(p.list, s) }
func (p *problems) addf(f string, a ...any) { p.list = append(p.list, fmt.Sprintf(f, a...)) }

func (p *problems) err() error {
	if len(p.list) == 0 {
		return nil
	}
	sort.Strings(p.list)
	return fmt.Errorf("bgh: %d problem(s):\n  %s", len(p.list), strings.Join(p.list, "\n  "))
}

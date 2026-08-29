package convert

import (
	"strconv"
	"strings"

	"github.com/DanielCoulbourne/battlegrounds-history-data/converter/bgh"
	"github.com/DanielCoulbourne/battlegrounds-history-data/converter/hslog"
)

// onTagChange handles the tags that move the game forward. Everything else has
// already gone into the entity table and is read from there when needed.
func (s *session) onTagChange(ev hslog.Event) {
	// The game entity carries the structure of the game: turns, phases, and the
	// end.
	if ev.EntityID == s.tab.GameEntityID || ev.EntityName == "GameEntity" {
		switch ev.Tag {
		case hslog.TagStep:
			switch ev.Value {
			case "MAIN_ACTION":
				s.beginRecruit()
			case "FINAL_WRAPUP":
				s.endGame()
			}
		case hslog.TagBoardVisualState:
			switch ev.Value {
			case "2":
				s.beginCombat()
			case "1":
				s.endCombat()
			}
		case hslog.TagState:
			if ev.Value == "COMPLETE" {
				s.endGame()
			}
		case hslog.TagProposedAttacker:
			s.attacker, _ = strconv.Atoi(ev.Value)
		case hslog.TagProposedDefender:
			// The defender of the swing now opening. It is announced on the game
			// entity a line or two AFTER the block opens, which is why the swing
			// is written here rather than at the block start: this is the first
			// moment both ends of it are known, and it is still before the damage.
			s.defender, _ = strconv.Atoi(ev.Value)
			s.writeAttack()
		}
		return
	}

	// Tags addressed to a display name rather than an entity. Your gold comes
	// this way, and so does the announcement of who is fighting whom.
	if ev.EntityID == 0 && ev.EntityName != "" {
		if ev.EntityName == s.tab.LocalName && ev.Tag == hslog.TagPlaystate &&
			(ev.Value == "LOST" || ev.Value == "WON") {
			s.eliminate(s.me)
			return
		}
		if ev.EntityName == s.tab.LocalName && ev.Tag == hslog.TagNextOpponent {
			if n, err := strconv.Atoi(ev.Value); err == nil && n >= 1 && n <= 8 {
				opp := seat(n)
				s.player(opp)
				s.b.EventFor(s.me, bgh.EvPairingAnnounced).Turn(s.round).
					Phase(bgh.PhaseRecruit).
					Data(bgh.Data{Opponent: opp}).Done()
			}
		}
		return
	}

	e := s.tab.Get(ev.EntityID)
	if e == nil {
		return
	}

	switch ev.Tag {
	case hslog.TagPlaystate:
		if ev.Value == "LOST" || ev.Value == "WON" {
			s.onPlaystate(e, ev.Value)
		}
	case hslog.TagProposedDefender:
		if n, err := strconv.Atoi(ev.Value); err == nil {
			s.defender = n
		}
	case hslog.TagDamage:
		s.onDamage(e, ev)
	case hslog.TagZone:
		if ev.Value == hslog.ZoneGraveyard && s.inCombat && s.inBlock("DEATHS") {
			s.b.Event(bgh.EvDeath).Turn(s.round).Phase(bgh.PhaseCombat).
				Data(bgh.Data{
					Combat: s.combatID(),
					Target: s.ref(e),
					Killer: s.refID(e.Int(hslog.TagLastAffectedBy)),
				}).Done()
		}
	case hslog.TagDivineShield:
		if s.inCombat && ev.Value == "0" {
			s.b.Event(bgh.EvDivineShieldLost).Turn(s.round).Phase(bgh.PhaseCombat).
				Data(bgh.Data{Combat: s.combatID(), Target: s.ref(e)}).Done()
		}
	}
}

// onDamage writes one damage event. The log reports a minion's running total,
// so the size of this hit is the rise since the last total — which is why the
// old value is read before the table is updated.
func (s *session) onDamage(e *hslog.Entity, ev hslog.Event) {
	if !s.inCombat || s.damageDelta <= 0 {
		return
	}
	lethal := false
	if hp := e.Int(hslog.TagHealth); hp > 0 {
		lethal = e.Int(hslog.TagDamage) >= hp
	}
	d := bgh.Data{
		Combat: s.combatID(),
		Target: s.ref(e),
		Amount: bgh.Int(s.damageDelta),
		Lethal: bgh.Bool(lethal),
	}
	if src := s.refID(e.Int(hslog.TagLastAffectedBy)); src != nil {
		d.Source = src
	}
	if e.CardType() == hslog.TypeHero {
		// Damage to a hero is what the fight cost, reported as it lands.
		s.b.Event(bgh.EvHeroDamage).Turn(s.round).Phase(bgh.PhaseCombat).
			Data(bgh.Data{Combat: s.combatID(), PlayerID: s.seatOfHero(e),
				Amount: bgh.Int(s.damageDelta)}).Done()
		return
	}
	s.b.Event(bgh.EvDamage).Turn(s.round).Phase(bgh.PhaseCombat).Data(d).Done()
}

func (s *session) onPlaystate(e *hslog.Entity, value string) {
	if e.Int(hslog.TagPlayerID) == 0 || value != "LOST" {
		return
	}
	s.eliminate(seat(e.Int(hslog.TagPlayerID)))
}

// eliminate records a seat leaving the game, with the place it finished in.
// The log announces the local player's exit by display name and everybody
// else's on their hero entity, so both routes arrive here.
func (s *session) eliminate(id string) {
	if s.gone[id] {
		return
	}
	s.gone[id] = true
	p := s.player(id)
	place := 0
	if h := s.heroOf(id); h != nil {
		place = h.Int(hslog.TagLeaderboardPlace)
	}
	d := bgh.Data{PlayerID: id, Turn: bgh.Int(s.round)}
	if place > 0 {
		d.Placement = bgh.Int(place)
		if p.Result == nil {
			p.Result = &bgh.Result{}
		}
		p.Result.Placement = bgh.Int(place)
	}
	p.Result.EliminatedOnTurn = bgh.Int(s.round)
	s.b.Event(bgh.EvPlayerEliminated).Turn(s.round).Phase(bgh.PhaseCombat).Data(d).Done()
}

// beginRecruit opens a recruit phase: the shopping half of a turn, and the only
// half this seat can act in.
func (s *session) beginRecruit() {
	if !s.inRecruit {
		s.round++
	}
	if s.inRecruit {
		return
	}
	s.inRecruit = true
	s.syncAll()

	s.b.Event(bgh.EvTurnStart).Turn(s.round).Phase(bgh.PhaseRecruit).
		Data(bgh.Data{Turn: bgh.Int(s.round), Gold: bgh.Int(s.gold())}).Done()

	if shop := s.shop(); len(shop) > 0 {
		s.b.EventFor(s.me, bgh.EvShopDealt).Turn(s.round).Phase(bgh.PhaseRecruit).
			Data(bgh.Data{Reason: "turnStart", Cards: shop}).Done()
	}
	s.writeState("turnStart")
	s.healthBefore[s.me] = s.healthOf(s.me)
}

// writeState records a full picture of this seat.
func (s *session) writeState(reason string) {
	eb := s.b.State(s.me, reason).Turn(s.round).Phase(bgh.PhaseRecruit).
		Seat(bgh.SeatState{
			Health: bgh.Int(s.healthOf(s.me)),
			Tier:   bgh.Int(s.tierOf(s.me)),
			Gold:   bgh.Int(s.gold()),
			Alive:  bgh.Bool(true),
		}).
		Zone(bgh.ZoneBoard, bgh.Zone{Cards: s.board(s.tab.LocalPlayerID), Capacity: bgh.Int(7)}).
		Zone(bgh.ZoneHand, bgh.Zone{Cards: s.hand(), Capacity: bgh.Int(10)}).
		Zone(bgh.ZoneTrinkets, bgh.Zone{Cards: s.trinkets()}).
		Standings(s.standings())
	eb.Done()
}

// standings is the leaderboard: one row per seat, carrying the hero, because
// the leaderboard shows every seat's hero portrait and that is very nearly the
// only durable fact this recording gets about the other seven.
func (s *session) standings() []bgh.Standing {
	var rows []bgh.Standing
	for _, p := range s.b.Document().Players {
		h := s.heroOf(p.ID)
		row := bgh.Standing{Player: p.ID}
		if h == nil {
			row.Health, row.Tier = bgh.Unknown(), bgh.Unknown()
			rows = append(rows, row)
			continue
		}
		row.Hero = &bgh.Card{Entity: "e" + strconv.Itoa(h.ID), CardID: h.CardID, Type: bgh.TypeHero}
		hp := max0(h.Int(hslog.TagHealth) + h.Int(hslog.TagArmor) - h.Int(hslog.TagDamage))
		row.Health = bgh.Known(hp)
		if t := h.Int(hslog.TagPlayerTechLevel); t > 0 {
			row.Tier = bgh.Known(t)
		} else {
			row.Tier = bgh.Unknown()
		}
		row.Alive = bgh.Bool(hp > 0)
		if pl := h.Int(hslog.TagLeaderboardPlace); pl > 0 {
			row.Placement = bgh.Int(pl)
		}
		rows = append(rows, row)
	}
	return rows
}

// beginCombat closes the recruit phase. The fight itself is not written until
// its first attack: at the moment the phase flag flips, the previous shop is
// still standing under the same controller the opponent's board will use, and
// reading it then is how a converter reports a shop as an enemy board.
func (s *session) beginCombat() {
	if s.inCombat {
		return
	}
	s.inRecruit = false
	s.inCombat = true
	s.combatNum++
	s.fightOpen = false
	s.healthBefore[s.me] = s.healthOf(s.me)
	if opp := s.currentOpponent(); opp != "" {
		s.healthBefore[opp] = s.healthOf(opp)
	}
}

// openFight writes combat_start, with both boards as they were sent in.
func (s *session) openFight() {
	if s.fightOpen {
		return
	}
	s.fightOpen = true
	opp := s.currentOpponent()
	if opp != "" {
		s.player(opp)
	}

	mine := bgh.Side{
		Player: s.me,
		Tier:   bgh.Int(s.tierOf(s.me)),
		Board:  s.board(s.tab.LocalPlayerID),
	}
	if hb, ok := s.healthBefore[s.me]; ok {
		mine.HealthBefore = bgh.Known(hb)
	}
	theirs := bgh.Side{Player: opp, Board: s.board(s.tab.DummyPlayerID)}
	if h := s.combatHero(); h != nil {
		theirs.Hero = &bgh.Card{Entity: "e" + strconv.Itoa(h.ID), CardID: h.CardID, Type: bgh.TypeHero}
		if t := h.Int(hslog.TagPlayerTechLevel); t > 0 {
			theirs.Tier = bgh.Int(t)
		}
		theirs.HealthBefore = bgh.Known(max0(h.Int(hslog.TagHealth) + h.Int(hslog.TagArmor) - h.Int(hslog.TagDamage)))
	} else {
		theirs.HealthBefore = bgh.Unknown()
	}

	s.b.Event(bgh.EvCombatStart).Turn(s.round).Phase(bgh.PhaseCombat).
		Data(bgh.Data{Combat: s.combatID(), Sides: []bgh.Side{mine, theirs}}).Done()
}

// endCombat writes combat_end. What this seat lost is exact. What the opponent
// lost is only knowable once the leaderboard shows it, so it is null when it is
// not yet known rather than guessed at.
func (s *session) endCombat() {
	if !s.inCombat {
		return
	}
	s.inCombat = false
	if !s.fightOpen {
		// A fight with no attacks: both boards were empty, or the log lost it.
		return
	}
	s.fightOpen = false
	s.syncAll()

	opp := s.currentOpponent()
	before := s.healthBefore[s.me]
	after := s.healthOf(s.me)

	mine := bgh.Side{
		Player: s.me, HealthBefore: bgh.Known(before), HealthAfter: bgh.Known(after),
		DamageTaken: bgh.Known(max0(before - after)),
		Eliminated:  after == 0,
	}
	theirs := bgh.Side{Player: opp}
	if opp != "" {
		ob, ok := s.healthBefore[opp]
		oa := s.healthOf(opp)
		if ok && oa >= 0 {
			theirs.HealthBefore = bgh.Known(ob)
			theirs.HealthAfter = bgh.Known(oa)
			theirs.DamageTaken = bgh.Known(max0(ob - oa))
			theirs.Eliminated = oa == 0
		} else {
			theirs.HealthBefore, theirs.HealthAfter, theirs.DamageTaken =
				bgh.Unknown(), bgh.Unknown(), bgh.Unknown()
		}
	}

	d := bgh.Data{Combat: s.combatID(), Sides: []bgh.Side{mine, theirs}}
	// The client states the result outright rather than leaving it to be worked
	// out from health, which matters for a draw: neither side loses health, and
	// that is not the same as nobody winning by chance.
	switch s.tab.NamedTags[s.tab.LocalName][hslog.TagWonLastCombat] {
	case "1":
		d.Winner = bgh.Str(s.me)
	case "0":
		if max0(before-after) > 0 && opp != "" {
			d.Winner = bgh.Str(opp)
		} else {
			d.Winner = nil // a draw, or a result the log did not state
		}
	}
	s.b.Event(bgh.EvCombatEnd).Turn(s.round).Phase(bgh.PhaseCombat).Data(d).Done()
}

func (s *session) endGame() {
	if s.ended {
		return
	}
	s.ended = true
	s.endCombat()
	s.syncAll()

	var places []bgh.Placement
	for _, p := range s.b.Document().Players {
		if p.Result != nil && p.Result.Placement != nil {
			places = append(places, bgh.Placement{Player: p.ID, Placement: p.Result.Placement})
		}
	}
	s.b.Event(bgh.EvGameEnd).Turn(s.round).Phase(bgh.PhaseEnd).
		Data(bgh.Data{Placements: places, Standings: s.standings()}).Done()
}

// finish closes the document. A game the log caught only part of is marked
// truncated rather than dropped: a partial recording is still a recording, and
// saying so is the whole point of the field.
func (s *session) finish() *bgh.Document {
	if !s.seatBound {
		// The log never named the local player, so there is nothing to anchor a
		// seat recording to. Return an empty document the caller can skip.
		return nil
	}
	if !s.ended {
		s.b.Truncate("The log ends before this game did.")
	}
	s.syncAll()
	return s.b.Document()
}

func (s *session) combatID() string { return "c" + strconv.Itoa(s.combatNum) }

func (s *session) inBlock(kind string) bool {
	for _, b := range s.blocks {
		if b.BlockType == kind {
			return true
		}
	}
	return false
}

// currentOpponent reads who this seat is fighting. The client announces it on
// the combat hero rather than by name, which is the reliable route: the display
// name only appears for the two players in the fight.
func (s *session) currentOpponent() string {
	if h := s.combatHero(); h != nil {
		if n := h.Int(hslog.TagPlayerID); n >= 1 && n <= 8 {
			return seat(n)
		}
	}
	return ""
}

func (s *session) combatHero() *hslog.Entity {
	var best *hslog.Entity
	for _, id := range s.tab.Order {
		e := s.tab.Get(id)
		if e.CardType() == hslog.TypeHero && e.Bool(hslog.TagCombatPhaseHero) &&
			e.Zone() == hslog.ZonePlay && !strings.HasPrefix(e.CardID, hslog.CardBobSkinPre) {
			best = e
		}
	}
	return best
}

func (s *session) seatOfHero(e *hslog.Entity) string {
	if n := e.Int(hslog.TagPlayerID); n >= 1 && n <= 8 {
		return seat(n)
	}
	return ""
}

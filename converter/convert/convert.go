// Package convert turns a Hearthstone Power.log into Battlegrounds History
// Data files, one per game.
//
// # What a client log can and cannot tell you
//
// The log records what your player did and what the game showed you. It records
// nothing about what anybody else decided. So every file this package writes is
// a one-seat recording: observer.scope is "seat", action entries exist for your
// seat alone, and the other seven seats are reached only through events and
// state entries. That is not a limitation of this converter. It is what the
// client knows.
//
// What you do get about the other seats is the leaderboard, and it is better
// than it sounds: every seat's hero, health, armor, tavern tier and placement,
// live, all game. Their heroes are recorded for that reason.
//
// What you get about an opponent's board is one snapshot per fight you were in,
// taken at the first attack. The game forgets it immediately afterwards, so it
// is written down at the moment it is true rather than derived later.
package convert

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/DanielCoulbourne/battlegrounds-history-data/converter/bgh"
	"github.com/DanielCoulbourne/battlegrounds-history-data/converter/hslog"
)

// Options control one conversion.
type Options struct {
	// IncludeNames writes each seat's display name into the file. It is off by
	// default: a battletag identifies a real person, and a recording is more
	// often shared than a log is.
	IncludeNames bool

	// Recorder names the program in the output. A caller that wraps this
	// package should say so here.
	Recorder bgh.Recorder
}

// Convert reads a whole log and returns one document per game it contains.
// A log holds every game since the client started, so this usually returns
// several, and a game the log only caught the middle of comes back marked
// truncated rather than dropped.
func Convert(r io.Reader, opt Options) ([]*bgh.Document, error) {
	var out []*bgh.Document
	var cur *session

	// The client prints a game's build number and the local player's name just
	// BEFORE it prints CREATE_GAME. Those lines belong to the game that is about
	// to start, so they are held and replayed into it. Without this, the first
	// game in every log comes out twice: once as an empty preamble and once for
	// real.
	var preamble []hslog.Event
	const maxPreamble = 32

	add := func(s *session) {
		if s == nil {
			return
		}
		if doc := s.finish(); doc != nil {
			out = append(out, doc)
		}
	}

	err := hslog.Split(r, func(ev hslog.Event) {
		switch {
		case ev.Kind == hslog.KindCreateGame:
			add(cur)
			cur = newSession(opt)
			for _, p := range preamble {
				cur.feed(p)
			}
			preamble = nil

		case ev.Kind == hslog.KindGameInfo:
			preamble = append(preamble, ev)
			if len(preamble) > maxPreamble {
				preamble = preamble[len(preamble)-maxPreamble:]
			}
			if cur == nil {
				return // nothing to attach it to yet
			}

		case cur == nil:
			// The log begins part way through a game, which is normal: a log
			// holds every game since the client started and can be rotated
			// mid-match. Record it anyway and say that it is partial.
			cur = newSession(opt)
			cur.startedMidGame = true
			for _, p := range preamble {
				cur.feed(p)
			}
		}
		cur.feed(ev)
	})
	if err != nil {
		return nil, fmt.Errorf("convert: read log: %w", err)
	}
	add(cur)
	return out, nil
}

// session converts one game.
type session struct {
	opt Options
	tab *hslog.Table
	b   *bgh.Builder

	startedMidGame bool
	seatBound      bool

	round     int // the visible round number, counted from the recruit phases
	inRecruit bool
	inCombat  bool
	ended     bool
	combatNum int
	fightOpen bool // a combat_start has been written for this fight
	rolled    bool // a shop refresh is in flight; its new cards arrive in the block

	// The seat this recording is of.
	me string

	// Where the last recruit phase left the seat, so a fight can report what it
	// cost without waiting for the leaderboard to catch up.
	healthBefore map[string]int

	// Per-fight bookkeeping.
	attacker, defender int
	swingOpen          bool // an ATTACK block is open and its swing not yet written
	// damageDelta is the size of the hit the current tag change reports. The
	// log prints a minion's running damage total, so the hit is the rise since
	// the last total, and the old total has to be read before the table is
	// updated.
	damageDelta int

	// Blocks currently open, innermost last.
	blocks []hslog.Event

	// The offer waiting for its answer. An offer is only complete once its
	// options have arrived, so it is held until the choice comes back.
	offerID     string
	offerType   string
	offerSource string
	offerCards  []bgh.Card
	offerN      int

	seen map[string]bool // player ids added
	gone map[string]bool // seats already reported as knocked out
}

func newSession(opt Options) *session {
	s := &session{
		opt:          opt,
		tab:          hslog.NewTable(),
		healthBefore: map[string]int{},
		seen:         map[string]bool{},
		gone:         map[string]bool{},
	}
	return s
}

// seat names a lobby seat. PLAYER_ID is 1..8 and is stable for the game, which
// is what a player id in the output needs to be.
func seat(playerID int) string { return "p" + strconv.Itoa(playerID) }

func (s *session) feed(ev hslog.Event) {
	// Read the old damage total before the table overwrites it, so the size of
	// this hit is known when the event is written.
	s.damageDelta = 0
	if ev.Kind == hslog.KindTagChange && ev.Tag == hslog.TagDamage {
		if was := s.tab.Get(ev.EntityID); was != nil {
			now, _ := strconv.Atoi(ev.Value)
			s.damageDelta = now - was.Int(hslog.TagDamage)
		}
	}
	s.tab.Apply(ev)

	// The builder cannot start until the local seat is known, because a seat
	// recording is defined by whose seat it is.
	if !s.seatBound {
		s.tryBindSeat()
		if !s.seatBound {
			return
		}
	}

	switch ev.Kind {
	case hslog.KindTagChange:
		s.onTagChange(ev)
	case hslog.KindBlockStart:
		s.blocks = append(s.blocks, ev)
		s.onBlockStart(ev)
	case hslog.KindBlockEnd:
		if n := len(s.blocks); n > 0 {
			s.onBlockEnd(s.blocks[n-1])
			s.blocks = s.blocks[:n-1]
		}
	case hslog.KindChoiceOffer:
		s.onChoiceOffer(ev)
	case hslog.KindChoiceOption:
		s.onChoiceOption(ev)
	case hslog.KindChosen:
		s.onChosen(ev)
	}
}

// tryBindSeat waits until the log has named the local player, then opens the
// document. DebugPrintGame names them outright, which is a better source than
// guessing from the order display names appear in: Bob and several opponents
// are addressed by name too.
func (s *session) tryBindSeat() {
	if s.tab.LocalPlayerID == 0 {
		return
	}
	// PLAYER_ID on the local hero is the lobby seat; the controller id is not.
	me := ""
	for _, id := range s.tab.Order {
		e := s.tab.Get(id)
		if e.CardType() == hslog.TypeHero && e.Controller() == s.tab.LocalPlayerID &&
			e.Int(hslog.TagPlayerID) > 0 {
			me = seat(e.Int(hslog.TagPlayerID))
			break
		}
	}
	if me == "" {
		return
	}
	s.me = me
	s.seatBound = true

	rec := s.opt.Recorder
	if rec.Name == "" {
		rec = bgh.Recorder{Name: "bgh-convert", Kind: bgh.RecorderTracker,
			URL: "https://github.com/DanielCoulbourne/battlegrounds-history-data"}
	}
	s.b = bgh.NewSeatRecording(s.gameID()+"-"+s.me, s.me).
		Recorder(rec).
		Detail(bgh.Detail{
			Combat:   bgh.CombatEvents,
			States:   bgh.StatesTurnStart,
			Entities: true,
		}).
		Note("Converted from a Hearthstone client log. A client log records what " +
			"this player did and what the game showed them, and nothing about what " +
			"the other seats decided.")
	s.b.Game(bgh.Game{
		ID:        s.gameID(),
		Mode:      bgh.ModeSolo,
		Patch:     s.tab.Build,
		SeatCount: 8,
	})
	if s.startedMidGame {
		s.b.Truncate("The log began part way through this game, so the earlier turns are missing.")
	}
	s.player(s.me)
	s.b.Event(bgh.EvGameStart).Phase(bgh.PhaseSetup).Done()
}

func (s *session) gameID() string {
	if g := s.tab.Get(s.tab.GameEntityID); g != nil {
		if seed := g.Tag(hslog.TagGameSeed); seed != "" {
			return "hs-" + seed
		}
	}
	return "hs-unknown"
}

// player registers a seat the first time it is mentioned, and returns it.
func (s *session) player(id string) *bgh.Player {
	if p := s.b.PlayerRef(id); p != nil {
		return p
	}
	n, _ := strconv.Atoi(strings.TrimPrefix(id, "p"))
	kind := bgh.KindUnknown
	if id == s.me {
		kind = bgh.KindHuman
	}
	p := s.b.AddPlayer(bgh.Player{ID: id, Seat: bgh.Int(n - 1), Kind: kind})
	s.seen[id] = true
	s.syncPlayer(p)
	return p
}

// heroOf finds a seat's hero entity. Every seat's hero sits under the dummy
// controller, set aside, carrying its PLAYER_ID — that is the leaderboard, and
// it is where an opponent's hero comes from.
func (s *session) heroOf(id string) *hslog.Entity {
	want, _ := strconv.Atoi(strings.TrimPrefix(id, "p"))
	var best *hslog.Entity
	for _, eid := range s.tab.Order {
		e := s.tab.Get(eid)
		if e.CardType() != hslog.TypeHero || e.Int(hslog.TagPlayerID) != want {
			continue
		}
		if strings.HasPrefix(e.CardID, hslog.CardBobSkinPre) ||
			e.CardID == hslog.CardHeroPlacehldr || e.CardID == "" {
			continue
		}
		// A hero spawned for the current fight is a copy. Prefer the
		// leaderboard original, which is the one that is not a combat hero.
		if e.Bool(hslog.TagCombatPhaseHero) && best != nil {
			continue
		}
		best = e
	}
	return best
}

// syncPlayer copies whatever the leaderboard currently says about a seat onto
// its player entry. Heroes and results are learned late, so this runs often.
func (s *session) syncPlayer(p *bgh.Player) {
	h := s.heroOf(p.ID)
	if h == nil {
		return
	}
	if p.Hero == nil {
		p.Hero = &bgh.Card{Entity: "e" + strconv.Itoa(h.ID), CardID: h.CardID, Type: bgh.TypeHero}
	}
	if p.StartingHealth == nil && h.Int(hslog.TagHealth) > 0 {
		p.StartingHealth = bgh.Int(h.Int(hslog.TagHealth))
		if a := h.Int(hslog.TagArmor); a > 0 {
			p.StartingArmor = bgh.Int(a)
		}
	}
	if p.Result == nil {
		p.Result = &bgh.Result{}
	}
	p.Result.Health = bgh.Int(max0(h.Int(hslog.TagHealth) + h.Int(hslog.TagArmor) - h.Int(hslog.TagDamage)))
	if t := h.Int(hslog.TagPlayerTechLevel); t > 0 {
		p.Result.Tier = bgh.Int(t)
	}
	if pl := h.Int(hslog.TagLeaderboardPlace); pl > 0 {
		p.Result.Placement = bgh.Int(pl)
	}
	// An opponent's trinkets arrive as database ids rather than card ids. That
	// number IS the card's identity, so it is recorded as one: a reader with a
	// card database can resolve it, and one without still knows which card it
	// is talking about.
	if p.ID != s.me {
		p.Result.Trinkets = nil
		for _, t := range []struct {
			tag  string
			tier string
		}{
			{hslog.TagFirstTrinketDBF, bgh.TrinketLesser},
			{hslog.TagSecondTrinketDBF, bgh.TrinketGreater},
		} {
			if dbf := h.Int(t.tag); dbf > 0 {
				p.Result.Trinkets = append(p.Result.Trinkets, bgh.Card{
					Type: bgh.TypeTrinket, TrinketTier: t.tier, DbfID: dbf,
				})
			}
		}
	}
}

func (s *session) syncAll() {
	// Every seat the leaderboard knows about, so an opponent's hero is recorded
	// even if this seat never fights them.
	for _, eid := range s.tab.Order {
		e := s.tab.Get(eid)
		if e.CardType() != hslog.TypeHero {
			continue
		}
		if n := e.Int(hslog.TagPlayerID); n >= 1 && n <= 8 {
			s.player(seat(n))
		}
	}
	for i := range s.b.Document().Players {
		s.syncPlayer(&s.b.Document().Players[i])
	}
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

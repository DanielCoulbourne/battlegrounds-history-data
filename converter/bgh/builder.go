package bgh

import "time"

// Builder assembles a Document. It owns seq numbering, so entries come out
// strictly in order with no repeats — one of the rules a JSON Schema cannot
// check, removed from the caller's hands entirely.
//
// A Builder is not safe for use from more than one goroutine.
type Builder struct {
	doc  Document
	seq  int
	seen map[string]bool // player ids, so a typo is caught at Validate
}

// New starts a lobby-wide recording: one that could see every seat. Only a
// program that runs the whole game, or reads a complete server-side replay, may
// honestly claim this.
func New(id string) *Builder { return newBuilder(id, Observer{Scope: ScopeLobby}) }

// NewSeatRecording starts a one-seat recording: what a program watching a live
// client produces. Everything in the file must be something that seat could
// have observed, and only that seat may have action entries.
func NewSeatRecording(id, seat string) *Builder {
	return newBuilder(id, Observer{Scope: ScopeSeat, Seat: seat})
}

func newBuilder(id string, obs Observer) *Builder {
	return &Builder{
		doc: Document{
			Format:       Format,
			SpecVersion:  Version,
			CardIDScheme: "hearthstone",
			Recording: Record{
				ID:         id,
				RecordedAt: time.Now().UTC().Format(time.RFC3339),
				Observer:   obs,
			},
		},
		seen: map[string]bool{},
	}
}

// Recorder names the program producing the file.
func (b *Builder) Recorder(r Recorder) *Builder { b.doc.Recording.Recorder = &r; return b }

// Detail states how much the file contains. Set it: without it, a reader has to
// read an absence as a fact, and it will read it wrong.
func (b *Builder) Detail(d Detail) *Builder { b.doc.Recording.Detail = &d; return b }

// Truncate marks a recording that stops before the game did, and says why. A
// log that began mid-game is the common case and is not a failure.
func (b *Builder) Truncate(reason string) *Builder {
	b.doc.Recording.Truncated = true
	b.doc.Recording.TruncationReason = reason
	return b
}

// Note attaches a sentence for a person. Never authoritative.
func (b *Builder) Note(s string) *Builder { b.doc.Recording.Note = s; return b }

// Game sets the lobby-wide facts.
func (b *Builder) Game(g Game) *Builder { b.doc.Game = g; return b }

// GameRef returns the game object for in-place edits as facts arrive.
func (b *Builder) GameRef() *Game { return &b.doc.Game }

// AddPlayer registers a seat and returns a pointer for later edits, because a
// seat's hero, and certainly its result, are learned long after it first
// appears.
func (b *Builder) AddPlayer(p Player) *Player {
	b.seen[p.ID] = true
	b.doc.Players = append(b.doc.Players, p)
	return &b.doc.Players[len(b.doc.Players)-1]
}

// PlayerRef finds a registered seat, or nil.
func (b *Builder) PlayerRef(id string) *Player {
	for i := range b.doc.Players {
		if b.doc.Players[i].ID == id {
			return &b.doc.Players[i]
		}
	}
	return nil
}

// Document returns the assembled document. Call Validate on it before writing.
func (b *Builder) Document() *Document { return &b.doc }

// EntryBuilder collects one entry. Finish it with Done.
type EntryBuilder struct {
	b *Builder
	e Entry
}

// Action starts an action entry: something a player chose to do.
func (b *Builder) Action(actor, verb string) *EntryBuilder {
	return &EntryBuilder{b: b, e: Entry{Kind: KindAction, Actor: actor, Action: verb}}
}

// Event starts an event entry: something the game did. Leave the actor empty
// for an event that concerns the whole lobby; use EventFor otherwise.
func (b *Builder) Event(verb string) *EntryBuilder {
	return &EntryBuilder{b: b, e: Entry{Kind: KindEvent, Event: verb}}
}

// EventFor starts an event that happened to one seat.
func (b *Builder) EventFor(actor, verb string) *EntryBuilder {
	return &EntryBuilder{b: b, e: Entry{Kind: KindEvent, Event: verb, Actor: actor}}
}

// State starts a state entry: a full picture of one seat at one moment.
func (b *Builder) State(actor, reason string) *EntryBuilder {
	return &EntryBuilder{b: b, e: Entry{Kind: KindState, Actor: actor, Reason: reason}}
}

// Turn sets which turn the entry belongs to.
func (eb *EntryBuilder) Turn(n int) *EntryBuilder { eb.e.Turn = Int(n); return eb }

// Phase sets setup, recruit, combat or end.
func (eb *EntryBuilder) Phase(p string) *EntryBuilder { eb.e.Phase = p; return eb }

// At sets the wall-clock time.
func (eb *EntryBuilder) At(t time.Time) *EntryBuilder {
	eb.e.At = t.UTC().Format(time.RFC3339)
	return eb
}

// Note attaches a sentence for a person. A reader must never parse it.
func (eb *EntryBuilder) Note(s string) *EntryBuilder { eb.e.Note = s; return eb }

// Data sets the payload.
func (eb *EntryBuilder) Data(d Data) *EntryBuilder { eb.e.Data = &d; return eb }

// Refused marks an action the game turned down, with the reason. A refused
// action changed nothing, and a reader must not apply it.
func (eb *EntryBuilder) Refused(why string) *EntryBuilder {
	eb.e.Accepted = Bool(false)
	eb.e.Error = why
	return eb
}

// Decision attaches notes on how the choice was made.
func (eb *EntryBuilder) Decision(d Decision) *EntryBuilder { eb.e.Decision = &d; return eb }

// Seat sets the numbers on a state entry.
func (eb *EntryBuilder) Seat(s SeatState) *EntryBuilder { eb.e.Player = &s; return eb }

// Zone adds one zone to a state entry. A zone left out was not written down; a
// zone present and empty was written down and was empty.
func (eb *EntryBuilder) Zone(name string, z Zone) *EntryBuilder {
	if eb.e.Zones == nil {
		eb.e.Zones = map[string]Zone{}
	}
	if z.Cards == nil {
		z.Cards = []Card{}
	}
	eb.e.Zones[name] = z
	return eb
}

// Standings sets the leaderboard on a state entry.
func (eb *EntryBuilder) Standings(rows []Standing) *EntryBuilder { eb.e.Standings = rows; return eb }

// Next names the seat this player will fight.
func (eb *EntryBuilder) Next(player, ghostOf string) *EntryBuilder {
	eb.e.NextOpponent = &NextOpponent{Player: player, GhostOf: ghostOf}
	return eb
}

// Ext attaches producer data.
func (eb *EntryBuilder) Ext(e Ext) *EntryBuilder { eb.e.Ext = e; return eb }

// Done appends the entry and returns its seq number, so a caller that needs to
// refer back to it — an offer, say — can hold on to it.
func (eb *EntryBuilder) Done() int {
	eb.b.seq++
	eb.e.Seq = eb.b.seq
	eb.b.doc.History = append(eb.b.doc.History, eb.e)
	return eb.e.Seq
}

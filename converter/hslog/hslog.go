// Package hslog reads a Hearthstone Power.log into a stream of events.
//
// It knows nothing about Battlegrounds. It turns lines into structured events
// and stops there; package state accumulates them into an entity table, and
// package convert projects that into a history file.
//
// # The one line shape
//
// Every line in the file looks like this, with no date and seven fractional
// digits of clock:
//
//	D 20:59:32.3470285 GameState.DebugPrintPower() - CREATE_GAME
//
// The parser reads only the GameState stream. The client also prints a
// PowerTaskList stream carrying the same events in a different format at a
// different time; accepting both would count everything twice.
//
// # Indentation
//
// Indentation lives inside the payload, after the " - ". It does two unrelated
// jobs: it nests BLOCK_START/BLOCK_END, and it attaches "tag=NAME value=V"
// lines to the entity header above them. Only the second matters here, and only
// lines that actually begin with "tag=" are folded, because META_DATA and
// SUB_SPELL_START print column-aligned continuation lines that are more indented
// than their header and are not tags.
package hslog

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// Kind names what a line said.
type Kind int

// The event kinds.
const (
	KindUnknown Kind = iota
	KindCreateGame
	KindGameEntity
	KindPlayer
	KindFullEntity
	KindShowEntity
	KindHideEntity
	KindTagChange
	KindBlockStart
	KindBlockEnd
	KindMetaData
	KindGameInfo     // a DebugPrintGame key/value
	KindSendOption   // the local player clicked something
	KindChoiceOffer  // a set of cards was offered
	KindChoiceOption // one option of that set
	KindChosen       // what was taken
)

// Event is one thing a line said.
type Event struct {
	Kind   Kind
	Time   string // HH:MM:SS.fffffff, as printed. There is no date in the file.
	Indent int    // payload indentation, in spaces

	// Who it is about. EntityID is 0 when the line named nobody, or named
	// somebody only by display name, in which case EntityName carries it.
	EntityID   int
	EntityName string
	CardID     string
	PlayerID   int

	// KindTagChange.
	Tag   string
	Value string

	// KindBlockStart.
	BlockType string
	Target    int
	TargetRef string

	// KindFullEntity, KindShowEntity, KindGameEntity, KindPlayer: the tag block
	// that followed the header, already folded in.
	Tags map[string]string

	// KindMetaData.
	Meta string
	Data int

	// KindGameInfo and KindSendOption carry loose key/value pairs.
	Fields map[string]string

	Raw string
}

// Int reads one of the folded tags as a number.
func (e *Event) Int(tag string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(e.Tags[tag]))
	return n
}

var (
	reLine = regexp.MustCompile(`^D (\d{2}:\d{2}:\d{2}\.\d+) (\w+)\.(\w+)\(\) - (.*)$`)

	// A TAG_CHANGE's entity reference may be a bare id, a display name, the word
	// GameEntity, or a bracket descriptor whose entityName can itself contain
	// square brackets. So: take everything up to the first " tag=", and dig the
	// id out of it by search rather than by position.
	reTagChange = regexp.MustCompile(`^TAG_CHANGE Entity=(.+?) tag=(\S+) value=(.+?)\s*$`)
	reBracketID = regexp.MustCompile(`\bid=(\d+)`)
	reEntityNm  = regexp.MustCompile(`^\[entityName=(.*) id=\d+ zone=`)

	reGameEntity = regexp.MustCompile(`^GameEntity EntityID=(\d+)`)
	rePlayer     = regexp.MustCompile(`^Player EntityID=(\d+) PlayerID=(\d+)`)
	reFullEntity = regexp.MustCompile(`^FULL_ENTITY - Creating ID=(\d+) CardID=(\S*)\s*$`)
	reShowEntity = regexp.MustCompile(`^SHOW_ENTITY - Updating Entity=(.+?) CardID=(\S*)\s*$`)
	reHideEntity = regexp.MustCompile(`^HIDE_ENTITY - Entity=(.+?) tag=(\S+) value=(.+?)\s*$`)
	reBlockStart = regexp.MustCompile(`^BLOCK_START BlockType=(\S+) Entity=(.+?)( EffectCardId=| EffectIndex=| Target=| SubOption=)`)
	// Target runs to " SubOption=", which always follows it, so a square bracket
	// inside an entityName cannot cut the match short.
	reBlockTgt = regexp.MustCompile(` Target=(\[.*\]|\S+) SubOption=`)
	reMetaData = regexp.MustCompile(`^META_DATA - Meta=(\S+) Data=(\d+)`)
	reTagLine  = regexp.MustCompile(`^tag=(\S+) value=(.+?)\s*$`)
	reKeyVals  = regexp.MustCompile(`(\w+)=([^\s]+)`)
)

// Stream names the printer this parser accepts.
const Stream = "GameState"

// Parser turns lines into events. Feed one line at a time; it returns zero, one
// or two events, because an entity header is only complete once a line that is
// not one of its tags arrives.
//
// A Parser is not safe for use from more than one goroutine.
type Parser struct {
	pending      *Event
	pendingIndot int
}

// NewParser returns a parser ready for the first line.
func NewParser() *Parser { return &Parser{} }

// Feed reads one line.
func (p *Parser) Feed(line string) []Event {
	m := reLine.FindStringSubmatch(strings.TrimRight(line, "\r\n"))
	if m == nil {
		return nil
	}
	when, printer, method, payload := m[1], m[2], m[3], m[4]
	if printer != Stream {
		return nil
	}

	indent := len(payload) - len(strings.TrimLeft(payload, " "))
	body := strings.TrimLeft(payload, " ")

	// A tag line belongs to the entity header above it, if that header is less
	// indented. Anything else closes the header.
	if p.pending != nil && indent > p.pendingIndot && method == "DebugPrintPower" {
		if tm := reTagLine.FindStringSubmatch(body); tm != nil {
			p.pending.Tags[tm[1]] = strings.TrimSpace(tm[2])
			return nil
		}
	}

	var out []Event
	if done := p.flush(); done != nil {
		out = append(out, *done)
	}

	ev := Event{Time: when, Indent: indent, Raw: body}

	switch method {
	case "DebugPrintGame":
		// "BuildNumber=248348", or "PlayerID=3, PlayerName=coulbourne#1741".
		ev.Kind = KindGameInfo
		ev.Fields = map[string]string{}
		for _, kv := range strings.Split(body, ", ") {
			if k, v, ok := strings.Cut(kv, "="); ok {
				ev.Fields[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
		}
		return append(out, ev)

	case "SendOption":
		ev.Kind = KindSendOption
		ev.Fields = map[string]string{}
		for _, kv := range reKeyVals.FindAllStringSubmatch(body, -1) {
			ev.Fields[kv[1]] = kv[2]
		}
		return append(out, ev)

	case "DebugPrintEntityChoices":
		switch {
		case strings.HasPrefix(body, "id="):
			ev.Kind = KindChoiceOffer
			ev.Fields = map[string]string{}
			for _, kv := range reKeyVals.FindAllStringSubmatch(body, -1) {
				ev.Fields[kv[1]] = kv[2]
			}
		case strings.HasPrefix(body, "Entities["):
			ev.Kind = KindChoiceOption
			_, ref, _ := strings.Cut(body, "=")
			ev.EntityID, ev.EntityName, ev.CardID = ParseRef(ref)
		case strings.HasPrefix(body, "Source="):
			// The Source names which sort of choice this is: a trinket slot, a
			// dark gift, a triple reward. Keep it as a field on the offer.
			ev.Kind = KindChoiceOffer
			ev.Fields = map[string]string{"Source": strings.TrimPrefix(body, "Source=")}
			_, _, ev.CardID = ParseRef(strings.TrimPrefix(body, "Source="))
		default:
			return out
		}
		return append(out, ev)

	case "DebugPrintEntitiesChosen":
		if !strings.HasPrefix(body, "Entities[") {
			return out
		}
		ev.Kind = KindChosen
		_, ref, _ := strings.Cut(body, "=")
		ev.EntityID, ev.EntityName, ev.CardID = ParseRef(ref)
		return append(out, ev)

	case "DebugPrintPower":
		// fall through
	default:
		return out
	}

	switch {
	case body == "CREATE_GAME":
		ev.Kind = KindCreateGame
		return append(out, ev)

	case body == "BLOCK_END":
		ev.Kind = KindBlockEnd
		return append(out, ev)

	case strings.HasPrefix(body, "TAG_CHANGE "):
		m := reTagChange.FindStringSubmatch(body)
		if m == nil {
			return out
		}
		ev.Kind = KindTagChange
		ev.EntityID, ev.EntityName, _ = ParseRef(m[1])
		ev.Tag, ev.Value = m[2], strings.TrimSpace(m[3])
		return append(out, ev)

	case strings.HasPrefix(body, "GameEntity "):
		m := reGameEntity.FindStringSubmatch(body)
		if m == nil {
			return out
		}
		ev.Kind, ev.Tags = KindGameEntity, map[string]string{}
		ev.EntityID, _ = strconv.Atoi(m[1])
		p.hold(&ev, indent)
		return out

	case strings.HasPrefix(body, "Player "):
		m := rePlayer.FindStringSubmatch(body)
		if m == nil {
			return out
		}
		ev.Kind, ev.Tags = KindPlayer, map[string]string{}
		ev.EntityID, _ = strconv.Atoi(m[1])
		ev.PlayerID, _ = strconv.Atoi(m[2])
		p.hold(&ev, indent)
		return out

	case strings.HasPrefix(body, "FULL_ENTITY"):
		m := reFullEntity.FindStringSubmatch(body)
		if m == nil {
			return out
		}
		ev.Kind, ev.Tags = KindFullEntity, map[string]string{}
		ev.EntityID, _ = strconv.Atoi(m[1])
		ev.CardID = m[2] // may be empty: the server hides some cards from you
		p.hold(&ev, indent)
		return out

	case strings.HasPrefix(body, "SHOW_ENTITY"):
		m := reShowEntity.FindStringSubmatch(body)
		if m == nil {
			return out
		}
		ev.Kind, ev.Tags = KindShowEntity, map[string]string{}
		ev.EntityID, ev.EntityName, _ = ParseRef(m[1])
		ev.CardID = m[2]
		p.hold(&ev, indent)
		return out

	case strings.HasPrefix(body, "HIDE_ENTITY"):
		m := reHideEntity.FindStringSubmatch(body)
		if m == nil {
			return out
		}
		ev.Kind = KindHideEntity
		ev.EntityID, ev.EntityName, _ = ParseRef(m[1])
		ev.Tag, ev.Value = m[2], strings.TrimSpace(m[3])
		return append(out, ev)

	case strings.HasPrefix(body, "BLOCK_START"):
		m := reBlockStart.FindStringSubmatch(body)
		if m == nil {
			return out
		}
		ev.Kind = KindBlockStart
		ev.BlockType = m[1]
		ev.EntityID, ev.EntityName, ev.CardID = ParseRef(m[2])
		if t := reBlockTgt.FindStringSubmatch(body); t != nil {
			ev.TargetRef = t[1]
			ev.Target, _, _ = ParseRef(t[1])
		}
		return append(out, ev)

	case strings.HasPrefix(body, "META_DATA"):
		m := reMetaData.FindStringSubmatch(body)
		if m == nil {
			return out
		}
		ev.Kind = KindMetaData
		ev.Meta = m[1]
		ev.Data, _ = strconv.Atoi(m[2])
		return append(out, ev)
	}
	return out
}

// Flush returns the entity header still being built, if any. Call it once at
// the end of a file.
func (p *Parser) Flush() []Event {
	if done := p.flush(); done != nil {
		return []Event{*done}
	}
	return nil
}

func (p *Parser) hold(ev *Event, indent int) {
	held := *ev
	p.pending, p.pendingIndot = &held, indent
}

func (p *Parser) flush() *Event {
	done := p.pending
	p.pending = nil
	return done
}

// ParseRef reads an entity reference in any of the four shapes the log uses:
// a bare id, the word GameEntity, a bracket descriptor, or a player's display
// name. It returns the id, the name, and the card id, each zero or empty when
// the reference did not carry it.
//
// Only id and cardId may be trusted out of a bracket descriptor. Its zone,
// zonePos and player describe the entity BEFORE the change the line is
// reporting, so reading them is how a parser ends up confidently wrong.
func ParseRef(s string) (id int, name, cardID string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, "", ""
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n, "", ""
	}
	if !strings.HasPrefix(s, "[") {
		// GameEntity, or a player addressed by battletag. Gold arrives this way,
		// so dropping it is why a first parser reports no gold all game.
		return 0, s, ""
	}
	if m := reBracketID.FindStringSubmatch(s); m != nil {
		id, _ = strconv.Atoi(m[1])
	}
	if m := reEntityNm.FindStringSubmatch(s); m != nil {
		name = m[1]
	}
	// cardId runs to the next " player=" and may be empty.
	if i := strings.Index(s, " cardId="); i >= 0 {
		rest := s[i+len(" cardId="):]
		if j := strings.Index(rest, " player="); j >= 0 {
			cardID = rest[:j]
		}
	}
	return id, name, cardID
}

// Split reads a whole log and calls fn once per event. Games are not separated
// here; a caller watches for KindCreateGame.
func Split(r io.Reader, fn func(Event)) error {
	p := NewParser()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		for _, ev := range p.Feed(sc.Text()) {
			fn(ev)
		}
	}
	for _, ev := range p.Flush() {
		fn(ev)
	}
	return sc.Err()
}

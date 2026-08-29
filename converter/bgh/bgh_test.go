package bgh

import (
	"encoding/json"
	"strings"
	"testing"
)

// demo builds the smallest document that exercises the parts most likely to be
// got wrong: a one-seat recording, an offer answered by a choice, a hero known
// only by name, both trinket slots, and an opponent health the recorder could
// not see.
func demo() *Builder {
	b := NewSeatRecording("test", "p1").
		Recorder(Recorder{Name: "test", Kind: RecorderTracker}).
		Detail(Detail{Combat: CombatBoards, States: StatesTurnStart, Entities: true}).
		Truncate("one turn")
	b.Game(Game{ID: "g", Mode: ModeSolo, SeatCount: 2})
	b.AddPlayer(Player{ID: "p1", Seat: Int(0), Kind: KindHuman,
		Hero: &Card{CardID: "TB_BaconShop_HERO_30", Name: "Nefarian", Type: TypeHero}})
	b.AddPlayer(Player{ID: "p2", Seat: Int(3), Kind: KindUnknown,
		Hero: &Card{Name: "Deathwing", Type: TypeHero, Unknown: true}})

	b.EventFor("p1", EvTurnStart).Turn(10).Phase(PhaseRecruit).
		Data(Data{Turn: Int(10), Gold: Int(10)}).Done()

	b.State("p1", "turnStart").Turn(10).Phase(PhaseRecruit).
		Seat(SeatState{Health: Int(27), Tier: Int(3), Gold: Int(10), UpgradeCost: Known(5)}).
		Zone(ZoneBoard, Zone{}).
		Zone(ZoneTrinkets, Zone{Cards: []Card{
			{Entity: "t1", CardID: "BG30_MagicItem_709", Type: TypeTrinket, TrinketTier: TrinketLesser},
			{Entity: "t2", CardID: "BG36_MagicItem_220", Type: TypeTrinket, TrinketTier: TrinketGreater},
		}}).
		Standings([]Standing{
			{Player: "p1", Health: Known(27), Tier: Known(3), Alive: Bool(true)},
			{Player: "p2", Health: Known(22), Tier: Unknown(), Alive: Bool(true),
				Hero: &Card{Name: "Deathwing", Type: TypeHero, Unknown: true}},
		}).
		Next("p2", "").Done()

	b.EventFor("p1", EvOffer).Turn(10).Phase(PhaseRecruit).Data(Data{
		ID: "o1", OfferType: OfferGreaterTrinket, Mandatory: true,
		Options: []OfferOpt{
			{Cards: []Card{{CardID: "BG36_MagicItem_220", Type: TypeTrinket, TrinketTier: TrinketGreater}}},
			{Cards: []Card{{CardID: "BG30_MagicItem_709", Type: TypeTrinket, TrinketTier: TrinketGreater}}},
		},
	}).Done()
	b.Action("p1", ActChoose).Turn(10).Phase(PhaseRecruit).
		Data(Data{Offer: "o1", OptionIndex: Int(1)}).Done()

	b.Action("p1", ActBuy).Turn(10).Phase(PhaseRecruit).
		Data(Data{From: At(ZoneShop, 1, "s2"), Cost: Int(3), Gold: Int(7)}).Done()
	b.Action("p1", ActEndTurn).Turn(10).Phase(PhaseRecruit).Done()

	b.Event(EvCombatEnd).Turn(10).Phase(PhaseCombat).Data(Data{
		Combat: "c10", Winner: Str("p1"),
		Sides: []Side{
			{Player: "p1", HealthBefore: Known(27), HealthAfter: Known(27), DamageTaken: Known(0)},
			{Player: "p2", HealthBefore: Known(22), HealthAfter: Unknown(), DamageTaken: Unknown()},
		},
	}).Done()
	return b
}

func TestDemoDocumentValidates(t *testing.T) {
	if err := demo().Document().Validate(); err != nil {
		t.Fatal(err)
	}
}

// TestSeqRisesWithoutGaps: seq numbering is the builder's job precisely so a
// caller cannot get it wrong.
func TestSeqRisesWithoutGaps(t *testing.T) {
	doc := demo().Document()
	for i, e := range doc.History {
		if e.Seq != i+1 {
			t.Fatalf("entry %d has seq %d, want %d", i, e.Seq, i+1)
		}
	}
}

// TestUnknownMarshalsToNullAndAbsentIsAbsent is the heart of the honesty rule:
// a health the recorder could not see is null, and a field that does not apply
// is not there at all. Collapsing the two would let a reader mistake "I could
// not see it" for "it was zero".
func TestUnknownMarshalsToNullAndAbsentIsAbsent(t *testing.T) {
	body, err := json.Marshal(Side{Player: "p2", HealthAfter: Unknown()})
	if err != nil {
		t.Fatal(err)
	}
	got := string(body)
	if !strings.Contains(got, `"healthAfter":null`) {
		t.Errorf("an unknown health should marshal to null, got %s", got)
	}
	if strings.Contains(got, "healthBefore") {
		t.Errorf("a health that was never set should be absent, got %s", got)
	}
	body, err = json.Marshal(Side{Player: "p2", HealthAfter: Known(0)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"healthAfter":0`) {
		t.Errorf("a known zero should marshal to 0, got %s", body)
	}
}

// TestSeatRecordingRefusesAnotherSeatsAction is the rule that keeps a one-seat
// recording honest. You never saw the other player decide, so you may not write
// down that they decided anything.
func TestSeatRecordingRefusesAnotherSeatsAction(t *testing.T) {
	b := demo()
	b.Action("p2", ActBuy).Data(Data{From: Slot(ZoneShop, 0)}).Done()
	err := b.Document().Validate()
	if err == nil {
		t.Fatal("want an error, got none")
	}
	if !strings.Contains(err.Error(), "cannot have watched another player decide") {
		t.Fatalf("want the seat-scope complaint, got: %v", err)
	}
}

// TestLobbyRecordingAllowsEverySeatsActions: the same file is fine once the
// recorder claims it could see the whole lobby.
func TestLobbyRecordingAllowsEverySeatsActions(t *testing.T) {
	b := demo()
	b.Document().Recording.Observer = Observer{Scope: ScopeLobby}
	b.Action("p2", ActBuy).Data(Data{From: Slot(ZoneShop, 0)}).Done()
	if err := b.Document().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestVerbOutsideTheVocabularyNeedsThePrefix(t *testing.T) {
	b := demo()
	b.Action("p1", "teleport").Done()
	err := b.Document().Validate()
	if err == nil || !strings.Contains(err.Error(), `must start with "x-"`) {
		t.Fatalf("want the vocabulary complaint, got: %v", err)
	}

	b = demo()
	b.Action("p1", "x-teleport").Done()
	if err := b.Document().Validate(); err != nil {
		t.Fatalf("an x- prefixed verb is allowed: %v", err)
	}
}

func TestRequiredDataPerVerb(t *testing.T) {
	b := demo()
	b.Action("p1", ActBuy).Done() // no data.from
	err := b.Document().Validate()
	if err == nil || !strings.Contains(err.Error(), "needs data.from") {
		t.Fatalf("want the missing-field complaint, got: %v", err)
	}
}

func TestChoiceMustPointAtAnEarlierOffer(t *testing.T) {
	b := demo()
	b.Action("p1", ActChoose).Data(Data{Offer: "nope", OptionIndex: Int(0)}).Done()
	err := b.Document().Validate()
	if err == nil || !strings.Contains(err.Error(), "which no earlier entry made") {
		t.Fatalf("want the dangling-offer complaint, got: %v", err)
	}

	b = demo()
	b.Action("p1", ActChoose).Data(Data{Offer: "o1", OptionIndex: Int(5)}).Done()
	err = b.Document().Validate()
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("want the out-of-range complaint, got: %v", err)
	}
}

// TestAnEntityNameStaysWithOneCopy: entity names are how a reader follows one
// minion through a game, so reusing one for a different card is an error, not a
// shortcut.
func TestAnEntityNameStaysWithOneCopy(t *testing.T) {
	b := demo()
	b.EventFor("p1", EvShopDealt).Data(Data{Cards: []Card{
		{Entity: "t1", CardID: "BG31_803"},
	}}).Done()
	err := b.Document().Validate()
	if err == nil || !strings.Contains(err.Error(), "stays with one copy") {
		t.Fatalf("want the entity complaint, got: %v", err)
	}
}

func TestCardNeedsAnIdOrAnHonestUnknown(t *testing.T) {
	b := demo()
	b.EventFor("p1", EvShopDealt).Data(Data{Cards: []Card{{Name: "Something"}}}).Done()
	err := b.Document().Validate()
	if err == nil || !strings.Contains(err.Error(), "not marked unknown") {
		t.Fatalf("want the identity complaint, got: %v", err)
	}
}

func TestMinionTypeMapsFromCardData(t *testing.T) {
	for in, want := range map[string]string{
		"MECHANICAL": MtMech, "BEAST": MtBeast, "beast": MtBeast, "ALL": MtAll,
	} {
		got, ok := MinionType(in)
		if !ok || got != want {
			t.Errorf("MinionType(%q) = %q, %v; want %q, true", in, got, ok, want)
		}
	}
	if _, ok := MinionType("SQUIRREL"); ok {
		t.Error("an unknown race should report false so the caller decides what to do")
	}
}

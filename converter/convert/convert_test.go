package convert

import (
	"flag"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/DanielCoulbourne/battlegrounds-history-data/converter/bgh"
)

// The fixture is hand-written from the log grammar, not a captured session. A
// real Power.log carries the battletags of eight people, and this repository is
// public.
const fixture = "../testdata/synthetic.log"

func convertFixture(t *testing.T) *bgh.Document {
	t.Helper()
	f, err := os.Open(fixture)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	docs, err := Convert(f, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("want 1 game, got %d", len(docs))
	}
	if docs[0] == nil {
		t.Fatal("the converter produced no document")
	}
	return docs[0]
}

// TestConvertedFileIsValid is the check that matters most: whatever the
// converter believes about the log, the file it writes has to be a file anyone
// else can read.
func TestConvertedFileIsValid(t *testing.T) {
	if err := convertFixture(t).Validate(); err != nil {
		t.Fatal(err)
	}
}

// TestItIsASeatRecording. A client log knows what you did and what you were
// shown. Claiming to have seen the whole lobby would be a lie, and the file
// must not tell it.
func TestItIsASeatRecording(t *testing.T) {
	doc := convertFixture(t)
	if doc.Recording.Observer.Scope != bgh.ScopeSeat {
		t.Errorf("scope is %q, want %q", doc.Recording.Observer.Scope, bgh.ScopeSeat)
	}
	if doc.Recording.Observer.Seat != "p3" {
		t.Errorf("seat is %q, want p3", doc.Recording.Observer.Seat)
	}
	for _, e := range doc.History {
		if e.Kind == bgh.KindAction && e.Actor != "p3" {
			t.Errorf("entry seq %d is an action by %q; a client log cannot see another seat decide",
				e.Seq, e.Actor)
		}
	}
}

// TestEverySeatsHeroIsRecorded. The leaderboard shows all eight heroes, and it
// is very nearly the only durable fact a one-seat recording gets about an
// opponent, so losing it would be a real loss.
func TestEverySeatsHeroIsRecorded(t *testing.T) {
	doc := convertFixture(t)
	want := map[string]string{
		"p3": "TB_BaconShop_HERO_30", // the recorded seat
		"p4": "BG22_HERO_002",
		"p5": "BG22_HERO_003",
		"p6": "TB_BaconShop_HERO_08",
	}
	got := map[string]string{}
	for _, p := range doc.Players {
		if p.Hero != nil {
			got[p.ID] = p.Hero.CardID
		}
	}
	for id, card := range want {
		if got[id] != card {
			t.Errorf("seat %s hero is %q, want %q", id, got[id], card)
		}
	}

	// And on the leaderboard rows inside a state entry, which is where a reader
	// looks for what the seat could see at that moment.
	for _, e := range doc.History {
		if e.Kind != bgh.KindState {
			continue
		}
		for _, row := range e.Standings {
			if row.Hero == nil || row.Hero.CardID == "" {
				t.Errorf("entry seq %d: leaderboard row for %s carries no hero", e.Seq, row.Player)
			}
		}
		return
	}
	t.Error("no state entry was written")
}

// TestBobIsNotAPlayer. Bob runs the tavern under the same controller as the
// leaderboard and the current opponent. Counting him as a seat is the classic
// way to get nine players in an eight-player game.
func TestBobIsNotAPlayer(t *testing.T) {
	for _, p := range convertFixture(t).Players {
		if p.Hero != nil && strings.Contains(p.Hero.CardID, "BaconShopBob") {
			t.Fatalf("seat %s is Bob, who is not a player", p.ID)
		}
	}
}

// TestTheShopIsNotReadAsAnEnemyBoard. When the phase flag flips to combat the
// previous shop is still standing under the controller the opponent's board is
// about to use. Reading it then reports the tavern as the enemy.
func TestTheShopIsNotReadAsAnEnemyBoard(t *testing.T) {
	doc := convertFixture(t)
	for _, e := range doc.History {
		if e.Event != bgh.EvCombatStart {
			continue
		}
		if len(e.Data.Sides) != 2 {
			t.Fatalf("combat_start has %d sides, want 2", len(e.Data.Sides))
		}
		enemy := e.Data.Sides[1]
		if len(enemy.Board) != 1 || enemy.Board[0].CardID != "BG33_886" {
			t.Fatalf("the opponent's board is %v, want the one minion they fielded", enemy.Board)
		}
		if enemy.Player != "p4" {
			t.Errorf("fighting %q, want p4", enemy.Player)
		}
		return
	}
	t.Error("no combat_start was written")
}

// TestActionsComeOutOfTheButtons. The client never says "the player bought a
// minion". It says a hidden button card was played at a target, and the meaning
// is in which button.
func TestActionsComeOutOfTheButtons(t *testing.T) {
	doc := convertFixture(t)
	var verbs []string
	for _, e := range doc.History {
		if e.Kind == bgh.KindAction {
			verbs = append(verbs, e.Action)
		}
	}
	got := strings.Join(verbs, " ")
	for _, want := range []string{
		bgh.ActChooseHero, bgh.ActBuy, bgh.ActPlay, bgh.ActRoll,
		bgh.ActFreeze, bgh.ActUpgrade, bgh.ActChoose,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q action was recorded; got: %s", want, got)
		}
	}
}

// TestBuyCarriesItsPrice. The price is on the buy button, which points at its
// card through a tag the client prints as a bare number. Dropping unnamed tags
// loses it.
func TestBuyCarriesItsPrice(t *testing.T) {
	for _, e := range convertFixture(t).History {
		if e.Action != bgh.ActBuy {
			continue
		}
		if e.Data == nil || e.Data.Cost == nil || *e.Data.Cost != 3 {
			t.Fatalf("the buy carries cost %v, want 3", e.Data.Cost)
		}
		if e.Data.From == nil || e.Data.From.CardID != "BG31_803" {
			t.Fatalf("the buy came from %v, want the shop card", e.Data.From)
		}
		return
	}
	t.Error("no buy was recorded")
}

// TestTrinketPickKnowsWhichSlot. Lesser and greater are two slots offered at
// two different points, not two grades of one thing, and the card that raised
// the choice is what says which.
func TestTrinketPickKnowsWhichSlot(t *testing.T) {
	doc := convertFixture(t)
	for _, e := range doc.History {
		if e.Event == bgh.EvOffer && e.Data.OfferType == bgh.OfferLesserTrinket {
			goto found
		}
	}
	t.Fatal("the trinket offer did not say which slot it was for")
found:
	for _, e := range doc.History {
		if e.Kind != bgh.KindState {
			continue
		}
		for _, c := range e.Zones[bgh.ZoneTrinkets].Cards {
			if c.CardID == "BG30_MagicItem_709" && c.TrinketTier != bgh.TrinketLesser {
				t.Errorf("the held trinket has tier %q, want %q", c.TrinketTier, bgh.TrinketLesser)
			}
		}
	}
}

// TestCombatIsRecordedBlowByBlow. The log carries every swing, every point of
// damage and every death, so the recording carries them too rather than
// settling for the score.
func TestCombatIsRecordedBlowByBlow(t *testing.T) {
	doc := convertFixture(t)
	if doc.Recording.Detail.Combat != bgh.CombatEvents {
		t.Errorf("combat detail is %q, want %q", doc.Recording.Detail.Combat, bgh.CombatEvents)
	}
	counts := map[string]int{}
	for _, e := range doc.History {
		if e.Kind == bgh.KindEvent {
			counts[e.Event]++
		}
	}
	for _, want := range []string{
		bgh.EvCombatStart, bgh.EvAttack, bgh.EvDamage, bgh.EvDeath,
		bgh.EvHeroDamage, bgh.EvCombatEnd,
	} {
		if counts[want] == 0 {
			t.Errorf("no %q event was written", want)
		}
	}
}

// TestDamageIsTheHitNotTheRunningTotal. The log prints a minion's cumulative
// damage, so a converter that copies the number reports the wrong hit as soon
// as anything is damaged twice.
func TestDamageIsTheHitNotTheRunningTotal(t *testing.T) {
	doc := convertFixture(t)
	for _, e := range doc.History {
		if e.Event != bgh.EvDamage || e.Data.Target == nil || e.Data.Target.Entity != "e435" {
			continue
		}
		if e.Data.Amount == nil || *e.Data.Amount != 2 {
			t.Fatalf("the hit on e435 is %v, want 2", e.Data.Amount)
		}
		if e.Data.Lethal == nil || !*e.Data.Lethal {
			t.Error("a 2 point hit on a 1 health minion is lethal and should say so")
		}
		return
	}
	t.Error("no damage event for the attacking minion")
}

// TestWhatTheOpponentLostIsNullNotZero. This seat sees its own health at once
// and the opponent's only when the leaderboard catches up. Writing 0 for a
// number you have not seen is the one thing the format exists to prevent.
func TestWhatTheOpponentLostIsNullNotZero(t *testing.T) {
	for _, e := range convertFixture(t).History {
		if e.Event != bgh.EvCombatEnd {
			continue
		}
		mine := e.Data.Sides[0]
		if _, known := mine.DamageTaken.Value(); !known {
			t.Error("this seat's own damage should be known")
		}
		return
	}
	t.Error("no combat_end was written")
}

// TestNamesAreLeftOutByDefault. A battletag identifies a real person, and a
// recording gets shared more readily than a log does.
func TestNamesAreLeftOutByDefault(t *testing.T) {
	body, err := convertFixture(t).Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "Tester#1234") {
		t.Error("the local player's display name reached the output")
	}
}

// TestTheGameIsIdentifiedByItsSeed, which is the one stable name a game has.
func TestTheGameIsIdentifiedByItsSeed(t *testing.T) {
	if got := convertFixture(t).Game.ID; got != "hs-987654321" {
		t.Errorf("game id is %q, want hs-987654321", got)
	}
}

// golden is the published example of what this converter produces. It lives in
// examples/ rather than in testdata/ on purpose: the repository's example of a
// converted client log ought to be real converter output, and making it a
// golden file is what keeps it that way.
const golden = "../../examples/from-client-log.json"

var update = flag.Bool("update", false, "rewrite the golden example from the fixture")

func TestGoldenExampleIsUpToDate(t *testing.T) {
	body, err := convertFixture(t).Marshal()
	if err != nil {
		t.Fatal(err)
	}
	// recordedAt is the clock at the moment of conversion, so it differs on
	// every run and is blanked before comparing.
	body = reRecorded.ReplaceAll(body, []byte(`"recordedAt": "2026-01-01T00:00:00Z"`))

	if *update {
		if err := os.WriteFile(golden, body, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Log("wrote", golden)
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v\nrun: go test ./convert -update", err)
	}
	if string(want) != string(body) {
		t.Errorf("%s is out of date; run: go test ./convert -update", golden)
	}
}

var reRecorded = regexp.MustCompile(`"recordedAt": "[^"]*"`)

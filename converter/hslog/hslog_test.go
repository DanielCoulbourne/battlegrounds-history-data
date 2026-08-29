package hslog

import "testing"

// Each test here is one of the ways a Power.log parser goes wrong. They are
// small on purpose: every one of them cost somebody an afternoon.

func feed(t *testing.T, lines ...string) []Event {
	t.Helper()
	p := NewParser()
	var out []Event
	for _, l := range lines {
		out = append(out, p.Feed(l)...)
	}
	return append(out, p.Flush()...)
}

// TestOnlyTheGameStateStreamIsRead. The client prints the same events twice,
// once as GameState and once as PowerTaskList, in different formats and at
// different times. Reading both counts everything twice.
func TestOnlyTheGameStateStreamIsRead(t *testing.T) {
	evs := feed(t,
		"D 20:00:00.0000001 GameState.DebugPrintPower() - CREATE_GAME",
		"D 20:00:00.0000002 PowerTaskList.DebugPrintPower() - CREATE_GAME",
	)
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	if evs[0].Kind != KindCreateGame {
		t.Errorf("kind is %v, want KindCreateGame", evs[0].Kind)
	}
}

// TestTagsFoldIntoTheEntityAboveThem. Indentation does two unrelated jobs in
// this file, and only tag lines belong to the header above them.
func TestTagsFoldIntoTheEntityAboveThem(t *testing.T) {
	evs := feed(t,
		"D 20:00:00.0000001 GameState.DebugPrintPower() -     FULL_ENTITY - Creating ID=435 CardID=BG31_803",
		"D 20:00:00.0000002 GameState.DebugPrintPower() -         tag=ATK value=1",
		"D 20:00:00.0000003 GameState.DebugPrintPower() -         tag=HEALTH value=1",
		"D 20:00:00.0000004 GameState.DebugPrintPower() - CREATE_GAME",
	)
	if len(evs) != 2 {
		t.Fatalf("got %d events, want 2", len(evs))
	}
	e := evs[0]
	if e.EntityID != 435 || e.CardID != "BG31_803" {
		t.Fatalf("entity is %d/%q, want 435/BG31_803", e.EntityID, e.CardID)
	}
	if e.Int("ATK") != 1 || e.Int("HEALTH") != 1 {
		t.Errorf("tags are %v, want ATK 1 and HEALTH 1", e.Tags)
	}
}

// TestContinuationLinesAreNotTags. META_DATA prints a column-aligned Info line
// that is MORE indented than its header and is not a tag. Folding it would
// corrupt whichever entity happened to be open.
func TestContinuationLinesAreNotTags(t *testing.T) {
	evs := feed(t,
		"D 20:00:00.0000001 GameState.DebugPrintPower() -     FULL_ENTITY - Creating ID=1 CardID=X",
		"D 20:00:00.0000002 GameState.DebugPrintPower() -         tag=ATK value=1",
		"D 20:00:00.0000003 GameState.DebugPrintPower() -     META_DATA - Meta=DAMAGE Data=6 InfoCount=1",
		"D 20:00:00.0000004 GameState.DebugPrintPower() -                 Info[0] = [entityName=Ominous Seer id=927 zone=PLAY zonePos=1 cardId=BG31_330 player=11]",
	)
	for _, e := range evs {
		if e.Kind == KindFullEntity && len(e.Tags) != 1 {
			t.Errorf("the entity picked up %v; only its own tag belongs to it", e.Tags)
		}
	}
}

// TestTrailingSpaceOnEveryTagChange. Every TAG_CHANGE ends with a space. Value
// comparisons fail silently if it is not trimmed.
func TestTrailingSpaceOnEveryTagChange(t *testing.T) {
	evs := feed(t, "D 20:00:00.0000001 GameState.DebugPrintPower() -     TAG_CHANGE Entity=GameEntity tag=TURN value=2 ")
	if len(evs) != 1 || evs[0].Value != "2" {
		t.Fatalf("value is %q, want \"2\"", evs[0].Value)
	}
}

// TestUnnamedTagsAreKept. The client prints a tag name only when it has a
// string for the enum; new ones appear as bare numbers. One of them is how a
// buy button points at the card it buys, so dropping them loses the price.
func TestUnnamedTagsAreKept(t *testing.T) {
	evs := feed(t, "D 20:00:00.0000001 GameState.DebugPrintPower() -         TAG_CHANGE Entity=436 tag=2442 value=435 ")
	if len(evs) != 1 || evs[0].Tag != "2442" || evs[0].Value != "435" {
		t.Fatalf("got %+v, want tag 2442 value 435", evs[0])
	}
}

// TestEntityReferencesInAllFourShapes.
func TestEntityReferencesInAllFourShapes(t *testing.T) {
	cases := []struct {
		in     string
		id     int
		name   string
		cardID string
	}{
		{"436", 436, "", ""},
		{"GameEntity", 0, "GameEntity", ""},
		{"coulbourne#1741", 0, "coulbourne#1741", ""},
		{"[entityName=Reno Jackson id=93 zone=PLAY zonePos=0 cardId=TB_BaconShop_HERO_41 player=3]",
			93, "Reno Jackson", "TB_BaconShop_HERO_41"},
		// An entityName containing square brackets, which is why the id is found
		// by search rather than by carving up the string.
		{"[entityName=UNKNOWN ENTITY [cardType=INVALID] id=584 zone=PLAY zonePos=0 cardId= player=11]",
			584, "UNKNOWN ENTITY [cardType=INVALID]", ""},
		// An empty cardId running straight into " player=".
		{"[entityName=Thing id=968 zone=SETASIDE zonePos=0 cardId= player=11]", 968, "Thing", ""},
	}
	for _, c := range cases {
		id, name, cardID := ParseRef(c.in)
		if id != c.id || name != c.name || cardID != c.cardID {
			t.Errorf("ParseRef(%q) = %d, %q, %q; want %d, %q, %q",
				c.in, id, name, cardID, c.id, c.name, c.cardID)
		}
	}
}

// TestBlockStartSurvivesTheEffectCardIdBug. The client prints a .NET type name
// where a card id belongs, backtick and square brackets included.
func TestBlockStartSurvivesTheEffectCardIdBug(t *testing.T) {
	line := "D 20:00:00.0000001 GameState.DebugPrintPower() - BLOCK_START BlockType=PLAY " +
		"Entity=[entityName=Drag To Buy id=436 zone=PLAY zonePos=0 cardId=TB_BaconShop_DragBuy player=3] " +
		"EffectCardId=System.Collections.Generic.List`1[System.String] EffectIndex=0 " +
		"Target=[entityName=Suspicious Prisonguard id=435 zone=PLAY zonePos=1 cardId=BG36_345 player=11] SubOption=-1 "
	evs := feed(t, line)
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	e := evs[0]
	if e.BlockType != "PLAY" || e.EntityID != 436 || e.CardID != "TB_BaconShop_DragBuy" {
		t.Errorf("block is %+v, want the buy button", e)
	}
	if e.Target != 435 {
		t.Errorf("target is %d, want 435", e.Target)
	}
}

// TestAHiddenCardKeepsNoIdentity, and a later reveal fills it in without a
// re-creation erasing what is already known.
func TestAHiddenCardKeepsNoIdentity(t *testing.T) {
	tab := NewTable()
	for _, ev := range feed(t,
		"D 20:00:00.0000001 GameState.DebugPrintPower() -     FULL_ENTITY - Creating ID=968 CardID=",
		"D 20:00:00.0000002 GameState.DebugPrintPower() -         tag=ZONE value=SETASIDE",
		"D 20:00:00.0000003 GameState.DebugPrintPower() -     SHOW_ENTITY - Updating Entity=968 CardID=BG31_803",
		"D 20:00:00.0000004 GameState.DebugPrintPower() -         tag=ZONE value=PLAY",
		"D 20:00:00.0000005 GameState.DebugPrintPower() -     FULL_ENTITY - Creating ID=968 CardID=",
		"D 20:00:00.0000006 GameState.DebugPrintPower() -         tag=ZONE value=PLAY",
	) {
		tab.Apply(ev)
	}
	if got := tab.Get(968); got == nil || got.CardID != "BG31_803" {
		t.Fatalf("card id is %q, want BG31_803 kept through the later hidden re-creation", got.CardID)
	}
}

// TestGoldIsAddressedByName. Your gold is sent to your battletag, not to your
// player entity. Dropping name-addressed tags is why a first parser reports no
// gold all game.
func TestGoldIsAddressedByName(t *testing.T) {
	tab := NewTable()
	for _, ev := range feed(t,
		"D 20:00:00.0000001 GameState.DebugPrintGame() - PlayerID=3, PlayerName=Tester#1234",
		"D 20:00:00.0000002 GameState.DebugPrintPower() - TAG_CHANGE Entity=Tester#1234 tag=RESOURCES value=7 ",
		"D 20:00:00.0000003 GameState.DebugPrintPower() - TAG_CHANGE Entity=Tester#1234 tag=RESOURCES_USED value=3 ",
	) {
		tab.Apply(ev)
	}
	if tab.LocalName != "Tester#1234" || tab.LocalPlayerID != 3 {
		t.Fatalf("local player is %q/%d, want Tester#1234/3", tab.LocalName, tab.LocalPlayerID)
	}
	if got := tab.LocalTagInt(TagResources) - tab.LocalTagInt(TagResourcesUsed); got != 4 {
		t.Errorf("gold left is %d, want 4", got)
	}
}

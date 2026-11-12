package sdata

import (
	"os"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
)

func TestRealBlessingTiers(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := a.ReadFile("Character/KillStatus.SData")
	if err != nil {
		t.Fatal(err)
	}
	tiers, err := ParseBlessing(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(tiers) != 60 || BlessingMaximum(tiers, 0) != 12288 || BlessingMaximum(tiers, 1) != 12288 {
		t.Fatalf("unexpected blessing tiers: %d", len(tiers))
	}
}

// N1: Item.SData decrypts and parses to its last byte.
func TestRealItems(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := a.ReadFile("Item/Item.SData")
	if err != nil {
		t.Fatal(err)
	}
	items, err := ParseItems(b)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d items", len(items))
	if it := items[ItemKey{1, 1}]; it != nil {
		t.Logf("1/1 = %q model %d level %d", it.Name, it.Model, it.MinLevel)
	}
}

func TestRealCash(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := a.ReadFile("Cash.SData")
	if err != nil {
		t.Fatal(err)
	}
	products, err := ParseCash(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(products) != 61 {
		t.Fatalf("unexpected product count %d", len(products))
	}
	if products[0].Code != "Basicos_1" || products[0].Items[0] != (CashItem{ItemID: 100067, Count: 255}) {
		t.Fatalf("first product %+v", products[0])
	}
}

func TestRealMonsters(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := a.ReadFile("Monster/Monster.SData")
	if err != nil {
		t.Fatal(err)
	}
	ms, err := ParseMonsters(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) == 0 {
		t.Fatal("empty monsters")
	}
	t.Logf("%d monsters; first: %+v", len(ms), ms[0])
}

func TestRealSkills(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := a.ReadFile("Character/Skill.SData")
	if err != nil {
		t.Fatal(err)
	}
	skills, err := ParseSkills(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) == 0 {
		t.Fatal("empty skills")
	}
	t.Logf("%d skills; 1/1: %+v", len(skills), skills[SkillKey{1, 1}])
}

func TestRealNPCs(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := a.ReadFile("Npc/NpcQuest.SData")
	if err != nil {
		t.Fatal(err)
	}
	defs, err := ParseNPCs(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) == 0 {
		t.Fatal("empty NPC lists")
	}
	for _, id := range []uint16{2, 32, 33, 44} {
		if gate, ok := defs[NPCKey{2, id}]; ok {
			t.Logf("gatekeeper 2/%d faction %d: %+v", id, gate.Faction, gate.Gates)
		}
	}
	t.Logf("%d NPCs; merchant 1: %+v", len(defs), defs[NPCKey{1, 1}])
}

func TestRealQuestsAndNPCLinks(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := a.ReadFile("Npc/NpcQuest.SData")
	if err != nil {
		t.Fatal(err)
	}
	npcs, quests, err := ParseQuests(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(quests) == 0 || quests[0].Name == "" {
		t.Fatal("missing quest definitions")
	}
	links := 0
	for _, n := range npcs {
		for _, id := range append(n.StartQuests, n.EndQuests...) {
			if id == 0 || int(id) > len(quests) {
				t.Fatalf("invalid quest link %d", id)
			}
			links++
		}
	}
	if len(quests) < 226 || quests[225].EndNPCType != 8 || quests[225].EndNPCID != 53 || quests[225].UserSelect != 2 || quests[225].Results[1].ItemType != 33 || quests[225].Results[1].ItemID != 87 {
		t.Fatal("quest 226 reward fixture did not match original data")
	}
	if links == 0 {
		t.Fatal("no NPC quest links")
	}
	t.Logf("quests=%d NPC quest links=%d first=%q start=%d/%d type=%d level=%d..%d jobs=%v faction=%d", len(quests), links, quests[0].Name, quests[0].StartNPCType, quests[0].StartNPCID, quests[0].StartType, quests[0].MinLevel, quests[0].MaxLevel, quests[0].Jobs, quests[0].Faction)
}

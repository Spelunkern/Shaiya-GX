//go:build integration

package app

import (
	"os"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// Exercise the real panel routing and app acknowledgement handling without a GL
// context: shop click beside inventory, item drag, bar assignment and use.
func TestPanelsAgainstRapture(t *testing.T) {
	st, w, send, wait := testworld.Enter(t)
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := inputApp()
	a.state = st
	a.world = w
	a.arc = arc
	a.skillDefs = nil
	a.npcDefs = nil
	a.loadNPCs()
	var merchant uint32
	var index int
	wait(func() bool {
		for id, e := range st.Entities {
			if e.Kind != game.KindNpc || e.NpcType != 1 {
				continue
			}
			for i, it := range a.npcDefs[sdata.NPCKey{Type: 1, ID: e.TypeID}].Items {
				d := a.itemsItem(it.Type, it.TypeID)
				if i < 256 && d != nil && d.RecoveryHP > 0 && d.Special == 0 && it.Type == 25 && (merchant == 0 || id < merchant) {
					merchant, index = id, i
				}
			}
		}
		return merchant != 0
	})
	send(proto.NpcTalkPacket(merchant))
	a.shop.npcID = merchant
	click := func(x, y int, shift bool) {
		t.Helper()
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true, Shift: shift}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true, Shift: shift}, 1024, 768)
	}
	click(717, 715, false)
	if !a.inventory.open {
		t.Fatal("bottom inventory control did not open the panel")
	}
	// Trading must use the moved frames for controls, slots and cross-panel drops.
	drag := func(x, y, destinationX, destinationY int) {
		a.worldControls(&platform.Input{MouseX: x + 50, MouseY: y + 15, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: destinationX + 50, MouseY: destinationY + 15, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: destinationX + 50, MouseY: destinationY + 15, Clicked: true}, 1024, 768)
	}
	ix, iy := a.inventoryUI.Origin(1024, 768)
	drag(ix, iy, 650, 100)
	sx, sy := a.marketUI.Origin(1024, 768)
	drag(sx, sy, 50, 100)
	if !a.inventoryUI.Position.Moved || !a.marketUI.Position.Moved {
		t.Fatal("trade frames did not move")
	}
	ack := func(cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for !cond() {
			a.pumpWorld()
			if a.state == nil {
				t.Fatal("app disconnected:", a.login.Message)
			}
			if time.Now().After(deadline) {
				t.Fatal("app acknowledgement timed out")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	confirmQuantity := func(count rune) {
		t.Helper()
		if a.shop.quantity == nil {
			t.Fatal("stack trade did not open quantity dialog")
		}
		qx, qy := a.quantityUI.X, a.quantityUI.Y
		click(qx+30, qy+40, false)
		a.worldControls(&platform.Input{Chars: []rune{count}, Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
		click(qx+30, qy+70, false)
		if a.shop.quantity != nil {
			t.Fatal("quantity confirmation did not close dialog")
		}
	}
	money := st.Details.Money
	mx, my := a.marketUI.Origin(1024, 768)
	for page := 0; page < index/ui.MarketPageSize; page++ {
		click(mx+146, my+309, false)
	}
	if a.shop.page != index/ui.MarketPageSize {
		t.Fatal("shop page control did not reach purchased product")
	}
	mcx, mcy, _ := ui.MarketCell(index % ui.MarketPageSize)
	click(mx+mcx+1, my+mcy+1, false)
	if !a.shop.pending {
		t.Fatal("inventory prevented the shop click")
	}
	ack(func() bool { return !a.shop.pending && st.LastShop.Type != 0 })
	if st.LastShop.Result != 0 || st.Details.Money >= money {
		t.Fatal("shop purchase was not confirmed", st.LastShop)
	}
	item := st.LastShop
	// Choose its bag and cell through the same clicks as the visible inventory.
	x, y := a.inventoryUI.Origin(1024, 768)
	click(x+15+int(item.Bag-1)*47, y+255, false)
	cx, cy, _, ok := ui.InventoryCell(item.Bag, item.Slot)
	if !ok {
		t.Fatal("purchased slot has no inventory cell")
	}
	// Purchase two more units through the native quantity dialog.
	beforeBuyMoney := st.Details.Money
	beforeBuy := st.Inventory[[2]uint8{item.Bag, item.Slot}]
	a.worldControls(&platform.Input{MouseX: mx + mcx + 1, MouseY: my + mcy + 1, Pressed: true, MouseDown: true}, 1024, 768)
	if !a.shop.dragging {
		t.Fatal("catalog item did not begin dragging")
	}
	a.worldControls(&platform.Input{MouseX: x + cx + 1, MouseY: y + cy + 1, Clicked: true}, 1024, 768)
	if a.shop.quantity == nil || a.shop.pending || a.shop.dragging || st.Details.Money != beforeBuyMoney || st.Inventory[[2]uint8{item.Bag, item.Slot}] != beforeBuy {
		t.Fatal("purchase drag changed state before acknowledgement")
	}
	confirmQuantity('2')
	if !a.shop.pending {
		t.Fatal("confirmed purchase was not sent")
	}
	ack(func() bool { return !a.shop.pending })
	if st.LastShop.Result != 0 || st.Details.Money >= beforeBuyMoney || st.LastShop.Count != item.Count+2 {
		t.Fatal("purchase drag was not confirmed", st.LastShop)
	}
	item = st.LastShop
	// Sell two units through the dialog; gaps are also native sale targets.
	beforeSale := st.Inventory[[2]uint8{item.Bag, item.Slot}]
	beforeSaleMoney := st.Details.Money
	a.worldControls(&platform.Input{MouseX: x + cx + 1, MouseY: y + cy + 1, Pressed: true, MouseDown: true}, 1024, 768)
	a.worldControls(&platform.Input{MouseX: mx + 48, MouseY: my + 113, Clicked: true}, 1024, 768)
	if a.shop.quantity == nil || a.inventory.pending || a.inventory.dragging || st.Inventory[[2]uint8{item.Bag, item.Slot}] != beforeSale || st.Details.Money != beforeSaleMoney {
		t.Fatal("sale drag changed state before acknowledgement")
	}
	confirmQuantity('2')
	if !a.inventory.pending {
		t.Fatal("confirmed sale was not sent")
	}
	ack(func() bool { return !a.inventory.pending })
	if st.LastShop.Result != 0 || st.Details.Money <= beforeSaleMoney || st.LastShop.Count != item.Count-2 {
		t.Fatal("sale drag was not confirmed", st.LastShop)
	}
	item = st.LastShop
	// Drag to a free cell; only pumpWorld may apply the server's new slots.
	destination := uint8(23)
	for destination > 0 {
		if _, occupied := st.Inventory[[2]uint8{item.Bag, destination}]; !occupied {
			break
		}
		destination--
	}
	dx, dy, _, _ := ui.InventoryCell(item.Bag, destination)
	source := [2]uint8{item.Bag, item.Slot}
	dest := [2]uint8{item.Bag, destination}
	before := st.Inventory[source]
	a.worldControls(&platform.Input{MouseX: x + cx + 1, MouseY: y + cy + 1, Pressed: true, MouseDown: true}, 1024, 768)
	if !a.inventory.dragging {
		t.Fatal("item did not begin dragging")
	}
	a.worldControls(&platform.Input{MouseX: x + dx + 1, MouseY: y + dy + 1, Clicked: true}, 1024, 768)
	if !a.inventory.pending || a.inventory.dragging || a.inventory.selected || st.Inventory[source] != before {
		t.Fatal("drag changed inventory before server acknowledgement")
	}
	ack(func() bool { return !a.inventory.pending })
	if _, occupied := st.Inventory[source]; occupied || st.Inventory[dest].Type != item.Type || st.Inventory[dest].Count != item.Count {
		t.Fatal("drag destination was not confirmed", st.Inventory)
	}
	item.Slot = destination
	cx, cy = dx, dy
	// A direct inventory drag binds the second bar without moving the item.
	bx2, by2 := a.actionBar.SlotOrigin(12, 1024, 768)
	a.worldControls(&platform.Input{MouseX: x + cx + 1, MouseY: y + cy + 1, Pressed: true, MouseDown: true}, 1024, 768)
	a.worldControls(&platform.Input{MouseX: bx2 + 1, MouseY: by2 + 1, MouseDown: true}, 1024, 768)
	a.worldControls(&platform.Input{MouseX: bx2 + 1, MouseY: by2 + 1, Clicked: true}, 1024, 768)
	if q, _ := a.shortcutAt(12); q == nil || q.Bag != item.Bag || q.Number != uint16(item.Slot) || st.Inventory[dest].Count != item.Count {
		t.Fatal("inventory shortcut drop moved or lost the item", q)
	}

	click(x+cx+1, y+cy+1, false)
	if !a.inventory.selected {
		t.Fatal("purchased item could not be selected")
	}
	ax, ay := ui.ActionBarOrigin(1024, 768)
	click(ax+31, ay+11, true)
	assigned := false
	for _, q := range st.QuickSlots {
		if q.Bar == 0 && q.Slot == 0 && q.Bag == item.Bag && q.Number == uint16(item.Slot) {
			assigned = true
		}
	}
	if !assigned {
		t.Fatal("item could not be assigned with inventory open")
	}
	a.worldControls(&platform.Input{Chars: []rune{'1'}}, 1024, 768)
	ack(func() bool { return st.LastItemUse.ID == st.SelfID })
	if !a.inventory.open || st.LastItemUse.Type != item.Type || st.LastItemUse.Count != item.Count-1 {
		t.Fatal("bar item use was not confirmed", st.LastItemUse)
	}
	if item.Count == 1 && a.inventory.selected {
		t.Fatal("consumed item stayed selected")
	}
	t.Logf("UI click purchase, quantity purchase/sale, bag move, bar assignment/use confirmed with inventory open; %d/%d", item.Type, item.TypeID)
	// Learn through the category tab and native plus button.
	a.loadSkills()
	var learn sdata.SkillKey
	for _, key := range a.availableSkills() {
		if a.canLearn(key) {
			learn = key
			break
		}
	}
	if learn.ID == 0 {
		t.Fatal("level-four fixture has no learnable UI skill")
	}
	cost := a.skillDefs[learn].Points
	click(752, 715, false)
	if !a.skills.open || a.inventory.open {
		t.Fatal("bottom skill control did not replace inventory")
	}
	bx, by := ui.SkillBookOrigin(1024, 768)
	category := int(a.skillDefs[learn].Show) - 1
	click(bx+4, by+75+category*62, false)
	row := -1
	for i, key := range a.visibleSkills() {
		if key == learn {
			row = i
			break
		}
	}
	if row < 0 {
		t.Fatal("learnable skill missing from its category")
	}
	for row-a.skills.offset >= ui.SkillBookRows {
		a.worldControls(&platform.Input{MouseX: bx + 100, MouseY: by + 200, Wheel: -120}, 1024, 768)
	}
	beforePoints := st.Details.SkillPoint
	entry := row - a.skills.offset
	click(bx+62+135*(entry%2), by+96+43*(entry/2), false)
	if !a.skills.pending {
		t.Fatal("native learn button did not request skill")
	}
	ack(func() bool { return !a.skills.pending })
	if st.LastLearn.Result != 0 || st.LastLearn.ID != learn.ID || st.LastLearn.Level != learn.Level || st.Details.SkillPoint != beforePoints-uint16(cost) {
		t.Fatal("native skill learning not confirmed", st.LastLearn)
	}
	click(bx+304, by+9, false)
	if a.skills.open {
		t.Fatal("native skill close did not close the book")
	}
	t.Logf("native skill category/learn confirmed: %d/%d", learn.ID, learn.Level)
	click(mx+229, my+6, false)
	if a.shop.npcID != 0 {
		t.Fatal("native shop close did not end conversation")
	}
	send(proto.LogoutPacket())
	ack(func() bool { return st.Phase == game.PhaseCharSelect })
}

// Run before fixture character creation: the account has no faction yet.
func TestFactionAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_FACTION") != "1" {
		t.Skip("requires new disposable account")
	}
	st, w, _, wait := testworld.Connect(t)
	wait(func() bool { return st.Phase == game.PhaseFaction })
	v, err := ui.NewFaction(inputTextures{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := inputApp()
	a.state, a.world, a.factionUI = st, w, v
	click := func(x, y int) {
		t.Helper()
		a.updateCharacters(&platform.Input{MouseX: x, MouseY: y, Pressed: true}, 1024, 768, .01)
		a.updateCharacters(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768, .01)
	}
	x, y := ui.FactionPanel(1024, 768, 0)
	click(x+30, y+100)
	click(1024/2-112, 768/2+212)
	if st.Phase != game.PhaseFaction || a.factionUI.Busy {
		t.Fatal("selection sent before confirmation")
	}
	click(1024/2-77, 768/2+198)
	if st.Phase != game.PhaseFaction || a.factionUI.Busy {
		t.Fatal("first confirmation sent faction")
	}
	click(1024/2-77, 768/2+80)
	if !a.factionUI.Busy || st.Phase != game.PhaseFaction {
		t.Fatal("faction was not requested or changed optimistically")
	}
	deadline := time.Now().Add(5 * time.Second)
	for st.Phase != game.PhaseCharSelect || a.factionUI.Busy {
		a.pumpWorld()
		if a.state == nil {
			t.Fatal("faction selection disconnected")
		}
		if time.Now().After(deadline) {
			t.Fatal("faction confirmation timed out")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if st.Faction.Faction != 0 {
		t.Fatal("Rapture confirmed wrong faction", st.Faction)
	}
	t.Log("new-account faction selected through both UI confirmations; Rapture confirmed Light")
}

// Run once after faction selection and before any fixture character exists.
func TestCreationAgainstRapture(t *testing.T) {
	name := os.Getenv("SHAIYA_TEST_CREATE")
	if name == "" {
		t.Skip("requires empty disposable roster")
	}
	st, w, _, wait := testworld.Connect(t)
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
	create, err := ui.NewCharCreate(inputTextures{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := inputApp()
	a.state, a.world, a.create, a.creating = st, w, create, true
	create.Open(0, st.Faction.Faction, st.Faction.MaxMode)
	x, y := ui.CreationOrigin(1024, 768)
	click := func(cx, cy int) {
		t.Helper()
		a.updateCharacters(&platform.Input{MouseX: x + cx, MouseY: y + cy, Pressed: true}, 1024, 768, .01)
		a.updateCharacters(&platform.Input{MouseX: x + cx, MouseY: y + cy, Clicked: true}, 1024, 768, .01)
	}
	click(149, 291) // Elf, then Mage and female preview
	click(173, 390)
	click(149, 329)
	if create.Draft.Race != 1 || create.Draft.Class != 4 || create.Draft.Gender != 1 {
		t.Fatal("native preview choices", create.Draft)
	}
	click(71, 291) // restore Human/Fighter for the gameplay fixture
	click(71, 329)
	click(190, 441)
	click(190, 474)
	click(190, 508)
	click(149, 555) // Normal mode
	a.updateCharacters(&platform.Input{Chars: []rune(name)}, 1024, 768, .01)
	expected := create.Draft
	expected.Name = name
	click(38, 623)
	if !create.Busy || !a.creating || st.Slots[0].ID != 0 {
		t.Fatal("creation changed roster before acknowledgement")
	}
	deadline := time.Now().Add(5 * time.Second)
	for a.creating || st.Slots[0].ID == 0 {
		a.pumpWorld()
		if a.state == nil {
			t.Fatal("creation disconnected")
		}
		if time.Now().After(deadline) {
			t.Fatal("creation confirmation timed out", a.accountMessage.Text)
		}
		time.Sleep(10 * time.Millisecond)
	}
	got := st.Slots[0]
	if got.Name != expected.Name || got.Race != expected.Race || got.Class != expected.Class || got.Gender != expected.Gender || got.Hair != expected.Hair || got.Face != expected.Face || got.Height != expected.Height || got.Mode != expected.Mode {
		t.Fatal("created roster does not match native controls", got, expected)
	}
	t.Logf("native creation confirmed: race=%d class=%d sex=%d hair=%d face=%d height=%d mode=%d", got.Race, got.Class, got.Gender, got.Hair, got.Face, got.Height, got.Mode)
}

// The fixture must exist but not be in world. Delete/restore only this account's
// disposable character, then leave it available for the gameplay probes.
func TestAccountDialogsAgainstRapture(t *testing.T) {
	st, w, _, wait := testworld.Connect(t)
	wait(func() bool { return st.Phase == game.PhaseCharSelect && st.Slots[0].ID != 0 })
	a := inputApp()
	a.state, a.world = st, w
	var err error
	a.chars, err = ui.NewCharSelect(inputTextures{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	a.create, err = ui.NewCharCreate(inputTextures{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	a.chars.SetChars(st.Slots)
	original := st.Slots[0]
	frame := func(in platform.Input) { a.updateCharacters(&in, 1024, 768, .01) }
	ack := func() {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for a.account.op != 0 || a.create.Busy {
			a.pumpWorld()
			if a.state == nil {
				t.Fatal("account action disconnected")
			}
			if time.Now().After(deadline) {
				t.Fatal("account action did not acknowledge")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	enter := platform.Input{Keys: []platform.Key{platform.KeyEnter}}
	escape := platform.Input{Keys: []platform.Key{platform.KeyEscape}}

	// Rejected duplicate creation preserves the draft. Dismissing the dialog
	// must not reuse its Enter to send another request.
	a.creating = true
	a.create.Open(1, st.Faction.Faction, st.Faction.MaxMode)
	a.create.Name.Text = original.Name
	frame(enter)
	ack()
	if !a.accountMessage.Visible || !a.creating || a.create.Name.Text != original.Name {
		t.Fatal("duplicate creation did not show a modal with the draft intact")
	}
	frame(enter)
	if a.create.Busy || a.accountMessage.Visible {
		t.Fatal("dismissal resubmitted creation")
	}
	frame(escape)
	if a.creating {
		t.Fatal("creation did not close after dismissal")
	}

	x, y := 1024/16+30, 768/2-295+479
	clickDelete := func() {
		frame(platform.Input{MouseX: x, MouseY: y, Pressed: true})
		frame(platform.Input{MouseX: x, MouseY: y, Clicked: true})
	}
	clickDelete()
	if !a.accountMessage.Confirm || a.account.confirm.ID != original.ID || a.account.op != 0 {
		t.Fatal("delete did not open a confirmation bound to the character")
	}
	frame(escape)
	if st.Slots[0].PendingDelete || a.account.op != 0 {
		t.Fatal("cancel deleted the character")
	}
	clickDelete()
	frame(enter)
	if a.account.op != proto.DeleteCharacter || st.Slots[0].PendingDelete {
		t.Fatal("delete changed state before acknowledgement")
	}
	frame(enter)
	ack()
	if !st.Slots[0].PendingDelete || !a.accountMessage.Visible {
		t.Fatal("delete result not applied")
	}
	frame(enter)
	frame(enter)
	if st.Phase != game.PhaseCharSelect || a.account.op != 0 {
		t.Fatal("deleted character could start")
	}

	clickDelete() // Native Delete control offers restore for a pending deletion.
	frame(enter)
	if a.account.op != proto.RestoreCharacter || !st.Slots[0].PendingDelete {
		t.Fatal("restore changed state before acknowledgement")
	}
	ack()
	if st.Slots[0].PendingDelete || !a.accountMessage.Visible {
		t.Fatal("restore result not applied")
	}
	frame(escape)

	// An explicit server rejection releases the pending lock and is visible.
	a.sendAccountAction(proto.SelectCharacter, 0)
	ack()
	if !a.accountMessage.Visible || a.chars.Busy || st.Phase != game.PhaseCharSelect {
		t.Fatal("selection rejection did not release the UI")
	}
	frame(enter)
	if st.Phase != game.PhaseCharSelect || a.account.op != 0 {
		t.Fatal("notice dismissal started a character")
	}
	t.Log("duplicate creation, cancel, confirmed soft delete/restore and rejected selection passed through UI")
}

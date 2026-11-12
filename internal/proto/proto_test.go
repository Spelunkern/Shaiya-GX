package proto

import (
	"bytes"
	"testing"
)

func TestRestoreCharacterWriter(t *testing.T) {
	want := []byte{0x0f, 0x01, 0x44, 0x33, 0x22, 0x11}
	if got := RestoreCharacterPacket(0x11223344); !bytes.Equal(got, want) {
		t.Fatalf("0x59c240 restore writer: %x, want %x", got, want)
	}
}

// Sizes pinned to the original writers (Game.exe 0x597f60..0x59a000).
func TestWriterSizes(t *testing.T) {
	block := make([]byte, 128)
	key, _ := KeyReply(block)
	cases := map[string]struct {
		p    []byte
		size int
	}{
		"A101 0x5992a0": {key, 0x83},
		"A102 0x5992f0": {LoginRequestPacket("admin1", "pw"), 0x32},
		"A202 0x599440": {SelectServerPacket(1, ClientVersion), 7},
		"A301 0x599480": {WorldHelloPacket(1, [16]byte{}), 0x16},
		"0109 0x598460": {FactionPacket(1), 3},
		"0102 0x5980e0": {CreateCharacterPacket(NewCharacter{Name: "Abc"}), 3 + 0xb},
		"0103 0x598180": {DeleteCharacterPacket(1), 6},
		"010F 0x59c240": {RestoreCharacterPacket(1), 6},
		"0104 0x5981b0": {SelectCharacterPacket(1), 6},
		"0107 0x5981e0": {LogoutPacket(), 2},
		"0201 0x598210": {MapLoadedPacket(), 2},
		"0501 0x598980": {MovePacket(0, 0, 1, 2, 3), 0x11},
		"1101 0x599a60": {ChatPacket(ChannelNormal, false, "hola"), 5 + 3},
	}
	for name, c := range cases {
		if len(c.p) != c.size {
			t.Errorf("%s: %d bytes, original %d", name, len(c.p), c.size)
		}
	}
}

func TestReaderStopsOnShort(t *testing.T) {
	if _, err := DecodeDetails([]byte{0x05, 0x01, 1}); err == nil {
		t.Fatal("short 0x0105 must fail")
	}
}

func FuzzDecoders(f *testing.F) {
	f.Add([]byte{1, 1, 0, 0, 0, 0})
	f.Fuzz(func(t *testing.T, b []byte) {
		DecodeCharacterSlot(b)
		DecodeUserShape(b)
		DecodePools(b)
		DecodeMobHealth(b)
		DecodeAttackResult(b)
		DecodeResurrection(b)
		DecodeInventory(b)
		DecodeItemMove(b)
		DecodeItemAdd(b)
		DecodeGroundItem(b)
		DecodeSkills(b)
		DecodeQuickSlots(b)
		DecodeLearnResult(b)
		DecodeSkillResult(b)
		DecodeShopResult(b)
		DecodeItemUse(b)
		DecodeEquipment(b)
		DecodeDetails(b)
		DecodeServerList(b)
		DecodeLoginHello(b)
		DecodeChatByID(b)
	})
}

func TestReviveChoicePackets(t *testing.T) {
	for _, mode := range []ReviveMode{ReviveBind, ReviveParty, ReviveRuneBind, ReviveRuneHere} {
		p := ReviveChoicePacket(mode)
		if len(p) != 3 || p[0] != 0x53 || p[1] != 0x05 || p[2] != byte(mode) {
			t.Fatalf("mode %d: %x", mode, p)
		}
	}
}

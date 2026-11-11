// Command shaiyacli is the headless client: it logs in, picks a character,
// enters the world, walks, chats and logs out, using the same session and
// game packages as the windowed client.
//
//	SHAIYA_PASSWORD=... shaiyacli -user normal1 -slot 0
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/Spelunkern/shaiya-go-client/internal/game"
	"github.com/Spelunkern/shaiya-go-client/internal/proto"
	"github.com/Spelunkern/shaiya-go-client/internal/session"
)

func main() {
	login := flag.String("login", "127.0.0.1:30800", "login server address")
	worldPort := flag.Int("world-port", 30810, "world server port")
	user := flag.String("user", "", "account name")
	serverID := flag.Int("server", 0, "server id (0 = first listed)")
	slot := flag.Int("slot", -1, "character slot to play (-1 = first used)")
	faction := flag.Int("faction", 0, "country to pick if the account has none (0 light, 1 fury)")
	create := flag.String("create", "", "create a human fighter with this name in -slot first")
	chat := flag.String("say", "hola desde shaiyacli", "normal chat line to send in world")
	steps := flag.Int("steps", 5, "movement packets to send")
	stay := flag.Duration("stay", 3*time.Second, "time to stay in world before logging out")
	verbose := flag.Bool("v", false, "print every event")
	flag.Parse()
	password := os.Getenv("SHAIYA_PASSWORD")
	if *user == "" || password == "" {
		log.Fatal("need -user and SHAIYA_PASSWORD")
	}
	if err := run(*login, *worldPort, *user, password, *serverID, *slot, *faction, *create, *chat, *steps, *stay, *verbose); err != nil {
		log.Fatal(err)
	}
}

func run(addr string, port int, user, password string, serverID, slot, faction int, create, say string, steps int, stay time.Duration, verbose bool) error {
	l, err := session.Dial(addr, user, password)
	if err != nil {
		return err
	}
	log.Printf("login ok uid=%d permission=%d servers=%d", l.Result.UserUID, l.Result.Permission, len(l.Servers))
	if len(l.Servers) == 0 {
		return fmt.Errorf("no servers listed")
	}
	id := uint8(serverID)
	if serverID == 0 {
		id = l.Servers[0].ID
	}
	ip, err := l.SelectServer(id, proto.ClientVersion)
	if err != nil {
		return err
	}
	w, err := l.DialWorld(ip, port)
	if err != nil {
		return err
	}
	defer w.Close()
	log.Printf("world %s handshake ok", ip)

	st := game.New()
	// next waits until cond holds, applying every packet meanwhile.
	next := func(what string, timeout time.Duration, cond func() bool) error {
		deadline := time.After(timeout)
		for !cond() {
			select {
			case p, ok := <-w.Packets:
				if !ok {
					return fmt.Errorf("connection closed while waiting for %s: %v", what, w.Err())
				}
				ev, err := st.Apply(p)
				if err != nil {
					log.Printf("bad packet: %v", err)
				} else if verbose || (ev.Text != "" && ev.Op != proto.CharacterMove) {
					if ev.Text == "" {
						ev.Text = "(not modeled)"
					}
					log.Printf("<- 0x%04X %s", uint16(ev.Op), ev.Text)
				}
			case <-deadline:
				return fmt.Errorf("timed out waiting for %s (phase %s)", what, st.Phase)
			}
		}
		return nil
	}
	if err := next("character select", 10*time.Second, func() bool { return st.Phase == game.PhaseCharSelect || st.Phase == game.PhaseFaction }); err != nil {
		return err
	}
	if st.Phase == game.PhaseFaction {
		log.Printf("-> faction %d", faction)
		w.Send(proto.FactionPacket(uint8(faction)))
		if err := next("faction", 10*time.Second, func() bool { return st.Phase == game.PhaseCharSelect }); err != nil {
			return err
		}
	}
	if create != "" {
		if slot < 0 {
			return fmt.Errorf("-create needs -slot")
		}
		log.Printf("-> create %q in slot %d", create, slot)
		w.Send(proto.CreateCharacterPacket(proto.NewCharacter{Slot: uint8(slot), Race: 0, Mode: 1, Hair: 1, Face: 1, Height: 1, Class: 0, Gender: 0, Name: create}))
		if err := next("created character", 10*time.Second, func() bool { _, ok := st.Slots[uint8(slot)]; return ok }); err != nil {
			return err
		}
	}
	if slot < 0 {
		for i := uint8(0); i < 5 && slot < 0; i++ {
			if _, ok := st.Slots[i]; ok {
				slot = int(i)
			}
		}
	}
	ch, ok := st.Slots[uint8(slot)]
	if !ok {
		return fmt.Errorf("no character in slot %d (account has %d)", slot, len(st.Slots))
	}
	log.Printf("-> select %s (id %d)", ch.Name, ch.ID)
	w.Send(proto.SelectCharacterPacket(ch.ID))
	if err := next("selection", 10*time.Second, func() bool { return st.Phase == game.PhaseLoading }); err != nil {
		return err
	}
	// The original reports the map loaded after the initial state, whose
	// last packet is 0x010B; then the server places us (0x020B, 0x0201).
	if err := next("initial state", 15*time.Second, func() bool { return st.Unknown[proto.CharacterSkillBar] > 0 }); err != nil {
		return err
	}
	log.Printf("-> map loaded")
	w.Send(proto.MapLoadedPacket())
	if err := next("placement", 10*time.Second, func() bool { return st.Phase == game.PhaseInWorld }); err != nil {
		return err
	}
	log.Printf("in world: map %d at (%.1f, %.1f, %.1f), %d entities in view", st.MapID, st.Self.X, st.Self.Y, st.Self.Z, len(st.Entities))
	x, y, z := st.Self.X, st.Self.Y, st.Self.Z
	for i := 0; i < steps; i++ {
		x += float32(math.Cos(float64(i))) * 1.5
		z += float32(math.Sin(float64(i))) * 1.5
		w.Send(proto.MovePacket(1, 0, x, y, z))
		next("step", 300*time.Millisecond, func() bool { return false })
	}
	if steps > 0 {
		w.Send(proto.MovePacket(0, 0, x, y, z))
	}
	if say != "" {
		log.Printf("-> say %q", say)
		w.Send(proto.ChatPacket(proto.ChannelNormal, false, say))
	}
	next("stay", stay, func() bool { return false })
	counts := map[game.Kind]int{}
	for _, e := range st.Entities {
		counts[e.Kind]++
	}
	log.Printf("view: %d users, %d mobs, %d npcs; chat lines %d", counts[game.KindUser], counts[game.KindMob], counts[game.KindNpc], len(st.Chat))
	log.Printf("-> logout")
	w.Send(proto.LogoutPacket())
	if err := next("logout", 15*time.Second, func() bool { return st.Phase == game.PhaseCharSelect }); err != nil {
		return err
	}
	log.Printf("logout ok; opcodes not modeled yet: %d distinct", len(st.Unknown))
	if verbose {
		for op, n := range st.Unknown {
			log.Printf("   0x%04X x%d", uint16(op), n)
		}
	}
	return nil
}

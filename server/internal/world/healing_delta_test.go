package world

import "testing"

func TestMobHealingReportsEffectiveGain(t *testing.T) {
	const mapID = 60999
	s := shard(mapID)
	s.mu.Lock()
	old := s.entities.Mobs
	s.entities.Mobs = []MapMob{{ID: 991, CurrentHP: 98, MaxHP: 100}}
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.entities.Mobs = old; s.mu.Unlock() }()
	m, gain, ok := HealMob(mapID, 991, 500)
	if !ok || gain != 2 || m.CurrentHP != 100 {
		t.Fatal(m.CurrentHP, gain, ok)
	}
	_, gain, ok = HealMob(mapID, 991, 500)
	if !ok || gain != 0 {
		t.Fatal("full-health mob gained HP", gain, ok)
	}
}

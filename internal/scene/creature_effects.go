package scene

import (
	"fmt"
	"path"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type creatureBoneEffect struct {
	bone, index int
	attempted   bool
	instance    *Effect
}

func (a *Assets) monsterEmitter(index int) (*Effect, error) {
	if !a.monsterEffectTried {
		a.monsterEffectTried = true
		raw, err := a.Arc.ReadFile("Effect/Monster.EFT")
		if err != nil {
			raw, err = a.Arc.ReadFile("Monster/Monster.EFT")
		}
		if err != nil {
			return nil, err
		}
		a.monsterEffectLibrary, err = effect.ParseEFT(raw)
		if err != nil {
			return nil, err
		}
	}
	l := a.monsterEffectLibrary
	if l == nil || index < 0 || index >= len(l.Effects) {
		return nil, fmt.Errorf("monster emitter %d unavailable", index)
	}
	single := *l
	single.Sequences = []effect.Sequence{{Records: []effect.Record{{Effect: int32(index)}}}}
	return loadEffectLibrary(a, fmt.Sprintf("Effect/Monster.EFT#emitter%d", index), 0, &single)
}

func (m *CreatureActor) queueModelEffect(action int) {
	slot := action - 2
	if action == 5 {
		slot = 3
	}
	if slot < 0 || slot >= len(m.visual.Effects) {
		return
	}
	name := m.visual.Effects[slot]
	if strings.EqualFold(path.Ext(name), ".eft") && len(m.pendingEffects) < 4 {
		m.pendingEffects = append(m.pendingEffects, path.Base(strings.ReplaceAll(name, `\`, "/")))
	}
}

// Prepare once per frame, independently of main/reflection draw passes. Two
// uncached emitter creations per frame bound crowded-area preparation work.
func (m *CreatureActor) PrepareEffects(eye render.V3, dt float32, enabled, alive bool, budget *int) {
	if m == nil {
		return
	}
	origin := render.V3{m.World[12], m.World[13], m.World[14]}
	reach := float32(50)
	if len(m.boneEffects) > 0 {
		reach = 55
	}
	if !enabled || origin.Sub(eye).Len() > reach {
		m.boneEffects = nil
		m.boneEffectsReady = false
		m.attackEffects = nil
		m.pendingEffects = nil
		return
	}
	if !m.boneEffectsReady {
		for _, pair := range m.visual.BoneEffects {
			if pair[0] >= 0 && pair[1] >= 0 {
				m.boneEffects = append(m.boneEffects, creatureBoneEffect{bone: int(pair[0]), index: int(pair[1])})
			}
		}
		m.boneEffectsReady = true
	}
	poses := m.finals()
	for i := range m.boneEffects {
		e := &m.boneEffects[i]
		if e.bone >= len(poses) {
			continue
		}
		if e.instance == nil && !e.attempted && alive && *budget > 0 {
			*budget--
			e.attempted = true
			var err error
			e.instance, err = m.assets.monsterEmitter(e.index)
			if err != nil {
				m.assets.Missing[err.Error()] = true
			}
		}
		if e.instance != nil {
			if !alive {
				e.instance.Stop()
			}
			e.instance.SetBoneTransform(poses[e.bone].Mul(m.World))
			e.instance.Update(dt)
		}
	}
	for len(m.pendingEffects) > 0 && *budget > 0 {
		name := m.pendingEffects[0]
		m.pendingEffects = m.pendingEffects[1:]
		*budget--
		x, err := LoadEffect(m.assets, "Effect/"+name, 0)
		if err != nil {
			m.assets.Missing[err.Error()] = true
			continue
		}
		x.SetTransform(m.World)
		m.attackEffects = append(m.attackEffects, x)
	}
	kept := m.attackEffects[:0]
	for _, x := range m.attackEffects {
		if m.cueTime <= 0 && m.action != 5 {
			x.Stop()
		}
		x.Update(dt)
		if !x.Done() {
			kept = append(kept, x)
		}
	}
	clear(m.attackEffects[len(kept):])
	m.attackEffects = kept
}

func (m *CreatureActor) DrawEffects(s *render.Scene3D) {
	if m == nil || s.Shadow {
		return
	}
	for _, e := range m.boneEffects {
		if e.instance != nil {
			e.instance.Draw(s)
		}
	}
	for _, e := range m.attackEffects {
		e.Draw(s)
	}
}

// Package dungeon owns temporary dungeon-run state. Nothing in this package is
// written to a Player file: a run disappears safely when it ends.
package dungeon

import (
	"fmt"
	"strings"
	"sync"
)

const (
	MaxRooms           = 5
	StartingHealth     = 500
	StartingLives      = 3
	StartingSouls      = 0
	StartingDamage     = 20
	StartingCritChance = 0.0
	StartingCritMult   = 100.0
)

// Run is one private dungeon attempt for one player. Party support will use
// the same structure later by adding member GrowIDs to a shared run.
type Run struct {
	GrowID       string
	WorldName    string
	InstanceID   int
	Room         int
	Souls        int
	Health       int
	MaxHealth    int
	Lives        int
	Damage       int
	CritChance   float32
	CritMult     float32
	Abilities    []string
	BackpackOpen bool
	Objectives   map[Target]bool
}

// Target is a destructible dungeon objective. It is intentionally separate
// from normal-world blocks so completing a room never creates normal drops.
type Target struct{ X, Y int }

// Manager is deliberately memory-only. It prevents dungeon abilities, Souls,
// health, or inventory state leaking into normal worlds after a disconnect.
type Manager struct {
	mu   sync.RWMutex
	runs map[string]*Run
}

func NewManager() *Manager {
	return &Manager{runs: make(map[string]*Run)}
}

func (m *Manager) Start(growID string, userID int) *Run {
	key := strings.ToUpper(strings.TrimSpace(growID))
	r := &Run{
		GrowID:     growID,
		WorldName:  fmt.Sprintf("DUNGEON__%d%d", userID, 1),
		InstanceID: userID,
		Room:       1,
		Souls:      StartingSouls,
		Health:     StartingHealth,
		MaxHealth:  StartingHealth,
		Lives:      StartingLives,
		Damage:     StartingDamage,
		CritChance: StartingCritChance,
		CritMult:   StartingCritMult,
		Abilities:  []string{},
		Objectives: objectivesForRoom(1),
	}
	m.mu.Lock()
	m.runs[key] = r
	m.mu.Unlock()
	return clone(r)
}

func (m *Manager) Get(growID string) (*Run, bool) {
	m.mu.RLock()
	r, ok := m.runs[strings.ToUpper(strings.TrimSpace(growID))]
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return clone(r), true
}

func (m *Manager) End(growID string) {
	m.mu.Lock()
	delete(m.runs, strings.ToUpper(strings.TrimSpace(growID)))
	m.mu.Unlock()
}

// CompleteTarget marks one dungeon objective complete and awards temporary
// Souls. It returns an updated snapshot and whether that room is now clear.
func (m *Manager) CompleteTarget(growID string, x, y int) (*Run, bool, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[strings.ToUpper(strings.TrimSpace(growID))]
	if !ok {
		return nil, false, false
	}
	target := Target{X: x, Y: y}
	if !r.Objectives[target] {
		return clone(r), false, false
	}
	delete(r.Objectives, target)
	r.Souls += 15
	return clone(r), true, len(r.Objectives) == 0
}

func (m *Manager) AddAbility(growID, abilityName string, cost int) *Run {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[strings.ToUpper(strings.TrimSpace(growID))]
	if !ok || r.Souls < cost {
		return nil
	}
	r.Souls -= cost
	r.Abilities = append(r.Abilities, abilityName)
	if strings.Contains(abilityName, "+10 Dmg") {
		r.Damage += 10
	} else if strings.Contains(abilityName, "+25 Dmg") {
		r.Damage += 25
	} else if strings.Contains(abilityName, "+100 Max HP") {
		r.MaxHealth += 100
		r.Health += 100
	}
	return clone(r)
}

// Advance moves a cleared run to the next private room. The final return is
// true only after clearing room five.
func (m *Manager) Advance(growID string) (*Run, bool, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[strings.ToUpper(strings.TrimSpace(growID))]
	if !ok || len(r.Objectives) != 0 {
		return nil, false, false
	}
	if r.Room >= MaxRooms {
		return clone(r), false, true
	}
	r.Room++
	r.WorldName = fmt.Sprintf("DUNGEON__%d%d", r.InstanceID, r.Room)
	r.Objectives = objectivesForRoom(r.Room)
	return clone(r), true, false
}

func (m *Manager) IsTarget(growID string, x, y int) bool {
	m.mu.RLock()
	r, ok := m.runs[strings.ToUpper(strings.TrimSpace(growID))]
	if !ok {
		m.mu.RUnlock()
		return false
	}
	_, target := r.Objectives[Target{X: x, Y: y}]
	m.mu.RUnlock()
	return target
}

func clone(r *Run) *Run {
	if r == nil {
		return nil
	}
	c := *r
	if r.Objectives != nil {
		c.Objectives = make(map[Target]bool, len(r.Objectives))
		for target, active := range r.Objectives {
			c.Objectives[target] = active
		}
	}
	if len(r.Abilities) > 0 {
		c.Abilities = make([]string, len(r.Abilities))
		copy(c.Abilities, r.Abilities)
	}
	return &c
}

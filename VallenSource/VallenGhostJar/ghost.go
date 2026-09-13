package ghostjar

import (
	"encoding/binary"
	"math"
	"math/rand"
	"sync"
	"time"
)

// ─────────────────────────────────────────────
// Constants
// ─────────────────────────────────────────────

const (
	// Items
	ItemProtonPack         = 3716 // Ecto-Pack (Senjata Proton)
	ItemGhostJar           = 3720 // Empty Jar (Perangkap Hantu)
	ItemGhostInJar         = 3722 // Ghost in a Jar (Hantu Biasa)
	ItemShadowGhostInJar   = 6080 // Shadow Ghost in a Jar

	// Ghost NPC Types
	TypeGhostNormal = 1
	TypeJarTrap     = 2
	TypeGhostShadow = 13

	// Actions
	ActionSpawn  = 2
	ActionMove   = 3
	ActionCaught = 4

	// States
	StateNormal    = 0
	StateFleeing   = 500
	StateDespawned = 10
)

// ─────────────────────────────────────────────
// WorldGhost Structure
// ─────────────────────────────────────────────

type WorldGhost struct {
	Type   uint8 `json:"type"`   // 1 = Ghost, 2 = Jar Trap, 13 = Shadow Ghost
	ID     uint8 `json:"id"`     // Index ID ghost di world
	Action uint8 `json:"action"` // 2 = Spawn, 3 = Move, 4 = Caught

	LastPos [2]float32 `json:"last_pos"` // Posisi awal (pixel)
	NewPos  [2]float32 `json:"new_pos"`  // Posisi target (pixel)
	Speed   float32    `json:"speed"`    // Kecepatan (pixel/detik)

	IsCaught uint32 `json:"is_caught"` // 0 = Bebas, 1 = Tersedot jar
	State    uint32 `json:"state"`     // 0 = Normal, 500 = Fleeing/Shot, 10 = Despawned, atau ID target jar

	// Internal timing & movement
	Distance  float64    `json:"distance"`   // Total jarak dari LastPos ke NewPos
	Time      float64    `json:"time"`       // Waktu berjalan dalam detik
	VisualPos [2]float32 `json:"visual_pos"` // Posisi realtime saat ini
	MaxTime   float64    `json:"max_time"`   // Durasi waktu tempuh (distance / speed)

	// Owner NetID for trap jar
	OwnerNetID int `json:"owner_net_id"`
}

// ─────────────────────────────────────────────
// Callbacks Interface for Decoupled Server Integration
// ─────────────────────────────────────────────

type PlayerInfo struct {
	NetID   int
	GrowID  string
	UserID  int
	X       float32
	Y       float32
	HasMod  bool // true jika player sedang punya playmod 114
}

type Callbacks struct {
	BroadcastToWorld       func(worldName string, packet []byte)
	GetWorldPlayers        func(worldName string) []PlayerInfo
	SendTalkBubble         func(worldName string, netID int, message string)
	SendConsoleMessage     func(worldName string, netID int, message string)
	SendParticle           func(worldName string, x, y float32, particleID int32)
	SendItemCaughtAnim     func(worldName string, x, y float32, toNetID int32, itemID int32)
	GivePlayerItem         func(growID string, itemID int, count int) bool
	ApplyPlayerMod         func(worldName string, netID int, modID int)
}

// ─────────────────────────────────────────────
// GhostManager
// ─────────────────────────────────────────────

type GhostManager struct {
	mu     sync.RWMutex
	ghosts map[string][]*WorldGhost // worldName -> list of WorldGhost
	rnd    *rand.Rand
}

func NewManager() *GhostManager {
	return &GhostManager{
		ghosts: make(map[string][]*WorldGhost),
		rnd:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SpawnGhost menambahkan ghost baru ke world
func (m *GhostManager) SpawnGhost(worldName string, ghostType uint8, x, y float32, ownerNetID int) *WorldGhost {
	m.mu.Lock()
	defer m.mu.Unlock()

	worldGhosts := m.ghosts[worldName]
	id := uint8(len(worldGhosts))

	ghost := &WorldGhost{
		Type:       ghostType,
		ID:         id,
		Action:     ActionSpawn,
		LastPos:    [2]float32{x, y},
		NewPos:     [2]float32{x, y},
		VisualPos:  [2]float32{x, y},
		Speed:      33,
		State:      StateNormal,
		OwnerNetID: ownerNetID,
	}

	if ghostType == TypeJarTrap {
		ghost.MaxTime = 5.0 // Jar aktif selama 5 detik
		ghost.Distance = float64(ownerNetID) // Kompatibilitas referensi
	}

	m.ghosts[worldName] = append(worldGhosts, ghost)
	return ghost
}

// GetWorldGhosts mengembalikan semua ghost aktif di world
func (m *GhostManager) GetWorldGhosts(worldName string) []*WorldGhost {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var active []*WorldGhost
	for _, g := range m.ghosts[worldName] {
		if g.State != StateDespawned {
			active = append(active, g)
		}
	}
	return active
}

// ─────────────────────────────────────────────
// Packet Builders
// ─────────────────────────────────────────────

// BuildGhostUpdatePacket membuat Packet Type 34 untuk single ghost update
func BuildGhostUpdatePacket(ghost *WorldGhost) []byte {
	pkt := make([]byte, 60)
	binary.LittleEndian.PutUint32(pkt[0:], 4) // NET_MESSAGE_GAME_PACKET
	pkt[4] = 34                              // m_type = 34
	pkt[5] = ghost.Type                      // m_npc_type
	pkt[6] = ghost.ID                        // m_npc_id
	pkt[7] = ghost.Action                    // m_npc_action

	binary.LittleEndian.PutUint32(pkt[28:], math.Float32bits(ghost.LastPos[0])) // m_vec_x
	binary.LittleEndian.PutUint32(pkt[32:], math.Float32bits(ghost.LastPos[1])) // m_vec_y
	binary.LittleEndian.PutUint32(pkt[36:], math.Float32bits(ghost.NewPos[0]))  // m_vec2_x
	binary.LittleEndian.PutUint32(pkt[40:], math.Float32bits(ghost.NewPos[1]))  // m_vec2_y
	binary.LittleEndian.PutUint32(pkt[44:], math.Float32bits(ghost.Speed))      // m_npc_speed
	binary.LittleEndian.PutUint32(pkt[48:], ghost.IsCaught)                     // m_int_x
	binary.LittleEndian.PutUint32(pkt[52:], ghost.State)                        // m_npc_state / m_int_y
	binary.LittleEndian.PutUint32(pkt[56:], 0)                                  // m_data_size
	return pkt
}

// BuildWorldGhostsJoinPacket menyusun Packet Type 34 batch saat player masuk world
func (m *GhostManager) BuildWorldGhostsJoinPacket(worldName string) []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var active []*WorldGhost
	for _, g := range m.ghosts[worldName] {
		if g.State != StateDespawned {
			active = append(active, g)
		}
	}

	if len(active) == 0 {
		return nil
	}

	// 1 byte count + 30 bytes per ghost
	extraData := make([]byte, 1+len(active)*30)
	extraData[0] = uint8(len(active))

	offset := 1
	for _, g := range active {
		extraData[offset] = g.Type
		extraData[offset+1] = g.ID
		binary.LittleEndian.PutUint32(extraData[offset+2:], math.Float32bits(g.VisualPos[0]))
		binary.LittleEndian.PutUint32(extraData[offset+6:], math.Float32bits(g.VisualPos[1]))
		binary.LittleEndian.PutUint32(extraData[offset+10:], math.Float32bits(g.NewPos[0]))
		binary.LittleEndian.PutUint32(extraData[offset+14:], math.Float32bits(g.NewPos[1]))
		binary.LittleEndian.PutUint32(extraData[offset+18:], g.IsCaught)
		binary.LittleEndian.PutUint32(extraData[offset+22:], g.State)
		binary.LittleEndian.PutUint32(extraData[offset+26:], math.Float32bits(g.Speed))
		offset += 30
	}

	pkt := make([]byte, 60)
	binary.LittleEndian.PutUint32(pkt[0:], 4)                       // NET_MESSAGE_GAME_PACKET
	pkt[4] = 34                                                    // m_type = 34
	binary.LittleEndian.PutUint32(pkt[16:], 8)                     // m_flags = 8
	binary.LittleEndian.PutUint32(pkt[56:], uint32(len(extraData))) // m_data_size

	return append(pkt, extraData...)
}

// BuildParticlePacket membuat GamePacket Type 17 untuk efek slime particle
func BuildParticlePacket(x, y float32, particleID int32) []byte {
	pkt := make([]byte, 60)
	binary.LittleEndian.PutUint32(pkt[0:], 4) // NET_MESSAGE_GAME_PACKET
	pkt[4] = 17                              // m_type = 17
	binary.LittleEndian.PutUint32(pkt[8:], uint32(particleID))
	binary.LittleEndian.PutUint32(pkt[28:], math.Float32bits(x))
	binary.LittleEndian.PutUint32(pkt[32:], math.Float32bits(y))
	binary.LittleEndian.PutUint32(pkt[40:], math.Float32bits(float32(particleID)))
	return pkt
}

// BuildPunchAckPacket membuat GamePacket Type 21 untuk respon tembakan proton pack
func BuildPunchAckPacket() []byte {
	pkt := make([]byte, 60)
	binary.LittleEndian.PutUint32(pkt[0:], 4) // NET_MESSAGE_GAME_PACKET
	pkt[4] = 21                              // m_type = 21
	return pkt
}

// BuildGhostCaughtAnimationPacket membuat GamePacket Type 19 (animasi item melayang saat ghost tertangkap)
func BuildGhostCaughtAnimationPacket(x, y float32, toNetID int32, itemID int32) []byte {
	pkt := make([]byte, 60)
	binary.LittleEndian.PutUint32(pkt[0:], 4) // NET_MESSAGE_GAME_PACKET
	pkt[4] = 19                              // m_type = 19
	pkt[7] = 5                               // raw[3] = 5 di C++
	binary.LittleEndian.PutUint32(pkt[8:], uint32(toNetID))
	binary.LittleEndian.PutUint32(pkt[28:], math.Float32bits(x))
	binary.LittleEndian.PutUint32(pkt[32:], math.Float32bits(y))
	binary.LittleEndian.PutUint32(pkt[48:], uint32(itemID)) // punchX = 3722
	return pkt
}

// ─────────────────────────────────────────────
// Proton Pack Raycast & Scaring Logic (Punch.txt)
// ─────────────────────────────────────────────

func (m *GhostManager) HandleProtonPunch(worldName string, playerX, playerY, punchX, punchY float64) ([]*WorldGhost, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ghosts, ok := m.ghosts[worldName]
	if !ok || len(ghosts) == 0 {
		return nil, false
	}

	lineLength := math.Sqrt(math.Pow(punchX-playerX, 2) + math.Pow(punchY-playerY, 2))
	if lineLength == 0 {
		return nil, false
	}

	var affected []*WorldGhost
	for _, ghost := range ghosts {
		if ghost.State == StateDespawned || ghost.Type == TypeJarTrap {
			continue
		}

		ghostX := float64(ghost.VisualPos[0])
		ghostY := float64(ghost.VisualPos[1])

		// Jarak tegak lurus titik ghost ke garis tembakan
		distanceFromLine := math.Abs((punchY-playerY)*ghostX-(punchX-playerX)*ghostY+punchX*playerY-punchY*playerX) / lineLength
		// Proyeksi titik ke ruas garis tembak
		projection := ((ghostX-playerX)*(punchX-playerX) + (ghostY-playerY)*(punchY-playerY)) / math.Pow(lineLength, 2)

		if distanceFromLine <= 16 && projection >= 0 && projection <= 1 {
			xDist := ghostX - playerX
			yDist := ghostY - playerY

			newX := m.rnd.Float64() * math.Abs(xDist)
			newY := m.rnd.Float64() * math.Abs(yDist)

			if xDist > 0 {
				newX = -newX
			}
			if yDist > 0 {
				newY = -newY
			}

			ghost.LastPos[0] = float32(ghostX)
			ghost.LastPos[1] = float32(ghostY)

			ghost.NewPos[0] = float32(ghostX + newX)
			ghost.NewPos[1] = float32(ghostY + newY)

			ghost.Speed = 30
			ghost.Action = ActionMove
			ghost.State = StateFleeing

			dx := float64(ghost.NewPos[0] - ghost.LastPos[0])
			dy := float64(ghost.NewPos[1] - ghost.LastPos[1])
			ghost.Distance = math.Sqrt(dx*dx + dy*dy)
			ghost.Time = 0
			if ghost.Speed > 0 {
				ghost.MaxTime = ghost.Distance / float64(ghost.Speed)
			} else {
				ghost.MaxTime = 1.0
			}

			affected = append(affected, ghost)
		}
	}

	return affected, len(affected) > 0
}

// ─────────────────────────────────────────────
// AI Roaming, Slime & Jar Capture Loop (Thread.txt)
// ─────────────────────────────────────────────

func (m *GhostManager) UpdateTick(deltaTime float64, cb Callbacks) {
	m.mu.Lock()
	defer m.mu.Unlock()

	slimedQuotes := []string{"`2AIYEE! A ghost!``", "`2I've been slimed!``"}

	for worldName, worldGhosts := range m.ghosts {
		players := cb.GetWorldPlayers(worldName)

		for _, ghost := range worldGhosts {
			if ghost.State == StateDespawned {
				continue
			}

			// ── 1. Ghost Type 1 (Normal) & Type 13 (Shadow) ──
			if ghost.Type == TypeGhostNormal || ghost.Type == TypeGhostShadow {
				if ghost.Distance > 0 {
					ghost.Time += deltaTime

					vectorX := float64(ghost.NewPos[0]-ghost.LastPos[0]) / ghost.Distance
					vectorY := float64(ghost.NewPos[1]-ghost.LastPos[1]) / ghost.Distance

					displacementX := float64(ghost.Speed) * ghost.Time * vectorX
					displacementY := float64(ghost.Speed) * ghost.Time * vectorY

					currentX := float32(float64(ghost.LastPos[0]) + displacementX)
					currentY := float32(float64(ghost.LastPos[1]) + displacementY)

					ghost.VisualPos[0] = currentX
					ghost.VisualPos[1] = currentY

					// Cek jarak ke player untuk efek slime (<= 32 pixel)
					for _, pl := range players {
						distToPl := math.Sqrt(math.Pow(float64(ghost.VisualPos[0]-pl.X), 2) +
							math.Pow(float64(ghost.VisualPos[1]-pl.Y), 2))

						if distToPl <= 32 {
							if !pl.HasMod {
								if cb.SendParticle != nil {
									cb.SendParticle(worldName, ghost.VisualPos[0], ghost.VisualPos[1], 195)
								}
								if cb.SendTalkBubble != nil {
									quote := slimedQuotes[m.rnd.Intn(len(slimedQuotes))]
									cb.SendTalkBubble(worldName, pl.NetID, quote)
								}
								if cb.ApplyPlayerMod != nil {
									cb.ApplyPlayerMod(worldName, pl.NetID, 114)
								}
							}
						}
					}

					// Cek apakah ada Jar Trap aktif (Type 2, time >= 2s)
					for _, jar := range worldGhosts {
						if jar.Type == TypeJarTrap && jar.Time >= 2.0 && jar.State != StateDespawned {
							xDist := float64(ghost.VisualPos[0] - jar.LastPos[0])
							yDist := float64(ghost.VisualPos[1] - jar.LastPos[1])

							// Jar menyedot jika ghost 0 sampai 3 block di atas jar (-96 <= yDist < 0) dan xDist <= 32
							if (yDist >= -96 && yDist < 0) && (xDist <= 32 && xDist > -32) {
								ghost.LastPos[0] = ghost.VisualPos[0]
								ghost.LastPos[1] = ghost.VisualPos[1]

								ghost.NewPos[0] = jar.LastPos[0]
								ghost.NewPos[1] = jar.LastPos[1]

								ghost.Speed = 100 // Efek tersedot cepat
								ghost.Action = ActionMove
								ghost.IsCaught = 1
								ghost.State = uint32(jar.ID) // Simpan ID jar yang menyedot

								dx := float64(ghost.NewPos[0] - ghost.LastPos[0])
								dy := float64(ghost.NewPos[1] - ghost.LastPos[1])
								ghost.Distance = math.Sqrt(dx*dx + dy*dy)
								ghost.Time = 0
								if ghost.Speed > 0 {
									ghost.MaxTime = ghost.Distance / float64(ghost.Speed)
								} else {
									ghost.MaxTime = 0.5
								}

								if cb.BroadcastToWorld != nil {
									cb.BroadcastToWorld(worldName, BuildGhostUpdatePacket(ghost))
								}
								break
							}
						}
					}
				}

				// Saat ghost mencapai titik tujuan
				if ghost.Time >= ghost.MaxTime {
					if ghost.State == StateFleeing {
						ghost.State = StateNormal
					}

					// Jika ghost berhasil disedot masuk ke jar
					if ghost.IsCaught == 1 {
						for _, jar := range worldGhosts {
							if jar.State != StateDespawned && jar.ID == uint8(ghost.State) {
								ghost.LastPos[0] = ghost.NewPos[0]
								ghost.LastPos[1] = ghost.NewPos[1]

								ghost.Action = ActionCaught
								ghost.Type = jar.ID
								ghost.IsCaught = 0
								ghost.Speed = 0

								targetNetID := jar.OwnerNetID
								rewardGrowID := ""
								for _, pl := range players {
									if pl.NetID == targetNetID {
										rewardGrowID = pl.GrowID
										break
									}
								}

								if rewardGrowID != "" {
									if cb.SendTalkBubble != nil {
										cb.SendTalkBubble(worldName, targetNetID, "`3I caught a ghost!``")
									}
									if cb.SendConsoleMessage != nil {
										cb.SendConsoleMessage(worldName, targetNetID, "`3I caught a ghost!``")
									}
									if cb.GivePlayerItem != nil {
										cb.GivePlayerItem(rewardGrowID, ItemGhostInJar, 1)
									}
									if cb.SendItemCaughtAnim != nil {
										cb.SendItemCaughtAnim(worldName, jar.LastPos[0], jar.LastPos[1], int32(targetNetID), ItemGhostInJar)
									}
								}

								jar.State = StateDespawned
							}
						}

						if cb.BroadcastToWorld != nil {
							cb.BroadcastToWorld(worldName, BuildGhostUpdatePacket(ghost))
						}
						ghost.State = StateDespawned
						continue
					}

					// Roaming acak baru
					newX := float32(m.rnd.Intn(320) - 160) // -160 sampai +160 pixel
					newY := float32(m.rnd.Intn(320) - 160)

					ghost.LastPos[0] = ghost.NewPos[0]
					ghost.LastPos[1] = ghost.NewPos[1]

					ghost.NewPos[0] = ghost.LastPos[0] + newX
					ghost.NewPos[1] = ghost.LastPos[1] + newY

					ghost.Speed = float32(33 + m.rnd.Intn(12)) // 33 sampai 44
					ghost.Action = ActionMove
					ghost.State = StateNormal

					dx := float64(ghost.NewPos[0] - ghost.LastPos[0])
					dy := float64(ghost.NewPos[1] - ghost.LastPos[1])
					ghost.Distance = math.Sqrt(dx*dx + dy*dy)
					ghost.Time = 0
					if ghost.Speed > 0 {
						ghost.MaxTime = ghost.Distance / float64(ghost.Speed)
					} else {
						ghost.MaxTime = 2.0
					}

					if cb.BroadcastToWorld != nil {
						cb.BroadcastToWorld(worldName, BuildGhostUpdatePacket(ghost))
					}
				}

			} else if ghost.Type == TypeJarTrap {
				// ── 2. Ghost Type 2 (Jar Trap) ──
				ghost.Time += deltaTime
				if ghost.Time >= ghost.MaxTime {
					// Waktu toples habis di tanah, despawn
					ghost.State = StateDespawned
				}
			}
		}
	}
}

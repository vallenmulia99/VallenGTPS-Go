package server

/*
#cgo CFLAGS: -I${SRCDIR}/../..
#cgo windows LDFLAGS: -L${SRCDIR}/../../enet/lib -lenet -lws2_32 -lwinmm
#cgo linux LDFLAGS: -L${SRCDIR}/../../enet/lib -l:libenet_linux.a
#include "../../enet/enet.h"
#include <stdlib.h>

static inline void init_host_settings(ENetHost* host) {
	host->usingNewPacketForServer = 1;
	host->checksum = enet_crc32;
	enet_host_compress_with_range_coder(host);
}
*/
import "C"
import (
	"fmt"
	"runtime"
	"sync"
	"time"
	"unsafe"

	config "gtps/VallenSource/VallenConfig"
	database "gtps/VallenSource/VallenDatabase"
	dungeon "gtps/VallenSource/VallenDungeon"
	ghostjar "gtps/VallenSource/VallenGhostJar"
	player "gtps/VallenSource/VallenPlayer"
	variant "gtps/VallenSource/VallenVariant"
	world "gtps/VallenSource/VallenWorld"

)

// ─────────────────────────────────────────────
// Peer State
// ─────────────────────────────────────────────

type peerState struct {
	mu           sync.Mutex
	p            *player.Player
	growID       string
	password     string
	netID        int
	currentWorld string
	posX         float32
	posY         float32
	facingLeft   bool
}

// ─────────────────────────────────────────────
// Server Core Structure
// ─────────────────────────────────────────────

type Server struct {
	config  *config.Config
	db      *database.JSONDatabase
	host    *C.ENetHost
	running bool

	worldsMu sync.RWMutex
	worlds   map[string]*world.World

	peersMu sync.RWMutex
	peers   map[*C.ENetPeer]*peerState

	netIDCounter int
	tileHitsMu   sync.Mutex
	tileHits     map[string]int // "WORLDNAME:x:y" -> current hits
	tradesMu     sync.Mutex
	trades       map[int]*TradeSession // netID -> session
	dungeons     *dungeon.Manager
	ghosts       *ghostjar.GhostManager
	stopDone     chan struct{}
}

// New menginisialisasi instance Server baru
func New(cfg *config.Config, db *database.JSONDatabase) *Server {
	return &Server{
		config:   cfg,
		db:       db,
		worlds:   make(map[string]*world.World),
		peers:    make(map[*C.ENetPeer]*peerState),
		tileHits: make(map[string]int),
		trades:   make(map[int]*TradeSession),
		dungeons: dungeon.NewManager(),
		ghosts:   ghostjar.NewManager(),
		stopDone: make(chan struct{}),
	}
}

// Start menjalankan ENet host dan background event loop
func (s *Server) Start() error {
	var address C.ENetAddress
	address._type = C.ENET_ADDRESS_TYPE_IPV4
	address.port = C.enet_uint16(s.config.Port)
	for i := 0; i < 4; i++ {
		address.host[i] = 0
	}

	s.host = C.enet_host_create(
		C.ENET_ADDRESS_TYPE_IPV4,
		&address,
		1024, 2, 0, 0,
	)
	if s.host == nil {
		return fmt.Errorf("failed to create ENet host")
	}

	C.init_host_settings(s.host)

	s.running = true
	go s.eventLoop()
	s.startGhostLoop()
	return nil
}

// eventLoop memproses event ENet (Connect, Receive, Disconnect)
func (s *Server) eventLoop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(s.stopDone)

	var event C.ENetEvent
	for s.running {
		for C.enet_host_service(s.host, &event, 1) > 0 {
			switch event._type {
			case C.ENET_EVENT_TYPE_CONNECT:
				s.handleConnect(&event)
			case C.ENET_EVENT_TYPE_RECEIVE:
				s.handleReceive(&event)
			case C.ENET_EVENT_TYPE_DISCONNECT:
				s.handleDisconnect(&event)
			}
		}
	}
}

// handleReceive memproses raw ENet packet
func (s *Server) handleReceive(event *C.ENetEvent) {
	packet := event.packet
	defer C.enet_packet_destroy(packet)

	data := C.GoBytes(unsafe.Pointer(packet.data), C.int(packet.dataLength))
	if len(data) == 0 {
		return
	}

	packetType := data[0]

	switch packetType {
	case 2, 3: // Text / Action packet
		s.handleTextPacket(event.peer, data)
	case 4: // Game packet
		s.handleGamePacket(event.peer, data)
	}
}

// Stop menghentikan server dan merilis resource ENet
func (s *Server) Stop() {
	if !s.running {
		return
	}
	s.running = false
	if s.stopDone != nil {
		<-s.stopDone
	}
	if s.host != nil {
		C.enet_host_destroy(s.host)
		s.host = nil
	}
}
// ─────────────────────────────────────────────
// Background Ghost AI Loop
// ─────────────────────────────────────────────

func (s *Server) startGhostLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		defer ticker.Stop()
		for s.running {
			<-ticker.C
			s.ghosts.UpdateTick(0.1, ghostjar.Callbacks{
				BroadcastToWorld: func(worldName string, packet []byte) {
					s.broadcastToWorld(worldName, nil, packet, true)
				},
				GetWorldPlayers: func(worldName string) []ghostjar.PlayerInfo {
					s.peersMu.RLock()
					defer s.peersMu.RUnlock()

					var list []ghostjar.PlayerInfo
					for _, st := range s.peers {
						if st != nil && st.currentWorld == worldName && st.p != nil {
							st.mu.Lock()
							x, y := st.posX, st.posY
							st.mu.Unlock()
							list = append(list, ghostjar.PlayerInfo{
								NetID:  st.netID,
								GrowID: st.growID,
								UserID: st.p.UserID,
								X:      x,
								Y:      y,
							})
						}
					}
					return list
				},
				SendTalkBubble: func(worldName string, netID int, message string) {
					pkt := variant.New("OnTalkBubble", int32(netID), message).Pack()
					s.broadcastToWorld(worldName, nil, pkt, true)
				},
				SendConsoleMessage: func(worldName string, netID int, message string) {
					s.peersMu.RLock()
					defer s.peersMu.RUnlock()
					for peer, st := range s.peers {
						if st != nil && st.netID == netID {
							s.sendPacket(peer, variant.New("OnConsoleMessage", message).Pack(), true)
							break
						}
					}
				},
				SendParticle: func(worldName string, x, y float32, particleID int32) {
					pkt := ghostjar.BuildParticlePacket(x, y, particleID)
					s.broadcastToWorld(worldName, nil, pkt, true)
				},
				SendItemCaughtAnim: func(worldName string, x, y float32, toNetID int32, itemID int32) {
					pkt := ghostjar.BuildGhostCaughtAnimationPacket(x, y, toNetID, itemID)
					s.broadcastToWorld(worldName, nil, pkt, true)
				},
				GivePlayerItem: func(growID string, itemID int, count int) bool {
					s.peersMu.RLock()
					defer s.peersMu.RUnlock()
					for peer, st := range s.peers {
						if st != nil && st.growID == growID && st.p != nil {
							st.p.AddItem(itemID, count)
							s.sendPacket(peer, buildInventoryPacket(st.p, st.netID), true)
							s.db.SavePlayer(st.p)
							return true
						}
					}
					return false
				},
				ApplyPlayerMod: func(worldName string, netID int, modID int) {
					// Playmod placeholder
				},
			})
		}
	}()
}

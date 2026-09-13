package server

/*
#include "../../enet/enet.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"unsafe"

	player "gtps/VallenSource/VallenPlayer"
	variant "gtps/VallenSource/VallenVariant"
)

const maxTileActionDistance = 3 // tiles; normal punch/place range is two tiles.

func (s *Server) canReachTile(state *peerState, tileX, tileY int) bool {
	if state == nil {
		return false
	}
	state.mu.Lock()
	px, py := state.posX, state.posY
	state.mu.Unlock()
	return math.Abs(float64(px/32-float32(tileX))) <= maxTileActionDistance &&
		math.Abs(float64(py/32-float32(tileY))) <= maxTileActionDistance
}

func canReachPosition(state *peerState, x, y float32, maxTiles float32) bool {
	if state == nil {
		return false
	}
	state.mu.Lock()
	px, py := state.posX, state.posY
	state.mu.Unlock()
	return math.Abs(float64(px-x)) <= float64(maxTiles*32) &&
		math.Abs(float64(py-y)) <= float64(maxTiles*32)
}

// ─────────────────────────────────────────────
// Network Helpers
// ─────────────────────────────────────────────

// getPeerState mengambil state peer dari map (thread-safe read).
func (s *Server) getPeerState(peer *C.ENetPeer) *peerState {
	s.peersMu.RLock()
	defer s.peersMu.RUnlock()
	return s.peers[peer]
}

// sendPacket mengirim raw bytes ke peer via ENet.
func (s *Server) sendPacket(peer *C.ENetPeer, data []byte, reliable bool) {
	s.sendPacketOnChannel(peer, data, reliable, 0)
}

// sendPacketOnChannel sends a packet on the protocol channel selected by the
// packet family. Variants/text/map use channel 0; live game-state packets such
// as dropped-object changes use channel 1, matching the reference server.
func (s *Server) sendPacketOnChannel(peer *C.ENetPeer, data []byte, reliable bool, channel C.enet_uint8) {
	if len(data) == 0 {
		return
	}
	var flags C.enet_uint32
	if reliable {
		flags = C.ENET_PACKET_FLAG_RELIABLE
	}
	packet := C.enet_packet_create(
		unsafe.Pointer(&data[0]),
		C.size_t(len(data)),
		flags,
	)
	C.enet_peer_send(peer, channel, packet)
}

// broadcastToWorld mengirim paket ke semua peer yang ada di world tertentu.
// Jika excludePeer != nil, peer tersebut dilewati (tidak dikirim).
func (s *Server) broadcastToWorld(worldName string, excludePeer *C.ENetPeer, data []byte, reliable bool) {
	s.broadcastToWorldOnChannel(worldName, excludePeer, data, reliable, 0)
}

func (s *Server) broadcastToWorldOnChannel(worldName string, excludePeer *C.ENetPeer, data []byte, reliable bool, channel C.enet_uint8) {
	s.peersMu.RLock()
	defer s.peersMu.RUnlock()

	for p, st := range s.peers {
		if excludePeer != nil && p == excludePeer {
			continue
		}
		if st != nil && st.currentWorld == worldName {
			s.sendPacketOnChannel(p, data, reliable, channel)
		}
	}
}

// broadcastGamePacketToWorld is used for authoritative live game updates.
// The Growtopia client expects these, including object spawn/remove/update, on
// channel 1. Sending them on channel 0 can render an object but leave its
// client-side pickup state stale until the next map load.
func (s *Server) broadcastGamePacketToWorld(worldName string, excludePeer *C.ENetPeer, data []byte, reliable bool) {
	s.broadcastToWorldOnChannel(worldName, excludePeer, data, reliable, 1)
}

// getPeerAddress mengembalikan string alamat IP:port dari peer.
func (s *Server) getPeerAddress(peer *C.ENetPeer) string {
	if peer == nil {
		return "unknown"
	}
	addr := peer.address
	if addr._type == C.ENET_ADDRESS_TYPE_IPV4 {
		return fmt.Sprintf("%d.%d.%d.%d:%d",
			addr.host[0], addr.host[1], addr.host[2], addr.host[3], addr.port)
	}
	return fmt.Sprintf("port:%d", addr.port)
}

// ─────────────────────────────────────────────
// Movement Handler
// ─────────────────────────────────────────────

// handleMovement memproses paket gerak player dan menyebarkannya ke world.
func (s *Server) handleMovement(peer *C.ENetPeer, gp GamePacket) {
	state := s.getPeerState(peer)
	if state == nil || state.currentWorld == "" {
		return
	}

	state.mu.Lock()
	// Never accept a single movement packet that jumps more than four tiles.
	// Tile actions use this authoritative position, so this closes remote
	// break/place/pickup attempts based on forged movement packets.
	if math.Abs(float64(gp.PosX-state.posX)) > 128 || math.Abs(float64(gp.PosY-state.posY)) > 128 {
		state.mu.Unlock()
		return
	}
	state.posX = gp.PosX
	state.posY = gp.PosY
	// Reference protocol flags: 0x10 = moving/facing left, 0x20 = right.
	// This was reversed, which made manual inventory drops appear behind the
	// avatar even though the drop-position calculation itself was correct.
	state.facingLeft = (gp.State & 0x10) != 0
	if state.p != nil {
		state.p.PosX = gp.PosX
		state.p.PosY = gp.PosY
		state.p.FacingLeft = state.facingLeft
	}
	netID := state.netID
	currentWorld := state.currentWorld
	state.mu.Unlock()

	// Broadcast movement ke semua player di world yang sama
	gp.NetID = int32(netID)
	pkt := buildGamePacket(gp, nil)
	s.broadcastToWorld(currentWorld, peer, pkt, false)

	// Geiger radiation signal feedback
	if state.p != nil && state.p.GetClothing(5) == 2286 {
		w := s.getOrCreateWorld(currentWorld)
		if w != nil {
			pTileX := int(state.posX / 32)
			pTileY := int(state.posY / 32)
			dist := math.Hypot(float64(pTileX-w.GeigerX), float64(pTileY-w.GeigerY))
			if dist <= 1.5 {
				s.sendPacket(peer, variant.New("OnTalkBubble", int32(netID), "`w[GEIGER: OVERLOAD! PUNCH HERE!]``").Pack(), true)
				s.sendPacket(peer, variant.New("OnPlayPositioned", "audio/beep.wav").Pack(), true)
			} else if dist <= 6.0 && rand.Intn(4) == 0 {
				s.sendPacket(peer, variant.New("OnTalkBubble", int32(netID), "`4[GEIGER: RED - VERY CLOSE!]``").Pack(), true)
				s.sendPacket(peer, variant.New("OnPlayPositioned", "audio/beep.wav").Pack(), true)
			} else if dist <= 14.0 && rand.Intn(6) == 0 {
				s.sendPacket(peer, variant.New("OnTalkBubble", int32(netID), "`e[GEIGER: YELLOW - GETTING WARMER]``").Pack(), true)
			}
		}
	}
}

// ─────────────────────────────────────────────
// String & Data Parsing Helpers
// ─────────────────────────────────────────────

// parseURLEncoded mem-parse string format "key=val&key2=val2" menjadi map.
func parseURLEncoded(s string) map[string]string {
	kv := make(map[string]string)
	pairs := strings.Split(s, "&")
	for _, pair := range pairs {
		kv2 := strings.SplitN(pair, "=", 2)
		if len(kv2) == 2 {
			kv[kv2[0]] = kv2[1]
		}
	}
	return kv
}

// parsePipeMap mem-parse slice pipes menjadi map key-value (pasangan berurutan).
func parsePipeMap(pipes []string) map[string]string {
	kv := make(map[string]string)
	for i := 0; i < len(pipes)-1; i += 2 {
		key := pipes[i]
		val := pipes[i+1]
		kv[key] = val
	}
	return kv
}

// truncate memotong string s jika panjangnya melebihi n karakter.
func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

// ─────────────────────────────────────────────
// World & Player Helpers
// ─────────────────────────────────────────────

// addRecentWorld menambahkan world ke daftar RecentWorlds player (FIFO, max 6).
// Bug fix: loop benar menggunakan seluruh array.
func addRecentWorld(p *player.Player, name string) {
	for i := 0; i < len(p.RecentWorlds); i++ {
		if p.RecentWorlds[i] == name {
			// Geser nama yang ditemukan ke posisi terakhir
			copy(p.RecentWorlds[i:], p.RecentWorlds[i+1:])
			p.RecentWorlds[len(p.RecentWorlds)-1] = name
			return
		}
	}
	// Belum ada — geser semua ke kiri, tambahkan di akhir
	copy(p.RecentWorlds[:], p.RecentWorlds[1:])
	p.RecentWorlds[len(p.RecentWorlds)-1] = name
}

// getPublicHost membaca host publik dari server_data.php.
// Jika tidak ditemukan, fallback ke "127.0.0.1".
func (s *Server) getPublicHost() string {
	data, err := os.ReadFile("VallenSetting/server_data.php")
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "server|") {
				parts := strings.SplitN(line, "|", 2)
				if len(parts) == 2 && parts[1] != "" && parts[1] != "0.0.0.0" {
					return parts[1]
				}
			}
		}
	}
	return "127.0.0.1"
}

// ─────────────────────────────────────────────
// findPlayerOnline — helper lookup player online
// ─────────────────────────────────────────────

// findPeerByGrowID mencari peer dan state berdasarkan growID (case-insensitive).
// Mengembalikan (peer, state) atau (nil, nil) jika tidak ditemukan.
// CATATAN: Caller harus memegang peersMu.RLock() sebelum memanggil fungsi ini.
func (s *Server) findPeerByGrowIDLocked(growID string) (*C.ENetPeer, *peerState) {
	for p, st := range s.peers {
		if st != nil && strings.EqualFold(st.growID, growID) {
			return p, st
		}
	}
	return nil, nil
}

// sendToGrowID mengirim paket ke player berdasarkan GrowID jika sedang online.
func (s *Server) sendToGrowID(growID string, data []byte, reliable bool) bool {
	s.peersMu.RLock()
	defer s.peersMu.RUnlock()
	p, _ := s.findPeerByGrowIDLocked(growID)
	if p == nil {
		return false
	}
	s.sendPacket(p, data, reliable)
	return true
}

// sendConsoleToGrowID mengirim pesan OnConsoleMessage ke player online tertentu.
func (s *Server) sendConsoleToGrowID(growID string, msg string) bool {
	return s.sendToGrowID(growID, variant.New("OnConsoleMessage", msg).Pack(), true)
}
// ─────────────────────────────────────────────
// Sender Shortcuts (Clean & Lazy Style)
// ─────────────────────────────────────────────

// sendConsole mengirim pesan OnConsoleMessage ke peer dengan format variadic string.
func (s *Server) sendConsole(peer *C.ENetPeer, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	s.sendPacket(peer, variant.New("OnConsoleMessage", msg).Pack(), true)
}

// sendOverlay mengirim teks banner tengah layar (OnTextOverlay) dengan format variadic string.
func (s *Server) sendOverlay(peer *C.ENetPeer, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	s.sendPacket(peer, variant.New("OnTextOverlay", msg).Pack(), true)
}

// sendDialog mengirim dialog klasik GT (OnDialogRequest) dan membunyikan klik.
func (s *Server) sendDialog(peer *C.ENetPeer, dialogStr string) {
	s.sendPacket(peer, variant.New("OnDialogRequest", dialogStr).Pack(), true)
	s.playSound(peer, "audio/dialog_confirm.wav")
}

// sendRML membuka window native RML Growtopia (OnDialogRequestRML) dan membunyikan klik.
func (s *Server) sendRML(peer *C.ENetPeer, triggerName string) {
	s.sendPacket(peer, variant.New("OnDialogRequestRML", triggerName).Pack(), true)
	s.playSound(peer, "audio/dialog_confirm.wav")
}

// sendBubble menampilkan chat bubble di atas kepala netID tertentu.
func (s *Server) sendBubble(peer *C.ENetPeer, netID int32, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	s.sendPacket(peer, variant.New("OnTalkBubble", netID, msg, int32(0), int32(0)).Pack(), true)
}

// playSound memutar file efek suara audio di client.
func (s *Server) playSound(peer *C.ENetPeer, soundFile string) {
	pkt := variant.SendAction("play_sfx", fmt.Sprintf("file|%s\ndelayMS|0", soundFile))
	s.sendPacket(peer, pkt, true)
}

// broadcastConsole menyiarkan pesan console ke seluruh pemain di world tertentu.
func (s *Server) broadcastConsole(worldName string, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	s.broadcastToWorld(worldName, nil, variant.New("OnConsoleMessage", msg).Pack(), true)
}
// playWrenchSound memutar suara klik wrench/dialog konfirmasi standar.
func (s *Server) playWrenchSound(peer *C.ENetPeer) {
	s.playSound(peer, "audio/dialog_confirm.wav")
}
// isPlayerOnline mengecek apakah GrowID tertentu sedang online di server.
func (s *Server) isPlayerOnline(growID string) bool {
	s.peersMu.RLock()
	defer s.peersMu.RUnlock()
	for _, st := range s.peers {
		if st != nil && strings.EqualFold(st.growID, growID) {
			return true
		}
	}
	return false
}

// getPlayerWorld mengembalikan nama world tempat player tertentu berada (kosong jika offline).
func (s *Server) getPlayerWorld(growID string) string {
	s.peersMu.RLock()
	defer s.peersMu.RUnlock()
	for _, st := range s.peers {
		if st != nil && strings.EqualFold(st.growID, growID) {
			return st.currentWorld
		}
	}
	return ""
}

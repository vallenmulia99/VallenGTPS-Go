package server

/*
#include "../../enet/enet.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"log"
	"unsafe"
)

// ─────────────────────────────────────────────
// Connect & Disconnect Handlers
// ─────────────────────────────────────────────

// handleConnect dijalankan saat ENet menerima koneksi baru.
func (s *Server) handleConnect(event *C.ENetEvent) {
	peer := event.peer
	addr := s.getPeerAddress(peer)
	log.Printf("[ENet] New connection from %s", addr)

	state := &peerState{}
	s.peersMu.Lock()
	s.peers[peer] = state
	s.peersMu.Unlock()

	peer.data = unsafe.Pointer(state)

	// Kirim NET_MESSAGE_SERVER_HELLO (0x01, 0x00, 0x00, 0x00)
	hello := []byte{0x01, 0x00, 0x00, 0x00}
	s.sendPacket(peer, hello, true)
	log.Printf("[ENet] Sent NET_MESSAGE_SERVER_HELLO to %s", addr)
}

// handleDisconnect dijalankan saat koneksi ENet terputus.
func (s *Server) handleDisconnect(event *C.ENetEvent) {
	peer := event.peer
	addr := s.getPeerAddress(peer)
	log.Printf("[ENet] Disconnected: %s", addr)

	s.peersMu.Lock()
	state, ok := s.peers[peer]
	if ok {
		delete(s.peers, peer)
	}
	s.peersMu.Unlock()

	if ok && state != nil {
		s.cancelTrade(state.netID, fmt.Sprintf("`4Trade canceled: %s disconnected.``", state.growID))
		state.mu.Lock()
		currentWorld, netID := state.currentWorld, state.netID
		state.mu.Unlock()
		s.leaveWorld(peer, state, currentWorld, netID)
	}

	peer.data = nil
}

package server

/*
#include "../../enet/enet.h"
#include <stdlib.h>
*/
import "C"
import "log"

// ─────────────────────────────────────────────
// Game Packet Handler & Router
// ─────────────────────────────────────────────

// handleGamePacket memproses paket game (type 4) dan merutekan ke handler yang tepat.
func (s *Server) handleGamePacket(peer *C.ENetPeer, data []byte) {
	if len(data) < 60 {
		return
	}

	gp := parseGamePacket(data)

	switch gp.Type {
	case 0: // Movement / State update
		s.handleMovement(peer, gp)
	case 3: // Tile Change Request (Punch / Place)
		s.handleTileChange(peer, gp)
	case 7: // Tile Activate Request (Doors, Checkpoints)
		s.handleTileActivate(peer, gp)
	case 10: // Item Activate Request (Clothing equip/unequip, WL/DL/BGL compress/shatter)
		s.handleItemActivate(peer, gp)
	case 11: // Item Activate Object Request (Collect Gem / Item Drop)
		s.handleItemActivateObject(peer, gp)
	case 18: // Ping Reply
		// ping handled silently
	default:
		log.Printf("[Game] Unhandled game packet type: %d", gp.Type)
	}
}

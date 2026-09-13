package server

/*
#include "../../enet/enet.h"
#include <stdlib.h>
*/
import "C"

import (
	"encoding/binary"

	player "gtps/VallenSource/VallenPlayer"
)

// ─────────────────────────────────────────────
// Inventory Packet Builder
// ─────────────────────────────────────────────

// buildInventoryPacket menyusun paket PACKET_SEND_INVENTORY_STATE (type 9)
// berisi daftar item yang dimiliki player.
func buildInventoryPacket(p *player.Player, netID int) []byte {
	inv := []byte{}
	inv = append(inv, 0x01) // big backpack flag

	b4 := make([]byte, 4)
	binary.LittleEndian.PutUint32(b4, uint32(p.SlotSize))
	inv = append(inv, b4...)

	b2 := make([]byte, 2)
	binary.LittleEndian.PutUint16(b2, uint16(len(p.Inventory)))
	inv = append(inv, b2...)

	for _, item := range p.Inventory {
		ib := make([]byte, 4)
		binary.LittleEndian.PutUint16(ib[0:], uint16(item.ID))
		count := item.Count
		if count > 200 {
			count = 200
		}
		ib[2] = uint8(count)
		if p.IsEquipped(item.ID) {
			ib[3] = 1 // 0x01 = EQUIPPED indicator in inventory UI
		}
		inv = append(inv, ib...)
	}

	pkt := make([]byte, 60)
	binary.LittleEndian.PutUint32(pkt[0:], 4)           // NET_MESSAGE_GAME_PACKET
	binary.LittleEndian.PutUint32(pkt[4:], 9)           // PACKET_SEND_INVENTORY_STATE = 9
	binary.LittleEndian.PutUint32(pkt[8:], uint32(netID))
	binary.LittleEndian.PutUint32(pkt[16:], 8)          // S_EXTENDED
	binary.LittleEndian.PutUint32(pkt[56:], uint32(len(inv)))

	return append(pkt, inv...)
}

func (s *Server) sendInventory(peer *C.ENetPeer, state *peerState) {
	if state != nil && state.p != nil {
		s.sendPacket(peer, buildInventoryPacket(state.p, state.netID), true)
	}
}

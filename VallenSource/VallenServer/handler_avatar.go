package server

/*
#include "../../enet/enet.h"
*/
import "C"

import (
	"strconv"

	items "gtps/VallenSource/VallenItems"
	variant "gtps/VallenSource/VallenVariant"
)

// handleSetSkin persists the color selected by the client and refreshes the
// avatar for every player in the current world.
func (s *Server) handleSetSkin(peer *C.ENetPeer, pipes []string) {
	state := s.getPeerState(peer)
	if state == nil || state.p == nil {
		return
	}

	values := parsePipeMap(pipes)
	color, err := strconv.ParseUint(values["color"], 10, 32)
	if err != nil {
		return
	}

	state.p.SkinColor = uint32(color)
	s.db.SavePlayer(state.p)
	if state.currentWorld != "" {
		s.broadcastPlayerAppearance(state)
		return
	}
	// The client resets its preview avatar while at the world-select screen.
	// Refresh the local appearance there too, rather than waiting for a world.
	s.sendPlayerAppearance(peer, state)
}

// playerAppearancePackets mirrors the reference OnSetClothing plus character
// state pair. OnSpawn does not carry persistent skin color, and several client
// builds reset their local avatar to white until this pair arrives.
func playerAppearancePackets(state *peerState, clothingDelay int32) ([]byte, []byte) {
	if state == nil || state.p == nil {
		return nil, nil
	}

	p := state.p
	clothing := variant.NewWithNetID(
		"OnSetClothing",
		int32(state.netID),
		variant.Vec3f{X: float32(p.Clothing[0]), Y: float32(p.Clothing[1]), Z: float32(p.Clothing[2])},
		variant.Vec3f{X: float32(p.Clothing[3]), Y: float32(p.Clothing[4]), Z: float32(p.Clothing[5])},
		variant.Vec3f{X: float32(p.Clothing[6]), Y: float32(p.Clothing[7]), Z: float32(p.Clothing[8])},
		p.SkinColor,
		variant.Vec3f{X: float32(p.Clothing[9])},
	)
	clothing.Delay = clothingDelay
	clothingPkt := clothing.Pack()

	// The character-state packet carries hair color and makes the client apply
	// persisted appearance after a login-time or menu-time default reset.
	// Only send non-zero if the player actually has dyed hair (avoid green hair bug).
	characterState := uint32(0x14 | ((0x808000 + p.PunchEffect) << 8))
	hairColor := int32(0)
	if p.HairColor != 0 && p.HairColor != 0xFFFFFFFF && p.HairColor != 0xFF00FF00 {
		hairColor = int32(p.HairColor)
	}
	statePkt := buildGamePacket(GamePacket{
		Type:   int32(characterState),
		NetID:  int32(state.netID),
		Count:  125,
		ID:     int32(p.State),
		PosX:   1200,
		PosY:   200,
		SpeedX: 250,
		SpeedY: 1000,
		PunchX: hairColor,
	}, nil)
	return clothingPkt, statePkt
}

// sendPlayerAppearance applies the appearance directly to one client. It is
// intentionally usable outside a world (login and world-select preview).
func (s *Server) sendPlayerAppearance(peer *C.ENetPeer, state *peerState) {
	clothingPkt, statePkt := playerAppearancePackets(state, 0)
	if len(clothingPkt) > 0 {
		s.sendPacket(peer, clothingPkt, true)
	}
	if len(statePkt) > 0 {
		s.sendPacket(peer, statePkt, true)
	}
}

// broadcastPlayerAppearance refreshes every avatar in the active world,
// including the local client that selected the skin.
func (s *Server) broadcastPlayerAppearance(state *peerState) {
	if state == nil || state.currentWorld == "" {
		return
	}
	clothingPkt, statePkt := playerAppearancePackets(state, 0)
	if len(clothingPkt) > 0 {
		s.broadcastToWorld(state.currentWorld, nil, clothingPkt, true)
	}
	if len(statePkt) > 0 {
		s.broadcastToWorld(state.currentWorld, nil, statePkt, true)
	}
}

// equipOrUnequipClothing toggles equipped clothing on the player.
// It handles:
// - Slot validation and equip/unequip toggle
// - Hand slot punch effect auto-update
// - Appearance broadcast to world
// - Inventory update packet with 0x01 equipped flags
// - Audio cue (audio/change_clothes.wav)
// - Database persistence
func (s *Server) equipOrUnequipClothing(peer *C.ENetPeer, state *peerState, itemID int) bool {
	if state == nil || state.p == nil || itemID <= 0 || !state.p.HasItem(itemID, 1) {
		return false
	}

	item := items.GetItem(uint16(itemID))
	if item == nil || !item.IsClothing() {
		return false
	}

	slot := int(item.ClothSlot)
	if slot < 0 || slot >= len(state.p.Clothing) {
		return false
	}

	if state.p.Clothing[slot] == itemID {
		state.p.Clothing[slot] = 0 // Unequip
	} else {
		state.p.Clothing[slot] = itemID // Equip
	}

	// Auto-update PunchEffect when hand slot changes (slot 5 = hand)
	handItemID := state.p.Clothing[items.ClothHand]
	state.p.PunchEffect = items.GetPunchEffect(handItemID)

	s.playWorldSound(state.currentWorld, "audio/change_clothes.wav", state.posX, state.posY)
	s.broadcastPlayerAppearance(state)
	s.sendPacket(peer, buildInventoryPacket(state.p, state.netID), true)
	s.db.SavePlayer(state.p)
	return true
}

// handleRespawn follows the freeze, killed, delayed teleport, unfreeze flow
// used by ANALISIS-GTPS-WORK.
func (s *Server) handleRespawn(peer *C.ENetPeer) {
	state := s.getPeerState(peer)
	if state == nil || state.p == nil || state.currentWorld == "" {
		return
	}
	w := s.getOrCreateWorld(state.currentWorld)
	if w == nil {
		return
	}

	x, y := state.p.RespawnX, state.p.RespawnY
	if x == 0 && y == 0 {
		x, y = w.SpawnPixelX(), w.SpawnPixelY()
	}
	state.posX, state.posY = x, y

	s.sendPacket(peer, variant.NewWithNetID("OnSetFreezeState", int32(state.netID), int32(2)).Pack(), true)
	s.broadcastToWorld(state.currentWorld, nil, variant.NewWithNetID("OnKilled", int32(state.netID)).Pack(), true)
	s.playWorldSound(state.currentWorld, "audio/death.wav", state.posX, state.posY)

	setPos := variant.NewWithNetID("OnSetPos", int32(state.netID), variant.Vec2f{X: x, Y: y})
	setPos.Delay = 1900
	s.sendPacket(peer, setPos.Pack(), true)

	unfreeze := variant.NewWithNetID("OnSetFreezeState", int32(state.netID))
	unfreeze.Delay = 1900
	s.sendPacket(peer, unfreeze.Pack(), true)
	s.broadcastPlayerAppearance(state)
}

package server

/*
#include "../../enet/enet.h"
#include <stdlib.h>
*/
import "C"

import (
	"encoding/binary"
	"fmt"

	items "gtps/VallenSource/VallenItems"
	variant "gtps/VallenSource/VallenVariant"
	world "gtps/VallenSource/VallenWorld"
)

// ─────────────────────────────────────────────────────────────────────────────
// Item Activation & Pickup (CGO GamePackets)
// ─────────────────────────────────────────────────────────────────────────────

func (s *Server) handleItemActivate(peer *C.ENetPeer, gp GamePacket) {
	state := s.getPeerState(peer)
	if state == nil || state.p == nil {
		return
	}

	itemID := int(gp.ID)
	if itemID <= 0 || !state.p.HasItem(itemID, 1) {
		return
	}

	item := items.GetItem(uint16(itemID))
	if item == nil {
		return
	}

	// 1. Lock Compression & Shatter
	if itemID == 242 { // World Lock -> Compress to Diamond Lock
		if state.p.GetItemCount(242) >= 100 {
			state.p.RemoveItem(242, 100)
			state.p.AddItem(1796, 1)
			s.playSound(peer, "audio/dialog_confirm.wav")
			msg := "You compressed 100 `2World Lock`` into a `2Diamond Lock``!"
			s.sendBubble(peer, int32(state.netID), "%s", msg)
			s.sendConsole(peer, "%s", msg)
			s.sendInventory(peer, state)
			s.db.SavePlayer(state.p)
			return
		}
	} else if itemID == 1796 { // Diamond Lock
		if state.p.GetItemCount(1796) >= 100 {
			// Compress to BGL
			state.p.RemoveItem(1796, 100)
			state.p.AddItem(7188, 1)
			s.playSound(peer, "audio/dialog_confirm.wav")
			msg := "You compressed 100 `2Diamond Lock`` into a `2Blue Gem Lock``!"
			s.sendBubble(peer, int32(state.netID), "%s", msg)
			s.sendConsole(peer, "%s", msg)
			s.sendInventory(peer, state)
			s.db.SavePlayer(state.p)
			return
		} else {
			// Shatter to 100 WL
			state.p.RemoveItem(1796, 1)
			state.p.AddItem(242, 100)
			s.playSound(peer, "audio/dialog_confirm.wav")
			msg := "You shattered a `2Diamond Lock`` into 100 `2World Lock``!"
			s.sendBubble(peer, int32(state.netID), "%s", msg)
			s.sendConsole(peer, "%s", msg)
			s.sendInventory(peer, state)
			s.db.SavePlayer(state.p)
			return
		}
	} else if itemID == 7188 { // Blue Gem Lock -> Shatter to 100 DL
		state.p.RemoveItem(7188, 1)
		state.p.AddItem(1796, 100)
		s.playSound(peer, "audio/dialog_confirm.wav")
		msg := "You shattered a `2Blue Gem Lock`` into 100 `2Diamond Lock``!"
		s.sendBubble(peer, int32(state.netID), "%s", msg)
		s.sendConsole(peer, "%s", msg)
		s.sendInventory(peer, state)
		s.db.SavePlayer(state.p)
		return
	}

	// 2. Clothing Equip / Unequip
	if item.IsClothing() {
		s.equipOrUnequipClothing(peer, state, itemID)
		return
	}

	// 3. Consumables / Potions
	if isConsumableItem(itemID) {
		w := s.getOrCreateWorld(state.currentWorld)
		if s.handleUseConsumable(peer, state, w, int(state.posX/32), int(state.posY/32), itemID) {
			return
		}
	}
}

func (s *Server) handleItemActivateObject(peer *C.ENetPeer, gp GamePacket) {
	state := s.getPeerState(peer)
	if state == nil || state.p == nil || state.currentWorld == "" {
		return
	}

	w := s.getOrCreateWorld(state.currentWorld)
	if w == nil {
		return
	}

	targetUID := int(gp.ID)
	obj := w.FindObjectByUID(targetUID)
	if obj == nil && int(gp.UID) != targetUID {
		targetUID = int(gp.UID)
		obj = w.FindObjectByUID(targetUID)
	}
	if obj == nil {
		return
	}

	result := w.PickupObject(targetUID, func(itemID, count int) int {
		return state.p.AddItem(itemID, count)
	})

	if !result.Success {
		return
	}

	if result.IsGem {
		state.p.Gems += result.CollectedQty
		s.sendPacket(peer, variant.SetBux(state.p.Gems, 0).Pack(), true)
	} else {
		itemName := fmt.Sprintf("Item #%d", result.ItemID)
		itemRarity := 0
		if itm := items.GetItem(uint16(result.ItemID)); itm != nil {
			if itm.Name != "" {
				itemName = itm.Name
			}
			itemRarity = int(itm.Rarity)
		}
		if itemRarity >= 999 {
			s.sendConsole(peer, "Collected `w%d %s``.", result.CollectedQty, itemName)
		} else {
			s.sendConsole(peer, "Collected `w%d %s``. Rarity: `w%d``", result.CollectedQty, itemName, itemRarity)
		}
		s.sendInventory(peer, state)
	}

	if result.Overflow > 0 {
		obj := w.FindObjectByUID(targetUID)
		if obj != nil {
			updatePkt := buildItemChangeObjectPacket(world.BuildUpdatePacketData(obj))
			s.broadcastGamePacketToWorld(state.currentWorld, nil, updatePkt, true)
		}
	} else {
		removePkt := buildItemChangeObjectPacket(world.BuildRemovePacketData(targetUID, state.netID))
		s.broadcastGamePacketToWorld(state.currentWorld, nil, removePkt, true)
	}

	s.db.SavePlayer(state.p)
	s.db.SaveWorld(w)
}

func (s *Server) handleRefreshItemData(peer *C.ENetPeer) {
	if items.DB == nil {
		return
	}
	s.sendConsole(peer, "One moment, updating item data...")
	itemData := items.DB.RawData
	pkt := make([]byte, 60)
	binary.LittleEndian.PutUint32(pkt[0:], 4)
	binary.LittleEndian.PutUint32(pkt[4:], 0x10)
	binary.LittleEndian.PutUint32(pkt[16:], 8)
	binary.LittleEndian.PutUint32(pkt[56:], uint32(len(itemData)))
	pkt = append(pkt, itemData...)
	s.sendPacket(peer, pkt, true)
}

func (s *Server) spawnWorldDrop(w *world.World, itemID, count int, x, y float32, dropperNetID int) {
	if w == nil || itemID <= 0 || count <= 0 {
		return
	}

	// Gem auto-merge system (GrowTavern / Growtopia logic)
	if itemID == world.GemItemID {
		removedUIDs, createdUIDs := w.AddGems(count, x, y)
		for _, uid := range removedUIDs {
			removePkt := buildItemChangeObjectPacket(world.BuildRemovePacketData(uid, -2))
			s.broadcastGamePacketToWorld(w.Name, nil, removePkt, true)
		}
		for _, uid := range createdUIDs {
			obj := w.FindObjectByUID(uid)
			if obj == nil {
				continue
			}
			data := world.BuildSpawnPacketData(obj, dropperNetID)
			s.broadcastGamePacketToWorld(w.Name, nil, buildItemChangeObjectPacket(data), true)
		}
		return
	}

	changes := w.AddObject(itemID, count, x, y)
	for _, change := range changes {
		obj := w.FindObjectByUID(change.UID)
		if obj == nil {
			continue
		}
		var data world.ObjectPacketData
		if change.Created {
			data = world.BuildSpawnPacketData(obj, dropperNetID)
		} else {
			data = world.BuildUpdatePacketData(obj)
		}
		s.broadcastGamePacketToWorld(w.Name, nil, buildItemChangeObjectPacket(data), true)
	}
}

func (s *Server) playWorldSound(worldName, sound string, x, y float32) {
	pkt := variant.SendAction("play_sfx", fmt.Sprintf("file|%s\ndelayMS|0", sound))
	s.broadcastToWorld(worldName, nil, pkt, true)
}

func (s *Server) playWorldPunchSound(worldName string, x, y float32) {
	pkt := variant.New("OnPlayPositioned", "audio/punch.wav").Pack()
	s.broadcastToWorld(worldName, nil, pkt, true)
}

func buildItemChangeObjectPacket(data world.ObjectPacketData) []byte {
	return buildGamePacket(GamePacket{
		Type:  0x0e,
		NetID: data.NetID,
		UID:   data.UID,
		State: data.State,
		Count: data.Count,
		ID:    data.ItemID,
		PosX:  data.X,
		PosY:  data.Y,
	}, nil)
}

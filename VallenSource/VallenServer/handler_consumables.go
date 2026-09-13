package server

/*
#include "../../enet/enet.h"
*/
import "C"

import (
	"fmt"
	"strings"
	"time"

	dialog "gtps/VallenSource/VallenDialog"
	"gtps/VallenSource/VallenRole"
	"gtps/VallenSource/VallenVariant"
	"gtps/VallenSource/VallenWorld"
)

func isConsumableItem(id int) bool {
	return id == 1488 || id == 228 || id == 1778 || id == 9412 || id == 1404 || id == 1662 || id == 2580
}

// handleUseConsumable menangani saat player menggunakan item potion / consumable
func (s *Server) handleUseConsumable(peer *C.ENetPeer, state *peerState, w *world.World, tileX, tileY, itemID int) bool {
	if state == nil || state.p == nil || w == nil {
		return false
	}

	if !state.p.HasItem(itemID, 1) {
		return false
	}

	switch itemID {
	// 1. Experience Potion (+10,000 XP)
	case 1488:
		state.p.RemoveItem(itemID, 1)
		xpGain := 10000
		state.p.XP += xpGain
		for state.p.XP >= state.p.Level*100 {
			state.p.Level++
		}
		s.db.SavePlayer(state.p)
		s.sendInventory(peer, state)
		s.playWorldSound(w.Name, "audio/piano_nice.wav", state.posX, state.posY)
		s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), fmt.Sprintf("`oDrank Experience Potion! `2+%d XP`` (Level %d)``", xpGain, state.p.Level)).Pack(), true)
		s.sendConsole(peer, "`2You consumed an Experience Potion and gained `w%d XP``! (Current Level: %d)", xpGain, state.p.Level)
		return true

	// 2. Grow Spray Fertilizer (Age tree by 1 hour)
	case 228:
		tree, found := w.FindTree(tileX, tileY)
		if !found {
			s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4Target block is not a growing tree!``").Pack(), true)
			return true
		}
		state.p.RemoveItem(itemID, 1)
		// Kurangi waktu tanam mundur 1 jam (3600 detik) agar lebih cepat matang
		tree.LastPick = tree.LastPick.Add(-1 * time.Hour)
		s.db.SavePlayer(state.p)
		s.db.SaveWorld(w)
		s.sendInventory(peer, state)
		s.playWorldSound(w.Name, "audio/spray.wav", float32(tileX*32), float32(tileY*32))
		s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`2Sprayed Grow Spray! Tree aged by 1 hour.``").Pack(), true)
		return true

	// 3. Deluxe Grow Spray (Instantly matures tree / 24 hours advance)
	case 1778:
		tree, found := w.FindTree(tileX, tileY)
		if !found {
			s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4Target block is not a growing tree!``").Pack(), true)
			return true
		}
		state.p.RemoveItem(itemID, 1)
		// Mundurkan 24 jam agar langsung siap panen
		tree.LastPick = tree.LastPick.Add(-24 * time.Hour)
		s.db.SavePlayer(state.p)
		s.db.SaveWorld(w)
		s.sendInventory(peer, state)
		s.playWorldSound(w.Name, "audio/spray.wav", float32(tileX*32), float32(tileY*32))
		s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`2Deluxe Grow Spray instantly matured the tree!``").Pack(), true)
		return true

	// 4. Upgrade Backpack (+10 Slots)
	case 9412:
		state.p.RemoveItem(itemID, 1)
		state.p.SlotSize += 10
		if state.p.SlotSize > 250 {
			state.p.SlotSize = 250
		}
		s.db.SavePlayer(state.p)
		s.sendInventory(peer, state)
		s.playWorldSound(w.Name, "audio/piano_nice.wav", state.posX, state.posY)
		s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), fmt.Sprintf("`2Backpack upgraded! Total slots: %d``", state.p.SlotSize)).Pack(), true)
		s.sendConsole(peer, "`2Your backpack has been upgraded by 10 slots! (Total: %d)", state.p.SlotSize)
		return true

	// 5. Door Mover (Moves White Door to target tile)
	case 1404:
		isOwner := w.Owner == state.p.UserID || s.config.IsOwner(state.growID) || state.p.CanAccess(role.LevelEliteGuardian)
		if !isOwner {
			s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4You must own the world to use a Door Mover!``").Pack(), true)
			return true
		}
		if tileY >= w.Height-1 {
			s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4Cannot place White Door at the very bottom!``").Pack(), true)
			return true
		}
		// Pindahkan White Door lama
		for y := 0; y < w.Height; y++ {
			for x := 0; x < w.Width; x++ {
				t := w.GetTile(x, y)
				if t != nil && t.Foreground == 6 {
					t.Foreground = 0
					s.sendTileUpdate(w, x, y, t)
				}
			}
		}
		// Pasang White Door di tile baru
		newDoorTile := w.GetTile(tileX, tileY)
		if newDoorTile != nil {
			newDoorTile.Foreground = 6 // White Door (EXIT)
			s.sendTileUpdate(w, tileX, tileY, newDoorTile)
		}
		// Pasang Bedrock di bawah pintu
		bedrockTile := w.GetTile(tileX, tileY+1)
		if bedrockTile != nil {
			bedrockTile.Foreground = 8 // Bedrock
			s.sendTileUpdate(w, tileX, tileY+1, bedrockTile)
		}
		w.SpawnTileX, w.SpawnTileY = tileX, tileY
		w.Doors = []world.Door{{Label: "EXIT", TileX: tileX, TileY: tileY}}

		state.p.RemoveItem(itemID, 1)
		s.db.SavePlayer(state.p)
		s.db.SaveWorld(w)
		s.sendInventory(peer, state)
		s.playWorldSound(w.Name, "audio/door_shut.wav", float32(tileX*32), float32(tileY*32))
		s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`2Moved the White Door to this position!``").Pack(), true)
		s.broadcastToWorld(w.Name, nil, variant.New("OnConsoleMessage", fmt.Sprintf("`w%s moved the White Door to (%d, %d).``", state.growID, tileX, tileY)).Pack(), true)
		return true

	// 6. Spike Juice (Immune to death spikes & lava)
	case 1662:
		state.p.RemoveItem(itemID, 1)
		s.db.SavePlayer(state.p)
		s.sendInventory(peer, state)
		s.playWorldSound(w.Name, "audio/hub_open.wav", state.posX, state.posY)
		s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`2Drank Spike Juice! Immune to Death Spikes and Lava!``").Pack(), true)
		s.sendConsole(peer, "`2You drank a refreshing Spike Juice potion!``")
		return true

	// 7. Change of Address (Rename / Swap World Name)
	case 2580:
		isOwner := w.Owner == state.p.UserID || s.config.IsOwner(state.growID) || state.p.CanAccess(role.LevelEliteGuardian)
		if !isOwner || !w.HasWorldLock() {
			s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4You must own this world with a World Lock to use Change of Address!``").Pack(), true)
			return true
		}
		s.sendChangeAddressDialog(peer, state, w)
		return true
	}

	return false
}

// sendChangeAddressDialog menampilkan dialog input nama world untuk ditukar
func (s *Server) sendChangeAddressDialog(peer *C.ENetPeer, state *peerState, w *world.World) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wChange of Address``", 2580)
	d.AddSpacer("small")
	d.AddTextbox(fmt.Sprintf("`oYou are about to swap the name of `w%s`` with another world that you own.``", w.Name))
	d.AddSmallText("`7Note: Both worlds must be locked by your World Lock. Structures and blocks will stay intact; only the world names will be swapped!``")
	d.AddSpacer("small")
	d.AddTextInput("target_world_name", "Target World Name:", "", 24)
	d.AddSpacer("small")
	d.AddButton("btn_confirm_swap", "`2Swap Names!``")
	d.AddButton("btn_cancel", "Cancel")
	d.AddQuickExit()
	s.sendDialog(peer, d.EndDialog("change_address_dialog", "", ""))
}

// handleChangeAddressReturn memproses penukaran nama world
func (s *Server) handleChangeAddressReturn(c *Ctx) {
	if c.Player == nil || c.World == nil {
		return
	}
	if c.Button() != "btn_confirm_swap" {
		return
	}
	targetName := strings.ToUpper(strings.TrimSpace(c.Str("target_world_name")))
	if targetName == "" || targetName == c.World.Name {
		c.Error("Invalid target world name.")
		return
	}
	if !world.ValidateWorldName(targetName) {
		c.Error("World name contains invalid characters.")
		return
	}
	targetWorld := s.getOrCreateWorld(targetName)
	if targetWorld == nil {
		c.Error("Target world could not be loaded.")
		return
	}
	if targetWorld.Owner != c.Player.UserID && !s.config.IsOwner(c.Player.GrowID) && !c.Player.CanAccess(role.LevelEliteGuardian) {
		c.Error("You must own the target world with a World Lock!")
		return
	}
	if !targetWorld.HasWorldLock() || !c.World.HasWorldLock() {
		c.Error("Both worlds must have a World Lock placed!")
		return
	}
	if !c.Player.HasItem(2580, 1) {
		c.Error("You don't have a Change of Address!")
		return
	}

	// Swap names in DB and memory
	oldName1 := c.World.Name
	oldName2 := targetWorld.Name

	s.worldsMu.Lock()
	c.World.Name = oldName2
	targetWorld.Name = oldName1
	s.worlds[oldName1] = targetWorld
	s.worlds[oldName2] = c.World
	s.worldsMu.Unlock()

	c.Player.RemoveItem(2580, 1)
	c.Save()
	s.db.SaveWorld(c.World)
	s.db.SaveWorld(targetWorld)
	c.SyncInventory()

	c.Sound("audio/piano_nice.wav")
	c.Success("World names successfully swapped! (%s <-> %s)", oldName1, oldName2)
	c.Bubble("`2[World Name Changed to %s!]``", c.World.Name)
}

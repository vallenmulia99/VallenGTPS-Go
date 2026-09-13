package server

/*
#include "../../enet/enet.h"
#include <stdlib.h>
*/
import "C"
import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	dialog "gtps/VallenSource/VallenDialog"
	ghostjar "gtps/VallenSource/VallenGhostJar"
	items "gtps/VallenSource/VallenItems"
	role "gtps/VallenSource/VallenRole"
	variant "gtps/VallenSource/VallenVariant"
	world "gtps/VallenSource/VallenWorld"
)

// ─────────────────────────────────────────────
// Tile Change: Punch, Wrench, & Place
// ─────────────────────────────────────────────

// handleTileChange memproses punch (ID==18), wrench block (ID==32), dan placing item dari player.
func (s *Server) handleTileChange(peer *C.ENetPeer, gp GamePacket) {
	state := s.getPeerState(peer)
	if state == nil || state.currentWorld == "" || state.p == nil {
		return
	}
	if s.handleDungeonTileChange(peer, state, gp) {
		return
	}

	w := s.getOrCreateWorld(state.currentWorld)
	if w == nil {
		return
	}

	tileX := int(gp.PunchX)
	tileY := int(gp.PunchY)
	tile := w.GetTile(tileX, tileY)
	if tile == nil {
		return
	}
	if !s.canReachTile(state, tileX, tileY) {
		s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4That tile is too far away.``").Pack(), true)
		return
	}

	// ── 1. PUNCHING / BREAKING (ID == 18) ─────────────────────────────────
	if gp.ID == 18 {
		// Proton Pack laser shooting check (Hand Item 3716)
		if state.p.GetClothing(5) == ghostjar.ItemProtonPack {
			c := s.NewVallenCtxFromState(peer, state, "action|punch", nil, nil)
			if ghostjar.HandleProtonPackPunch(c, int(gp.PunchX), int(gp.PunchY), s.ghosts) {
				return
			}
		}

		// Geiger Counter extraction check
		if state.p.GetClothing(items.ClothHand) == 2286 {
			dist := math.Hypot(float64(tileX-w.GeigerX), float64(tileY-w.GeigerY))
			if dist <= 1.5 {
				s.handleGeigerExtract(peer, state, w, tileX, tileY)
				return
			}
		}

		// Check if punching another player
		targetPeer, targetState := s.findPeerAtTile(w.Name, tileX, tileY, state.netID)
		if targetState != nil {
			if w.PunchJammerActive {
				s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`(Punching is disabled in this world!)``").Pack(), true)
				s.playWorldSound(w.Name, "audio/punch_locked.wav", float32(tileX*32), float32(tileY*32))
				return
			}
			s.playWorldPunchSound(w.Name, float32(tileX*32), float32(tileY*32))
			knockDir := float32(16.0)
			if state.facingLeft {
				knockDir = -16.0
			}
			targetState.posX += knockDir
			if targetPeer != nil {
				s.sendPacket(targetPeer, variant.NewWithNetID("OnSetPos", int32(targetState.netID), variant.Vec2f{X: targetState.posX, Y: targetState.posY}).Pack(), true)
			}
			return
		}

		targetItemID := tile.Foreground
		isBackground := false
		if targetItemID == 0 {
			targetItemID = tile.Background
			isBackground = true
		}

		if targetItemID == 0 {
			return // tile kosong
		}

		// Check for strong blocks and main door
		if world.IsStrongBlock(targetItemID) {
			s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "It's too strong to break.").Pack(), true)
			return
		}
		if world.IsMainDoor(targetItemID) {
			s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "(stand over and punch to use)").Pack(), true)
			return
		}

		// Enforce lock protection
		isPrivileged := state.p.CanAccess(role.LevelEliteGuardian) || s.config.IsOwner(state.growID)
		if !w.CanEditTile(state.p.UserID, tileX, tileY, isPrivileged) {
			s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4This area is protected.``").Pack(), true)
			s.playWorldSound(state.currentWorld, "audio/punch_locked.wav", float32(tileX*32), float32(tileY*32))
			return
		}

		// Special check for locks
		if world.IsWorldLockItem(targetItemID) {
			lock := w.FindLockAt(tileX, tileY)
			if lock != nil && lock.Owner != state.p.UserID && !isPrivileged {
				s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4You cannot break someone else's lock!``").Pack(), true)
				s.playWorldSound(state.currentWorld, "audio/punch_locked.wav", float32(tileX*32), float32(tileY*32))
				return
			}
		}

		// Harvest mature trees
		if _, planted := w.FindTree(tileX, tileY); planted && !isBackground {
			seedInfo := items.GetItem(uint16(targetItemID))
			if seedInfo != nil {
				fruitID, fruitCount, seedID, ready := w.HarvestTreeWithCount(tileX, tileY, seedInfo.GrowTimeSec, time.Now())
				if ready {
					tile.Foreground = 0
					tile.State = [4]uint8{0, 0, 0, 0}
					tile.ResetHits()

					// Animasi harvest tile & sound
					breakPkt := buildGamePacket(GamePacket{
						Type:   3, // PACKET_TILE_CHANGE_REQ
						NetID:  int32(state.netID),
						ID:     18,
						PunchX: int32(tileX),
						PunchY: int32(tileY),
					}, nil)
					s.broadcastToWorld(state.currentWorld, nil, breakPkt, true)
					s.sendTileUpdate(w, tileX, tileY, tile)
					s.playWorldSound(w.Name, "audio/harvest.wav", float32(tileX*32), float32(tileY*32))

					// Spawn world drop hasil panen
					pixelX := float32(tileX * 32)
					pixelY := float32(tileY * 32)
					s.spawnWorldDrop(w, fruitID, fruitCount, pixelX, pixelY, -1)
					s.spawnWorldDrop(w, seedID, 1, pixelX+8, pixelY, -1)

					// Chance drop gems berdasarkan rarity
					if seedInfo.Rarity > 0 {
						gemCount := int(seedInfo.Rarity) / 4
						if gemCount > 100 {
							gemCount = 100
						}
						if gemCount > 0 {
							s.spawnWorldDrop(w, 112, gemCount, pixelX+float32(rand.Intn(16)), pixelY+float32(rand.Intn(16)), -1)
						}
					}

					s.db.SaveWorld(w)
					return
				} else {
					s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "This tree is still growing.").Pack(), true)
					s.playWorldSound(state.currentWorld, "audio/punch_locked.wav", float32(tileX*32), float32(tileY*32))
					return
				}
			}
		}

		// Get required hits from items.dat or defaults
		itemInfo := items.GetItem(uint16(targetItemID))
		maxHits := world.GetRequiredHits(targetItemID)
		rarity := 0
		
		if itemInfo != nil {
			if itemInfo.Hits > 0 {
				maxHits = int(itemInfo.Hits)
			}
			rarity = int(itemInfo.Rarity)
		}

		// Apply tool bonuses
		handItem := state.p.GetClothing(9) // hand slot
		
		// Digger's Spade bonus for Dirt/Cave bg
		if handItem == 2952 && (targetItemID == 2 || targetItemID == 14) {
			tile.SetHits(isBackground, 3)
			// TODO: Send particle effect
		}

		// ── Interactive Casino / Game Blocks (Roulette Wheel & Dice) ──
		if targetItemID == 758 { // Roulette Wheel
			s.handleRouletteSpin(peer, state, w, tileX, tileY)
			return
		}
		if targetItemID == 756 || targetItemID == 1360 { // Dice Block
			s.handleDiceRoll(peer, state, w, tileX, tileY)
			return
		}

		// ── Interactive Weather Machine toggle on punch (GrowTavern standard) ──
		if isWeatherMachine(itemInfo, targetItemID) {
			s.handleWeatherMachineToggle(peer, state, w, tileX, tileY, targetItemID)
			return
		}

		// ── Interactive Jammer toggle on punch (GrowTavern standard) ──
		if isJammerItem(targetItemID) {
			s.handleJammerPunchToggle(peer, state, w, tileX, tileY, targetItemID)
			return
		}

		// Apply damage to tile
		shouldBreak := tile.ApplyDamage(isBackground, maxHits)

		// Send damage animation
		damagePkt := buildGamePacket(GamePacket{
			Type:   8, // PACKET_TILE_APPLY_DAMAGE
			NetID:  int32(state.netID),
			ID:     6,
			PunchX: int32(tileX),
			PunchY: int32(tileY),
		}, nil)
		s.broadcastToWorld(state.currentWorld, nil, damagePkt, true)
		s.playWorldPunchSound(state.currentWorld, float32(tileX*32), float32(tileY*32))

		// Check if block should break
		if !shouldBreak {
			return // Not enough hits yet
		}

		// BLOCK BREAKS!
		brokenID := targetItemID

		// Safeguard: Empty machine/display before breaking
		if isVendingItem(brokenID) {
			vend := w.FindVendingAt(tileX, tileY)
			if vend != nil && (vend.Count > 0 || vend.LocksEarned > 0) {
				s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4Empty the Vending Machine before breaking it!``").Pack(), true)
				return
			}
			w.RemoveVending(tileX, tileY)
		}
		if isDisplayItem(brokenID) {
			disp := w.FindDisplayAt(tileX, tileY)
			if disp != nil && disp.ItemID > 0 {
				s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4Empty the Display Box before breaking it!``").Pack(), true)
				return
			}
			w.RemoveDisplay(tileX, tileY)
		}

		if isDonationItem(brokenID) {
			box := w.FindDonationAt(tileX, tileY)
			if box != nil && len(box.Donations) > 0 {
				s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4Empty the Donation Box before breaking it!``").Pack(), true)
				return
			}
			w.RemoveDonation(tileX, tileY)
		}
		
		// Clear tile
		if isBackground {
			tile.Background = 0
		} else {
			tile.Foreground = 0
		}
		tile.State = [4]uint8{0, 0, 0, 0}
		tile.ResetHits()

		// Remove lock if applicable
		if world.IsWorldLockItem(brokenID) {
			w.RemoveLock(tileX, tileY)
			unlockedMsg := fmt.Sprintf("`5[```w%s`` lock has been removed by %s`5]``", w.Name, state.growID)
			s.broadcastToWorld(w.Name, nil, variant.New("OnConsoleMessage", unlockedMsg).Pack(), true)
			s.broadcastToWorld(w.Name, nil, variant.New("OnTalkBubble", int32(state.netID), unlockedMsg).Pack(), true)
			s.broadcastLockRemoval(w, brokenID, tileX, tileY)

			if world.IsWorldLevelLock(brokenID) {
				w.Owner = 0
				w.OwnerName = ""
				revertedName := role.FormatPlayerNameInWorld(state.growID, state.p.Role, state.p.Title, "", false)
				s.broadcastPlayerNameChange(w.Name, state.netID, revertedName)
			}
		}

		// Revert weather if broken block was active weather machine
		itemInfoBroken := items.GetItem(uint16(brokenID))
		if isWeatherMachine(itemInfoBroken, brokenID) && w.Weather == getWeatherIDFromItem(brokenID) {
			w.Weather = 0
			s.broadcastToWorld(w.Name, nil, variant.New("OnSetCurrentWeather", int32(0)).Pack(), true)
		}

		// Broadcast tile break animation (PACKET_TILE_CHANGE_REQ) & tile update
		breakPkt := buildGamePacket(GamePacket{
			Type:   3, // PACKET_TILE_CHANGE_REQ
			NetID:  int32(state.netID),
			ID:     18,
			PunchX: int32(tileX),
			PunchY: int32(tileY),
		}, nil)
		s.broadcastToWorld(state.currentWorld, nil, breakPkt, true)
		s.sendTileUpdate(w, tileX, tileY, tile)
		s.playWorldSound(w.Name, getBreakSound(uint16(brokenID)), float32(tileX*32), float32(tileY*32))

		// Calculate drops using rarity-based system
		pixelX := float32(tileX * 32)
		pixelY := float32(tileY * 32)

		// A destroyed World Lock is a return item: give it back immediately to
		// the breaker when inventory has space, matching the reference behavior.
		if world.IsWorldLockItem(brokenID) {
			overflow := state.p.AddItem(brokenID, 1)
			if overflow > 0 {
				s.spawnWorldDrop(w, brokenID, overflow, pixelX, pixelY, -1)
			} else {
				s.sendPacket(peer, buildInventoryPacket(state.p, state.netID), true)
				s.sendPacket(peer, variant.New("OnConsoleMessage", "`2World Lock returned to your inventory.``").Pack(), true)
			}
		} else {
			// Use rarity-based drop calculation
			drops := world.CalculateDrops(brokenID, rarity)
			
			// Drop gems (in appropriate denominations)
			if drops.Gems > 0 {
				remaining := drops.Gems
				// Break into gem types: 10, 5, 1
				for _, gemSize := range []int{10, 5, 1} {
					for remaining >= gemSize {
						s.spawnWorldDrop(w, drops.GemItemID, gemSize, pixelX+float32(rand.Intn(16)), pixelY+float32(rand.Intn(16)), -1)
						remaining -= gemSize
					}
				}
			}
			
			// Drop seeds
			if drops.Seeds > 0 {
				s.spawnWorldDrop(w, drops.SeedItemID, drops.Seeds, pixelX, pixelY, -1)
			}
			
			// Drop blocks
			if drops.Blocks > 0 {
				s.spawnWorldDrop(w, drops.BlockItemID, drops.Blocks, pixelX, pixelY, -1)
			}
		}

		// Award XP based on rarity
		xpReward := world.CalculateXP(rarity)
		state.p.XP += xpReward
		if state.p.XP >= state.p.Level*100 {
			state.p.Level++
		}
		
		s.db.SavePlayer(state.p)
		s.db.SaveWorld(w)
		return
	}

	// ── 2. WRENCHING A BLOCK (ID == 32) ──────────────────────────────────
	if gp.ID == 32 {
		targetID := tile.Foreground
		if targetID == 0 {
			targetID = tile.Background
		}

		if targetID == 0 {
			return
		}

		s.handleWrenchTile(peer, state, w, tileX, tileY, targetID, tile)
		return
	}

	// ── 3. PLACING A BLOCK (ID != 18 && ID != 32) ─────────────────────────
	if gp.ID > 0 {
		placeItemID := int(gp.ID)

		// Verifikasi player punya item di inventory
		if !state.p.HasItem(placeItemID, 1) {
			return
		}

		// Ghost Jar item handling (Ghost release / Trap Jar)
		if placeItemID == ghostjar.ItemGhostInJar || placeItemID == ghostjar.ItemShadowGhostInJar || placeItemID == ghostjar.ItemGhostJar {
			c := s.NewVallenCtxFromState(peer, state, "action|place", nil, nil)
			if ghostjar.HandleGhostJarItem(c, tileX, tileY, placeItemID, s.ghosts) {
				return
			}
		}

		// ── Guard: Clothing Equip/Unequip saat item baju/rambut/celana di-tap di hotbar ──
		// Jika player tap menggunakan clothing, jangan tempatkan sebagai block melainkan EQUIP/UNEQUIP!
		itemInfo := items.GetItem(uint16(placeItemID))
		if itemInfo != nil && itemInfo.IsClothing() {
			s.equipOrUnequipClothing(peer, state, placeItemID)
			return
		}

		// Consumable items (XP Potion, Grow Spray, Deluxe Grow Spray, Backpack Upgrade, Door Mover, Spike Juice)
		if isConsumableItem(placeItemID) {
			if s.handleUseConsumable(peer, state, w, tileX, tileY, placeItemID) {
				return
			}
		}

		// Cek proteksi World Lock / Lock pada world
		isPrivileged := state.p.CanAccess(role.LevelEliteGuardian) || s.config.IsOwner(state.growID)
		if !w.CanEditTile(state.p.UserID, tileX, tileY, isPrivileged) {
			s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4This area is protected.``").Pack(), true)
			s.playWorldSound(state.currentWorld, "audio/cant_place_tile.wav", float32(tileX*32), float32(tileY*32))
			return
		}

		// Cek apakah item adalah lock
		isLockItem := world.IsWorldLockItem(placeItemID)
		
		// Check if trying to place a World Lock when one already exists
		if world.IsWorldLevelLock(placeItemID) && w.HasWorldLock() {
			s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4Only one World Lock can be placed in a world!``").Pack(), true)
			s.playWorldSound(state.currentWorld, "audio/cant_place_tile.wav", float32(tileX*32), float32(tileY*32))
			return
		}
		
		isClothing := itemInfo != nil && itemInfo.IsClothing()
		var itemType uint8 = 0
		if itemInfo != nil {
			itemType = itemInfo.Type
		}
		if err := w.ValidatePlacement(tile, isClothing, itemType); err != nil {
			s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4"+err.Error()+"``").Pack(), true)
			return
		}

		isBg := false
		if itemInfo != nil && itemInfo.Type == items.TypeBackground { // TypeBackground = 0x12
			isBg = true
		}

		// Clothing / consumable / fist / wrench → diblock dari tile placement oleh validatePlacement.
		// Ghost jar items sudah di-handle di atas dan return lebih awal.

		// Plant seeds as persistent tree state; the seed remains visible on the tile.
		if itemInfo != nil && itemInfo.Type == items.TypeSeed {
			// Check if placing seed on existing tree (splicing)
			if tile.Foreground != 0 {
				existingSeedID := tile.Foreground
				
				// Check if already spliced
				if tile.IsSpliced() {
					s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "It would be too dangerous to try to mix three seeds.").Pack(), true)
					return
				}
				
				// Try to find splice combination
				canSplice, resultSeedID, seed1Name, seed2Name, resultName := items.FindSpliceResult(existingSeedID, placeItemID)
				
				if !canSplice {
					errorMsg := fmt.Sprintf("Hmm, it looks like `w%s`` and `w%s`` can't be spliced.", 
						items.GetItem(uint16(existingSeedID)).Name, itemInfo.Name)
					s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), errorMsg).Pack(), true)
					return
				}
				
				// Splice success!
				successMsg := fmt.Sprintf("`w%s`` and `w%s`` have been spliced to make a `$%s``!", 
					seed1Name, seed2Name, resultName[:len(resultName)-5]) // Remove " Seed" suffix
				
				// Mark tile as spliced
				tile.MarkAsSpliced()
				tile.Foreground = resultSeedID
				
				// Reset tree growth time
				w.UpdateTreeTime(tileX, tileY, time.Now())
				
				// Remove seed from inventory
				state.p.RemoveItem(placeItemID, 1)
				
				// Send success message
				s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), successMsg, 0, 1).Pack(), true)
				s.sendTileUpdate(w, tileX, tileY, tile)
				s.sendPacket(peer, buildInventoryPacket(state.p, state.netID), true)
				s.db.SavePlayer(state.p)
				s.db.SaveWorld(w)
				return
			}
			
			// Normal planting: needs solid block or platform underneath
			tileBelow := w.GetTile(tileX, tileY+1)
			if tileBelow == nil || tileBelow.Foreground == 0 {
				s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4You must plant seeds on solid ground!``").Pack(), true)
				s.playWorldSound(state.currentWorld, "audio/cant_place_tile.wav", float32(tileX*32), float32(tileY*32))
				return
			}
			
			// Plant with random fruit count (1-3)
			fruitCount := rand.Intn(3) + 1
			if !w.PlantTreeWithFruit(tileX, tileY, placeItemID-1, fruitCount, time.Now()) {
				return
			}
		}

		if isBg {
			if tile.Background != 0 {
				return // Background sudah terisi
			}
			tile.Background = placeItemID
		} else {
			if tile.Foreground != 0 {
				return // Foreground sudah terisi
			}
			tile.Foreground = placeItemID
		}

		// Jika menaruh Lock, add ke lock list
		if isLockItem {
			lock := w.AddLock(placeItemID, tileX, tileY, state.p.UserID)
			lock.OwnerName = state.growID
			
			if world.IsWorldLevelLock(placeItemID) {
				w.Owner = state.p.UserID
				w.OwnerName = state.growID
				lockItemName := items.GetItem(uint16(placeItemID)).Name
				lockedMsg := fmt.Sprintf("`5[```w%s`` has been `$World Locked`` by %s with a %s`5]``", w.Name, state.growID, lockItemName)
				s.broadcastToWorld(w.Name, nil, variant.New("OnConsoleMessage", lockedMsg).Pack(), true)
				s.broadcastToWorld(w.Name, nil, variant.New("OnTalkBubble", int32(state.netID), lockedMsg).Pack(), true)
				
				// Update tile state to reflect public/private
				if lock.IsPublic {
					tile.State[2] |= world.TileStatePublic
				}

				// TURNS PLAYER NAME GREEN INSTANTLY!
				ownerOverheadName := role.FormatPlayerNameInWorld(state.growID, state.p.Role, state.p.Title, w.OwnerName, false)
				s.broadcastPlayerNameChange(w.Name, state.netID, ownerOverheadName)
			} else {
				// Tile lock placed (Small, Big, Huge, Builder)
				lockMsg := fmt.Sprintf("`5[```w%s`` placed a lock at (%d, %d)`5]``", state.growID, tileX, tileY)
				s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), lockMsg).Pack(), true)
			}

			// Broadcast Packet 15 (NET_GAME_PACKET_SEND_LOCK) so client draws the lock boundary
			s.broadcastLockPacket(w, lock, state.growID)
		}

		// Door/Portal placement - create empty door entry
		if itemInfo != nil && (itemInfo.Type == 0x09 || itemInfo.Type == 0x13) { // TYPE_DOOR or TYPE_PORTAL
			w.AddDoor("", "", "", tileX, tileY)
		}

		// Sign placement
		if itemInfo != nil && itemInfo.Type == 0x0A { // TYPE_SIGN
			w.UpdateSign("", tileX, tileY)
		}

		// Kurangi item dari inventory
		state.p.RemoveItem(placeItemID, 1)
		s.db.SavePlayer(state.p)
		s.db.SaveWorld(w)

		// Broadcast tile update ke world dengan format extra data lengkap
		s.sendTileUpdate(w, tileX, tileY, tile)
		if isLockItem {
			s.playWorldSound(w.Name, "audio/use_lock.wav", float32(tileX*32), float32(tileY*32))
		} else {
			s.playWorldSound(w.Name, "audio/tile_created.wav", float32(tileX*32), float32(tileY*32))
		}

		// Update inventory di client
		invPkt := buildInventoryPacket(state.p, state.netID)
		s.sendPacket(peer, invPkt, true)
		return
	}
}

// handleWrenchTile menampilkan dialog edit sesuai tipe block yang di-wrench
func (s *Server) handleWrenchTile(peer *C.ENetPeer, state *peerState, w *world.World, tileX, tileY, itemID int, tile *world.Tile) {
	itemInfo := items.GetItem(uint16(itemID))
	rawName := "Block"
	if itemInfo != nil && itemInfo.Name != "" {
		rawName = itemInfo.Name
	}

	// 0. Vending Machines & Display Blocks
	if isVendingItem(itemID) {
		s.handleVendingWrench(peer, state, w, tileX, tileY)
		return
	}
	if isDisplayItem(itemID) {
		s.handleDisplayWrench(peer, state, w, tileX, tileY)
		return
	}

	// 0.1 Weather Machines (Open Dialog on Wrench)
	if isWeatherMachine(itemInfo, itemID) {
		s.sendWeatherMachineDialog(peer, state, w, tileX, tileY, itemID)
		return
	}

	// 0.2 Donation Boxes
	if isDonationItem(itemID) {
		s.handleDonationWrench(peer, state, w, tileX, tileY)
		return
	}

	// 0.3 Casino & Entertainment Blocks (Roulette & Dice)
	if itemID == 758 || itemID == 756 || itemID == 1360 {
		s.handleCasinoWrench(peer, state, w, tileX, tileY, itemID)
		return
	}

	// 0.4 World Jammers (Open Dialog on Wrench)
	if isJammerItem(itemID) {
		s.sendJammerDialog(peer, state, w, tileX, tileY, itemID)
		return
	}

	// 1. Doors / Portals / Main Door
	if itemID == 6 || itemID == 12 || (itemInfo != nil && (itemInfo.Type == 0x09 || itemInfo.Type == items.TypeDoor || itemInfo.Type == 0x02)) {
		label := ""
		dest := ""
		doorID := ""
		for _, d := range w.Doors {
			if d.TileX == tileX && d.TileY == tileY {
				label = d.Label
				dest = d.Dest
				doorID = d.ID
				break
			}
		}

		doorDialog := fmt.Sprintf(
			"set_default_color|`o\n"+
				"add_label_with_icon|big|`wEdit %s``|left|%d|\n"+
				"add_spacer|small|\n"+
				"add_text_input|door_name|Label|%s|100|\n"+
				"add_text_input|door_target|Destination|%s|24|\n"+
				"add_smalltext|Destination format: `2WORLDNAME:ID`` or `:ID`` for current world.|left|\n"+
				"add_text_input|door_id|Door ID|%s|11|\n"+
				"add_smalltext|Set a unique ID to target this door from another door.|left|\n"+
				"add_checkbox|checkbox_locked|Is open to public|1\n"+
				"embed_data|tilex|%d\n"+
				"embed_data|tiley|%d\n"+
				"add_quick_exit|\n"+
				"end_dialog|door_edit|Cancel|OK|\n",
			rawName, itemID, label, dest, doorID, tileX, tileY,
		)
		s.sendPacket(peer, variant.New("OnDialogRequest", doorDialog).Pack(), true)
		return
	}

	// 2. Trees / Seeds
	if itemInfo != nil && itemInfo.Type == items.TypeSeed {
		cleanName := strings.ReplaceAll(rawName, "|", "")
		fruitID := int(itemInfo.ID) - 1
		fruitItem := items.GetItem(uint16(fruitID))
		fruitName := fmt.Sprintf("Item #%d", fruitID)
		if fruitItem != nil && fruitItem.Name != "" {
			fruitName = fruitItem.Name
		}

		tree, found := w.FindTree(tileX, tileY)
		growTimeSec := int(itemInfo.GrowTimeSec)
		if growTimeSec <= 0 {
			growTimeSec = int(itemInfo.Rarity)*10 + 30
		}

		var treeDialog string
		if found {
			elapsed := int(time.Since(tree.LastPick).Seconds())
			if elapsed >= growTimeSec {
				treeDialog = fmt.Sprintf(
					"set_default_color|`o\n"+
						"add_label_with_icon|big|`w%s``|left|%d|\n"+
						"add_spacer|small|\n"+
						"add_textbox|`2This tree is ready to harvest!``|left|\n"+
						"add_spacer|small|\n"+
						"add_textbox|Punch this tree to harvest `w%s`` and seeds.|left|\n"+
						"add_textbox|Rarity: `2%d``  Fruit Capacity: `2%d``|left|\n"+
						"add_spacer|small|\n"+
						"end_dialog|tree_info||OK|\n",
					cleanName, itemID, fruitName, itemInfo.Rarity, tree.FruitCount,
				)
			} else {
				remSec := growTimeSec - elapsed
				if remSec < 0 {
					remSec = 0
				}
				hours := remSec / 3600
				mins := (remSec % 3600) / 60
				secs := remSec % 60
				timeStr := ""
				if hours > 0 {
					timeStr = fmt.Sprintf("%dh %dm %ds", hours, mins, secs)
				} else if mins > 0 {
					timeStr = fmt.Sprintf("%dm %ds", mins, secs)
				} else {
					timeStr = fmt.Sprintf("%ds", secs)
				}

				treeDialog = fmt.Sprintf(
					"set_default_color|`o\n"+
						"add_label_with_icon|big|`w%s``|left|%d|\n"+
						"add_spacer|small|\n"+
						"add_textbox|`oThis tree is still growing.|left|\n"+
						"add_textbox|Time left: `2%s``|left|\n"+
						"add_spacer|small|\n"+
						"add_textbox|Will bear fruit: `w%s``|left|\n"+
						"add_textbox|Rarity: `2%d``|left|\n"+
						"add_spacer|small|\n"+
						"end_dialog|tree_info||OK|\n",
					cleanName, itemID, timeStr, fruitName, itemInfo.Rarity,
				)
			}
		} else {
			// Planted but no tree state tracked yet; initialize it
			w.PlantTreeWithFruit(tileX, tileY, fruitID, 3, time.Now())
			treeDialog = fmt.Sprintf(
				"set_default_color|`o\n"+
					"add_label_with_icon|big|`w%s``|left|%d|\n"+
					"add_spacer|small|\n"+
					"add_textbox|`oPlanted recently and started growing.|left|\n"+
					"add_textbox|Will bear fruit: `w%s``|left|\n"+
					"add_textbox|Rarity: `2%d``|left|\n"+
					"add_spacer|small|\n"+
					"end_dialog|tree_info||OK|\n",
				cleanName, itemID, fruitName, itemInfo.Rarity,
			)
		}
		s.sendPacket(peer, variant.New("OnDialogRequest", treeDialog).Pack(), true)
		return
	}

	// 3. Signs
	if itemID == 20 || (itemInfo != nil && itemInfo.Type == 0x0A) {
		label := ""
		for _, sg := range w.Signs {
			if sg.TileX == tileX && sg.TileY == tileY {
				label = sg.Label
				break
			}
		}

		signDialog := fmt.Sprintf(
			"set_default_color|`o\n"+
				"add_label_with_icon|big|`wEdit %s``|left|%d|\n"+
				"add_spacer|small|\n"+
				"add_textbox|What would you like to write on this sign?|left|\n"+
				"add_text_input|sign_text||%s|128|\n"+
				"embed_data|tilex|%d\n"+
				"embed_data|tiley|%d\n"+
				"add_quick_exit|\n"+
				"end_dialog|sign_edit|Cancel|OK|\n",
			rawName, itemID, label, tileX, tileY,
		)
		s.sendPacket(peer, variant.New("OnDialogRequest", signDialog).Pack(), true)
		return
	}

	// 3. Locks (World Lock, Small Lock, Big Lock, Builder's Lock)
	if world.IsWorldLockItem(itemID) {
		// Find lock at this position
		lock := w.FindLockAt(tileX, tileY)
		if lock == nil {
			s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4No lock data found!``").Pack(), true)
			return
		}

		// Check ownership
		isOwner := lock.Owner == state.p.UserID || s.config.IsOwner(state.growID) || state.p.CanAccess(9)
		if !isOwner {
			s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4You don't own this lock!``").Pack(), true)
			return
		}

		publicVal := "0"
		if lock.IsPublic {
			publicVal = "1"
		}

		disableMusicVal := "0"
		if lock.LockState&world.LockStateDisableMusic != 0 {
			disableMusicVal = "1"
		}

		// Access list display
		accessListStr := "Currently, you're the only one with access."
		accessCount := 0
		for _, userID := range lock.AccessList {
			if userID != 0 {
				accessCount++
			}
		}
		if accessCount > 0 {
			accessListStr = fmt.Sprintf("%d player(s) have access", accessCount)
		}

		var lockDialog string
		if world.IsTileLock(itemID) {
			if itemID == world.ItemBuilderLock {
				lockDialog = fmt.Sprintf(
					"set_default_color|`o\n"+
						"add_label_with_icon|big|`wEdit Builder's Lock``|left|4994|\n"+
						"add_spacer|small|\n"+
						"add_textbox|`oAccess list:``|left|\n"+
						"add_textbox|%s|left|\n"+
						"add_spacer|small|\n"+
						"add_player_picker|playerNetID|`wAdd Player``|\n"+
						"add_spacer|small|\n"+
						"add_checkbox|checkbox_public|Allow anyone to Build or Break|%s\n"+
						"add_checkbox|checkbox_buildonly|Only Allow Building!|0\n"+
						"add_checkbox|checkbox_admins|Admins Are Limited|0\n"+
						"add_button|recalcLock|`wRe-apply lock``|noflags|0|0|\n"+
						"embed_data|tilex|%d\n"+
						"embed_data|tiley|%d\n"+
						"add_quick_exit|\n"+
						"end_dialog|lock_edit|Cancel|OK|\n",
					accessListStr, publicVal, tileX, tileY,
				)
			} else {
				// Small Lock (202), Big Lock (204), Huge Lock (206)
				lockDialog = fmt.Sprintf(
					"set_default_color|`o\n"+
						"add_label_with_icon|big|`wEdit %s``|left|%d|\n"+
						"add_spacer|small|\n"+
						"add_textbox|`oAccess list:``|left|\n"+
						"add_textbox|%s|left|\n"+
						"add_spacer|small|\n"+
						"add_player_picker|playerNetID|`wAdd Player``|\n"+
						"add_spacer|small|\n"+
						"add_checkbox|checkbox_public|Allow anyone to Build and Break|%s\n"+
						"add_checkbox|checkbox_ignore|Ignore empty air|0\n"+
						"add_button|recalcLock|`wRe-apply lock``|noflags|0|0|\n"+
						"embed_data|tilex|%d\n"+
						"embed_data|tiley|%d\n"+
						"add_quick_exit|\n"+
						"end_dialog|lock_edit|Cancel|OK|\n",
					rawName, itemID, accessListStr, publicVal, tileX, tileY,
				)
			}
		} else if itemID == world.ItemRoyalLock || itemID == world.ItemRoboticLock {
			// Royal Lock / Robotic Lock
			lockDialog = fmt.Sprintf(
				"set_default_color|`o\n"+
					"add_label_with_icon|big|`wEdit %s``|left|%d|\n"+
					"add_spacer|small|\n"+
					"add_textbox|`oAccess list:``|left|\n"+
					"add_textbox|%s|left|\n"+
					"add_spacer|small|\n"+
					"add_player_picker|playerNetID|`wAdd Player``|\n"+
					"add_spacer|small|\n"+
					"add_checkbox|checkbox_public|Allow anyone to Build and Break|%s\n"+
					"add_textbox|`wYe Royal Options:``|left|\n"+
					"add_checkbox|checkbox_silence|Silence, Peasants!|0\n"+
					"add_checkbox|checkbox_rainbows|Royal Rainbows!|0\n"+
					"add_checkbox|checkbox_disable_music|Disable Custom Music Blocks|%s\n"+
					"add_text_input|minimum_entry_level|Minimum World Level: |%d|3|\n"+
					"add_button|getKey|Get World Key|noflags|0|0|\n"+
					"embed_data|tilex|%d\n"+
					"embed_data|tiley|%d\n"+
					"add_quick_exit|\n"+
					"end_dialog|lock_edit|Cancel|OK|\n",
				rawName, itemID, accessListStr, publicVal, disableMusicVal, w.MinimumEntryLevel, tileX, tileY,
			)
		} else {
			// World Lock (242), Diamond Lock (1796), Blue Gem Lock (7188), etc.
			lockDialog = fmt.Sprintf(
				"set_default_color|`o\n"+
					"add_label_with_icon|big|`wEdit %s``|left|%d|\n"+
					"add_spacer|small|\n"+
					"add_textbox|`oAccess list:``|left|\n"+
					"add_textbox|%s|left|\n"+
					"add_spacer|small|\n"+
					"add_player_picker|playerNetID|`wAdd Player``|\n"+
					"add_spacer|small|\n"+
					"add_checkbox|checkbox_public|Allow anyone to Build and Break|%s\n"+
					"add_checkbox|checkbox_disable_music|Disable Custom Music Blocks|%s\n"+
					"add_text_input|minimum_entry_level|Minimum World Level: |%d|3|\n"+
					"add_smalltext|Only players with this level or higher can enter the world.|\n"+
					"add_button|getKey|Get World Key|noflags|0|0|\n"+
					"embed_data|tilex|%d\n"+
					"embed_data|tiley|%d\n"+
					"add_quick_exit|\n"+
					"end_dialog|lock_edit|Cancel|OK|\n",
				rawName, itemID, accessListStr, publicVal, disableMusicVal, w.MinimumEntryLevel, tileX, tileY,
			)
		}
		s.sendPacket(peer, variant.New("OnDialogRequest", lockDialog).Pack(), true)
		return
	}
}

// ─────────────────────────────────────────────
// Tile Activation (Doors, Portals)
// ─────────────────────────────────────────────

// handleTileActivate memproses aktivasi tile seperti Main Door / EXIT.
func (s *Server) handleTileActivate(peer *C.ENetPeer, gp GamePacket) {
	state := s.getPeerState(peer)
	if state == nil || state.currentWorld == "" {
		return
	}

	w := s.getOrCreateWorld(state.currentWorld)
	if w == nil {
		return
	}

	tileX := int(gp.PunchX)
	tileY := int(gp.PunchY)
	tile := w.GetTile(tileX, tileY)
	if tile == nil {
		return
	}
	if !s.canReachTile(state, tileX, tileY) {
		return
	}
	if s.handleDungeonDoor(peer, state, tileX, tileY) {
		return
	}

	itemInfo := items.GetItem(uint16(tile.Foreground))

	// Main Door / EXIT - quit to world select
	if tile.Foreground == 6 {
		s.playWorldSound(state.currentWorld, "audio/door_shut.wav", float32(tileX*32), float32(tileY*32))
		s.handleQuitToExit(peer)
		return
	}

	// Door / Portal - teleport to destination
	if itemInfo != nil && (itemInfo.Type == items.TypeDoor || itemInfo.Type == 0x02 || itemInfo.Type == 0x09) {
		s.playWorldSound(state.currentWorld, "audio/door_open.wav", float32(tileX*32), float32(tileY*32))
		door := w.FindDoorAt(tileX, tileY)
		if door == nil || door.Dest == "" {
			// No destination set, teleport to spawn with zoom effect
			s.sendPacket(peer, variant.New("OnSetPos", []float32{float32(w.SpawnTileX * 32), float32(w.SpawnTileY * 32)}).Pack(), true)
			s.sendPacket(peer, variant.New("OnZoomCamera", []float32{10000.0, 1000.0}, int32(state.netID)).Pack(), true)
			s.sendPacket(peer, variant.New("OnSetFreezeState", int32(0), int32(state.netID)).Pack(), true)
			// TODO: play audio/teleport.wav
			return
		}

		// Parse destination
		destWorld, doorID, valid := world.ParseDoorDestination(door.Dest)
		if !valid {
			s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4Invalid door destination!``").Pack(), true)
			return
		}

		// If no world name, use current world
		if destWorld == "" {
			destWorld = state.currentWorld
		}

		// Validate world name
		if !world.ValidateWorldName(destWorld) {
			s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4Invalid world name!``").Pack(), true)
			return
		}

		// Quit current world (skip world select menu)
		s.handleQuitToExit(peer)

		// Join destination world
		s.joinWorld(peer, state, destWorld, doorID)
		return
	}

	// Checkpoint - set spawn point
	if itemInfo != nil && itemInfo.Type == 0x0D { // TYPE_CHECKPOINT
		// Get old checkpoint position from player's current spawn (convert pixel to tile)
		oldX := int(state.p.RespawnX / 32)
		oldY := int(state.p.RespawnY / 32)

		// Update spawn point (convert tile to pixel)
		state.p.RespawnX = float32(tileX * 32)
		state.p.RespawnY = float32(tileY * 32)

		// Toggle checkpoint tiles
		w.ToggleCheckpoint(oldX, oldY, tileX, tileY)

		// Send tile updates for both checkpoints
		if oldTile := w.GetTile(oldX, oldY); oldTile != nil {
			s.sendTileUpdate(w, oldX, oldY, oldTile)
		}
		s.sendTileUpdate(w, tileX, tileY, tile)

		// Save
		s.db.SavePlayer(state.p)
		s.db.SaveWorld(w)

		s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`2Checkpoint set!``").Pack(), true)
		s.playWorldSound(state.currentWorld, "audio/beep.wav", float32(tileX*32), float32(tileY*32))
		return
	}
}

// ─────────────────────────────────────────────
// Tile Update Broadcast (Dengan Extra Data GTPS yang Tepat)
// ─────────────────────────────────────────────

// sendTileUpdate mengirim paket PACKET_SEND_TILE_UPDATE_DATA lengkap dengan extra data ke semua player di world.
func (s *Server) sendTileUpdate(w *world.World, tileX, tileY int, tile *world.Tile) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint16(buf[0:], uint16(tile.Foreground))
	binary.LittleEndian.PutUint16(buf[2:], uint16(tile.Background))
	buf[4] = tile.State[0]
	buf[5] = tile.State[1]
	buf[6] = tile.State[2]
	buf[7] = tile.State[3]

	// Extra data per tipe item sesuai spesifikasi protokol Growtopia
	itemInfo := items.GetItem(uint16(tile.Foreground))
	itemType := uint8(0)
	if itemInfo != nil {
		itemType = itemInfo.Type
	}

	switch {
	case tile.Foreground == 6 || tile.Foreground == 12 || itemType == 0x09 || itemType == 0x13: // DOOR / MAIN_DOOR / PORTAL
		buf = append(buf, 0x01) // type: door
		label := ""
		for _, d := range w.Doors {
			if d.TileX == tileX && d.TileY == tileY {
				label = d.Label
				break
			}
		}
		labelBytes := make([]byte, 2)
		binary.LittleEndian.PutUint16(labelBytes, uint16(len(label)))
		buf = append(buf, labelBytes...)
		buf = append(buf, []byte(label)...)
		buf = append(buf, 0x00) // null terminator

	case tile.Foreground == 20 || itemType == 0x0A: // SIGN
		buf = append(buf, 0x02) // type: sign
		label := ""
		for _, sg := range w.Signs {
			if sg.TileX == tileX && sg.TileY == tileY {
				label = sg.Label
				break
			}
		}
		labelBytes := make([]byte, 2)
		binary.LittleEndian.PutUint16(labelBytes, uint16(len(label)))
		buf = append(buf, labelBytes...)
		buf = append(buf, []byte(label)...)
		b4 := make([]byte, 4)
		binary.LittleEndian.PutUint32(b4, 0xFFFFFFFF)
		buf = append(buf, b4...)

	case world.IsWorldLockItem(tile.Foreground) || itemType == 0x03: // LOCK
		buf = append(buf, 0x03) // type: lock
		lock := w.FindLockAt(tileX, tileY)
		ownerID := uint32(w.Owner)
		lockState := uint8(w.LockState)
		var accessList []uint32

		if lock != nil {
			ownerID = uint32(lock.Owner)
			lockState = uint8(lock.LockState)
			for _, uid := range lock.AccessList {
				if uid != 0 {
					accessList = append(accessList, uint32(uid))
				}
			}
		} else {
			for _, uid := range w.AccessList {
				if uid != 0 {
					accessList = append(accessList, uint32(uid))
				}
			}
		}

		buf = append(buf, lockState)
		bOwner := make([]byte, 4)
		binary.LittleEndian.PutUint32(bOwner, ownerID)
		buf = append(buf, bOwner...)

		bAccessCount := make([]byte, 4)
		binary.LittleEndian.PutUint32(bAccessCount, uint32(len(accessList)))
		buf = append(buf, bAccessCount...)

		for _, uid := range accessList {
			bUID := make([]byte, 4)
			binary.LittleEndian.PutUint32(bUID, uid)
			buf = append(buf, bUID...)
		}

	case isVendingItem(tile.Foreground):
		buf = append(buf, 0x18) // type: 24 (vending)
		vend := w.FindVendingAt(tileX, tileY)
		itemID := int32(0)
		price := int32(0)
		if vend != nil {
			itemID = int32(vend.ItemID)
			price = int32(vend.Price)
		}
		bItem := make([]byte, 4)
		binary.LittleEndian.PutUint32(bItem, uint32(itemID))
		buf = append(buf, bItem...)
		bPrice := make([]byte, 4)
		binary.LittleEndian.PutUint32(bPrice, uint32(price))
		buf = append(buf, bPrice...)

	case isDisplayItem(tile.Foreground):
		buf = append(buf, 0x17) // type: 23 (display)
		disp := w.FindDisplayAt(tileX, tileY)
		itemID := int32(0)
		if disp != nil {
			itemID = int32(disp.ItemID)
		}
		bItem := make([]byte, 4)
		binary.LittleEndian.PutUint32(bItem, uint32(itemID))
		buf = append(buf, bItem...)

	default:
		buf = append(buf, 0x00) // standard block tanpa extra data
	}

	pkt := buildGamePacket(GamePacket{
		Type:   5, // PACKET_SEND_TILE_UPDATE_DATA
		PunchX: int32(tileX),
		PunchY: int32(tileY),
		State:  8, // S_EXTENDED
	}, buf)

	s.broadcastToWorld(w.Name, nil, pkt, true)
}

// getBreakSound returns the authentic block breaking sound based on item material.
func getBreakSound(itemID uint16) string {
	itemInfo := items.GetItem(itemID)
	if itemInfo != nil {
		name := strings.ToLower(itemInfo.Name)
		if strings.Contains(name, "wood") || strings.Contains(name, "tree") || strings.Contains(name, "plank") || strings.Contains(name, "fence") {
			return "audio/wood_break.wav"
		}
		if strings.Contains(name, "metal") || strings.Contains(name, "steel") || strings.Contains(name, "iron") || strings.Contains(name, "safe") || strings.Contains(name, "robot") {
			return "audio/metal_destroy.wav"
		}
	}
	return "audio/rock_destroy.wav"
}

// broadcastLockPacket sends Packet 15 (NET_GAME_PACKET_SEND_LOCK) to world peers
func (s *Server) broadcastPlayerNameChange(worldName string, netID int, formattedName string) {
	pkt := variant.NewWithNetID("OnNameChanged", int32(netID), formattedName).Pack()
	s.broadcastToWorld(worldName, nil, pkt, true)
}

func (s *Server) broadcastLockPacket(w *world.World, lock *world.Lock, ownerName string) {
	if w == nil || lock == nil {
		return
	}
	ownerID := uint32(lock.Owner)
	if ownerID == 0 {
		ownerID = world.HashGrowID(ownerName)
	}
	lockedTiles := w.GetLockedTiles(lock)
	lockPkt := buildLockGamePacket(lock, ownerID, lockedTiles)
	s.broadcastGamePacketToWorld(w.Name, nil, lockPkt, true)
}

// broadcastLockRemoval sends Packet 15 with NetID=-1 to clear lock boundary
func (s *Server) broadcastLockRemoval(w *world.World, lockItemID, tileX, tileY int) {
	gp := GamePacket{
		Type:   15,
		NetID:  -1,
		State:  0x08,
		PunchX: int32(tileX),
		PunchY: int32(tileY),
		ID:     int32(lockItemID),
	}
	s.broadcastGamePacketToWorld(w.Name, nil, buildGamePacket(gp, nil), true)
}

// buildLockGamePacket builds Packet 15 with optional locked tile indices
func buildLockGamePacket(lock *world.Lock, ownerHash uint32, lockedTileIndices []uint16) []byte {
	var extraData []byte
	if len(lockedTileIndices) > 0 {
		extraData = make([]byte, len(lockedTileIndices)*2)
		for i, idx := range lockedTileIndices {
			binary.LittleEndian.PutUint16(extraData[i*2:], idx)
		}
	}

	gp := GamePacket{
		Type:   15, // PACKET_SEND_LOCK
		NetID:  int32(ownerHash),
		State:  0x08,
		PunchX: int32(lock.TileX),
		PunchY: int32(lock.TileY),
		ID:     int32(lock.ItemID),
	}
	buf := buildGamePacket(gp, extraData)
	if len(lockedTileIndices) > 0 {
		binary.LittleEndian.PutUint16(buf[12:], uint16(len(lockedTileIndices)))
		binary.LittleEndian.PutUint16(buf[16:], 8)
	}
	return buf
}

func isWeatherMachine(itemInfo *items.Item, itemID int) bool {
	if itemInfo != nil && (itemInfo.Type == 41 || itemInfo.Type == 81 || itemInfo.Type == 89) {
		return true
	}
	return itemID == 934 || itemID == 940 || itemID == 946 || itemID == 1006 || itemID == 1210 ||
		itemID == 1490 || itemID == 1484 || itemID == 1774 || itemID == 2038 || itemID == 6854 ||
		itemID == 3832 || itemID == 5000 || itemID == 5654 || itemID == 8188 || itemID == 8558
}

func getWeatherIDFromItem(itemID int) int {
	switch itemID {
	case 934: // Night
		return 1
	case 940: // Arid
		return 2
	case 946: // Rain
		return 3
	case 1006: // Snow
		return 4
	case 1210: // Spooky
		return 5
	case 1490: // Nothingness
		return 6
	case 1484: // Undersea
		return 7
	case 1774: // Warp
		return 8
	case 2038: // Comet
		return 9
	case 6854: // Digital Rain
		return 15
	case 3832: // City
		return 17
	case 5000: // Pagoda
		return 23
	case 5654: // Autumn
		return 24
	case 8188: // Spring
		return 27
	case 8558: // Subzero
		return 28
	default:
		return 1
	}
}

func (s *Server) handleWeatherMachineToggle(peer *C.ENetPeer, state *peerState, w *world.World, tileX, tileY, itemID int) {
	if state == nil || state.p == nil || w == nil {
		return
	}
	canEdit := w.CanEditTile(state.p.UserID, tileX, tileY, state.p.CanAccess(role.LevelEliteGuardian))
	if !canEdit {
		s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4You don't have access to this Weather Machine!``").Pack(), true)
		return
	}

	targetWeather := getWeatherIDFromItem(itemID)
	if w.Weather == targetWeather {
		w.Weather = 0
		msg := fmt.Sprintf("`o%s turned the Weather Machine `4OFF``.", state.growID)
		s.broadcastToWorld(w.Name, nil, variant.New("OnConsoleMessage", msg).Pack(), true)
		s.broadcastToWorld(w.Name, nil, variant.New("OnTalkBubble", int32(state.netID), msg).Pack(), true)
	} else {
		w.Weather = targetWeather
		msg := fmt.Sprintf("`o%s turned the Weather Machine `2ON``.", state.growID)
		s.broadcastToWorld(w.Name, nil, variant.New("OnConsoleMessage", msg).Pack(), true)
		s.broadcastToWorld(w.Name, nil, variant.New("OnTalkBubble", int32(state.netID), msg).Pack(), true)
	}

	s.db.SaveWorld(w)
	s.broadcastToWorld(w.Name, nil, variant.New("OnSetCurrentWeather", int32(w.Weather)).Pack(), true)
	s.playWorldSound(w.Name, "audio/weather_switch.wav", float32(tileX*32), float32(tileY*32))
}

func (s *Server) sendWeatherMachineDialog(peer *C.ENetPeer, state *peerState, w *world.World, tileX, tileY, itemID int) {
	canEdit := w.CanEditTile(state.p.UserID, tileX, tileY, state.p.CanAccess(role.LevelEliteGuardian))
	itemName := "Weather Machine"
	if def := items.GetItem(uint16(itemID)); def != nil && def.Name != "" {
		itemName = def.Name
	}

	targetWeather := getWeatherIDFromItem(itemID)
	statusStr := "`4INACTIVE (OFF)``"
	if w.Weather == targetWeather {
		statusStr = "`2ACTIVE (ON)``"
	}

	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", fmt.Sprintf("`w%s``", itemName), itemID)
	d.AddSpacer("small")
	d.EmbedData("tilex", tileX)
	d.EmbedData("tiley", tileY)
	d.EmbedData("item_id", itemID)
	d.AddTextbox(fmt.Sprintf("Current Machine Status: %s", statusStr))
	d.AddSmallText("You can also simply punch this machine to toggle the weather.")
	d.AddSpacer("small")

	if canEdit {
		d.AddButton("btn_toggle_weather", "`wToggle Weather ON/OFF``")
		d.AddButton("btn_retrieve_weather", "`4Retrieve Machine to Backpack``")
	}

	d.AddQuickExit()
	s.sendDialog(peer, d.EndDialog("weather_wrench_dialog", "Close", ""))
}

func (s *Server) handleWeatherDialogReturn(c *Ctx) {
	if c.Player == nil || c.World == nil {
		return
	}
	tileX, tileY := c.Int("tilex"), c.Int("tiley")
	itemID := c.Int("item_id")

	canEdit := c.World.CanEditTile(c.Player.UserID, tileX, tileY, c.Player.CanAccess(role.LevelEliteGuardian))
	if !canEdit {
		c.Error("You don't have access to this Weather Machine!")
		return
	}

	btn := c.Button()
	if btn == "btn_toggle_weather" {
		s.handleWeatherMachineToggle(c.Peer, c.State, c.World, tileX, tileY, itemID)
		s.sendWeatherMachineDialog(c.Peer, c.State, c.World, tileX, tileY, itemID)
		return
	}
	if btn == "btn_retrieve_weather" {
		t := c.World.GetTile(tileX, tileY)
		if t != nil && t.Foreground == itemID {
			t.Foreground = 0
			s.sendTileUpdate(c.World, tileX, tileY, t)
			c.Player.AddItem(itemID, 1)
			if c.World.Weather == getWeatherIDFromItem(itemID) {
				c.World.Weather = 0
				s.broadcastToWorld(c.World.Name, nil, variant.New("OnSetCurrentWeather", int32(0)).Pack(), true)
			}
			c.Save()
			c.SaveWorld()
			c.SyncInventory()
			c.Sound("audio/tile_created.wav")
			c.Success("Retrieved %s to backpack.", items.GetItem(uint16(itemID)).Name)
		}
	}
}

func isRouletteRed(n int) bool {
	switch n {
	case 1, 3, 5, 7, 9, 12, 14, 16, 18, 19, 21, 23, 25, 27, 30, 32, 34, 36:
		return true
	default:
		return false
	}
}

func (s *Server) handleRouletteSpin(peer *C.ENetPeer, state *peerState, w *world.World, tileX, tileY int) {
	// Visual spin animation on tile
	spinTilePkt := buildGamePacket(GamePacket{
		Type:   3, // PACKET_TILE_CHANGE_REQ
		NetID:  int32(state.netID),
		PunchX: int32(tileX),
		PunchY: int32(tileY),
		State:  8,
	}, nil)
	s.broadcastToWorld(w.Name, nil, spinTilePkt, true)
	s.playWorldSound(w.Name, "audio/roulette_spin.wav", float32(tileX*32), float32(tileY*32))

	num := rand.Intn(37) // 0 to 36
	var color string
	if num == 0 {
		color = "`2" // Green
	} else if isRouletteRed(num) {
		color = "`4" // Red
	} else {
		color = "`b" // Dark Blue / Black
	}

	// Authentic GrowTavern text format:
	// "`7[``" + get_player_nick(peer) + " spun the wheel and got " + color + to_string(get37) + "!`7]``"
	wheelMsg := fmt.Sprintf("`7[```w%s`` spun the wheel and got %s%d!`7]``", state.growID, color, num)

	time.AfterFunc(2000*time.Millisecond, func() {
		s.broadcastToWorld(w.Name, nil, variant.New("OnConsoleMessage", wheelMsg).Pack(), true)
		s.broadcastToWorld(w.Name, nil, variant.New("OnTalkBubble", int32(state.netID), wheelMsg).Pack(), true)
	})
}

func (s *Server) handleDiceRoll(peer *C.ENetPeer, state *peerState, w *world.World, tileX, tileY int) {
	rollTilePkt := buildGamePacket(GamePacket{
		Type:   3,
		NetID:  int32(state.netID),
		PunchX: int32(tileX),
		PunchY: int32(tileY),
		State:  8,
	}, nil)
	s.broadcastToWorld(w.Name, nil, rollTilePkt, true)
	s.playWorldSound(w.Name, "audio/dice.wav", float32(tileX*32), float32(tileY*32))

	num := rand.Intn(6) + 1 // 1 to 6
	diceMsg := fmt.Sprintf("`7[```w%s`` rolled a `2%d!`7]``", state.growID, num)

	time.AfterFunc(1500*time.Millisecond, func() {
		s.broadcastToWorld(w.Name, nil, variant.New("OnConsoleMessage", diceMsg).Pack(), true)
		s.broadcastToWorld(w.Name, nil, variant.New("OnTalkBubble", int32(state.netID), diceMsg).Pack(), true)
	})
}

func (s *Server) handleCasinoWrench(peer *C.ENetPeer, state *peerState, w *world.World, tileX, tileY, itemID int) {
	canEdit := w.CanEditTile(state.p.UserID, tileX, tileY, state.p.CanAccess(role.LevelEliteGuardian))
	itemName := "Roulette Wheel"
	if itemID == 756 || itemID == 1360 {
		itemName = "Dice Block"
	}

	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", fmt.Sprintf("`w%s``", itemName), itemID)
	d.AddSpacer("small")
	d.EmbedData("tilex", tileX)
	d.EmbedData("tiley", tileY)
	d.EmbedData("block_id", itemID)

	if itemID == 758 {
		d.AddTextbox("`oPunch this wheel to spin random numbers 0-36 (Green, Red, Black)!``")
	} else {
		d.AddTextbox("`oPunch this block to roll dice 1-6!``")
	}

	if canEdit {
		d.AddSpacer("small")
		d.AddButton("btn_retrieve_casino", "`4Retrieve Block to Backpack``")
	}

	d.AddQuickExit()
	s.sendDialog(peer, d.EndDialog("casino_wrench", "Close", ""))
}

func (s *Server) handleCasinoWrenchReturn(c *Ctx) {
	if c.Player == nil || c.World == nil {
		return
	}
	if c.Button() == "btn_retrieve_casino" {
		tileX, tileY := c.Int("tilex"), c.Int("tiley")
		blockID := c.Int("block_id")
		canEdit := c.World.CanEditTile(c.Player.UserID, tileX, tileY, c.Player.CanAccess(role.LevelEliteGuardian))
		if !canEdit {
			c.Error("You don't have permission to remove this block!")
			return
		}
		t := c.World.GetTile(tileX, tileY)
		if t != nil && t.Foreground == blockID {
			t.Foreground = 0
			s.sendTileUpdate(c.World, tileX, tileY, t)
			c.Player.AddItem(blockID, 1)
			c.Save()
			c.SaveWorld()
			c.SyncInventory()
			c.Sound("audio/tile_created.wav")
			c.Success("Retrieved %s to backpack.", items.GetItem(uint16(blockID)).Name)
		}
	}
}

func isJammerItem(id int) bool {
	return id == 1276 || id == 1278 || id == 226
}

func (s *Server) handleJammerPunchToggle(peer *C.ENetPeer, state *peerState, w *world.World, tileX, tileY, itemID int) {
	if state == nil || state.p == nil || w == nil {
		return
	}
	canEdit := w.CanEditTile(state.p.UserID, tileX, tileY, state.p.CanAccess(role.LevelEliteGuardian))
	if !canEdit {
		s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4You don't have access to this Jammer!``").Pack(), true)
		return
	}

	itemName := "Punch Jammer"
	statusStr := "`4DISABLED``"

	if itemID == 1276 { // Punch Jammer
		w.PunchJammerActive = !w.PunchJammerActive
		if w.PunchJammerActive {
			statusStr = "`2ACTIVE``"
		}
	} else if itemID == 1278 { // Zombie Jammer
		itemName = "Zombie Jammer"
		w.ZombieJammerActive = !w.ZombieJammerActive
		if w.ZombieJammerActive {
			statusStr = "`2ACTIVE``"
		}
	} else if itemID == 226 { // Signal Jammer
		itemName = "Signal Jammer"
		statusStr = "`2ACTIVE``"
	}

	msg := fmt.Sprintf("`5[```w%s`` set the `w%s`` to %s`5]``", state.growID, itemName, statusStr)
	s.broadcastToWorld(w.Name, nil, variant.New("OnConsoleMessage", msg).Pack(), true)
	s.broadcastToWorld(w.Name, nil, variant.New("OnTalkBubble", int32(state.netID), msg).Pack(), true)
	s.playWorldSound(w.Name, "audio/hum.wav", float32(tileX*32), float32(tileY*32))
	s.db.SaveWorld(w)
}

func (s *Server) sendJammerDialog(peer *C.ENetPeer, state *peerState, w *world.World, tileX, tileY, itemID int) {
	canEdit := w.CanEditTile(state.p.UserID, tileX, tileY, state.p.CanAccess(role.LevelEliteGuardian))
	itemName := "Punch Jammer"
	statusStr := "`4DISABLED (OFF)``"

	if itemID == 1276 {
		if w.PunchJammerActive {
			statusStr = "`2ACTIVE (ON)``"
		}
	} else if itemID == 1278 {
		itemName = "Zombie Jammer"
		if w.ZombieJammerActive {
			statusStr = "`2ACTIVE (ON)``"
		}
	} else if itemID == 226 {
		itemName = "Signal Jammer"
		statusStr = "`2ACTIVE (ON)``"
	}

	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", fmt.Sprintf("`w%s``", itemName), itemID)
	d.AddSpacer("small")
	d.EmbedData("tilex", tileX)
	d.EmbedData("tiley", tileY)
	d.EmbedData("item_id", itemID)
	d.AddTextbox(fmt.Sprintf("Current Status: %s", statusStr))
	d.AddSmallText("You can also simply punch this device to toggle its protection.")
	d.AddSpacer("small")

	if canEdit {
		d.AddButton("btn_toggle_jammer", "`wToggle Jammer ON/OFF``")
		d.AddButton("btn_retrieve_jammer", "`4Retrieve Jammer to Backpack``")
	}

	d.AddQuickExit()
	s.sendDialog(peer, d.EndDialog("jammer_wrench_dialog", "Close", ""))
}

func (s *Server) handleJammerDialogReturn(c *Ctx) {
	if c.Player == nil || c.World == nil {
		return
	}
	tileX, tileY := c.Int("tilex"), c.Int("tiley")
	itemID := c.Int("item_id")

	canEdit := c.World.CanEditTile(c.Player.UserID, tileX, tileY, c.Player.CanAccess(role.LevelEliteGuardian))
	if !canEdit {
		c.Error("You don't have access to this Jammer!")
		return
	}

	btn := c.Button()
	if btn == "btn_toggle_jammer" {
		s.handleJammerPunchToggle(c.Peer, c.State, c.World, tileX, tileY, itemID)
		s.sendJammerDialog(c.Peer, c.State, c.World, tileX, tileY, itemID)
		return
	}
	if btn == "btn_retrieve_jammer" {
		t := c.World.GetTile(tileX, tileY)
		if t != nil && t.Foreground == itemID {
			t.Foreground = 0
			s.sendTileUpdate(c.World, tileX, tileY, t)
			c.Player.AddItem(itemID, 1)
			if itemID == 1276 {
				c.World.PunchJammerActive = false
			} else if itemID == 1278 {
				c.World.ZombieJammerActive = false
			}
			c.Save()
			c.SaveWorld()
			c.SyncInventory()
			c.Sound("audio/tile_created.wav")
			c.Success("Retrieved %s to backpack.", items.GetItem(uint16(itemID)).Name)
		}
	}
}

func (s *Server) findPeerAtTile(worldName string, tileX, tileY, excludeNetID int) (*C.ENetPeer, *peerState) {
	s.peersMu.RLock()
	defer s.peersMu.RUnlock()
	for p, st := range s.peers {
		if st != nil && st.currentWorld == worldName && st.netID != excludeNetID && st.p != nil {
			stTileX := int(st.posX / 32)
			stTileY := int(st.posY / 32)
			if stTileX == tileX && stTileY == tileY {
				return p, st
			}
		}
	}
	return nil, nil
}

func (s *Server) handleGeigerExtract(peer *C.ENetPeer, state *peerState, w *world.World, tileX, tileY int) {
	state.p.RemoveItem(2286, 1)
	state.p.AddItem(2204, 1)
	state.p.Clothing[items.ClothHand] = 2204

	roll := rand.Intn(100)
	var rewardID int
	var rewardCount int
	rewardName := "Radioactive Chemical"

	if roll < 60 {
		rewardID = 2288 // Radioactive Chemical
		rewardCount = rand.Intn(3) + 1
		rewardName = fmt.Sprintf("%d Radioactive Chemical", rewardCount)
		state.p.AddItem(rewardID, rewardCount)
	} else if roll < 85 {
		crystals := []int{2244, 2246, 2240} // White, Black, Red Crystal
		rewardID = crystals[rand.Intn(len(crystals))]
		rewardCount = 1
		rewardName = items.GetItem(uint16(rewardID)).Name
		state.p.AddItem(rewardID, rewardCount)
	} else {
		gemAmt := (rand.Intn(3) + 1) * 1000
		state.p.Gems += gemAmt
		s.sendPacket(peer, variant.SetBux(state.p.Gems, 0).Pack(), true)
		rewardName = fmt.Sprintf("%d Gems", gemAmt)
	}

	w.GeigerX = rand.Intn(w.Width-6) + 3
	w.GeigerY = rand.Intn(w.Height-15) + 5

	s.db.SavePlayer(state.p)
	s.db.SaveWorld(w)
	s.sendInventory(peer, state)
	s.broadcastPlayerAppearance(state)

	msg := fmt.Sprintf("`oFound radiation hotspot! `2Received %s``! (Geiger Counter battery depleted)``", rewardName)
	s.broadcastToWorld(w.Name, nil, variant.New("OnConsoleMessage", fmt.Sprintf("`5[GEIGER] `w%s `ofound radiation and received `2%s``!``", state.growID, rewardName)).Pack(), true)
	s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), msg).Pack(), true)
	s.playWorldSound(w.Name, "audio/piano_nice.wav", float32(tileX*32), float32(tileY*32))
}

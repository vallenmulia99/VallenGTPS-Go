package server

/*
#include "../../enet/enet.h"
#include <stdlib.h>
*/
import "C"
import (
	"encoding/binary"
	"fmt"
	"log"
	"strings"
	"time"

	role "gtps/VallenSource/VallenRole"
	variant "gtps/VallenSource/VallenVariant"
	world "gtps/VallenSource/VallenWorld"
)

// ─────────────────────────────────────────────
// enter_game — tampilkan World Select Menu
// ─────────────────────────────────────────────

// handleEnterGame mengirimkan World Select Menu dan welcome packet ke player.
func (s *Server) handleEnterGame(peer *C.ENetPeer) {
	state := s.getPeerState(peer)
	if state == nil {
		return
	}

	state.mu.Lock()
	p := state.p
	growID := state.growID
	netID := state.netID
	state.mu.Unlock()

	if p == nil {
		return
	}

	log.Printf("[Game] enter_game: %s", growID)

	// 1. Welcome messages
	s.sendPacket(peer, variant.New("OnConsoleMessage", fmt.Sprintf("Welcome back, `w%s````. No friends are online.", growID)).Pack(), true)
	s.sendPacket(peer, variant.New("OnConsoleMessage", "`5Personal Settings active:`` `#Can customize profile``").Pack(), true)

	// 2. Inventory state
	invPkt := buildInventoryPacket(p, netID)
	s.sendPacket(peer, invPkt, true)

	// 3. SetBux (gems)
	s.sendPacket(peer, variant.SetBux(p.Gems, 0).Pack(), true)

	// 4. SetHasGrowID
	s.sendPacket(peer, variant.New("SetHasGrowID", int32(1), growID, "").Pack(), true)

	// 5. OnTodaysDate
	now := time.Now()
	s.sendPacket(peer, variant.New("OnTodaysDate", int32(now.Month()), int32(now.Day()), int32(0), int32(0)).Pack(), true)

	// 6. World select menu
	menuPkt := s.buildWorldSelectMenu(state)
	s.sendPacket(peer, menuPkt, true)

	// 7. Console hint
	s.sendPacket(peer, variant.New("OnConsoleMessage", fmt.Sprintf("Where would you like to go? (`w%d`` online)", len(s.peers))).Pack(), true)

	// 8. Ping request
	pingPkt := make([]byte, 60)
	binary.LittleEndian.PutUint32(pingPkt[0:], 4)    // NET_MESSAGE_GAME_PACKET
	binary.LittleEndian.PutUint32(pingPkt[4:], 0x16) // PACKET_PING_REQUEST = 22
	s.sendPacket(peer, pingPkt, true)

	// 9. Feature enable flags
	s.sendPacket(peer, variant.New("OnSetFeatureEnableFlags", "EA8DEAcGAgEOBQgKCQ0MEQQ=").Pack(), true)

	// World select can reset the local avatar preview. Re-send persisted
	// clothing/skin immediately, then it is sent once more after OnSpawn when
	// entering a world.
	s.sendPlayerAppearance(peer, state)

	log.Printf("[Game] Sent World Select Menu to %s", growID)
}

// buildWorldSelectMenu menyusun packet OnRequestWorldSelectMenu.
func (s *Server) buildWorldSelectMenu(state *peerState) []byte {
	recentSection := ""
	if state.p != nil {
		for _, wname := range state.p.RecentWorlds {
			if wname != "" {
				recentSection += fmt.Sprintf("add_floater|%s|0|0.5|3417414143\n", wname)
			}
		}
	}

	menuText := fmt.Sprintf(
		"add_filter|\n"+
			"add_heading|Top Worlds<ROW2>|\n"+
			"add_floater|START|0|0.5|3529161471\n"+
			"add_heading|My Worlds<CR>|\n"+
			"add_heading|Recently Visited Worlds<CR>|\n%s",
		recentSection,
	)

	return variant.New("OnRequestWorldSelectMenu", menuText, int32(1)).Pack()
}

// ─────────────────────────────────────────────
// join_request — masuk ke world tertentu
// ─────────────────────────────────────────────

// handleJoinRequest memproses permintaan masuk ke world.
func (s *Server) handleJoinRequest(peer *C.ENetPeer, pipes []string) {
	state := s.getPeerState(peer)
	if state == nil {
		return
	}

	var worldName string
	for i := 0; i < len(pipes)-1; i++ {
		if pipes[i] == "name" {
			worldName = pipes[i+1]
			break
		}
	}
	worldName = strings.ToUpper(strings.TrimSpace(worldName))
	if worldName == "" {
		worldName = "START"
	}
	if !world.ValidateWorldName(worldName) {
		s.sendPacket(peer, variant.New("OnFailedToEnterWorld").Pack(), true)
		return
	}
	if strings.HasPrefix(worldName, "DUNGEON") {
		run, allowed := s.dungeons.Get(state.growID)
		if !allowed || run.WorldName != worldName {
			s.sendPacket(peer, variant.New("OnFailedToEnterWorld").Pack(), true)
			return
		}
	}

	s.joinWorld(peer, state, worldName, "")
}

// joinWorld joins a specific world, optionally targeting a door by ID
func (s *Server) joinWorld(peer *C.ENetPeer, state *peerState, worldName, doorID string) {
	if state == nil || state.p == nil || !world.ValidateWorldName(worldName) {
		s.sendPacket(peer, variant.New("OnFailedToEnterWorld").Pack(), true)
		return
	}

	state.mu.Lock()
	currentWorld := state.currentWorld
	currentNetID := state.netID
	state.mu.Unlock()

	if currentWorld == worldName {
		if doorID == "" {
			return
		}
		w := s.getOrCreateWorld(worldName)
		door := w.FindDoorByID(doorID)
		if door == nil {
			return
		}
		x, y := float32(door.TileX*32), float32(door.TileY*32)
		state.mu.Lock()
		state.posX, state.posY = x, y
		state.mu.Unlock()
		s.sendPacket(peer, variant.NewWithNetID("OnSetPos", int32(state.netID), variant.Vec2f{X: x, Y: y}).Pack(), true)
		return
	}

	// Remove the old avatar and its visitor slot before the new world map is
	// sent. This is the same leave-before-join lifecycle used by the reference
	// server and prevents ghost players/counts when hopping worlds.
	if currentWorld != "" {
		s.leaveWorld(peer, state, currentWorld, currentNetID)
	}

	log.Printf("[World] %s joining world: %s (doorID: %s)", state.growID, worldName, doorID)

	w := s.getOrCreateWorld(worldName)
	if strings.HasPrefix(worldName, "DUNGEON") {
		w.Weather = 25 // 25 = FIRE_HAZE (DUNGEON WEATHER ID!)
	}

	isPrivileged := state.p != nil && (state.p.AdminLevel >= role.LevelEliteGuardian || s.config.IsOwner(state.growID))
	if w.IsBanned(state.growID) && !isPrivileged {
		s.sendConsole(peer, "`4You are temporarily banned from `w%s``!``", worldName)
		s.handleQuitToExit(peer)
		return
	}

	// 1. Kirim world map data
	mapPkt := w.BuildMapDataPacket()
	s.sendPacket(peer, mapPkt, true)

	// Kirim Packet 15 (NET_GAME_PACKET_SEND_LOCK) untuk semua lock di world agar client merender border
	for i := range w.Locks {
		lck := &w.Locks[i]
		ownerID := uint32(lck.Owner)
		if ownerID == 0 {
			ownerName := lck.OwnerName
			if ownerName == "" && lck.ItemID == world.ItemWorldLock {
				ownerName = w.OwnerName
			}
			ownerID = world.HashGrowID(ownerName)
		}
		lockedTiles := w.GetLockedTiles(lck)
		s.sendPacket(peer, buildLockGamePacket(lck, ownerID, lockedTiles), true)
	}

	w.NetIDCounter++
	netID := w.NetIDCounter

	// Determine spawn position - use door location if doorID specified
	spawnX := int(w.SpawnPixelX())
	spawnY := int(w.SpawnPixelY())
	
	if doorID != "" {
		targetDoor := w.FindDoorByID(doorID)
		if targetDoor != nil {
			spawnX = targetDoor.TileX * 32
			spawnY = targetDoor.TileY * 32
		}
	}

	state.mu.Lock()
	state.netID = netID
	state.currentWorld = worldName
	state.posX = float32(spawnX)
	state.posY = float32(spawnY)
	state.mu.Unlock()

	if state.p != nil {
		// Checkpoints are local to this world; never carry coordinates from a
		// previous world into the respawn handler.
		state.p.RespawnX = float32(spawnX)
		state.p.RespawnY = float32(spawnY)
		addRecentWorld(state.p, worldName)
		s.db.SavePlayer(state.p)
	}

	growID := state.growID

	// 2. Spawn player lain yang sudah ada di world ini ke player yang baru join
	s.peersMu.RLock()
	for otherPeer, otherState := range s.peers {
		if otherPeer == peer || otherState == nil || otherState.currentWorld != worldName || otherState.p == nil {
			continue
		}
		otherRole := role.GetRole(otherState.p.Role)
		otherHasAccess := false
		for _, uid := range w.AccessList {
			if uid != 0 && uid == otherState.p.UserID {
				otherHasAccess = true
				break
			}
		}
		otherName := role.FormatPlayerNameInWorld(otherState.growID, otherState.p.Role, otherState.p.Title, w.OwnerName, otherHasAccess)
		otherMstate, otherSmstate := 0, 0
		if otherRole.IsDeveloper {
			otherSmstate = 1
		} else if otherRole.IsModerator {
			otherMstate = 1
		}

		otherSpawnText := fmt.Sprintf(
			"spawn|avatar\nnetID|%d\nuserID|%d\ncolrect|0|0|20|30\nposXY|%d|%d\nname|%s\ncountry|id\ninvis|0\nmstate|%d\nsmstate|%d\nonlineID|\n",
			otherState.netID, otherState.p.UserID, int(otherState.posX), int(otherState.posY), otherName, otherMstate, otherSmstate,
		)
		// Include clothing fields agar baju/hair player lain terlihat saat join world
		otherSpawnText += avatarSpawnFields(otherState.p)
		s.sendPacket(peer, buildSpawnPacket(otherSpawnText), true)

		// Kirim juga OnSetClothing & characterState player lain ke peer yang baru join
		clothingPkt, statePkt := playerAppearancePackets(otherState, 0)
		if len(clothingPkt) > 0 {
			s.sendPacket(peer, clothingPkt, true)
		}
		if len(statePkt) > 0 {
			s.sendPacket(peer, statePkt, true)
		}
	}
	s.peersMu.RUnlock()

	localRole := role.GetRole(state.p.Role)
	localHasAccess := false
	for _, uid := range w.AccessList {
		if uid != 0 && uid == state.p.UserID {
			localHasAccess = true
			break
		}
	}
	localName := role.FormatPlayerNameInWorld(growID, state.p.Role, state.p.Title, w.OwnerName, localHasAccess)
	localMstate, localSmstate := 0, 0
	if localRole.IsDeveloper {
		localSmstate = 1
	} else if localRole.IsModerator {
		localMstate = 1
	}

	// 3. OnSpawn untuk local player
	// avatarSpawnFields() wajib disertakan agar clothing (hair/shirt/pants/hand dll)
	// tampil ke client saat pertama masuk world (tanpa ini clothing invisible)
	localSpawnText := fmt.Sprintf(
		"spawn|avatar\nnetID|%d\nuserID|%d\ncolrect|0|0|20|30\nposXY|%d|%d\nname|%s\ncountry|id\ninvis|0\nmstate|%d\nsmstate|%d\nonlineID|\ntype|local\n",
		netID, state.p.UserID, spawnX, spawnY, localName, localMstate, localSmstate,
	)
	localSpawnText += avatarSpawnFields(state.p)
	s.sendPacket(peer, buildSpawnPacket(localSpawnText), true)

	// 4. Broadcast spawn player baru ke semua player lain di world (juga include clothing)
	broadcastSpawnText := fmt.Sprintf(
		"spawn|avatar\nnetID|%d\nuserID|%d\ncolrect|0|0|20|30\nposXY|%d|%d\nname|%s\ncountry|id\ninvis|0\nmstate|%d\nsmstate|%d\nonlineID|\n",
		netID, state.p.UserID, spawnX, spawnY, localName, localMstate, localSmstate,
	)
	broadcastSpawnText += avatarSpawnFields(state.p)
	s.broadcastToWorld(worldName, peer, buildSpawnPacket(broadcastSpawnText), true)

	// 5. OnSetPos
	s.sendPacket(peer, variant.NewWithNetID("OnSetPos", int32(netID), variant.Vec2f{X: float32(spawnX), Y: float32(spawnY)}).Pack(), true)
	s.broadcastPlayerAppearance(state)

	// 6. Inventory state & Profile HUD bound to current netID
	if state.p != nil {
		s.sendPacket(peer, buildInventoryPacket(state.p, netID), true)
		s.sendPacket(peer, variant.SetBux(state.p.Gems, 0).Pack(), true)
		s.sendPacket(peer, variant.New("SetHasGrowID", int32(1), growID, "").Pack(), true)
	}

	// 7. Welcome message
	s.sendPacket(peer, variant.New("OnConsoleMessage", fmt.Sprintf("World `w%s`` entered. There are `w%d`` other people here, `w%d`` online.", worldName, w.Visitors, len(s.peers))).Pack(), true)

	// 8. Kirim status ghost aktif di world jika ada
	if ghostJoinPkt := s.ghosts.BuildWorldGhostsJoinPacket(worldName); len(ghostJoinPkt) > 0 {
		s.sendPacket(peer, ghostJoinPkt, true)
	}

	w.Visitors++
	log.Printf("[World] %s successfully entered world %s (spawn: %d, %d)", growID, worldName, spawnX, spawnY)

	// If entering a dungeon world, activate the Dungeon UI immediately
	if strings.HasPrefix(worldName, "DUNGEON") {
		run, ok := s.dungeons.Get(state.growID)
		if !ok {
			run = s.dungeons.Start(state.growID, state.p.UserID)
		}
		s.sendDungeonUI(peer, state, run)
	}
}

// buildSpawnPacket matches the reference OnSpawn header exactly. In
// particular, the packet's delay/id field is 0xFFFFFFFF (-1), not zero. Some
// client versions use this marker while constructing the local world avatar;
// using zero makes them reset appearance to white after world entry.
func buildSpawnPacket(spawnText string) []byte {
	pkt := variant.NewWithNetID("OnSpawn", -1, spawnText)
	pkt.Delay = -1
	return pkt.Pack()
}

// ─────────────────────────────────────────────
// quit_to_exit — keluar dari world
// ─────────────────────────────────────────────

// handleQuitToExit memproses exit player dari world kembali ke World Select Menu.
func (s *Server) handleQuitToExit(peer *C.ENetPeer) {
	state := s.getPeerState(peer)
	if state == nil {
		return
	}

	state.mu.Lock()
	currentWorld, netID := state.currentWorld, state.netID
	state.mu.Unlock()
	s.leaveWorld(peer, state, currentWorld, netID)

	menuPkt := s.buildWorldSelectMenu(state)
	s.sendPacket(peer, menuPkt, true)
	s.sendPacket(peer, variant.New("OnConsoleMessage", "Where would you like to go?").Pack(), true)
}

// leaveWorld performs the one authoritative leave sequence shared by manual
// exit, door travel, world hopping, and disconnect: remove avatar, decrement
// visitors once, then clear the player's current-world state.
func (s *Server) leaveWorld(peer *C.ENetPeer, state *peerState, currentWorld string, netID int) {
	if state == nil || currentWorld == "" {
		return
	}

	s.cancelTrade(netID, fmt.Sprintf("`4Trade canceled: %s left the world.``", state.growID))

	state.mu.Lock()
	if state.currentWorld != currentWorld || state.netID != netID {
		state.mu.Unlock()
		return
	}
	state.currentWorld = ""
	state.mu.Unlock()

	// DungeonWorldUI is client-owned (loaded from the official game assets),
	// so explicitly disable its model before returning to a normal world/menu.
	if run, ok := s.dungeons.Get(state.growID); ok && run.WorldName == currentWorld {
		s.sendPacket(peer, variant.New("OnSetDungeonWorldInfo", int32(0)).Pack(), true)
	}

	// Dungeon state is deliberately session-only. Leaving the private dungeon
	// (including a disconnect) discards its Souls, health, lives and abilities.
	if run, ok := s.dungeons.Get(state.growID); ok && run.WorldName == currentWorld {
		s.dungeons.End(state.growID)
		log.Printf("[Dungeon] Ended run for %s after leaving %s", state.growID, currentWorld)
	}

	s.worldsMu.Lock()
	if w, ok := s.worlds[currentWorld]; ok && w.Visitors > 0 {
		w.Visitors--
	}
	s.worldsMu.Unlock()

	removePkt := variant.New("OnRemove", fmt.Sprintf("netID|%d\n", netID)).Pack()
	s.broadcastToWorld(currentWorld, peer, removePkt, true)
}

// ─────────────────────────────────────────────
// World Management
// ─────────────────────────────────────────────

// getOrCreateWorld mendapatkan world dari cache atau DB, atau membuat baru jika belum ada.
func (s *Server) getOrCreateWorld(name string) *world.World {
	s.worldsMu.Lock()
	defer s.worldsMu.Unlock()

	if w, ok := s.worlds[name]; ok {
		return w
	}

	w, exists := s.db.GetWorld(name)
	if !exists {
		w = world.New(name)
		s.db.SaveWorld(w)
		log.Printf("[World] Created new world in DB: %s", name)
	}
	s.worlds[name] = w
	return w
}

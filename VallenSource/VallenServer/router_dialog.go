package server

/*
#include "../../enet/enet.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	command "gtps/VallenSource/VallenCommand"
	events "gtps/VallenSource/VallenEvents"
	friends "gtps/VallenSource/VallenFriends"
	items "gtps/VallenSource/VallenItems"
	player "gtps/VallenSource/VallenPlayer"
	role "gtps/VallenSource/VallenRole"
	variant "gtps/VallenSource/VallenVariant"
	wardrobe "gtps/VallenSource/VallenWardrobe"
	world "gtps/VallenSource/VallenWorld"
)

// ─────────────────────────────────────────────
// Dialog Return Handlers
// ─────────────────────────────────────────────

// handleDialogReturn menangani packet kembalian dari dialog UI (action|dialog_return)
func (s *Server) handleDialogReturn(c *Ctx) {
	dialogName := c.Str("dialog_name")

	if c.Player == nil {
		return
	}

	log.Printf("[Dialog] dialog_return: %s from %s", dialogName, c.Player.GrowID)

	switch dialogName {
	case "weather_wrench_dialog":
		s.handleWeatherDialogReturn(c)
	case "jammer_wrench_dialog":
		s.handleJammerDialogReturn(c)
	case "change_address_dialog":
		s.handleChangeAddressReturn(c)
	case "casino_wrench":
		s.handleCasinoWrenchReturn(c)
	case "donation_owner":
		s.handleDonationOwnerReturn(c)
	case "donation_visitor":
		if c.Button() == "btn_pick_donation" {
			s.sendDonationPickList(c.Peer, c.State, c.Int("tilex"), c.Int("tiley"))
		}
	case "donation_pick":
		s.handleDonationPickReturn(c)
	case "donation_confirm":
		s.handleDonationConfirmReturn(c)
	case "billboard_edit":
		s.handleBillboardEditReturn(c)
	case "wrench_custom_dialog":
		s.handleWrenchCustomizationReturn(c)
	case "send_pm_dialog":
		s.handlePMSendReturn(c)
	case "report_player_dialog":
		s.handleReportPlayerReturn(c)
	case "vending_owner":
		s.handleVendingOwnerReturn(c.Peer, c.State, c.Values)
	case "vending_deposit":
		s.handleVendingDepositReturn(c.Peer, c.State, c.Values)
	case "vending_buyer":
		s.handleVendingBuyerReturn(c.Peer, c.State, c.Values)
	case "display_box":
		s.handleDisplayBoxReturn(c.Peer, c.State, c.Values)
	case "display_pick":
		s.handleDisplayPickReturn(c.Peer, c.State, c.Values)
	case "trade_invite":
		s.handleTradeInviteResponse(c.Peer, c.State, c.Values)
	case "trade_session":
		s.handleTradeSessionReturn(c.Peer, c.State, c.Values)
	case "trade_add_item":
		s.handleTradeAddItemReturn(c.Peer, c.State, c.Values)
	case "trade_set_count":
		s.handleTradeSetCountReturn(c.Peer, c.State, c.Values)
	case "trade_confirm":
		s.handleTradeConfirmReturn(c.Peer, c.State, c.Values)
	case "dungeon_start_ui":
		s.handleDungeonDialogReturn(c.Peer, c.State, c.Values)
	case "drop_item":
		items.HandleDropDialogReturn(c.V(), s.spawnWorldDrop)
	case "trash_item":
		items.HandleTrashDialogReturn(c.V())
	case "socialportal", "SocialPortal":
		friends.HandleSocialPortalReturn(c.V(), s.isPlayerOnline, s.getPlayerWorld)
	case "friends", "friends_edit", "friends_options", "all_friends", "trade_history", "community_hub", "proxy_menu", "bg_settings", "guild_info":
		friends.HandleFriendsDialogReturn(c.V(), s.isPlayerOnline, s.getPlayerWorld)
	case "grole_edit":
		s.handleGroleSelectionDialogReturn(c.Peer, c.State, c.Values)
	case "grole_confirm":
		s.handleGroleConfirmDialogReturn(c.Peer, c.State, c.Values)
	case "door_edit":
		s.handleDoorEditDialogReturn(c)
	case "sign_edit":
		s.handleSignEditDialogReturn(c)
	case "lock_edit":
		s.handleLockEditDialogReturn(c)
	case "popup":
		s.handlePopupDialogReturn(c)
	case "wrench_social":
		s.handleSocialAction(c.Peer, c.State, c.Str("buttonClicked"), c.Str("netID"))
	case "find_item_dialog":
		s.handleFindItemDialogReturn(c.Peer, c.State, c.Values)
	case "find_give_confirm":
		s.handleFindGiveConfirmReturn(c.Peer, c.State, c.Values)
	case "statsblock":
		s.handleStatsblockDialogReturn(c.Peer, c.State, c.Values)
	case "wardrobe_main_ui", "wardrobe_save_ui", "wardrobe_load_ui":
		wardrobe.HandleDialogReturn(c.V())
	case "mailbox_dialog":
		events.HandleMailboxDialogReturn(c.V())
	case "wl_bank_dialog":
		events.HandleWLBankDialogReturn(c.V())
	case "titles_dialog":
		events.HandleTitlesDialogReturn(c.V())
	case "status_picker_dialog":
		events.HandleOnlineStatusDialogReturn(c.V())
	case "personalize_dialog":
		events.HandlePersonalizeProfileDialogReturn(c.V())
	case "notebook_dialog":
		events.HandleNotebookDialogReturn(c.V())
	case "daily_bonus_dialog":
		events.HandleDailyBonusDialogReturn(c.V())
	case "my_worlds_dialog":
		events.HandleMyWorldsDialogReturn(c.V())
	case "piggy_bank_dialog":
		events.HandlePiggyBankDialogReturn(c.V())
	case "daily_challenge_dialog":
		events.HandleDailyChallengeDialogReturn(c.V())
	case "fruit_mixer_dialog":
		events.HandleFruitMixerDialogReturn(c.V())
	default:
		log.Printf("[Dialog] Unhandled dialog: %s", dialogName)
	}
}

// handleGroleSelectionDialogReturn validates the first page then opens the
// confirmation page. No player data is changed at this point.
func (s *Server) handleGroleSelectionDialogReturn(peer *C.ENetPeer, state *peerState, kv map[string]string) {
	if state.p == nil {
		return
	}
	if strings.EqualFold(kv["buttonClicked"], "cancel") {
		return
	}

	targetGrowID, newAdminLevel, errMsg := command.ReadRoleSelection(
		state.growID,
		state.p.AdminLevel,
		s.config,
		kv,
	)

	if errMsg != "" {
		s.sendPacket(peer, variant.New("OnConsoleMessage", fmt.Sprintf("`4%s``", errMsg)).Pack(), true)
		return
	}

	s.sendPacket(peer, variant.New("OnDialogRequest", command.RenderRoleConfirmDialog(targetGrowID, newAdminLevel)).Pack(), true)
}

// handleGroleConfirmDialogReturn is the only place that persists a role.
// It validates again because dialog return data is client-controlled.
func (s *Server) handleGroleConfirmDialogReturn(peer *C.ENetPeer, state *peerState, kv map[string]string) {
	button := strings.TrimSpace(kv["buttonClicked"])
	// Some client builds submit end_dialog confirmation without buttonClicked.
	// The embedded role data is still present, so only an explicit Cancel may
	// stop the role update.
	if state.p == nil || strings.EqualFold(button, "cancel") {
		return
	}
	log.Printf("[Role] Confirmation from %s: button=%q target=%q level=%q", state.growID, button, kv["target_growid"], kv["role_level"])
	targetGrowID := strings.TrimSpace(kv["target_growid"])
	newAdminLevel, err := strconv.Atoi(kv["role_level"])
	if targetGrowID == "" || err != nil {
		s.sendPacket(peer, variant.New("OnConsoleMessage", "`4Role confirmation data is invalid.``").Pack(), true)
		return
	}
	if errMsg := command.ValidateRoleChange(state.growID, state.p.AdminLevel, s.config, newAdminLevel); errMsg != "" {
		s.sendPacket(peer, variant.New("OnConsoleMessage", fmt.Sprintf("`4%s``", errMsg)).Pack(), true)
		return
	}

	// Cari target player (online atau offline DB)
	var targetPlayer *player.Player
	var targetPeer *C.ENetPeer
	var targetState *peerState

	s.peersMu.RLock()
	for p, st := range s.peers {
		if st != nil && strings.EqualFold(st.growID, targetGrowID) {
			targetPlayer = st.p
			targetPeer = p
			targetState = st
			break
		}
	}
	s.peersMu.RUnlock()

	if targetPlayer == nil {
		p, exists := s.db.GetPlayer(targetGrowID)
		if exists {
			targetPlayer = p
		}
	}

	if targetPlayer == nil {
		s.sendPacket(peer, variant.New("OnConsoleMessage", "`4Target player not found!``").Pack(), true)
		return
	}

	// Update role berdasarkan admin level
	targetRole := role.GetRoleByAdminLevel(newAdminLevel)
	targetPlayer.SetRole(targetRole.ID)
	s.db.SavePlayer(targetPlayer)
	if targetPeer != nil && targetState != nil {
		s.refreshOnlineRoleAvatar(targetPeer, targetState)
	}

	// Notifikasi pengirim
	successMsg := fmt.Sprintf("Role `w%s`` berhasil diubah menjadi %s%s %s%s`` `7(AdminLevel: %d)``!", targetGrowID, targetRole.NameColor, targetRole.Name, targetRole.TagColor, targetRole.Tag, targetRole.AdminLevel)
	s.sendPacket(peer, variant.New("OnConsoleMessage", fmt.Sprintf("`2%s``", successMsg)).Pack(), true)

	// Notifikasi target jika online
	if targetPeer != nil {
		msg := fmt.Sprintf("`2Your role has been updated to %s%s %s%s``!``",
			targetRole.NameColor, targetRole.Name, targetRole.TagColor, targetRole.Tag)
		s.sendPacket(targetPeer, variant.New("OnConsoleMessage", msg).Pack(), true)
	}

	log.Printf("[Role] %s updated %s to role %s (AdminLevel %d)", state.growID, targetGrowID, targetRole.Name, newAdminLevel)
}

// refreshOnlineRoleAvatar makes the changed name, tag, and moderator flags
// visible immediately. OnSpawn is the client packet that owns these fields,
// so a clothing-only update cannot refresh a role badge.
func (s *Server) refreshOnlineRoleAvatar(peer *C.ENetPeer, state *peerState) {
	if peer == nil || state == nil || state.p == nil {
		return
	}

	state.mu.Lock()
	worldName := state.currentWorld
	netID := state.netID
	posX, posY := state.posX, state.posY
	p := state.p
	state.mu.Unlock()
	if worldName == "" || netID <= 0 {
		return
	}

	// Remove the old avatar on every client, then spawn it again with the role
	// fields rebuilt from VallenRole.
	removePkt := variant.New("OnRemove", fmt.Sprintf("netID|%d\n", netID)).Pack()
	s.broadcastToWorld(worldName, nil, removePkt, true)

	roleInfo := role.GetRole(p.Role)
	mstate, smstate := 0, 0
	if roleInfo.IsModerator {
		mstate = 1
	}
	if roleInfo.IsDeveloper {
		smstate = 1
	}
	formattedName := role.FormatPlayerNameWithTitle(state.growID, p.Role, p.Title)

	localSpawn := fmt.Sprintf(
		"spawn|avatar\nnetID|%d\nuserID|%d\ncolrect|0|0|20|30\nposXY|%d|%d\nname|%s\ncountry|id\ninvis|0\nmstate|%d\nsmstate|%d\nonlineID|\ntype|local\n",
		netID, p.UserID, int(posX), int(posY), formattedName, mstate, smstate,
	)
	s.sendPacket(peer, buildSpawnPacket(localSpawn), true)

	remoteSpawn := fmt.Sprintf(
		"spawn|avatar\nnetID|%d\nuserID|%d\ncolrect|0|0|20|30\nposXY|%d|%d\nname|%s\ncountry|id\ninvis|0\nmstate|%d\nsmstate|%d\nonlineID|\n",
		netID, p.UserID, int(posX), int(posY), formattedName, mstate, smstate,
	)
	s.broadcastToWorld(worldName, peer, buildSpawnPacket(remoteSpawn), true)
	s.sendPacket(peer, variant.NewWithNetID("OnSetPos", int32(netID), variant.Vec2f{X: posX, Y: posY}).Pack(), true)
	s.broadcastPlayerAppearance(state)
}

// handleDoorEditDialogReturn memproses perubahan label dan destination door (100% English RGT)
func (s *Server) handleDoorEditDialogReturn(c *Ctx) {
	if c.World == nil {
		return
	}

	tileX, tileY := c.Int("tilex"), c.Int("tiley")
	doorName := c.Str("door_name")
	doorTarget := c.Str("door_target")
	doorID := c.Str("door_id")

	c.World.UpdateDoor(doorName, doorTarget, doorID, tileX, tileY)
	c.SaveWorld()

	if tile := c.World.GetTile(tileX, tileY); tile != nil {
		s.sendTileUpdate(c.World, tileX, tileY, tile)
	}

	c.Success("Door updated! (Label: %s, Dest: %s, ID: %s)", doorName, doorTarget, doorID)
}

// handleSignEditDialogReturn memproses perubahan text sign (100% English RGT)
func (s *Server) handleSignEditDialogReturn(c *Ctx) {
	if c.World == nil {
		return
	}

	tileX, tileY := c.Int("tilex"), c.Int("tiley")
	signText := c.Str("sign_text")

	updated := false
	for i := range c.World.Signs {
		if c.World.Signs[i].TileX == tileX && c.World.Signs[i].TileY == tileY {
			c.World.Signs[i].Label = signText
			updated = true
			break
		}
	}

	if !updated {
		c.World.Signs = append(c.World.Signs, world.Sign{
			Label: signText,
			TileX: tileX,
			TileY: tileY,
		})
	}

	c.SaveWorld()
	c.Success("Sign updated: %s", signText)
}

// handleLockEditDialogReturn memproses setting lock (100% English RGT)
func (s *Server) handleLockEditDialogReturn(c *Ctx) {
	if c.World == nil || c.Player == nil {
		return
	}

	tileX, tileY := c.Int("tilex"), c.Int("tiley")
	lock := c.World.FindLockAt(tileX, tileY)
	if lock == nil {
		c.Error("No lock found at this position!")
		return
	}

	// Check ownership
	if lock.Owner != c.Player.UserID && !s.config.IsOwner(c.Player.GrowID) && !c.Player.CanAccess(role.LevelEliteGuardian) {
		c.Error("You don't own this lock!")
		return
	}

	// Update lock settings
	isPublic := c.Str("checkbox_public") == "1"
	minLvl := c.Int("minimum_entry_level")
	if minLvl < 1 {
		minLvl = 1
	}

	c.World.UpdateLockPublic(tileX, tileY, isPublic)

	// Update world-level settings if this is the World Lock
	if lock.ItemID == world.ItemWorldLock {
		c.World.MinimumEntryLevel = minLvl
		disableMusic := c.Str("checkbox_disable_music") == "1"
		if disableMusic {
			lock.LockState |= world.LockStateDisableMusic
		} else {
			lock.LockState &= ^world.LockStateDisableMusic
		}
		c.World.UpdateLockState(tileX, tileY, lock.LockState)
	}

	// Add player from picker
	if selectedNetID := c.Int("playerNetID"); selectedNetID > 0 {
		s.peersMu.RLock()
		for _, candidate := range s.peers {
			if candidate != nil && candidate.currentWorld == c.World.Name && candidate.netID == selectedNetID && candidate.p != nil {
				if candidate.p.UserID != lock.Owner {
					c.World.AddToAccessList(tileX, tileY, candidate.p.UserID)
				}
				break
			}
		}
		s.peersMu.RUnlock()
	}

	c.SaveWorld()

	if tile := c.World.GetTile(tileX, tileY); tile != nil {
		s.sendTileUpdate(c.World, tileX, tileY, tile)
	}
	s.broadcastLockPacket(c.World, lock, c.Player.GrowID)

	publicStr := "PRIVATE"
	if isPublic {
		publicStr = "PUBLIC"
	}
	c.WorldBroadcast("`2%s`` has set the lock to `$%s``", c.Player.GrowID, publicStr)
}

// handlePopupDialogReturn menangani tombol aksi dari WrenchMenu
func (s *Server) handlePopupDialogReturn(c *Ctx) {
	button := c.Button()

	switch button {
	case "open_worldlock_storage":
		events.SendWLBankDialog(c.V())
		return
	case "wardrobe_customization":
		wardrobe.OpenNative(c.V())
		return
	case "title_edit":
		events.SendTitlesDialog(c.V())
		return
	case "set_online_status":
		events.SendOnlineStatusDialog(c.V())
		return
	case "open_personlize_profile":
		events.SendPersonalizeProfileDialog(c.V())
		return
	case "notebook_edit":
		events.SendNotebookDialog(c.V())
		return
	case "bonus":
		events.SendDailyBonusDialog(c.V())
		return
	case "seed_diary_customization":
		events.SendSeedDiaryDialog(c.V())
		return
	case "alist":
		events.SendAchievementsDialog(c.V())
		return
	case "emojis":
		events.SendGrowmojisDialog(c.V())
		return
	case "my_worlds":
		events.SendMyWorldsDialog(c.V())
		return
	case "goals", "marvelous_missions":
		events.SendDailyChallengeDialog(c.V())
		return
	case "billboard_edit":
		s.sendBillboardEditDialog(c)
		return
	case "wrench_customization":
		s.sendWrenchCustomizationDialog(c)
		return
	case "trade_scan":
		s.sendTradeScanDialog(c)
		return
	case "renew_pvp_license", "pets":
		s.sendBattlePetLicenseDialog(c)
		return
	}

	if _, ok := wrenchPreviewActions[button]; ok {
		s.sendWrenchPreviewDialog(c.Peer, c.State, button)
		return
	}

	if button == "sendpm" {
		netID, _ := strconv.Atoi(c.Str("netID"))
		s.sendPMDialog(c, netID)
		return
	}

	if button == "report_player" {
		netID, _ := strconv.Atoi(c.Str("netID"))
		s.sendReportPlayerDialog(c, netID)
		return
	}

	if button == "friend_add" || button == "ignore_player" || button == "pull_player" || button == "kick_player" || button == "ban_player" {
		s.handleSocialAction(c.Peer, c.State, button, c.Str("netID"))
		return
	}

	if button == "trade" || button == "show_clothes" {
		s.handleWrenchInfoAction(c, button, c.Str("netID"))
		return
	}

	if button == "admin_grole" {
		netIDStr := c.Str("netID")
		if netIDStr != "" {
			netID, _ := strconv.Atoi(netIDStr)
			s.peersMu.RLock()
			var targetName string
			targetAdminLevel := role.LevelPlayer
			for _, st := range s.peers {
				if st != nil && st.netID == netID {
					targetName = st.growID
					if st.p != nil {
						targetAdminLevel = st.p.AdminLevel
					}
					break
				}
			}
			s.peersMu.RUnlock()

			if targetName != "" {
				if c.Player.AdminLevel < role.LevelEliteGuardian && !s.config.IsOwner(c.Player.GrowID) {
					c.Error("You don't have permission to manage roles.")
					return
				}
				c.Dialog(command.RenderRoleDialog(targetName, targetAdminLevel))
			}
		}
	}
}

// handleSocialAction applies the safe, JSON-backed social actions exposed by WrenchMenu.
func (s *Server) handleSocialAction(peer *C.ENetPeer, state *peerState, action, netIDText string) {
	if state == nil || state.p == nil {
		return
	}
	netID, err := strconv.Atoi(netIDText)
	if err != nil {
		return
	}
	var target *player.Player
	s.peersMu.RLock()
	for _, st := range s.peers {
		if st != nil && st.netID == netID && st.currentWorld == state.currentWorld {
			target = st.p
			break
		}
	}
	s.peersMu.RUnlock()
	if target == nil || strings.EqualFold(target.GrowID, state.growID) {
		return
	}
	switch action {
	case "friend_add":
		state.p.SetFriend(target.GrowID, false, false, false)
		s.sendConsole(peer, "`2Friend added.``")
	case "ignore_player":
		state.p.SetFriend(target.GrowID, true, false, false)
		s.sendConsole(peer, "`2Player ignored.``")
	case "report_player":
		s.sendConsole(peer, "`2Report received.``")
	case "pull_player":
		isWorldOwner := false
		if w := s.getOrCreateWorld(state.currentWorld); w != nil && w.Owner != 0 && w.Owner == state.p.UserID {
			isWorldOwner = true
		}
		if !isWorldOwner && state.p.AdminLevel < role.LevelEliteGuardian {
			s.sendConsole(peer, "`4Only world owner or staff can pull players!``")
			return
		}
		var targetPeer *C.ENetPeer
		var targetSt *peerState
		s.peersMu.RLock()
		for p, st := range s.peers {
			if st != nil && st.currentWorld == state.currentWorld && strings.EqualFold(st.growID, target.GrowID) {
				targetPeer = p
				targetSt = st
				break
			}
		}
		s.peersMu.RUnlock()
		if targetSt != nil && targetPeer != nil {
			targetSt.posX = state.posX
			targetSt.posY = state.posY
			s.sendPacket(targetPeer, variant.NewWithNetID("OnSetPos", int32(targetSt.netID), variant.Vec2f{X: state.posX, Y: state.posY}).Pack(), true)
			s.broadcastToWorld(state.currentWorld, nil, variant.New("OnTalkBubble", int32(targetSt.netID), "`5[PULLED]``").Pack(), true)
		}
		return
	case "kick_player":
		isWorldOwner := false
		if w := s.getOrCreateWorld(state.currentWorld); w != nil && w.Owner != 0 && w.Owner == state.p.UserID {
			isWorldOwner = true
		}
		if !isWorldOwner && state.p.AdminLevel < role.LevelEliteGuardian {
			s.sendConsole(peer, "`4Only world owner or staff can kick players!``")
			return
		}
		var targetPeer *C.ENetPeer
		s.peersMu.RLock()
		for p, st := range s.peers {
			if st != nil && st.currentWorld == state.currentWorld && strings.EqualFold(st.growID, target.GrowID) {
				targetPeer = p
				break
			}
		}
		s.peersMu.RUnlock()
		if targetPeer != nil {
			s.handleQuitToExit(targetPeer)
			s.broadcastConsole(state.currentWorld, "`4%s was kicked from this world!``", target.GrowID)
		}
		return
	case "ban_player":
		isWorldOwner := false
		w := s.getOrCreateWorld(state.currentWorld)
		if w != nil && w.Owner != 0 && w.Owner == state.p.UserID {
			isWorldOwner = true
		}
		if !isWorldOwner && state.p.AdminLevel < role.LevelEliteGuardian {
			s.sendConsole(peer, "`4Only world owner or staff can ban players!``")
			return
		}
		if w != nil {
			w.BanUser(target.GrowID, 1*time.Hour)
			s.db.SaveWorld(w)
		}
		var targetPeer *C.ENetPeer
		s.peersMu.RLock()
		for p, st := range s.peers {
			if st != nil && st.currentWorld == state.currentWorld && strings.EqualFold(st.growID, target.GrowID) {
				targetPeer = p
				break
			}
		}
		s.peersMu.RUnlock()
		if targetPeer != nil {
			s.handleQuitToExit(targetPeer)
		}
		s.broadcastConsole(state.currentWorld, "`4%s was banned from this world for 1 hour!``", target.GrowID)
		s.playWorldSound(state.currentWorld, "audio/door_shut.wav", state.posX, state.posY)
		return
	}
	if action != "report_player" {
		s.db.SavePlayer(state.p)
	}
}

func (s *Server) handleWrenchInfoAction(c *Ctx, action, netIDText string) {
	netID, err := strconv.Atoi(netIDText)
	if err != nil {
		return
	}
	var target *player.Player
	s.peersMu.RLock()
	for _, st := range s.peers {
		if st != nil && st.netID == netID && st.currentWorld == c.State.currentWorld {
			target = st.p
			break
		}
	}
	s.peersMu.RUnlock()
	if target == nil || target == c.Player {
		return
	}
	switch action {
	case "trade":
		s.handleTradeInvite(c.Peer, c.State, netID)
	case "show_clothes", "set_online_status":
		s.sendPlayerStatusDialog(c, target)
	}
}

// handleFindItemDialogReturn menangani aksi dari dialog /find (search, navigasi page, klik item).
func (s *Server) handleFindItemDialogReturn(peer *C.ENetPeer, state *peerState, kv map[string]string) {
	if state.p == nil {
		return
	}
	btn := strings.TrimSpace(kv["buttonClicked"])
	if strings.EqualFold(btn, "close") {
		return
	}

	// 1. User klik tombol Cari
	if btn == "find_search_btn" {
		query := strings.TrimSpace(kv["find_query"])
		s.sendPacket(peer, variant.New("OnDialogRequest", command.RenderFindDialog(query, 0)).Pack(), true)
		return
	}

	// 2. User klik tombol pagination (find_page_X)
	if strings.HasPrefix(btn, "find_page_") {
		pageStr := strings.TrimPrefix(btn, "find_page_")
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 0 {
			page = 0
		}
		query := kv["current_query"]
		if query == "" {
			query = kv["find_query"]
		}
		s.sendPacket(peer, variant.New("OnDialogRequest", command.RenderFindDialog(query, page)).Pack(), true)
		return
	}

	// 3. User memilih item tertentu (find_select_X)
	if strings.HasPrefix(btn, "find_select_") {
		idStr := strings.TrimPrefix(btn, "find_select_")
		itemID, err := strconv.Atoi(idStr)
		if err != nil || itemID <= 0 {
			return
		}
		s.sendPacket(peer, variant.New("OnDialogRequest", command.RenderFindGiveDialog(itemID)).Pack(), true)
		return
	}
}

// handleFindGiveConfirmReturn memproses kuantitas item yang diminta player.
func (s *Server) handleFindGiveConfirmReturn(peer *C.ENetPeer, state *peerState, kv map[string]string) {
	if state.p == nil {
		return
	}
	btn := strings.TrimSpace(kv["buttonClicked"])
	if strings.EqualFold(btn, "cancel") {
		return
	}

	// Cek hak akses
	if state.p.AdminLevel < role.LevelEliteGuardian && !s.config.IsOwner(state.growID) {
		s.sendPacket(peer, variant.New("OnConsoleMessage", "`4You do not have permission to get items.``").Pack(), true)
		return
	}

	idStr := strings.TrimSpace(kv["itemID"])
	itemID, err := strconv.Atoi(idStr)
	if err != nil || itemID <= 0 {
		return
	}

	countStr := strings.TrimSpace(kv["count"])
	count, err := strconv.Atoi(countStr)
	if err != nil || count <= 0 {
		count = 200
	}
	if count > 200 {
		count = 200
	}

	overflow := state.p.AddItem(itemID, count)
	s.sendPacket(peer, buildInventoryPacket(state.p, state.netID), true)
	s.db.SavePlayer(state.p)

	added := count - overflow
	item := items.GetItem(uint16(itemID))
	itemName := fmt.Sprintf("Item #%d", itemID)
	if item != nil && item.Name != "" {
		itemName = item.Name
	}

	s.sendPacket(peer, variant.New("OnConsoleMessage", fmt.Sprintf("`2Received `w%d `2of `w%s`2!``", added, itemName)).Pack(), true)
}

// handleStatsblockDialogReturn menangani interaksi dialog /growscan (REMOVED - command deleted)
func (s *Server) handleStatsblockDialogReturn(peer *C.ENetPeer, state *peerState, kv map[string]string) {
	// Command growscan has been removed
	return
}

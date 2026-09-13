package server

/*
#include "../../enet/enet.h"
#include <stdlib.h>
*/
import "C"
import (
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	player "gtps/VallenSource/VallenPlayer"
	role "gtps/VallenSource/VallenRole"
	variant "gtps/VallenSource/VallenVariant"
)

// ─────────────────────────────────────────────
// Auth: protocol (ltoken authentication)
// ─────────────────────────────────────────────

// handleProtocol memproses paket login awal menggunakan ltoken.
func (s *Server) handleProtocol(peer *C.ENetPeer, pipes []string) {
	state := s.getPeerState(peer)
	if state == nil {
		return
	}

	var token string
	for i := 0; i < len(pipes)-1; i++ {
		if pipes[i] == "ltoken" {
			token = pipes[i+1]
			break
		}
	}

	if token == "" {
		s.sendPacket(peer, variant.SendAction("logon_fail", ""), true)
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		log.Printf("[Auth] Failed to decode token: %v", err)
		s.sendPacket(peer, variant.SendAction("logon_fail", ""), true)
		return
	}

	tokenStr := string(decoded)
	tokenKV := parseURLEncoded(tokenStr)

	growID := tokenKV["growId"]
	password := tokenKV["password"]

	if growID == "" || password == "" {
		s.sendPacket(peer, variant.SendAction("logon_fail", ""), true)
		return
	}

	log.Printf("[Auth] Login: %s", growID)

	p, exists := s.db.GetPlayer(growID)
	if !exists {
		// Registrasi player baru (starter items dari setting.json)
		p = player.New(growID, password)
		p.UserID = s.db.NextUserID()
		p.Gems = s.config.Setting.NewPlayer.StartGems
		p.AddItem(32, 1) // Wrench (always required)
		for _, item := range s.config.Setting.NewPlayer.StartItems {
			if item.ID > 0 && item.Count > 0 {
				p.AddItem(item.ID, item.Count)
			}
		}
		s.db.SavePlayer(p)
		log.Printf("[Auth] New player registered: %s (UID: %d) with %d gems", growID, p.UserID, p.Gems)
	} else if p.Password != password {
		log.Printf("[Auth] Invalid password for: %s", growID)
		s.sendPacket(peer, variant.SendAction("logon_fail", ""), true)
		return
	} else {
		// Pastikan player lama juga memiliki Wrench (ID 32)
		if !p.HasItem(32, 1) {
			p.AddItem(32, 1)
			s.db.SavePlayer(p)
		}
	}
	if p.EnsureAppearanceDefaults() {
		s.db.SavePlayer(p)
	}

	// Jika akun termasuk owner di setting.json, otomatis berikan role Monarch (Level 999)
	if s.config.IsOwner(p.GrowID) && p.AdminLevel < role.LevelMonarch {
		p.SetRole(6)
		s.db.SavePlayer(p)
	}

	// Only a successfully authenticated session may replace an existing login.
	s.disconnectOtherSession(peer, growID)

	state.mu.Lock()
	state.p = p
	state.growID = p.GrowID
	state.password = p.Password
	state.mu.Unlock()

	serverHost := s.config.Host
	if serverHost == "0.0.0.0" || serverHost == "" {
		serverHost = s.getPublicHost()
	}

	pkt := variant.New("OnSendToServer",
		s.config.Port,
		int32(0),
		int32(p.UserID),
		fmt.Sprintf("%s|0|0", serverHost),
		int32(1),
		growID,
	).Pack()

	s.sendPacket(peer, pkt, true)
	log.Printf("[Auth] Sent OnSendToServer for %s -> %s:%d", growID, serverHost, s.config.Port)
}

// disconnectOtherSession prevents the same account from controlling one shared
// inventory through two concurrent peers.
func (s *Server) disconnectOtherSession(current *C.ENetPeer, growID string) {
	s.peersMu.RLock()
	type oldSession struct {
		peer  *C.ENetPeer
		state *peerState
	}
	stale := make([]oldSession, 0, 1)
	for peer, state := range s.peers {
		if peer == current || state == nil || !strings.EqualFold(state.growID, growID) {
			continue
		}
		stale = append(stale, oldSession{peer: peer, state: state})
	}
	s.peersMu.RUnlock()

	for _, session := range stale {
		session.state.mu.Lock()
		currentWorld, netID := session.state.currentWorld, session.state.netID
		session.state.mu.Unlock()

		// Remove the old avatar before its network disconnect event arrives. This
		// prevents the same account briefly existing twice in a shared world.
		s.leaveWorld(session.peer, session.state, currentWorld, netID)

		// Detach authenticated state so a queued packet from the old connection
		// cannot mutate the shared player object during handoff.
		session.state.mu.Lock()
		session.state.p = nil
		session.state.growID = ""
		session.state.password = ""
		session.state.mu.Unlock()

		log.Printf("[Auth] Disconnecting previous session for %s", growID)
		C.enet_peer_disconnect_later(session.peer, 0)
	}
}

// ─────────────────────────────────────────────
// Auth: tankIDName (sub-server login)
// ─────────────────────────────────────────────

// handleTankIDName memproses login ke sub-server setelah OnSendToServer.
func (s *Server) handleTankIDName(peer *C.ENetPeer, pipes []string) {
	var growID string
	for i := 0; i < len(pipes)-1; i++ {
		if pipes[i] == "tankIDName" {
			growID = pipes[i+1]
			break
		}
	}

	if growID == "" {
		C.enet_peer_disconnect_later(peer, 0)
		return
	}

	log.Printf("[Game] tankIDName: %s", growID)

	state := s.getPeerState(peer)
	if state == nil {
		return
	}

	p, exists := s.db.GetPlayer(growID)
	if !exists {
		log.Printf("[Game] Player not found: %s", growID)
		C.enet_peer_disconnect_later(peer, 0)
		return
	}
	if p.EnsureAppearanceDefaults() {
		s.db.SavePlayer(p)
	}

	s.netIDCounter++
	netID := s.netIDCounter

	state.mu.Lock()
	state.p = p
	state.growID = p.GrowID
	state.netID = netID
	state.mu.Unlock()

	// 1. OnOverrideGDPRFromServer
	s.sendPacket(peer, variant.New("OnOverrideGDPRFromServer", int32(18), int32(1), int32(0), int32(1)).Pack(), true)

	// 2. Items hash matching client items.dat cache (from setting.json)
	itemsHash := s.config.Setting.Server.ItemsHash

	// 3. OnSuperMainStartAcceptLogonHrdxs47254722215a (driven by setting.json)
	superMainMeta := fmt.Sprintf("proto=225|choosemusic=%s|active_holiday=0|wing_week_day=0|ubi_week_day=0|server_tick=0|game_theme=%s|clash_active=0|drop_lavacheck_faster=1|isPayingUser=1|usingStoreNavigation=1|enableInventoryTab=1|bigBackpack=1|seed_diary_hash=4266294761|m_clientBits=|eventButtons={\"EventButtonData\":[{\"active\":false,\"buttonAction\":\"eventmenu\",\"buttonTemplate\":\"BaseEventButton\",\"counter\":0,\"counterMax\":0,\"itemIdIcon\":6244,\"name\":\"ClashEventButton\",\"order\":9,\"rcssClass\":\"clash-event\",\"text\":\"\"},{\"active\":true,\"buttonAction\":\"dailychallengemenu\",\"buttonTemplate\":\"BaseEventButton\",\"counter\":0,\"counterMax\":0,\"itemIdIcon\":23,\"name\":\"DailyChallenge\",\"order\":10,\"rcssClass\":\"daily_challenge\",\"text\":\"\"},{\"active\":true,\"buttonAction\":\"openPiggyBank\",\"buttonTemplate\":\"BaseEventButton\",\"counter\":0,\"counterMax\":0,\"name\":\"PiggyBankButton\",\"order\":20,\"rcssClass\":\"piggybank\",\"text\":\"\"},{\"active\":false,\"buttonAction\":\"showdungeonsui\",\"buttonTemplate\":\"DungeonEventButton\",\"counter\":0,\"counterMax\":20,\"name\":\"ScrollsPurchaseButton\",\"order\":30,\"rcssClass\":\"scrollbank\",\"text\":\"\"},{\"active\":false,\"buttonAction\":\"show_mailbox_ui\",\"buttonTemplate\":\"BaseEventButton\",\"counter\":0,\"counterMax\":0,\"name\":\"MailboxButton\",\"order\":30,\"rcssClass\":\"mailbox\",\"text\":\"\"},{\"active\":false,\"buttonAction\":\"show_auction_ui\",\"buttonTemplate\":\"BaseEventButton\",\"counter\":0,\"counterMax\":0,\"name\":\"AuctionButton\",\"order\":30,\"rcssClass\":\"auction\",\"text\":\"\"},{\"active\":false,\"buttonTemplate\":\"ActiveAuctionEventButton\",\"counter\":0,\"counterMax\":20,\"name\":\"ActiveAuctionButton\",\"order\":30,\"rcssClass\":\"activeauction\",\"text\":\"\"},{\"active\":false,\"buttonAction\":\"eventmenu\",\"buttonTemplate\":\"BaseEventButton\",\"counter\":0,\"counterMax\":0,\"itemIdIcon\":6244,\"name\":\"ClashEventButton\",\"order\":21,\"rcssClass\":\"clash-event\",\"text\":\"\"},{\"active\":false,\"buttonAction\":\"show_bingo_ui\",\"buttonTemplate\":\"BaseEventButton\",\"counter\":0,\"counterMax\":0,\"name\":\"WinterBingoButton\",\"order\":49,\"rcssClass\":\"wf-bingo\",\"text\":\"\"},{\"active\":false,\"buttonAction\":\"show_bingo_ui\",\"buttonTemplate\":\"BaseEventButton\",\"name\":\"UbiBingoButton\",\"order\":50,\"rcssClass\":\"ubi-bingo\",\"text\":\"\"},{\"active\":false,\"buttonAction\":\"winterrallymenu\",\"buttonTemplate\":\"BaseEventButton\",\"counter\":0,\"counterMax\":0,\"name\":\"WinterRallyButton\",\"order\":50,\"rcssClass\":\"winter-rally\",\"text\":\"\"},{\"active\":false,\"buttonAction\":\"leaderboardBtnClicked\",\"buttonTemplate\":\"BaseEventButton\",\"counter\":0,\"counterMax\":0,\"name\":\"AnniversaryLeaderboardButton\",\"order\":50,\"rcssClass\":\"anniversary-leaderboard\",\"text\":\"\"},{\"active\":false,\"buttonAction\":\"euphoriaBtnClicked\",\"buttonTemplate\":\"BaseEventButton\",\"counter\":0,\"counterMax\":0,\"name\":\"AnniversaryEuphoriaButton\",\"order\":50,\"rcssClass\":\"anniversary-euphoria\",\"text\":\"\"},{\"active\":false,\"buttonAction\":\"openLnySparksPopup\",\"buttonTemplate\":\"BaseEventButton\",\"counter\":0,\"counterMax\":5,\"name\":\"LnyButton\",\"order\":50,\"rcssClass\":\"cny\",\"text\":\"\"},{\"active\":false,\"buttonAction\":\"ShowValentinesQuestDialog\",\"buttonTemplate\":\"EventButtonWithCounter\",\"counter\":0,\"counterMax\":100,\"name\":\"ValentinesButton\",\"order\":50,\"rcssClass\":\"valentines_day\",\"text\":\"\"},{\"active\":false,\"buttonAction\":\"showegseeventui\",\"buttonTemplate\":\"EventButtonWithCounter\",\"counter\":0,\"counterMax\":20,\"name\":\"EasterButton\",\"order\":50,\"rcssClass\":\"easter_event\",\"text\":\"\"},{\"active\":false,\"buttonAction\":\"openStPatrickPiggyBank\",\"buttonTemplate\":\"BaseEventButton\",\"name\":\"StPatrickPBButton\",\"order\":50,\"rcssClass\":\"st_patrick_event\",\"text\":\"\"},{\"active\":false,\"buttonAction\":\"dailyrewardmenu\",\"buttonTemplate\":\"BaseEventButton\",\"counter\":0,\"counterMax\":1,\"name\":\"CincoPinataButton\",\"order\":50,\"rcssClass\":\"cinco_pinata_event\",\"text\":\"\"},{\"active\":false,\"buttonAction\":\"show_fruit_mixer_dialog\",\"buttonTemplate\":\"BaseEventButton\",\"counter\":0,\"counterMax\":0,\"name\":\"SPP_TropicalFruitsButton\",\"order\":50,\"rcssClass\":\"spp_tropical_fruits\",\"text\":\"\"},{\"active\":false,\"buttonAction\":\"claimprogressbar\",\"buttonTemplate\":\"EventButtonWithCounter\",\"counter\":0,\"counterMax\":0,\"name\":\"SummerfestButton\",\"order\":50,\"rcssClass\":\"summerfest\",\"text\":\"\"},{\"active\":false,\"buttonAction\":\"claimprogressbar\",\"buttonTemplate\":\"EventButtonWithCounter\",\"counter\":0,\"counterMax\":15,\"name\":\"HalloweenButton\",\"order\":50,\"rcssClass\":\"halloween\",\"text\":\"\"}]}",
		s.config.Setting.Server.ChooseMusic,
		s.config.Setting.Server.GameTheme,
	)
	// Activate rich event buttons for client HUD (driven by setting.json events)
	superMainMeta = strings.ReplaceAll(superMainMeta, "\"active\":true", "\"active\":false")
	for _, act := range s.config.ActiveEventButtons() {
		superMainMeta = strings.ReplaceAll(
			superMainMeta,
			fmt.Sprintf("\"active\":false,\"buttonAction\":\"%s\"", act),
			fmt.Sprintf("\"active\":true,\"buttonAction\":\"%s\"", act),
		)
	}
	superMain := variant.New(
		"OnSuperMainStartAcceptLogonHrdxs47254722215a",
		itemsHash,
		"ubistatic-a.akamaihd.net",
		"0098/150726456789/cache/",
		"cc.cz.madkite.freedom org.aqua.gg idv.aqua.bulldog com.cih.gamecih2 com.cih.gamecih com.cih.game_cih cn.maocai.gamekiller com.gmd.speedtime org.dax.attack com.x0.strai.frep com.x0.strai.free org.cheatengine.cegui org.sbtools.gamehack com.skgames.traffikrider org.sbtoods.gamehaca com.skype.ralder org.cheatengine.cegui.xx.multi1458919170111 com.prohiro.macro me.autotouch.autotouch com.cygery.repetitouch.free com.cygery.repetitouch.pro com.proziro.zacro com.slash.gamebuster",
		superMainMeta,
	).Pack()
	s.sendPacket(peer, superMain, true)
	// Apply the persisted avatar as soon as the sub-server accepts login. This
	// prevents a new account or a returning account from briefly/defaulting to
	// white before it enters a world.
	s.sendPlayerAppearance(peer, state)

	log.Printf("[Game] Accepted logon for %s (netid=%d, hash=%d)", growID, netID, itemsHash)
}

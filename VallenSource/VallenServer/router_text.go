package server

/*
#include "../../enet/enet.h"
#include <stdlib.h>
*/
import "C"
import (
	"log"
	"strings"

	events "gtps/VallenSource/VallenEvents"
	friends "gtps/VallenSource/VallenFriends"
	items "gtps/VallenSource/VallenItems"
	store "gtps/VallenSource/VallenStore"
	wardrobe "gtps/VallenSource/VallenWardrobe"
)

// ─────────────────────────────────────────────
// Text Packet Handler & Action Router
// ─────────────────────────────────────────────

// handleTextPacket memproses paket teks (type 2/3) dan merutekan ke handler yang tepat.
func (s *Server) handleTextPacket(peer *C.ENetPeer, data []byte) {
	if len(data) < 5 {
		return
	}

	header := strings.Trim(string(data[4:]), "\x00\r\n\t ")
	log.Printf("[Packet] Received: %s", truncate(header, 120))

	// Normalisasi semua jenis line ending menjadi pipe
	flat := strings.ReplaceAll(header, "\r\n", "|")
	flat = strings.ReplaceAll(flat, "\n", "|")
	flat = strings.ReplaceAll(flat, "\r", "|")
	rawPipes := strings.Split(flat, "|")

	var pipes []string
	for _, p := range rawPipes {
		cleaned := strings.Trim(p, "\x00\r\n\t ")
		if cleaned != "" {
			pipes = append(pipes, cleaned)
		}
	}
	if len(pipes) == 0 {
		return
	}

	// Tentukan action dari pipes
	var action string
	if pipes[0] == "protocol" || pipes[0] == "tankIDName" {
		action = pipes[0]
	} else if len(pipes) >= 2 {
		action = pipes[0] + "|" + pipes[1]
	} else {
		action = pipes[0]
	}

	log.Printf("[Action] %s from %s", action, s.getPeerAddress(peer))

	// Buat Context pemalas terpadu untuk action ini
	c := s.NewCtx(peer, action, pipes, nil)

	// Route ke handler yang tepat
	switch {
	case action == "protocol" || action == "action|protocol":
		s.handleProtocol(peer, pipes)

	case action == "tankIDName":
		s.handleTankIDName(peer, pipes)

	case action == "action|enter_game":
		s.handleEnterGame(peer)

	case action == "action|join_request":
		s.handleJoinRequest(peer, pipes)

	case action == "action|respawn" || action == "action|respawn_spike":
		s.handleRespawn(peer)

	case action == "action|setSkin":
		s.handleSetSkin(peer, pipes)

	case strings.HasPrefix(action, "action|quit"):
		s.handleQuitToExit(peer)

	case action == "action|refresh_item_data":
		s.handleRefreshItemData(peer)

	case action == "action|store":
		store.HandleStore(c.V())

	case action == "action|storenavigate":
		store.HandleStoreNavigate(c.V())

	case action == "action|buy":
		store.HandleBuy(c.V())

	case action == "action|killstore":
		// Store dialog closed by client

	case action == "action|drop":
		items.HandleDropRequest(c.V())

	case action == "action|trash":
		items.HandleTrashRequest(c.V())

	case action == "action|info":
		items.HandleItemInfo(c.V())

	case action == "action|itemfavourite":
		items.HandleItemFavourite(c.V())

	case action == "action|friends":
		friends.HandleSocialPortal(c.V())

	case action == "action|startdungeonbtn":
		if c.State != nil {
			s.startDungeonRun(peer, c.State)
		}

	case action == "action|dungeonBackpackClicked" || action == "action|dungeon_ui.backpack_click":
		s.handleDungeonBackpack(peer)

	case strings.HasPrefix(action, "action|wardrobe_"):
		wardrobe.HandleAction(c.V())

	case strings.HasPrefix(action, "action|") && events.IsSuperMainAction(strings.TrimPrefix(action, "action|")):
		events.HandleSuperMainAction(c.V(), strings.TrimPrefix(action, "action|"), func() {
			s.handleDungeonMenu(c.Peer, c.State)
		})

	case action == "action|wrench":
		s.handleWrench(c)

	case action == "action|input":
		s.handleInput(peer, pipes)

	case strings.HasPrefix(action, "action|dialog_return"):
		s.handleDialogReturn(c)

	default:
		log.Printf("[WARN] Unhandled action: %s", action)
	}
}

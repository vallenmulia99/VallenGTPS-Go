package server

/*
#include "../../enet/enet.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"log"
	"strings"

	command "gtps/VallenSource/VallenCommand"
	player "gtps/VallenSource/VallenPlayer"
	role "gtps/VallenSource/VallenRole"
	variant "gtps/VallenSource/VallenVariant"
)

// ─────────────────────────────────────────────
// Chat & Input Handler
// ─────────────────────────────────────────────

// handleInput memproses input teks dari player — chat biasa atau command (/).
func (s *Server) handleInput(peer *C.ENetPeer, pipes []string) {
	var text string
	for i := 0; i < len(pipes)-1; i++ {
		if pipes[i] == "text" {
			text = pipes[i+1]
			break
		}
	}
	if text == "" && len(pipes) > 2 {
		text = pipes[2]
	}
	if text == "" {
		return
	}

	state := s.getPeerState(peer)
	if state == nil {
		return
	}
	growID := state.growID

	// Command dimulai dengan /
	if strings.HasPrefix(text, "/") {
		s.handleCommand(peer, state, text)
		return
	}

	log.Printf("[Chat] %s: %s", growID, text)

	displayName := growID
	if state.p.Title != "" {
		displayName = fmt.Sprintf("%s %s", growID, state.p.Title)
	}

	// Broadcast chat bubble & console ke semua player di world yang sama
	bubbleMsg, consoleMsg := role.FormatChat(displayName, state.p.Role, text)
	bubblePkt := variant.New("OnTalkBubble", int32(state.netID), bubbleMsg, int32(0)).Pack()
	consolePkt := variant.New("OnConsoleMessage", consoleMsg).Pack()

	s.broadcastToWorld(state.currentWorld, nil, bubblePkt, true)
	s.broadcastToWorld(state.currentWorld, nil, consolePkt, true)
}

// ─────────────────────────────────────────────
// Command Dispatcher (Bridge ke VallenCommand)
// ─────────────────────────────────────────────

// handleCommand membangun CommandCtx dan mendispatch ke VallenCommand registry.
func (s *Server) handleCommand(peer *C.ENetPeer, state *peerState, text string) {
	if state.p == nil {
		s.sendPacket(peer, variant.New("OnConsoleMessage", "`4You are not logged in yet.``").Pack(), true)
		return
	}

	// Owner dari setting.json harus dapat membuka semua command walaupun akun tersebut
	// baru dibuat dan AdminLevel yang tersimpan masih 0.
	roleLevel := state.p.AdminLevel
	if s.config.IsOwner(state.growID) {
		roleLevel = role.LevelMonarch
	}

	isWorldOwner := false
	if state.currentWorld != "" {
		if w := s.getOrCreateWorld(state.currentWorld); w != nil {
			if w.Owner != 0 && w.Owner == state.p.UserID {
				isWorldOwner = true
			}
		}
	}

	ctx := &command.Ctx{
		GrowID:       state.growID,
		RoleLevel:    roleLevel,
		WorldName:    state.currentWorld,
		IsWorldOwner: isWorldOwner,
		RawText:      text,
		Send: func(msg string) {
			s.sendPacket(peer, variant.New("OnConsoleMessage", msg).Pack(), true)
		},
		SendDialog: func(dialogStr string) {
			s.sendPacket(peer, variant.New("OnDialogRequest", dialogStr).Pack(), true)
		},
		GetPlayer: func(growID string) (*player.Player, bool) {
			s.peersMu.RLock()
			for _, st := range s.peers {
				if st != nil && strings.EqualFold(st.growID, growID) {
					p := st.p
					s.peersMu.RUnlock()
					return p, true
				}
			}
			s.peersMu.RUnlock()
			return s.db.GetPlayer(growID)
		},
		Warp: func(worldName string) {
			s.joinWorld(peer, state, worldName, "")
		},
		BroadcastWorld: func(msg string) {
			if state.currentWorld != "" {
				s.broadcastToWorld(state.currentWorld, nil, variant.New("OnConsoleMessage", msg).Pack(), true)
			}
		},
		BroadcastAll: func(msg string) {
			s.peersMu.RLock()
			pkt := variant.New("OnConsoleMessage", msg).Pack()
			for p := range s.peers {
				s.sendPacket(p, pkt, true)
			}
			s.peersMu.RUnlock()
		},
		GetOnlinePlayers: func() []string {
			s.peersMu.RLock()
			defer s.peersMu.RUnlock()
			var names []string
			for _, st := range s.peers {
				if st != nil && st.growID != "" {
					names = append(names, st.growID)
				}
			}
			return names
		},
		GetSelfPos: func() (float32, float32) {
			return state.posX, state.posY
		},
		AddSelfItem: func(itemID, count int) int {
			overflow := state.p.AddItem(itemID, count)
			s.sendPacket(peer, buildInventoryPacket(state.p, state.netID), true)
			s.db.SavePlayer(state.p)
			return overflow
		},
		AddSelfGems: func(count int) {
			state.p.Gems += count
			s.sendPacket(peer, variant.SetBux(state.p.Gems, 0).Pack(), true)
			s.db.SavePlayer(state.p)
		},
		RespawnSelf: func() {
			s.handleRespawn(peer)
		},
		PullPlayer: func(targetName string) bool {
			if state.currentWorld == "" {
				return false
			}
			var targetPeer *C.ENetPeer
			var targetSt *peerState
			s.peersMu.RLock()
			for p, st := range s.peers {
				if st != nil && st.currentWorld == state.currentWorld && strings.EqualFold(st.growID, targetName) {
					targetPeer = p
					targetSt = st
					break
				}
			}
			s.peersMu.RUnlock()
			if targetSt == nil || targetPeer == nil {
				return false
			}
			targetSt.posX = state.posX
			targetSt.posY = state.posY
			s.sendPacket(targetPeer, variant.NewWithNetID("OnSetPos", int32(targetSt.netID), variant.Vec2f{X: state.posX, Y: state.posY}).Pack(), true)
			s.broadcastToWorld(state.currentWorld, nil, variant.New("OnTalkBubble", int32(targetSt.netID), "`5[PULLED]``").Pack(), true)
			return true
		},
		KickPlayer: func(targetName string) bool {
			if state.currentWorld == "" {
				return false
			}
			var targetPeer *C.ENetPeer
			var targetSt *peerState
			s.peersMu.RLock()
			for p, st := range s.peers {
				if st != nil && st.currentWorld == state.currentWorld && strings.EqualFold(st.growID, targetName) {
					targetPeer = p
					targetSt = st
					break
				}
			}
			s.peersMu.RUnlock()
			if targetSt == nil || targetPeer == nil {
				return false
			}
			s.handleQuitToExit(targetPeer)
			s.broadcastToWorld(state.currentWorld, nil, variant.New("OnConsoleMessage", fmt.Sprintf("`4%s was kicked from this world!``", targetName)).Pack(), true)
			return true
		},
		SaveWorld: func() {
			if state.currentWorld != "" {
				if w := s.getOrCreateWorld(state.currentWorld); w != nil {
					s.db.SaveWorld(w)
				}
			}
		},
		GetWorld: func() interface{} {
			if state.currentWorld == "" {
				return nil
			}
			return s.getOrCreateWorld(state.currentWorld)
		},
		SetSkinColor: func(skinColor uint32) bool {
			if state.p == nil {
				return false
			}
			state.p.SkinColor = skinColor
			s.db.SavePlayer(state.p)
			if state.currentWorld != "" {
				s.broadcastPlayerAppearance(state)
			} else {
				s.sendPlayerAppearance(peer, state)
			}
			return true
		},
		SetNick: func(nick string) bool {
			if state.p == nil {
				return false
			}
			state.p.Prefix = nick
			s.db.SavePlayer(state.p)
			if state.currentWorld != "" {
				s.broadcastPlayerAppearance(state)
			} else {
				s.sendPlayerAppearance(peer, state)
			}
			return true
		},
	}

	result := command.Dispatch(ctx)

	switch result {
	case command.DispatchNotFound:
		s.sendPacket(peer, variant.New("OnConsoleMessage", "`4Unknown command. Type `5/help`4 for available commands.``").Pack(), true)
	case command.DispatchPermissionDenied:
		s.sendPacket(peer, variant.New("OnConsoleMessage", "`4You don't have permission to use this command.``").Pack(), true)
	case command.DispatchMissingArgs:
		parts := strings.Fields(text)
		if len(parts) > 0 {
			cmdName := strings.TrimPrefix(strings.ToLower(parts[0]), "/")
			if def, ok := command.Get(cmdName); ok {
				s.sendPacket(peer, variant.New("OnConsoleMessage", command.FormatUsage(def.Usage)).Pack(), true)
			}
		}
	}
}

// ─────────────────────────────────────────────
// Helpers — format pesan
// ─────────────────────────────────────────────

// formatOnlineCount mengembalikan string jumlah player online.
func formatOnlineCount(n int) string {
	return fmt.Sprintf("`w%d``", n)
}
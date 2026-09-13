package server

/*
#include "../../enet/enet.h"
*/
import "C"

import (
	"fmt"
	"log"

	dungeon "gtps/VallenSource/VallenDungeon"
	variant "gtps/VallenSource/VallenVariant"
)

func (s *Server) handleDungeonMenu(peer *C.ENetPeer, state *peerState) {
	c := s.NewVallenCtxFromState(peer, state, "action|showdungeonsui", nil, nil)
	dungeon.HandleDungeonMenu(c)
}

func (s *Server) handleDungeonDialogReturn(peer *C.ENetPeer, state *peerState, kv map[string]string) {
	c := s.NewVallenCtxFromState(peer, state, "action|dialog_return", nil, kv)
	dungeon.HandleDungeonDialogReturn(c, s.dungeons, func() {
		s.startDungeonRun(peer, state)
	})
}

func (s *Server) handleDungeonShopBuy(peer *C.ENetPeer, state *peerState, item string) {
	c := s.NewVallenCtxFromState(peer, state, "action|dialog_return", nil, nil)
	dungeon.HandleDungeonShopBuy(c, s.dungeons, item)
}

func (s *Server) startDungeonRun(peer *C.ENetPeer, state *peerState) {
	if existing, ok := s.dungeons.Get(state.growID); ok {
		if existing.WorldName == state.currentWorld {
			s.sendPacket(peer, variant.New("OnConsoleMessage", "`4You are already in a dungeon run.``").Pack(), true)
			return
		}
		s.dungeons.End(state.growID)
	}

	run := s.dungeons.Start(state.growID, state.p.UserID)
	w := s.getOrCreateWorld(run.WorldName)
	dungeon.BuildRoom(w, run)
	s.db.SaveWorld(w)
	s.joinWorld(peer, state, run.WorldName, "")
	s.sendPacket(peer, variant.New("OnConsoleMessage", fmt.Sprintf("`2Dungeon started: Room %d/%d.``", run.Room, dungeon.MaxRooms)).Pack(), true)
	log.Printf("[Dungeon] Started private run for %s in %s", state.growID, run.WorldName)
}

func (s *Server) handleDungeonTileChange(peer *C.ENetPeer, state *peerState, gp GamePacket) bool {
	run, ok := s.dungeons.Get(state.growID)
	if !ok || run.WorldName != state.currentWorld {
		return false
	}
	x, y := int(gp.PunchX), int(gp.PunchY)
	w := s.getOrCreateWorld(state.currentWorld)
	tile := w.GetTile(x, y)
	if tile == nil {
		return true
	}
	if !s.canReachTile(state, x, y) {
		return true
	}
	if gp.ID != 18 {
		s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4Dungeon tiles cannot be edited.``").Pack(), true)
		return true
	}
	if !s.dungeons.IsTarget(state.growID, x, y) {
		s.playWorldSound(state.currentWorld, "audio/punch_locked.wav", float32(x*32), float32(y*32))
		return true
	}

	broken := tile.ApplyDamage(true, 3)
	damagePkt := buildGamePacket(GamePacket{Type: 8, NetID: int32(state.netID), ID: 6, PunchX: int32(x), PunchY: int32(y)}, nil)
	s.broadcastToWorld(state.currentWorld, nil, damagePkt, true)
	s.playWorldPunchSound(state.currentWorld, float32(x*32), float32(y*32))
	if !broken {
		return true
	}
	tile.Foreground = 0
	tile.ResetHits()
	breakPkt := buildGamePacket(GamePacket{
		Type:   3, // PACKET_TILE_CHANGE_REQ
		NetID:  int32(state.netID),
		ID:     18,
		PunchX: int32(x),
		PunchY: int32(y),
	}, nil)
	s.broadcastToWorld(state.currentWorld, nil, breakPkt, true)
	s.sendTileUpdate(w, x, y, tile)
	s.playWorldSound(state.currentWorld, "audio/rock_destroy.wav", float32(x*32), float32(y*32))
	updated, completed, roomClear := s.dungeons.CompleteTarget(state.growID, x, y)
	if completed {
		s.sendDungeonUI(peer, state, updated)
		s.sendPacket(peer, variant.New("OnConsoleMessage", "`2+15 Dungeon Souls.``").Pack(), true)
	}
	if roomClear {
		s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`2Room cleared! Use the NEXT ROOM door.``").Pack(), true)
	}
	return true
}

func (s *Server) handleDungeonDoor(peer *C.ENetPeer, state *peerState, tileX, tileY int) bool {
	run, ok := s.dungeons.Get(state.growID)
	if !ok || run.WorldName != state.currentWorld || tileX != 90 || tileY != 42 {
		return false
	}
	if len(run.Objectives) != 0 {
		s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), "`4Destroy every Trial Stone first.``").Pack(), true)
		s.playWorldSound(state.currentWorld, "audio/punch_locked.wav", float32(tileX*32), float32(tileY*32))
		return true
	}
	next, advanced, finished := s.dungeons.Advance(state.growID)
	if finished {
		s.sendPacket(peer, variant.New("OnDialogRequest", dungeon.FinishDialog(run)).Pack(), true)
		s.dungeons.End(state.growID)
		return true
	}
	if !advanced {
		return true
	}
	w := s.getOrCreateWorld(next.WorldName)
	dungeon.BuildRoom(w, next)
	s.db.SaveWorld(w)
	s.playWorldSound(state.currentWorld, "audio/door_open.wav", float32(tileX*32), float32(tileY*32))
	s.joinWorld(peer, state, next.WorldName, "")
	s.sendDungeonUI(peer, state, next)
	s.sendPacket(peer, variant.New("OnConsoleMessage", fmt.Sprintf("`2Dungeon Room %d/%d started.``", next.Room, dungeon.MaxRooms)).Pack(), true)
	return true
}

func (s *Server) handleDungeonBackpack(peer *C.ENetPeer) {
	state := s.getPeerState(peer)
	c := s.NewVallenCtxFromState(peer, state, "action|dungeon_backpack", nil, nil)
	dungeon.HandleDungeonBackpack(c, s.dungeons)
}

func (s *Server) sendDungeonUI(peer *C.ENetPeer, state *peerState, run *dungeon.Run) {
	if run == nil || state == nil || state.p == nil {
		return
	}

	netID := int32(state.netID)

	// 1. Send Dungeon Event Button dataset (Scrolls button)
	scrollsJson := `{"active":true,"buttonAction":"showdungeonsui","buttonState":0,"buttonTemplate":"DungeonEventButton","counter":20,"counterMax":20,"itemIdIcon":0,"name":"ScrollsPurchaseButton","notification":0,"order":30,"rcssClass":"scrollbank","text":"20/20","visibilityFlag":8}`
	s.sendPacket(peer, variant.New("OnEventButtonDataSet", "ScrollsPurchaseButton", int32(1), scrollsJson).Pack(), true)

	// 2. Send Tank Packet Type 19 for ScrollsPurchaseButton
	pkt19 := buildGamePacket(GamePacket{
		Type:   19,
		NetID:  -1,
		UID:    netID,
		State:  8,
		ID:     461,
		PosX:   295.0,
		PosY:   342.0,
		PunchX: 14730,
		PunchY: 10,
	}, []byte("ScrollsPurchaseButton\x00"))
	s.sendPacket(peer, pkt19, true)

	// 3. XML Bindings & World info
	s.sendPacket(peer, variant.New("OnSetDungeonXMLIndexes", dungeon.NPCIndexes, dungeon.ProjectileIndexes, dungeon.AbilityIndexes).Pack(), true)
	s.sendPacket(peer, variant.New("OnSetDungeonWorldInfo", int32(run.Room)).Pack(), true)
	s.sendPacket(peer, variant.New("OnSetDungeonSouls", int32(run.Souls)).Pack(), true)

	// 4. Mission Timer (5 minutes countdown for the room)
	s.sendPacket(peer, variant.New("OnSetMissionTimer", int32(300)).Pack(), true)

	// 5. TalkBubble & Banner Notification
	s.sendPacket(peer, variant.New("OnTalkBubble", netID, "You have 20 Scrolls remaining.", int32(0), int32(1)).Pack(), true)
	s.sendPacket(peer, variant.New("OnAddNotification", "interface/large/begin.rttex", "Destroy the Monsters!", "").Pack(), true)
	s.sendPacket(peer, variant.New("OnConsoleMessage", "Destroy the Monsters!").Pack(), true)

	log.Printf("[DungeonUI] Activated dungeon UI: world=%s netID=%d room=%d/%d souls=%d", run.WorldName, netID, run.Room, dungeon.MaxRooms, run.Souls)
}

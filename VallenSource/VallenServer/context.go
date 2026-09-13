package server

/*
#include "../../enet/enet.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"strconv"
	"strings"

	vallenctx "gtps/VallenSource/VallenContext"
	player "gtps/VallenSource/VallenPlayer"
	variant "gtps/VallenSource/VallenVariant"
	world "gtps/VallenSource/VallenWorld"
)

// ─────────────────────────────────────────────────────────────────────────────
// Ctx — The Pragmatic / Lazy Developer Context
// ─────────────────────────────────────────────────────────────────────────────
// Ctx membungkus peer, state, player, world, dan parameter input menjadi satu
// objek. Semua method dibuat seringkas mungkin agar developer cukup memanggil
// 1 baris kode tanpa perlu manual packing variant, parsing pipe, atau type conversion.

type Ctx struct {
	Server *Server
	Peer   *C.ENetPeer
	State  *peerState
	Player *player.Player
	World  *world.World
	Action string
	Pipes  []string
	Values map[string]string
	VCtx   *vallenctx.Ctx
}

// NewCtx membuat instance Ctx baru dari peer.
func (s *Server) NewCtx(peer *C.ENetPeer, action string, pipes []string, values map[string]string) *Ctx {
	state := s.getPeerState(peer)
	var p *player.Player
	var w *world.World

	if state != nil {
		p = state.p
		if state.currentWorld != "" {
			w = s.getOrCreateWorld(state.currentWorld)
		}
	}

	if values == nil && len(pipes) > 0 {
		values = parsePipeMap(pipes)
	}
	if values == nil {
		values = make(map[string]string)
	}

	vctx := s.NewVallenCtxFromState(peer, state, action, pipes, values)

	return &Ctx{
		Server: s,
		Peer:   peer,
		State:  state,
		Player: p,
		World:  w,
		Action: action,
		Pipes:  pipes,
		Values: values,
		VCtx:   vctx,
	}
}

// V mengembalikan *vallenctx.Ctx universal untuk dipassing ke domain packages.
func (c *Ctx) V() *vallenctx.Ctx {
	return c.VCtx
}

// NewVallenCtxFromState membuat vallenctx.Ctx universal dari peerState.
func (s *Server) NewVallenCtxFromState(peer *C.ENetPeer, state *peerState, action string, pipes []string, values map[string]string) *vallenctx.Ctx {
	var p *player.Player
	var w *world.World
	netID := 0
	var posX, posY float32

	if state != nil {
		p = state.p
		netID = state.netID
		posX = state.posX
		posY = state.posY
		if state.currentWorld != "" {
			w = s.getOrCreateWorld(state.currentWorld)
		}
	}

	if values == nil && len(pipes) > 0 {
		values = parsePipeMap(pipes)
	}
	if values == nil {
		values = make(map[string]string)
	}

	c := &vallenctx.Ctx{
		Player: p,
		World:  w,
		Action: action,
		Pipes:  pipes,
		Values: values,
		NetID:  netID,
		PosX:   posX,
		PosY:   posY,
		Config: s.config, // inject config reference
	}

	c.SendRawPacket = func(data []byte, reliable bool) {
		s.sendPacket(peer, data, reliable)
	}
	c.BroadcastToWorld = func(worldName string, data []byte, reliable bool) {
		s.broadcastToWorld(worldName, nil, data, reliable)
	}
	c.OnSavePlayer = func() {
		if p != nil {
			s.db.SavePlayer(p)
		}
	}
	c.OnSaveWorld = func() {
		if w != nil {
			s.db.SaveWorld(w)
		}
	}
	c.OnSyncInventory = func() {
		if state != nil && p != nil {
			s.sendInventory(peer, state)
		}
	}
	c.OnJoinWorld = func(worldName, doorID string) {
		if state != nil {
			s.joinWorld(peer, state, worldName, doorID)
		}
	}
	c.OnSendTileUpdate = func(tw *world.World, x, y int, tile *world.Tile) {
		s.sendTileUpdate(tw, x, y, tile)
	}
	c.OnBroadcastState = func() {
		if state != nil {
			s.broadcastPlayerAppearance(state)
		}
	}

	return c
}

// ─────────────────────────────────────────────
// Value Getters (Zero-boilerplate)
// ─────────────────────────────────────────────

// Str mengembalikan nilai string dari Values map, dipotong spasi.
func (c *Ctx) Str(key string) string {
	return strings.TrimSpace(c.Values[key])
}

// Int mengembalikan nilai integer dari Values map. Jika gagal, return 0.
func (c *Ctx) Int(key string) int {
	v, _ := strconv.Atoi(strings.TrimSpace(c.Values[key]))
	return v
}

// Button mengembalikan tombol yang diklik pada dialog return (buttonClicked).
func (c *Ctx) Button() string {
	return c.Str("buttonClicked")
}

// NetID mengembalikan NetID target dari dialog/action (atau netID sendiri jika tidak ada).
func (c *Ctx) NetID() int {
	if n := c.Int("netID"); n > 0 {
		return n
	}
	if n := c.Int("netid"); n > 0 {
		return n
	}
	if c.State != nil {
		return c.State.netID
	}
	return 0
}

// ─────────────────────────────────────────────
// Messaging Shortcuts (100% English RGT Format)
// ─────────────────────────────────────────────

// Console mengirim pesan OnConsoleMessage ke player (otomatis format string).
func (c *Ctx) Console(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	c.Server.sendPacket(c.Peer, variant.New("OnConsoleMessage", msg).Pack(), true)
}

// Error mengirim pesan console dengan format warna merah `4 (Growtopia Error).
func (c *Ctx) Error(format string, a ...interface{}) {
	msg := fmt.Sprintf("`4"+format+"``", a...)
	c.Console("%s", msg)
}

// Success mengirim pesan console dengan format warna hijau `2 (Growtopia Success).
func (c *Ctx) Success(format string, a ...interface{}) {
	msg := fmt.Sprintf("`2"+format+"``", a...)
	c.Console("%s", msg)
}

// Info mengirim pesan console dengan format warna kuning/oranye `9 (Growtopia Info).
func (c *Ctx) Info(format string, a ...interface{}) {
	msg := fmt.Sprintf("`9"+format+"``", a...)
	c.Console("%s", msg)
}

// Overlay mengirim teks banner besar di tengah layar player (OnTextOverlay).
func (c *Ctx) Overlay(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	c.Server.sendPacket(c.Peer, variant.New("OnTextOverlay", msg).Pack(), true)
}

// Bubble menampilkan chat bubble di atas kepala avatar player di world.
func (c *Ctx) Bubble(format string, a ...interface{}) {
	if c.State == nil || c.World == nil {
		return
	}
	msg := fmt.Sprintf(format, a...)
	pkt := variant.New("OnTalkBubble", int32(c.State.netID), msg, int32(0), int32(0)).Pack()
	c.Server.broadcastToWorld(c.World.Name, nil, pkt, true)
}

// Dialog membuka dialog klasik GT (OnDialogRequest) dan membunyikan klik dialog.
func (c *Ctx) Dialog(dialogStr string) {
	if c.Player != nil && c.Player.BorderColor != "" && c.Player.BorderColor != "Default" {
		dialogStr = fmt.Sprintf("set_border_color|%s\nset_bg_color|%s\n%s", c.Player.BorderColor, c.Player.BgColor, dialogStr)
	}
	c.Server.sendPacket(c.Peer, variant.New("OnDialogRequest", dialogStr).Pack(), true)
	c.Sound("audio/dialog_confirm.wav")
}

// RML membuka window RML modern asli Growtopia (OnDialogRequestRML) dan membunyikan klik.
func (c *Ctx) RML(triggerName string) {
	c.Server.sendPacket(c.Peer, variant.New("OnDialogRequestRML", triggerName).Pack(), true)
	c.Sound("audio/dialog_confirm.wav")
}

// Sound memutar efek suara audio di client player.
func (c *Ctx) Sound(filePath string) {
	pkt := variant.SendAction("play_sfx", fmt.Sprintf("file|%s\ndelayMS|0", filePath))
	c.Server.sendPacket(c.Peer, pkt, true)
}

// WorldBroadcast menyiarkan pesan console ke seluruh pemain di world aktif.
func (c *Ctx) WorldBroadcast(format string, a ...interface{}) {
	if c.World == nil {
		return
	}
	msg := fmt.Sprintf(format, a...)
	c.Server.broadcastToWorld(c.World.Name, nil, variant.New("OnConsoleMessage", msg).Pack(), true)
}

// ─────────────────────────────────────────────
// Inventory & Storage Shortcuts (Auto-Sync)
// ─────────────────────────────────────────────

// HasItem mengecek apakah player memiliki sejumlah item tertentu di inventory.
func (c *Ctx) HasItem(itemID, count int) bool {
	if c.Player == nil {
		return false
	}
	return c.Player.HasItem(itemID, count)
}

// AddItem menambahkan item ke tas player, otomatis save DB dan refresh tampilan inventory.
func (c *Ctx) AddItem(itemID, count int) int {
	if c.Player == nil {
		return count
	}
	overflow := c.Player.AddItem(itemID, count)
	c.Save()
	c.SyncInventory()
	return overflow
}

// RemoveItem membuang item dari tas player, otomatis save DB dan refresh tampilan inventory.
func (c *Ctx) RemoveItem(itemID, count int) bool {
	if c.Player == nil {
		return false
	}
	ok := c.Player.RemoveItem(itemID, count)
	if ok {
		c.Save()
		c.SyncInventory()
	}
	return ok
}

// SyncInventory memperbarui tampilan tas/backpack di layar client player.
func (c *Ctx) SyncInventory() {
	if c.Player != nil && c.State != nil {
		c.Server.sendInventory(c.Peer, c.State)
	}
}

// Save menyimpan data akun player ke database JSON.
func (c *Ctx) Save() {
	if c.Player != nil {
		c.Server.db.SavePlayer(c.Player)
	}
}

// SaveWorld menyimpan state world saat ini ke database JSON.
func (c *Ctx) SaveWorld() {
	if c.World != nil {
		c.Server.db.SaveWorld(c.World)
	}
}

// IsWorldOwner memeriksa apakah player adalah pemilik sah dari world aktif.
func (c *Ctx) IsWorldOwner() bool {
	if c.World == nil || c.Player == nil {
		return false
	}
	return c.World.Owner != 0 && c.World.Owner == c.Player.UserID
}

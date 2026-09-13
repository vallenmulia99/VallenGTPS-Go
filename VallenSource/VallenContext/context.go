package vallenctx

import (
	"fmt"
	"strconv"
	"strings"

	player "gtps/VallenSource/VallenPlayer"
	variant "gtps/VallenSource/VallenVariant"
	world "gtps/VallenSource/VallenWorld"
)

// ─────────────────────────────────────────────────────────────────────────────
// Ctx — The Universal Lazy Context
// ─────────────────────────────────────────────────────────────────────────────
// Ctx dirancang murni Go (zero-CGO) dan bebas cyclic dependency.
// Semua folder domain (VallenGhostJar, VallenDungeon, VallenStore, VallenWardrobe, dll)
// dapat mengimpor package ini dan memanggil fungsi-fungsi pemalas dengan mudah.

type Ctx struct {
	Player *player.Player
	World  *world.World
	Action string
	Pipes  []string
	Values map[string]string
	NetID  int
	PosX   float32
	PosY   float32

	// Config reference (read-only access to server settings)
	Config interface{} // points to *config.Config, but kept as interface{} to avoid import cycle

	// Network Callbacks (Diinjeksi oleh VallenServer)
	SendRawPacket     func(data []byte, reliable bool)
	BroadcastToWorld  func(worldName string, data []byte, reliable bool)
	OnSavePlayer      func()
	OnSaveWorld       func()
	OnSyncInventory   func()
	OnJoinWorld       func(worldName, doorID string)
	OnSendTileUpdate  func(w *world.World, x, y int, tile *world.Tile)
	OnBroadcastState  func()
}

// ─────────────────────────────────────────────
// Value Getters (Zero-boilerplate)
// ─────────────────────────────────────────────

// Str mengambil value string dari Values map, dipotong spasi.
func (c *Ctx) Str(key string) string {
	if c.Values == nil {
		return ""
	}
	return strings.TrimSpace(c.Values[key])
}

// Int mengambil value integer dari Values map. Return 0 jika kosong/invalid.
func (c *Ctx) Int(key string) int {
	if c.Values == nil {
		return 0
	}
	v, _ := strconv.Atoi(strings.TrimSpace(c.Values[key]))
	return v
}

// Button mengambil nama tombol yang diklik (buttonClicked).
func (c *Ctx) Button() string {
	return c.Str("buttonClicked")
}

// TargetNetID mengambil NetID target (atau NetID sendiri jika 0).
func (c *Ctx) TargetNetID() int {
	if n := c.Int("netID"); n > 0 {
		return n
	}
	if n := c.Int("netid"); n > 0 {
		return n
	}
	return c.NetID
}

// ─────────────────────────────────────────────
// Messaging Shortcuts (100% English RGT Format)
// ─────────────────────────────────────────────

// Console mengirim pesan OnConsoleMessage (otomatis format string).
func (c *Ctx) Console(format string, a ...interface{}) {
	if c.SendRawPacket == nil {
		return
	}
	msg := fmt.Sprintf(format, a...)
	c.SendRawPacket(variant.New("OnConsoleMessage", msg).Pack(), true)
}

// Success mengirim pesan console berwarna hijau `2.
func (c *Ctx) Success(format string, a ...interface{}) {
	c.Console("`2" + format + "``", a...)
}

// Error mengirim pesan console berwarna merah `4.
func (c *Ctx) Error(format string, a ...interface{}) {
	c.Console("`4" + format + "``", a...)
}

// Info mengirim pesan console berwarna kuning/oranye `9.
func (c *Ctx) Info(format string, a ...interface{}) {
	c.Console("`9" + format + "``", a...)
}

// Overlay mengirim teks banner besar di tengah layar (OnTextOverlay).
func (c *Ctx) Overlay(format string, a ...interface{}) {
	if c.SendRawPacket == nil {
		return
	}
	msg := fmt.Sprintf(format, a...)
	c.SendRawPacket(variant.New("OnTextOverlay", msg).Pack(), true)
}

// Bubble mengirim chat bubble di atas kepala player.
func (c *Ctx) Bubble(format string, a ...interface{}) {
	if c.BroadcastToWorld == nil || c.World == nil {
		return
	}
	msg := fmt.Sprintf(format, a...)
	pkt := variant.New("OnTalkBubble", int32(c.NetID), msg, int32(0), int32(0)).Pack()
	c.BroadcastToWorld(c.World.Name, pkt, true)
}

// Dialog membuka dialog klasik GT dan membunyikan klik dialog.
func (c *Ctx) Dialog(dialogStr string) {
	if c.SendRawPacket == nil {
		return
	}
	if c.Player != nil && c.Player.BorderColor != "" && c.Player.BorderColor != "Default" {
		dialogStr = fmt.Sprintf("set_border_color|%s\nset_bg_color|%s\n%s", c.Player.BorderColor, c.Player.BgColor, dialogStr)
	}
	c.SendRawPacket(variant.New("OnDialogRequest", dialogStr).Pack(), true)
	c.Sound("audio/dialog_confirm.wav")
}

// RML membuka window RML native Growtopia dan membunyikan klik.
func (c *Ctx) RML(triggerName string) {
	if c.SendRawPacket == nil {
		return
	}
	c.SendRawPacket(variant.New("OnDialogRequestRML", triggerName).Pack(), true)
	c.Sound("audio/dialog_confirm.wav")
}

// Sound memutar file suara audio di client player.
func (c *Ctx) Sound(filePath string) {
	if c.SendRawPacket == nil {
		return
	}
	pkt := variant.SendAction("play_sfx", fmt.Sprintf("file|%s\ndelayMS|0", filePath))
	c.SendRawPacket(pkt, true)
}

// WorldBroadcast menyiarkan pesan console ke seluruh pemain di world aktif.
func (c *Ctx) WorldBroadcast(format string, a ...interface{}) {
	if c.BroadcastToWorld == nil || c.World == nil {
		return
	}
	msg := fmt.Sprintf(format, a...)
	c.BroadcastToWorld(c.World.Name, variant.New("OnConsoleMessage", msg).Pack(), true)
}

// ─────────────────────────────────────────────
// Inventory & Storage Shortcuts (Auto-Sync & Auto-Save)
// ─────────────────────────────────────────────

// HasItem mengecek kepemilikan item di inventory.
func (c *Ctx) HasItem(itemID, count int) bool {
	if c.Player == nil {
		return false
	}
	return c.Player.HasItem(itemID, count)
}

// AddItem menambah item ke tas player, otomatis save DB dan refresh tampilan tas di layar.
func (c *Ctx) AddItem(itemID, count int) int {
	if c.Player == nil {
		return count
	}
	overflow := c.Player.AddItem(itemID, count)
	c.Save()
	c.SyncInventory()
	return overflow
}

// RemoveItem membuang item dari tas player, otomatis save DB dan refresh tampilan tas di layar.
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

// SyncInventory menyegarkan tampilan backpack di layar player.
func (c *Ctx) SyncInventory() {
	if c.OnSyncInventory != nil {
		c.OnSyncInventory()
	}
}

// Save menyimpan data akun player ke database JSON.
func (c *Ctx) Save() {
	if c.OnSavePlayer != nil {
		c.OnSavePlayer()
	}
}

// SaveWorld menyimpan state world aktif ke database JSON.
func (c *Ctx) SaveWorld() {
	if c.OnSaveWorld != nil {
		c.OnSaveWorld()
	}
}

// IsWorldOwner mengecek apakah player adalah pemilik sah dari world aktif.
func (c *Ctx) IsWorldOwner() bool {
	if c.World == nil || c.Player == nil {
		return false
	}
	return c.World.Owner != 0 && c.World.Owner == c.Player.UserID
}

// Warp memindahkan player ke world lain.
func (c *Ctx) Warp(worldName string) {
	if c.OnJoinWorld != nil {
		c.OnJoinWorld(worldName, "")
	}
}

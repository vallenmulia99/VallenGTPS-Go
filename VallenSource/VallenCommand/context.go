package command

import (
	"fmt"
	"strconv"
	"strings"

	player "gtps/VallenSource/VallenPlayer"
)

// Ctx adalah konteks eksekusi command gaya "code malas"
type Ctx struct {
	GrowID       string
	RoleLevel    int
	WorldName    string
	IsWorldOwner bool
	Args         []string
	RawText      string

	// Callbacks ke server
	Send             func(msg string)
	SendDialog       func(dialogStr string)
	GetPlayer        func(growID string) (*player.Player, bool)
	Warp             func(worldName string)
	BroadcastWorld   func(msg string)
	BroadcastAll     func(msg string)
	GetOnlinePlayers func() []string
	GetSelfPos       func() (float32, float32)
	AddSelfItem      func(itemID, count int) int
	AddSelfGems      func(count int)
	RespawnSelf      func()
	PullPlayer       func(targetName string) bool
	KickPlayer       func(targetName string) bool
	SaveWorld        func()
	GetWorld         func() interface{}
	SetSkinColor     func(skinColor uint32) bool
	SetNick          func(nick string) bool
}

// ─────────────────────────────────────────────
// Helper Methods Sakti (Code Malas)
// ─────────────────────────────────────────────

// Reply mengirim pesan teks biasa/kustom ke console player
func (c *Ctx) Reply(format string, a ...interface{}) {
	if c.Send != nil {
		c.Send(fmt.Sprintf(format, a...))
	}
}

// Error mengirim pesan error merah (`4...``)
func (c *Ctx) Error(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	c.Reply("`4%s``", msg)
}

// Success mengirim pesan sukses hijau (`2...``)
func (c *Ctx) Success(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	c.Reply("`2%s``", msg)
}

// Info mengirim pesan info kuning/biru (`9...``)
func (c *Ctx) Info(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	c.Reply("`9%s``", msg)
}

// Dialog membuka dialog UI GTPS ke player
func (c *Ctx) Dialog(dialogStr string) {
	if c.SendDialog != nil {
		c.SendDialog(dialogStr)
	}
}

// Arg mengambil argumen ke-i secara aman (tidak akan panic jika index out of range)
func (c *Ctx) Arg(i int) string {
	if i >= 0 && i < len(c.Args) {
		return c.Args[i]
	}
	return ""
}

// Int mengambil argumen ke-i sebagai integer, atau defaultVal jika kosong/gagal parse
func (c *Ctx) Int(i int, defaultVal int) int {
	valStr := c.Arg(i)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

// Float mengambil argumen ke-i sebagai float64, atau defaultVal jika kosong/gagal parse
func (c *Ctx) Float(i int, defaultVal float64) float64 {
	valStr := c.Arg(i)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return defaultVal
	}
	return val
}

// Remaining menggabungkan semua argumen mulai dari startIndex dengan spasi
func (c *Ctx) Remaining(startIndex int) string {
	if startIndex >= len(c.Args) {
		return ""
	}
	return strings.Join(c.Args[startIndex:], " ")
}

// HasArgs mengecek apakah player memberikan minimal n argumen
func (c *Ctx) HasArgs(n int) bool {
	return len(c.Args) >= n
}

// AddItem menambah item ke inventory player
func (c *Ctx) AddItem(itemID, count int) int {
	if c.AddSelfItem != nil {
		return c.AddSelfItem(itemID, count)
	}
	return 0
}

// AddGems menambah gems player
func (c *Ctx) AddGems(count int) {
	if c.AddSelfGems != nil {
		c.AddSelfGems(count)
	}
}

// Respawn merespawn player
func (c *Ctx) Respawn() {
	if c.RespawnSelf != nil {
		c.RespawnSelf()
	}
}

// WarpTo melakukan warp ke world tertentu
func (c *Ctx) WarpTo(worldName string) {
	if c.Warp != nil {
		c.Warp(worldName)
	}
}

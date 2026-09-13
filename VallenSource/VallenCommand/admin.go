package command

import (
	"fmt"
)

func init() {
	// ─────────────────────────────────────────────
	// /find - Cari & Ambil Item via UI Dialog
	// ─────────────────────────────────────────────
	Add("find", LevelMod).
		Alias("itemfind", "searchitem").
		SetUsage("/find [nama_item]").
		SetDesc("Mencari item di database (non-seed) via dialog interaktif").
		Exec(func(c *Ctx) {
			query := c.Remaining(0)
			c.Dialog(RenderFindDialog(query, 0))
		})

	// ─────────────────────────────────────────────
	// /grole - Berikan Role ke Player via UI Dialog
	// ─────────────────────────────────────────────
	Add("grole", LevelMod).
		SetUsage("/grole <nama_player>").
		SetDesc("Membuka UI dialog untuk mengatur role player").
		SetMinArgs(1).
		Exec(func(c *Ctx) {
			targetName := c.Arg(0)
			targetPlayer, exists := c.GetPlayer(targetName)
			if !exists || targetPlayer == nil {
				c.Error("Player '%s' tidak ditemukan!", targetName)
				return
			}
			c.Dialog(RenderRoleDialog(targetName, targetPlayer.AdminLevel))
		})

	// ─────────────────────────────────────────────
	// /give - Memberikan item ke diri sendiri
	// ─────────────────────────────────────────────
	Add("give", LevelAdmin).
		Alias("item").
		SetUsage("/give <item_id> [count]").
		SetDesc("Memberikan item ke diri sendiri").
		SetMinArgs(1).
		Exec(func(c *Ctx) {
			itemID := c.Int(0, 0)
			if itemID <= 0 {
				c.Error("Invalid item ID: %s", c.Arg(0))
				return
			}
			count := c.Int(1, 200)
			if count <= 0 {
				count = 200
			}
			if count > 200 {
				count = 200
			}

			c.AddItem(itemID, count)
			c.Success("Given item ID %d (x%d) to your inventory.", itemID, count)
		})

	// ─────────────────────────────────────────────
	// /gems - Menambah gems ke diri sendiri
	// ─────────────────────────────────────────────
	Add("gems", LevelAdmin).
		SetUsage("/gems <amount>").
		SetDesc("Menambah gems ke akun kamu").
		SetMinArgs(1).
		Exec(func(c *Ctx) {
			amount := c.Int(0, 0)
			if amount <= 0 {
				c.Error("Please specify a valid amount of gems!")
				return
			}
			c.AddGems(amount)
			c.Success("Added %d gems!", amount)
		})

	// ─────────────────────────────────────────────
	// /pull - Menarik player di world yang sama
	// ─────────────────────────────────────────────
	Add("pull", LevelMod).
		SetUsage("/pull <nama_player>").
		SetDesc("Menarik player di world yang sama ke posisi kamu").
		SetMinArgs(1).
		Exec(func(c *Ctx) {
			target := c.Arg(0)
			if c.PullPlayer == nil || !c.PullPlayer(target) {
				c.Error("Player '%s' tidak ditemukan di world ini!", target)
				return
			}
			c.Success("Player '%s' berhasil ditarik ke posisimu.", target)
		})

	// ─────────────────────────────────────────────
	// /kick - Menendang player dari world
	// ─────────────────────────────────────────────
	Add("kick", LevelMod).
		SetUsage("/kick <nama_player>").
		SetDesc("Menendang player dari world saat ini").
		SetMinArgs(1).
		Exec(func(c *Ctx) {
			target := c.Arg(0)
			if c.KickPlayer == nil || !c.KickPlayer(target) {
				c.Error("Player '%s' tidak ditemukan di world ini!", target)
				return
			}
			c.Success("Player '%s' telah ditendang dari world.", target)
		})

	// ─────────────────────────────────────────────
	// /bc - Broadcast pesan ke seluruh server
	// ─────────────────────────────────────────────
	Add("bc", LevelMod).
		Alias("broadcast", "announcement").
		SetUsage("/bc <pesan>").
		SetDesc("Mengirim broadcast ke semua player online").
		SetMinArgs(1).
		Exec(func(c *Ctx) {
			msg := c.Remaining(0)
			if c.BroadcastAll != nil {
				c.BroadcastAll(fmt.Sprintf("`e[BROADCAST] `w%s: `c%s``", c.GrowID, msg))
			}
		})

	// ─────────────────────────────────────────────
	// /save - Menyimpan world saat ini ke database
	// ─────────────────────────────────────────────
	Add("save", LevelAdmin).
		SetDesc("Menyimpan data world saat ini ke database").
		Exec(func(c *Ctx) {
			if c.SaveWorld != nil {
				c.SaveWorld()
				c.Success("World '%s' berhasil disimpan ke database.", c.WorldName)
			}
		})

	// ─────────────────────────────────────────────
	// /nick - Mengatur prefix/title player
	// ─────────────────────────────────────────────
	Add("nick", LevelAdmin).
		SetUsage("/nick <nama/prefix>").
		SetDesc("Mengatur prefix nama karakter kamu").
		SetMinArgs(1).
		Exec(func(c *Ctx) {
			nick := c.Remaining(0)
			if c.SetNick != nil && c.SetNick(nick) {
				c.Success("Prefix nama diubah menjadi: %s", nick)
			}
		})

	// ─────────────────────────────────────────────
	// /skin - Mengatur warna kulit karakter
	// ─────────────────────────────────────────────
	Add("skin", LevelMod).
		SetUsage("/skin <hex_color/decimal>").
		SetDesc("Mengubah warna kulit karakter (misal: 0x884422)").
		SetMinArgs(1).
		Exec(func(c *Ctx) {
			val := uint32(c.Int(0, 0))
			if c.SetSkinColor != nil && c.SetSkinColor(val) {
				c.Success("Skin color updated to: %d", val)
			}
		})
}

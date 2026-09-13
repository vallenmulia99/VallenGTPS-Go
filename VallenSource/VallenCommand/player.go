package command

import (
	"strings"
)

func init() {
	// ─────────────────────────────────────────────
	// /help - Menampilkan daftar command yang tersedia
	// ─────────────────────────────────────────────
	Add("help", LevelPlayer).
		Alias("commands", "cmd").
		SetDesc("Menampilkan daftar perintah yang bisa digunakan").
		Exec(func(c *Ctx) {
			allCmds := GetAll()
			var available []string
			for _, cmd := range allCmds {
				if c.RoleLevel >= cmd.MinLevel {
					available = append(available, "/"+cmd.Name)
				}
			}
			c.Reply("`oAvailable Commands (`2%d``): `w%s``", len(available), strings.Join(available, "`, `w"))
		})

	// ─────────────────────────────────────────────
	// /warp - Teleport ke world lain
	// ─────────────────────────────────────────────
	Add("warp", LevelPlayer).
		Alias("join", "go").
		SetUsage("/warp <world_name>").
		SetDesc("Pergi ke world tujuan").
		SetMinArgs(1).
		Exec(func(c *Ctx) {
			world := strings.ToUpper(strings.TrimSpace(c.Arg(0)))
			if world == "" {
				c.Error("Please specify a world name!")
				return
			}
			c.WarpTo(world)
		})

	// ─────────────────────────────────────────────
	// /respawn - Respawn karakter ke white door
	// ─────────────────────────────────────────────
	Add("respawn", LevelPlayer).
		Alias("kill", "suicide").
		SetDesc("Respawn karakter kamu kembali ke White Door").
		Exec(func(c *Ctx) {
			c.Respawn()
			c.Success("Respawned!")
		})

	// ─────────────────────────────────────────────
	// /online - Melihat jumlah dan daftar player online
	// ─────────────────────────────────────────────
	Add("online", LevelPlayer).
		Alias("who").
		SetDesc("Melihat siapa saja yang sedang online").
		Exec(func(c *Ctx) {
			if c.GetOnlinePlayers == nil {
				return
			}
			players := c.GetOnlinePlayers()
			c.Reply("`oPlayers Online (`2%d``): `w%s``", len(players), strings.Join(players, "`, `w"))
		})

	// ─────────────────────────────────────────────
	// /pos - Melihat posisi koordinat kamu
	// ─────────────────────────────────────────────
	Add("pos", LevelPlayer).
		SetDesc("Melihat koordinat X dan Y kamu di world saat ini").
		Exec(func(c *Ctx) {
			if c.GetSelfPos == nil {
				return
			}
			x, y := c.GetSelfPos()
			c.Info("Current Position: X: `w%.1f``, Y: `w%.1f`` in `w%s``", x, y, c.WorldName)
		})
}

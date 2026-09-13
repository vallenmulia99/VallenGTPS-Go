package ghostjar

import (
	vallenctx "gtps/VallenSource/VallenContext"
)

// ─────────────────────────────────────────────────────────────────────────────
// Ghost Jar Handlers (Domain: VallenGhostJar)
// ─────────────────────────────────────────────────────────────────────────────

// HandleGhostJarItem memproses penempatan toples kosong (perangkap) atau pelepasan hantu.
func HandleGhostJarItem(c *vallenctx.Ctx, tileX, tileY, itemID int, gm *GhostManager) bool {
	if c.Player == nil || c.World == nil || gm == nil {
		return false
	}

	switch itemID {
	case ItemGhostInJar, ItemShadowGhostInJar:
		// Melepas Ghost dari dalam toples
		ghostType := uint8(TypeGhostNormal)
		if itemID == ItemShadowGhostInJar {
			ghostType = uint8(TypeGhostShadow)
		}

		pixelX := float32(tileX*32 + 16)
		pixelY := float32(tileY*32 + 16)

		ghost := gm.SpawnGhost(c.World.Name, ghostType, pixelX, pixelY, c.NetID)
		c.BroadcastToWorld(c.World.Name, BuildGhostUpdatePacket(ghost), true)
		c.Sound("audio/punch_glass.wav")

		// Kurangi toples hantu dari tas (otomatis sync tas & save DB)
		c.RemoveItem(itemID, 1)
		return true

	case ItemGhostJar:
		// Menaruh Empty Jar di tanah sebagai jebakan (Trap)
		tile := c.World.GetTile(tileX, tileY)
		tileBelow := c.World.GetTile(tileX, tileY+1)

		// Syarat: tile kosong (fg == 0) dan di bawahnya ada pijakan blok (fg != 0)
		if tile == nil || tileBelow == nil || tile.Foreground != 0 || tileBelow.Foreground == 0 {
			c.Bubble("Put the jar on the ground!")
			return true
		}

		pixelX := float32(tileX*32 + 16)
		pixelY := float32(tileY*32 + 16)

		ghost := gm.SpawnGhost(c.World.Name, TypeJarTrap, pixelX, pixelY, c.NetID)
		c.BroadcastToWorld(c.World.Name, BuildGhostUpdatePacket(ghost), true)

		// Kurangi toples kosong dari tas (otomatis sync tas & save DB)
		c.RemoveItem(itemID, 1)
		return true
	}

	return false
}

// HandleProtonPackPunch memproses tembakan laser senjata Proton Pack ke arah hantu.
func HandleProtonPackPunch(c *vallenctx.Ctx, punchX, punchY int, gm *GhostManager) bool {
	if c.Player == nil || c.World == nil || gm == nil {
		return false
	}

	targetX := float64(punchX*32 + 16)
	targetY := float64(punchY*32 + 16)

	affected, hit := gm.HandleProtonPunch(c.World.Name, float64(c.PosX), float64(c.PosY), targetX, targetY)
	if hit {
		for _, g := range affected {
			c.BroadcastToWorld(c.World.Name, BuildGhostUpdatePacket(g), true)
		}
	}

	// Kirim paket visual respon tembakan proton pack (Packet Type 21)
	c.SendRawPacket(BuildPunchAckPacket(), true)
	return true
}

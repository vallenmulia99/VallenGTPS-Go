package wardrobe

import (
	vallenctx "gtps/VallenSource/VallenContext"
	variant "gtps/VallenSource/VallenVariant"
)

// ─────────────────────────────────────────────────────────────────────────────
// Wardrobe Native Handlers (Domain: VallenWardrobe)
// ─────────────────────────────────────────────────────────────────────────────

// OpenNative memicu window Wardrobe asli Growtopia (WardrobeMain.rml).
func OpenNative(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	c.RML("show_wardrobe_main_ui")
}

// HandleAction memproses aksi dari tombol window RML Wardrobe.
func HandleAction(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	actionName := c.Str("action")
	preset := c.Str("preset")

	switch actionName {
	case "wardrobe_unequip_all":
		for i := range c.Player.Clothing {
			c.Player.Clothing[i] = 0
		}
		c.Save()
		if c.OnBroadcastState != nil {
			c.OnBroadcastState()
		}
		c.Success("All clothing unequipped!")

	case "wardrobe_save_ui":
		if preset == "two" {
			c.Player.Outfit2 = c.Player.Clothing
			c.Save()
			SendPresetUpdate(c, 1, c.Player.Outfit2)
			c.Success("Saved current outfit to Preset 2!")
		} else {
			c.Player.Outfit1 = c.Player.Clothing
			c.Save()
			SendPresetUpdate(c, 0, c.Player.Outfit1)
			c.Success("Saved current outfit to Preset 1!")
		}

	case "wardrobe_load_ui":
		SendPresetUpdate(c, 0, c.Player.Outfit1)
		SendPresetUpdate(c, 1, c.Player.Outfit2)

	case "wardrobe_preset_load":
		if preset == "two" {
			ApplyPreset(c, c.Player.Outfit2, 2)
		} else {
			ApplyPreset(c, c.Player.Outfit1, 1)
		}
	}
}

// SendPresetUpdate menyinkronkan visual data preset ke window client.
func SendPresetUpdate(c *vallenctx.Ctx, presetIdx int, outfit [10]int) {
	if c.Player == nil || c.SendRawPacket == nil {
		return
	}
	args := []interface{}{
		"OnWardrobePresetUpdate",
		int32(c.NetID),
		int32(presetIdx),
	}
	for _, v := range outfit {
		args = append(args, int32(v))
	}
	c.SendRawPacket(variant.New(args[0].(string), args[1:]...).Pack(), true)
}

// ApplyPreset memasang setelan pakaian dari preset terpilih ke avatar player.
func ApplyPreset(c *vallenctx.Ctx, preset [10]int, num int) {
	equippedCount := 0
	for slot, itemID := range preset {
		if itemID > 0 && (c.HasItem(itemID, 1) || c.Player.Clothing[slot] == itemID) {
			c.Player.Clothing[slot] = itemID
			equippedCount++
		} else {
			c.Player.Clothing[slot] = 0
		}
	}
	c.Save()
	if c.OnBroadcastState != nil {
		c.OnBroadcastState()
	}
	c.Success("Equipped Outfit %d (%d items worn)!", num, equippedCount)
}

// HandleDialogReturn fallback jika client versi lama mengirim via dialog_return.
func HandleDialogReturn(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	btn := c.Button()
	switch btn {
	case "wardrobe_unequip_all":
		for i := range c.Player.Clothing {
			c.Player.Clothing[i] = 0
		}
		c.Save()
		if c.OnBroadcastState != nil {
			c.OnBroadcastState()
		}
		c.Success("All clothing unequipped!")
	case "wardrobe_do_save_1":
		c.Player.Outfit1 = c.Player.Clothing
		c.Save()
		SendPresetUpdate(c, 0, c.Player.Outfit1)
		c.Success("Saved current outfit to Preset 1!")
	case "wardrobe_do_save_2":
		c.Player.Outfit2 = c.Player.Clothing
		c.Save()
		SendPresetUpdate(c, 1, c.Player.Outfit2)
		c.Success("Saved current outfit to Preset 2!")
	case "wardrobe_do_equip_1":
		ApplyPreset(c, c.Player.Outfit1, 1)
	case "wardrobe_do_equip_2":
		ApplyPreset(c, c.Player.Outfit2, 2)
	}
}

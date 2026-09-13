package items

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"

	vallenctx "gtps/VallenSource/VallenContext"
	variant "gtps/VallenSource/VallenVariant"
	world "gtps/VallenSource/VallenWorld"
)

const maxFavoriteItems = 20

// ─────────────────────────────────────────────────────────────────────────────
// 1. DROP ITEM (action|drop & drop_item dialog)
// ─────────────────────────────────────────────────────────────────────────────

func HandleDropRequest(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	itemID := c.Int("itemID")
	if itemID <= 0 || !c.HasItem(itemID, 1) {
		return
	}

	item := GetItem(uint16(itemID))
	if item != nil && item.Category&CatUntradeable != 0 {
		c.Overlay("You can't drop that.")
		return
	}

	name := fmt.Sprintf("Item #%d", itemID)
	if item != nil && item.Name != "" {
		name = item.Name
	}
	owned := c.Player.GetItemCount(itemID)

	dialog := fmt.Sprintf(
		"set_default_color|`o\n"+
			"add_label_with_icon|big|`wDrop %s``|left|%d|\n"+
			"add_textbox|How many to `4drop``? (you have %d)|left|\n"+
			"add_text_input|count||%d|5|\n"+
			"embed_data|itemID|%d\n"+
			"end_dialog|drop_item|Cancel|OK|\n",
		strings.ReplaceAll(name, "|", ""), itemID, owned, owned, itemID,
	)
	c.Dialog(dialog)
}

func HandleDropDialogReturn(c *vallenctx.Ctx, onSpawnDrop func(w *world.World, itemID, count int, x, y float32, dropperNetID int)) {
	if c.Player == nil || c.World == nil {
		return
	}
	if strings.EqualFold(c.Button(), "cancel") {
		return
	}

	itemID := c.Int("itemID")
	count := c.Int("count")
	if itemID <= 0 || count <= 0 {
		return
	}

	owned := c.Player.GetItemCount(itemID)
	if owned == 0 {
		return
	}
	if count > owned {
		count = owned
	}

	item := GetItem(uint16(itemID))
	if item != nil && item.Category&CatUntradeable != 0 {
		c.Overlay("You can't drop that.")
		return
	}

	scatter := float32(rand.Intn(12))
	var dropX float32
	if c.Player.FacingLeft {
		dropX = c.PosX - (scatter + 20)
	} else {
		dropX = c.PosX + 20 + scatter
	}
	dropY := c.PosY + float32(rand.Intn(16))

	worldMaxX := float32(c.World.Width * 32)
	worldMaxY := float32(c.World.Height * 32)
	if dropX < 0 || dropX >= worldMaxX || dropY < 0 || dropY >= worldMaxY {
		c.Overlay("You can't drop that here, face somewhere with open space.")
		return
	}

	dropTile := c.World.GetTile(int(dropX/32), int(dropY/32))
	if dropTile == nil {
		c.Overlay("You can't drop that here, face somewhere with open space.")
		return
	}
	if dropTile.Foreground != 0 {
		fgItem := GetItem(uint16(dropTile.Foreground))
		if dropTile.Foreground == 6 || world.IsMainDoor(dropTile.Foreground) {
			c.Overlay("You can't drop items on the white door.")
			return
		}
		if fgItem != nil && (fgItem.Collision == 1 || fgItem.Type == TypeDoor) {
			c.Overlay("You can't drop that here, face somewhere with open space.")
			return
		}
	}

	spotCount := 0
	for _, obj := range c.World.DroppedItems {
		if math.Abs(float64(obj.X-dropX)) <= 16 && math.Abs(float64(obj.Y-dropY)) <= 16 {
			spotCount++
		}
	}
	if spotCount > 20 {
		c.Overlay("You can't drop that here, find an emptier spot!")
		return
	}

	if !c.RemoveItem(itemID, count) {
		return
	}

	if onSpawnDrop != nil {
		onSpawnDrop(c.World, itemID, count, dropX, dropY, c.NetID)
	}
	c.Sound("audio/object_spawn.wav")
	c.SaveWorld()

	log.Printf("[Drop] %s dropped %d of item %d at %.1f,%.1f", c.Player.GrowID, count, itemID, dropX, dropY)
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. TRASH & RECYCLE (action|trash & trash_item dialog)
// ─────────────────────────────────────────────────────────────────────────────

func HandleTrashRequest(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	itemID := c.Int("itemID")
	if itemID <= 0 || !c.HasItem(itemID, 1) {
		return
	}

	item := GetItem(uint16(itemID))
	if item != nil && (item.Type == TypeFist || item.Type == TypeWrench) {
		c.Overlay("You'd be sorry if you lost that!")
		return
	}

	name := fmt.Sprintf("Item #%d", itemID)
	if item != nil && item.Name != "" {
		name = item.Name
	}
	owned := c.Player.GetItemCount(itemID)
	recycleGems := 0
	if item != nil && item.Rarity > 0 && item.Rarity < 999 {
		recycleGems = int(item.Rarity) / 2
		if recycleGems < 1 {
			recycleGems = 1
		}
	}

	dialog := fmt.Sprintf(
		"set_default_color|`o\n"+
			"add_label_with_icon|big|`4Recycle`` `w%s``|left|%d|\n"+
			"add_textbox|How many to `4recycle``? (you have %d)|left|\n"+
			"add_text_input|count||0|5|\n"+
			"add_smalltext|`4Warning: This will permanently delete this item!``|left|\n"+
			"add_smalltext|`2You will receive `$%d Gems `2per item recycled.``|left|\n"+
			"embed_data|itemID|%d\n"+
			"end_dialog|trash_item|Cancel|OK|\n",
		strings.ReplaceAll(name, "|", ""), itemID, owned, recycleGems, itemID,
	)
	c.Dialog(dialog)
}

func HandleTrashDialogReturn(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	if strings.EqualFold(c.Button(), "cancel") {
		return
	}

	itemID := c.Int("itemID")
	count := c.Int("count")
	if itemID <= 0 || count <= 0 || !c.HasItem(itemID, 1) {
		return
	}

	item := GetItem(uint16(itemID))
	if item != nil && (item.Type == TypeFist || item.Type == TypeWrench) {
		return
	}

	if owned := c.Player.GetItemCount(itemID); count > owned {
		count = owned
	}

	if !c.RemoveItem(itemID, count) {
		return
	}

	// Clean up favorites
	if !c.Player.HasItem(itemID, 1) {
		for i, fav := range c.Player.Favorites {
			if fav == itemID {
				c.Player.Favorites = append(c.Player.Favorites[:i], c.Player.Favorites[i+1:]...)
				break
			}
		}
	}

	gemsGained := 0
	if item != nil && item.Rarity > 0 && item.Rarity < 999 {
		perItem := int(item.Rarity) / 2
		if perItem < 1 {
			perItem = 1
		}
		gemsGained = perItem * count
	}

	if gemsGained > 0 {
		c.Player.Gems += gemsGained
		c.SendRawPacket(variant.SetBux(c.Player.Gems, 0).Pack(), true)
	}

	c.Save()
	c.Sound("audio/trash.wav")

	name := fmt.Sprintf("Item #%d", itemID)
	if item != nil && item.Name != "" {
		name = item.Name
	}

	if gemsGained > 0 {
		c.Success("Recycled %d `w%s`` and received `$%d Gems``!``", count, name, gemsGained)
	} else {
		c.Console("%d `w%s`` recycled.", count, name)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. ITEM INFO & FAVORITE (action|info & action|itemfavourite)
// ─────────────────────────────────────────────────────────────────────────────

func HandleItemInfo(c *vallenctx.Ctx) {
	itemID := c.Int("itemID")
	if itemID <= 0 {
		return
	}
	item := GetItem(uint16(itemID))
	if item == nil {
		return
	}

	name := strings.ReplaceAll(item.Name, "|", "")
	var b strings.Builder
	fmt.Fprintf(&b, "set_default_color|`o\nadd_label_with_icon|big|`wAbout %s``|left|%d|\nadd_spacer|small|\n", name, itemID)
	if item.Description != "" {
		fmt.Fprintf(&b, "add_textbox|%s|left|\n", strings.ReplaceAll(item.Description, "|", ""))
	} else {
		b.WriteString("add_textbox|`oNo additional description is available for this item.``|left|\n")
	}
	if item.Rarity < 999 {
		fmt.Fprintf(&b, "add_spacer|small|\nadd_textbox|Rarity: `w%d``|left|\n", item.Rarity)
	}
	for _, prop := range ItemPropertyText(item.Property) {
		fmt.Fprintf(&b, "add_textbox|%s|left|\n", prop)
	}
	fmt.Fprintf(&b, "add_spacer|small|\nembed_data|itemID|%d\nend_dialog|continue||OK|\n", itemID)
	c.Dialog(b.String())
}

func HandleItemFavourite(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	itemID := c.Int("itemID")
	if itemID <= 0 || !c.HasItem(itemID, 1) {
		return
	}

	idx := -1
	for i, fav := range c.Player.Favorites {
		if fav == itemID {
			idx = i
			break
		}
	}

	if idx >= 0 {
		c.Player.Favorites = append(c.Player.Favorites[:idx], c.Player.Favorites[idx+1:]...)
		c.SendRawPacket(variant.New("OnFavItemUpdated", int32(itemID), int32(0)).Pack(), true)
	} else {
		if len(c.Player.Favorites) >= maxFavoriteItems {
			c.Bubble("You cannot favorite any more items. Remove some from your list and try again.")
			c.Error("You cannot favorite any more items. Remove some from your list and try again.")
			return
		}
		c.Player.Favorites = append(c.Player.Favorites, itemID)
		c.SendRawPacket(variant.New("OnFavItemUpdated", int32(itemID), int32(1)).Pack(), true)
	}
	c.Save()
}

func ItemPropertyText(property uint8) []string {
	text := make([]string, 0, 4)
	if property == 0x0e {
		text = append(text, "`oA lock makes it so only you (and designated friends) can edit an area.``")
	}
	if property&0x01 != 0 {
		text = append(text, "`1This item can be placed in two directions, depending on the direction you're facing.``")
	}
	if property&0x02 != 0 {
		text = append(text, "`1This item has special properties you can adjust with the Wrench.``")
	}
	if property&0x04 != 0 {
		text = append(text, "`1This item never drops any seeds.``")
	}
	if property&0x08 != 0 {
		text = append(text, "`1This item can't be destroyed - smashing it will return it to your backpack if you have room!``")
	}
	if property&0x10 != 0 {
		text = append(text, "`1This item can be transmuted.``")
	}
	return text
}

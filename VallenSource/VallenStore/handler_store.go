package store

import (
	"fmt"
	"strings"

	vallenctx "gtps/VallenSource/VallenContext"
	items "gtps/VallenSource/VallenItems"
	player "gtps/VallenSource/VallenPlayer"
	variant "gtps/VallenSource/VallenVariant"
)

// ─────────────────────────────────────────────────────────────────────────────
// Store Handlers (Domain: VallenStore)
// ─────────────────────────────────────────────────────────────────────────────

// HandleStore menampilkan halaman utama Store (Home Tab)
func HandleStore(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}

	location := c.Str("location")
	if location == "" || location == "gem" || location == "bottommenu" || location == "pausemenu" {
		SendMainStorePage(c)
		return
	}

	tab := LocationToTab(location)
	if tab > 0 {
		SendStoreTab(c, tab, "")
	} else {
		SendMainStorePage(c)
	}
}

// HandleBuy memproses action|buy (klik tab navigasi toko atau tombol pembelian item)
func HandleBuy(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}

	item := c.Str("item")
	if item == "" {
		SendMainStorePage(c)
		return
	}

	// 1. Cek apakah ini navigasi tab store
	if item == "main" || item == "main_menu" || item == "tab1" {
		SendMainStorePage(c)
		return
	}
	tab := LocationToTab(item)
	if tab > 0 {
		SendStoreTab(c, tab, "")
		return
	}

	// 2. Jika bukan tab, berarti ini pembelian item!
	ProcessPurchase(c, item)
}

// HandleStoreNavigate memproses action|storenavigate (deep link / shortcut)
func HandleStoreNavigate(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}

	item := c.Str("item")
	selection := c.Str("selection")

	if item == "" || item == "main" || item == "main_menu" {
		SendMainStorePage(c)
		return
	}

	tab := LocationToTab(item)
	if tab > 0 {
		SendStoreTab(c, tab, selection)
		return
	}

	SendMainStorePage(c)
}

// ─────────────────────────────────────────────
// Store Pages
// ─────────────────────────────────────────────

// SendMainStorePage mengirim OnStoreRequest halaman utama Store
func SendMainStorePage(c *vallenctx.Ctx) {
	paymentSite := "https://discord.gg/zzWHgzaF7J"

	page := "set_description_text|Welcome to the `2Growtopia Store``! Select the item you'd like more info on.`o `wThanks for being a supporter of Growtopia!\n" +
		BuildTabButtons(0) +
		"add_banner|interface/large/gui_shop_featured_header.rttex|0|1|\n" +
		fmt.Sprintf("add_button|gems_glory|Road To Glory|interface/large/store_buttons/store_buttons30.rttex|{0}|0|0|0||||-1|-1|/interface/large/gui_shop_buybanner.rttex|1|0|`2You Get:`` Road To Glory and 120k Gems Instantly.|1||||||0|0|CustomParams:|\n") +
		fmt.Sprintf("add_button|grow_shop|`oGrowtopia Shop``|interface/large/store_buttons/store_buttons36.rttex|%s|1|1|0|0|Open Shop||-1|-1|interface/large/gui_shop_buybanner.rttex|0|1|Visit our shop for more options!|1||||||0|0|CustomParams:|\n", paymentSite) +
		"add_banner|interface/large/gui_shop_featured_header.rttex|0|2|\n" +
		"add_button|gems_bundle06|Gem Abundance|interface/large/store_buttons/store_buttons37.rttex|{0}|2|1|0||||-1|-1||-1|-1|`2You Get:`` 19,680,000 Gems, 3,680 World Locks, 10 Growtokens and 5 Megaphones.|1||||||0|0|CustomParams:|\n" +
		"add_button|gems_bundle05|Gem Bounty|interface/large/store_buttons/store_buttons34.rttex|{0}|0|6|0||||-1|-1||-1|-1|`2You Get:`` 10,080,000 Gems, 2,100 World Locks, 6 Growtokens and 3 Megaphones.|1||||||0|0|CustomParams:|\n" +
		"add_button|gems_fountain|Gem Fountain|interface/large/store_buttons/store_buttons2.rttex|{0}|0|2|0||||-1|-1||-1|-1|`2You Get:`` 922,500 Gems, 260 World Locks and 1 Growtoken.|1||||||0|0|CustomParams:|\n" +
		"add_button|gems_chest|Chest o' Gems|interface/large/store_buttons/store_buttons.rttex|{0}|0|5|0||||-1|-1||-1|-1|`2You Get:`` 280,000 Gems and 84 World Locks.|1||||||0|0|CustomParams:|\n"

	c.SendRawPacket(variant.New("OnStoreRequest", page).Pack(), true)
}

// SendStoreTab mengirim katalog item per kategori tab
func SendStoreTab(c *vallenctx.Ctx, tab int, selection string) {
	header := TabHeader(tab, c.Player)

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(BuildTabButtons(tab))

	tabItems := GetTabItems(tab)
	for _, si := range tabItems {
		cost := si.Cost
		if si.Btn == "upgrade_backpack" {
			if !CanUpgradeBackpack(c.Player.SlotSize) {
				continue
			}
			cost = BackpackCost(c.Player.SlotSize)
		}

		sb.WriteString(fmt.Sprintf(
			"add_button|%s|%s|%s|%s|%s|%s|%d|0|||-1|-1||-1|-1||1||||||0|0|CustomParams:|\n",
			si.Btn, si.Name, si.RTTX, si.Description, si.Tex1, si.Tex2, cost,
		))
	}

	if selection != "" {
		sb.WriteString(fmt.Sprintf("select_item|%s\n", selection))
	}

	c.SendRawPacket(variant.New("OnStoreRequest", sb.String()).Pack(), true)
}

// ─────────────────────────────────────────────
// Purchase Logic
// ─────────────────────────────────────────────

// ProcessPurchase menangani pembelian item toko oleh player
func ProcessPurchase(c *vallenctx.Ctx, btn string) {
	p := c.Player

	// ── 1. Khusus Backpack Upgrade ──
	if btn == "upgrade_backpack" {
		if !CanUpgradeBackpack(p.SlotSize) {
			c.Sound("audio/bleep_fail.wav")
			c.SendRawPacket(variant.New("OnStorePurchaseResult", "Your backpack is already at maximum size!").Pack(), true)
			return
		}
		cost := BackpackCost(p.SlotSize)
		if p.Gems < cost {
			c.Sound("audio/bleep_fail.wav")
			c.SendRawPacket(variant.New("OnStorePurchaseResult",
				fmt.Sprintf("You can't afford `0Upgrade Backpack`` (`w10 Slots``)! You're `$%d`` Gems short.", cost-p.Gems)).Pack(), true)
			return
		}

		p.Gems -= cost
		p.SlotSize += SlotsPerUpgrade
		c.Sound("audio/piano_nice.wav")
		c.SendRawPacket(variant.SetBux(p.Gems, 0).Pack(), true)

		successMsg := fmt.Sprintf("You've purchased `0Upgrade Backpack (10 Slots)`` for `$%d`` Gems.\nYou have `$%d`` Gems left.\n\n`5Received: ```0Backpack Upgrade``\n", cost, p.Gems)
		c.SendRawPacket(variant.New("OnStorePurchaseResult", successMsg).Pack(), true)
		c.Console("You've purchased `0Upgrade Backpack`` (`010 Slots``) for `$%d`` Gems.\nYou have `$%d`` Gems left.", cost, p.Gems)

		c.Save()
		c.SyncInventory()
		SendStoreTab(c, TabLocks, "upgrade_backpack")
		return
	}

	// ── 2. Item Normal dari storeItems ──
	si := FindItem(btn)
	if si == nil {
		c.Sound("audio/bleep_fail.wav")
		c.SendRawPacket(variant.New("OnStorePurchaseResult", fmt.Sprintf("Item `0%s`` not found in store.", btn)).Pack(), true)
		return
	}

	cost := si.Cost

	// Cek saldo mata uang
	if si.IsGrowtokenItem() {
		gtCost := si.GrowtokenCost()
		playerGT := p.GetItemCount(GrowtokenItemID)
		if playerGT < gtCost {
			c.Sound("audio/bleep_fail.wav")
			c.SendRawPacket(variant.New("OnStorePurchaseResult",
				fmt.Sprintf("You can't afford `0%s``! You're `$%d`` `2Growtokens`` short.", si.Name, gtCost-playerGT)).Pack(), true)
			return
		}
	} else {
		if p.Gems < cost {
			c.Sound("audio/bleep_fail.wav")
			c.SendRawPacket(variant.New("OnStorePurchaseResult",
				fmt.Sprintf("You can't afford `0%s``! You're `$%d`` Gems short.", si.Name, cost-p.Gems)).Pack(), true)
			return
		}
	}

	// Reward items
	var rewards []Reward
	dynamic := GenerateRewards(btn)
	if dynamic != nil {
		rewards = dynamic
	} else {
		rewards = si.Rewards
	}

	if len(rewards) == 0 {
		c.Sound("audio/bleep_fail.wav")
		c.SendRawPacket(variant.New("OnStorePurchaseResult", fmt.Sprintf("`0%s`` is not available at this time.", si.Name)).Pack(), true)
		return
	}

	// Cek kapasitas tas
	capacityCheck := make([]player.Item, 0, len(rewards))
	for _, reward := range rewards {
		if reward.ID != BackpackUpgradeID {
			capacityCheck = append(capacityCheck, player.Item{ID: reward.ID, Count: reward.Count})
		}
	}
	if !p.CanAddItems(capacityCheck) {
		c.Sound("audio/bleep_fail.wav")
		c.SendRawPacket(variant.New("OnStorePurchaseResult", "You don't have enough space in your inventory for that! You may be carrying too many of one of the items or you don't have enough free slots.").Pack(), true)
		return
	}

	// Berikan reward
	var receivedNames []string
	for _, r := range rewards {
		if r.ID == BackpackUpgradeID {
			p.SlotSize += SlotsPerUpgrade
			receivedNames = append(receivedNames, fmt.Sprintf("%d Backpack Slots", SlotsPerUpgrade))
		} else {
			p.AddItem(r.ID, r.Count)
			name := fmt.Sprintf("Item #%d", r.ID)
			if itm := items.GetItem(uint16(r.ID)); itm != nil && itm.Name != "" {
				name = itm.Name
			}
			receivedNames = append(receivedNames, fmt.Sprintf("%dx %s", r.Count, name))
		}
	}

	// Potong saldo & notifikasi
	var successMsg string
	if si.IsGrowtokenItem() {
		gtCost := si.GrowtokenCost()
		p.RemoveItem(GrowtokenItemID, gtCost)
		remaining := p.GetItemCount(GrowtokenItemID)
		successMsg = fmt.Sprintf("You've purchased `0%s`` for `$%d`` `2Growtokens``.\nYou have `$%d`` `2Growtokens`` left.\n\n`5Received: ```0%s``\n",
			si.Name, gtCost, remaining, strings.Join(receivedNames, ", "))
	} else {
		p.Gems -= cost
		successMsg = fmt.Sprintf("You've purchased `0%s`` for `$%d`` Gems.\nYou have `$%d`` Gems left.\n\n`5Received: ```0%s``\n",
			si.Name, cost, p.Gems, strings.Join(receivedNames, ", "))
		c.SendRawPacket(variant.SetBux(p.Gems, 0).Pack(), true)
	}

	c.Sound("audio/piano_nice.wav")
	c.SendRawPacket(variant.New("OnStorePurchaseResult", successMsg).Pack(), true)
	c.Console("You've purchased `0%s``.\n`5Received: ```0%s``", si.Name, strings.Join(receivedNames, ", "))
	c.Save()
	c.SyncInventory()

	// Refresh tab tempat item tersebut berada
	if si.Tab > 0 {
		SendStoreTab(c, si.Tab, si.Btn)
	} else {
		SendMainStorePage(c)
	}
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

// LocationToTab memetakan string location/item ke tab number
func LocationToTab(location string) int {
	switch strings.ToLower(location) {
	case "locks", "locks_menu", "tab4":
		return TabLocks
	case "itempack", "itempack_menu", "tab2":
		return TabItemPack
	case "bigitems", "bigitems_menu", "bigiitems", "bigiitems_menu", "tab6":
		return TabBigItems
	case "weather", "weather_menu", "tab3":
		return TabWeather
	case "token", "token_menu", "tab5":
		return TabToken
	}
	return 0
}

// BuildTabButtons menyusun tombol navigasi tab store
func BuildTabButtons(activeTab int) string {
	active := func(tab int) string {
		if tab == activeTab {
			return "1"
		}
		return "0"
	}
	return "enable_tabs|1\n" +
		fmt.Sprintf("add_tab_button|main_menu|Home|interface/large/btn_shop.rttex||%s|0|0|0||||-1|-1|||0|0|CustomParams:|\n", active(0)) +
		fmt.Sprintf("add_tab_button|locks_menu|Locks And Stuff|interface/large/btn_shop.rttex||%s|1|0|0||||-1|-1|||0|0|CustomParams:|\n", active(TabLocks)) +
		fmt.Sprintf("add_tab_button|itempack_menu|Item Packs|interface/large/btn_shop.rttex||%s|3|0|0||||-1|-1|||0|0|CustomParams:|\n", active(TabItemPack)) +
		fmt.Sprintf("add_tab_button|bigitems_menu|Awesome Items|interface/large/btn_shop.rttex||%s|4|0|0||||-1|-1|||0|0|CustomParams:|\n", active(TabBigItems)) +
		fmt.Sprintf("add_tab_button|weather_menu|Weather Machines|interface/large/btn_shop.rttex|Tired of the same sunny sky? We offer alternatives within...|%s|5|0|0||||-1|-1|||0|0|CustomParams:|\n", active(TabWeather)) +
		fmt.Sprintf("add_tab_button|token_menu|Growtoken Items|interface/large/btn_shop.rttex||%s|2|0|0||||-1|-1|||0|0|CustomParams:|\n", active(TabToken))
}

// TabHeader mengembalikan header teks deskripsi per tab
func TabHeader(tab int, p *player.Player) string {
	switch tab {
	case TabLocks:
		return "set_description_text|`2Locks And Stuff!`` Select the item you'd like more info on, or BACK to go back.\n"
	case TabItemPack:
		return "set_description_text|`2Item Packs!`` Select the item you'd like more info on, or BACK to go back.\n"
	case TabBigItems:
		return "set_description_text|`2Awesome Items!`` Select the item you'd like more info on, or BACK to go back.\n"
	case TabWeather:
		return "set_description_text|`2Weather Machines!`` Select the item you'd like more info on, or BACK to go back.\n"
	case TabToken:
		gtCount := 0
		if p != nil {
			gtCount = p.GetItemCount(GrowtokenItemID)
		}
		return fmt.Sprintf("set_description_text|`2Spend your Growtokens!`` (You have `5%d``) You earn Growtokens from Crazy Jim and Sales-Man. Select the item you'd like more info on, or BACK to go back.\n", gtCount)
	}
	return "set_description_text|Welcome to the `2Growtopia Store``!\n"
}

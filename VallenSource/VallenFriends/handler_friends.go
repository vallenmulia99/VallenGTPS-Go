package friends

import (
	"fmt"
	"strings"

	vallenctx "gtps/VallenSource/VallenContext"
)

// ─────────────────────────────────────────────────────────────────────────────
// Friends & Social Portal Handlers (Domain: VallenFriends)
// ─────────────────────────────────────────────────────────────────────────────

// HandleSocialPortal membuka dialog Social Portal resmi
func HandleSocialPortal(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	SendSocialPortal(c)
}

// SendSocialPortal menampilkan menu utama Social Portal sesuai arsitektur GrowTavern
func SendSocialPortal(c *vallenctx.Ctx) {
	dialog := "set_default_color|`o\n" +
		"add_label_with_icon|big|`wSocial Portal`` |left|1366|\n" +
		"add_spacer|small|\n" +
		"add_textbox|`oConnect with other players, view your friends, community updates, and server tools.``|left|\n" +
		"add_spacer|small|\n" +
		"add_button|show_friends|`wShow Friends``|noflags|0|0|\n" +
		"add_button|showguild|`wGrow Guild``|noflags|0|0|\n" +
		"add_button|trade_history|`wTrade History``|noflags|0|0|\n" +
		"add_button|community_hub|`wCommunity Hub``|noflags|0|0|\n" +
		"add_button|proxy_menu|`wProxy Menu``|noflags|0|0|\n" +
		"add_button|Background_Setting|`wBackground Setting``|noflags|0|0|\n" +
		"add_spacer|small|\n" +
		"add_quick_exit|\n" +
		"end_dialog|SocialPortal|Cancel||\n"
	c.Dialog(dialog)
}

// HandleSocialPortalReturn merutekan tombol dari menu Social Portal
func HandleSocialPortalReturn(c *vallenctx.Ctx, isOnline func(name string) bool, getPlayerWorld func(name string) string) {
	if c.Player == nil {
		return
	}

	btn := c.Button()
	switch btn {
	case "show_friends", "showfriend":
		SendFriendsList(c, false, isOnline, getPlayerWorld)
	case "showguild":
		SendGuildHub(c)
	case "trade_history":
		SendTradeHistory(c)
	case "community_hub", "communityhub":
		SendCommunityHub(c)
	case "proxy_menu":
		SendProxyMenu(c)
	case "Background_Setting":
		SendBackgroundSettings(c)
	case "Social_Portal", "back":
		SendSocialPortal(c)
	case "", "cancel", "close":
		return
	default:
		if strings.HasPrefix(btn, "friend_") {
			targetName := strings.TrimPrefix(btn, "friend_")
			SendFriendDetail(c, targetName, isOnline, getPlayerWorld)
			return
		}
	}
}

// SendFriendsList menampilkan daftar pertemanan interaktif
func SendFriendsList(c *vallenctx.Ctx, showAll bool, isOnline func(name string) bool, getPlayerWorld func(name string) string) {
	if c.Player == nil {
		return
	}

	friends := c.Player.Friends
	onlineCount := 0
	for _, f := range friends {
		if !f.Ignore && !f.Block && isOnline != nil && isOnline(f.Name) {
			onlineCount++
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "set_default_color|`o\nadd_label_with_icon|big|`w%d of %d Friends Online``|left|1366|\n", onlineCount, len(friends))
	b.WriteString("add_spacer|small|\n")

	if len(friends) == 0 {
		b.WriteString("add_textbox|`oYou currently have no friends. That's just sad. To make some, press a person's wrench icon, then choose `5Add as friend``.``|left|\n")
	} else {
		shown := 0
		for _, f := range friends {
			name := strings.ReplaceAll(f.Name, "|", "")
			online := isOnline != nil && isOnline(f.Name)

			if !showAll && !online {
				continue
			}

			shown++
			if online {
				worldName := ""
				if getPlayerWorld != nil {
					worldName = getPlayerWorld(f.Name)
				}
				if worldName == "" {
					worldName = "Online"
				}
				fmt.Fprintf(&b, "add_button|friend_%s|`2%s `o(`5%s`o)``|noflags|0|0|\n", name, name, worldName)
			} else {
				status := "`4Offline``"
				if f.Ignore || f.Block {
					status = "`8Ignored``"
				}
				fmt.Fprintf(&b, "add_button|friend_%s|`8%s `o(%s)``|noflags|0|0|\n", name, name, status)
			}
		}

		if shown == 0 && !showAll {
			b.WriteString("add_textbox|`oNone of your friends are currently online.``|left|\n")
		}
	}

	b.WriteString("add_spacer|small|\n")
	if !showAll {
		b.WriteString("add_button|friend_all|Show offline and ignored too|noflags|0|0|\n")
	} else {
		b.WriteString("add_button|friend_online_only|Show online only|noflags|0|0|\n")
	}
	b.WriteString("add_button|all_friends|Manage Friends|noflags|0|0|\n")
	b.WriteString("add_button|friends_options|Friend Options|noflags|0|0|\n")
	b.WriteString("add_button|back|Back|noflags|0|0|\n")
	b.WriteString("add_button||Close|noflags|0|0|\n")
	b.WriteString("add_quick_exit|\n")
	b.WriteString("end_dialog|friends|||\n")

	c.Dialog(b.String())
}

// SendFriendDetail menampilkan menu detail dan opsi aksi untuk satu teman (GrowTavern Info_Friend)
func SendFriendDetail(c *vallenctx.Ctx, targetName string, isOnline func(name string) bool, getPlayerWorld func(name string) string) {
	if c.Player == nil || targetName == "" {
		return
	}

	isMuted := false
	isBlocked := false
	for _, f := range c.Player.Friends {
		if strings.EqualFold(f.Name, targetName) {
			isMuted = f.Mute
			isBlocked = f.Block
			break
		}
	}

	online := isOnline != nil && isOnline(targetName)
	cleanName := strings.ReplaceAll(targetName, "|", "")

	var b strings.Builder
	fmt.Fprintf(&b, "set_default_color|`o\nadd_label_with_icon|big|`w%s``|left|1366|\nadd_spacer|small|\n", cleanName)
	fmt.Fprintf(&b, "embed_data|friendID|%s\n", cleanName)

	if online {
		worldName := ""
		if getPlayerWorld != nil {
			worldName = getPlayerWorld(targetName)
		}
		if worldName == "" {
			worldName = "EXIT"
		}

		fmt.Fprintf(&b, "add_textbox|`o%s is `2online`` now in the world `5%s``.|left|\n", cleanName, worldName)
		b.WriteString("add_spacer|small|\n")
		fmt.Fprintf(&b, "add_button|goto_friend|`wWarp to `5%s``|noflags|0|0|\n", worldName)
		fmt.Fprintf(&b, "embed_data|target_world|%s\n", worldName)
		fmt.Fprintf(&b, "add_button|msg_%s|`5Send Message (/msg)``|noflags|0|0|\n", cleanName)
	} else {
		fmt.Fprintf(&b, "add_textbox|`o%s is `4offline``.|left|\n", cleanName)
		b.WriteString("add_spacer|small|\n")
	}

	if isMuted {
		b.WriteString("add_button|unmute_friend|Unmute Friend|noflags|0|0|\n")
	} else {
		b.WriteString("add_button|mute_friend|Mute Friend|noflags|0|0|\n")
	}

	if isBlocked {
		b.WriteString("add_button|unblock_trade|Enable Trade|noflags|0|0|\n")
	} else {
		b.WriteString("add_button|block_trade|Disable Trade|noflags|0|0|\n")
	}

	b.WriteString("add_button|remove_friend|`4Remove as friend``|noflags|0|0|\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_button|back_to_friends|Back|noflags|0|0|\n")
	b.WriteString("add_quick_exit|\n")
	b.WriteString("end_dialog|friends_edit|||\n")

	c.Dialog(b.String())
}

// SendAllFriendsManage menampilkan daftar semua teman dengan checkbox untuk aksi massal
func SendAllFriendsManage(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}

	var b strings.Builder
	b.WriteString("set_default_color|`o\nadd_label_with_icon|big|`wManage All Friends``|left|1366|\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_textbox|`oSelect friends below to apply quick batch actions:``|left|\n")
	b.WriteString("add_spacer|small|\n")

	if len(c.Player.Friends) == 0 {
		b.WriteString("add_textbox|`oYou don't have any friends to manage yet.``|left|\n")
	} else {
		for i, f := range c.Player.Friends {
			cleanName := strings.ReplaceAll(f.Name, "|", "")
			status := ""
			if f.Mute {
				status += " <Muted>"
			}
			if f.Block {
				status += " <Trade Blocked>"
			}
			fmt.Fprintf(&b, "add_checkbox|cf_%d|%s%s|0\n", i, cleanName, status)
		}
		b.WriteString("add_spacer|small|\n")
		b.WriteString("add_button|batch_remove|`4Remove Selected``|noflags|0|0|\n")
		b.WriteString("add_button|batch_mute|Toggle Mute|noflags|0|0|\n")
		b.WriteString("add_button|batch_block|Toggle Trade Block|noflags|0|0|\n")
	}

	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_button|back_to_friends|Back|noflags|0|0|\n")
	b.WriteString("add_quick_exit|\n")
	b.WriteString("end_dialog|all_friends|||\n")

	c.Dialog(b.String())
}

// SendFriendsOptions menampilkan preferensi privasi pertemanan
func SendFriendsOptions(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}

	showLoc := "1"
	if !c.Player.ShowLocation {
		showLoc = "0"
	}
	showNotif := "1"
	if !c.Player.ShowNotifications {
		showNotif = "0"
	}

	var b strings.Builder
	b.WriteString("set_default_color|`o\nadd_label_with_icon|big|`wFriend Options``|left|1366|\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_textbox|`oManage your privacy and friend notification settings:``|left|\n")
	b.WriteString("add_spacer|small|\n")
	fmt.Fprintf(&b, "add_checkbox|checkbox_public|Show location to friends|%s\n", showLoc)
	fmt.Fprintf(&b, "add_checkbox|checkbox_notifications|Show friend notifications|%s\n", showNotif)
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_button|save_friend_options|`2Save Options``|noflags|0|0|\n")
	b.WriteString("add_button|back_to_friends|Back|noflags|0|0|\n")
	b.WriteString("add_quick_exit|\n")
	b.WriteString("end_dialog|friends_options|||\n")

	c.Dialog(b.String())
}

// SendTradeHistory menampilkan riwayat transaksi barter/trade player
func SendTradeHistory(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}

	var b strings.Builder
	fmt.Fprintf(&b, "set_default_color|`o\nadd_label_with_icon|small|`w%s's Trade History``|left|242|\n", c.Player.GrowID)
	b.WriteString("add_spacer|small|\n")

	if len(c.Player.TradeHistory) == 0 {
		b.WriteString("add_textbox|`oNothing to show yet. Trades will be recorded here automatically when you trade with other players!``|left|\n")
	} else {
		for _, record := range c.Player.TradeHistory {
			fmt.Fprintf(&b, "add_smalltext|%s|left|\n", record)
			b.WriteString("add_spacer|small|\n")
		}
	}

	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_button|back|Back|noflags|0|0|\n")
	b.WriteString("add_button||Close|noflags|0|0|\n")
	b.WriteString("add_quick_exit|\n")
	b.WriteString("end_dialog|trade_history|||\n")

	c.Dialog(b.String())
}

// SendCommunityHub menampilkan pusat komunitas resmi server
func SendCommunityHub(c *vallenctx.Ctx) {
	var b strings.Builder
	b.WriteString("set_default_color|`o\nadd_label_with_icon|big|`wCommunity Hub``|left|1366|\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_textbox|`oWelcome to the `2VALLEN GTPS`` Community Hub! Connect with us and stay updated.``|left|\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_textbox|`9Discord Server: `whttps://discord.gg/zzWHgzaF7J``|left|\n")
	b.WriteString("add_textbox|`2Server Website: `whttps://vallengtps.com``|left|\n")
	b.WriteString("add_textbox|`eServer Status: `2ONLINE - High Performance``|left|\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_smalltext|`5Rules & Fairplay Guidelines:``<CR>`o1. Cheating or exploiting bugs will result in a permanent ban.<CR>2. Respect fellow players, no harassment or spam.<CR>3. Keep your account secure and never share passwords!|left|\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_button|back|Back|noflags|0|0|\n")
	b.WriteString("add_button||Close|noflags|0|0|\n")
	b.WriteString("add_quick_exit|\n")
	b.WriteString("end_dialog|community_hub|||\n")

	c.Dialog(b.String())
}

// SendProxyMenu menampilkan fitur utilitas bawaan in-game (Proxy / Fast Casino Tools)
func SendProxyMenu(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}

	wheelVal := "0"
	if c.Player.Proxy.Roulette {
		wheelVal = "1"
	}
	remeVal := "0"
	if c.Player.Proxy.Reme {
		remeVal = "1"
	}
	spinVal := "0"
	if c.Player.Proxy.FastSpin {
		spinVal = "1"
	}

	var b strings.Builder
	b.WriteString("set_default_color|`o\nadd_label_with_icon|big|`wProxy Menu - Built-in Tools``|left|1366|\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_smalltext|`5REMINDER: `oAll these features are natively built into `wVALLEN GTPS``, no third-party client or proxy is needed!|left|\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_label_with_icon|small|`wFast Modes & Casino Utilities``|left|758|\n")
	b.WriteString("add_smalltext|`o>> Convenient options for hosters and fast gameplay:``|left|\n")
	fmt.Fprintf(&b, "add_checkbox|proxy_wheel|`$Real/Fake Roulette Wheel Display``|%s\n", wheelVal)
	fmt.Fprintf(&b, "add_checkbox|proxy_reme|`$Reme Mode (Show Real-time Calculation)``|%s\n", remeVal)
	fmt.Fprintf(&b, "add_checkbox|proxy_spin|`$Fast Wheel Spin Speed``|%s\n", spinVal)
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_button|apply_proxy|`2Update Proxy Settings``|noflags|0|0|\n")
	b.WriteString("add_button|back|Back|noflags|0|0|\n")
	b.WriteString("add_quick_exit|\n")
	b.WriteString("end_dialog|proxy_menu|Cancel||\n")

	c.Dialog(b.String())
}

// SendBackgroundSettings menampilkan kustomisasi warna tema dialog dan transparansi
func SendBackgroundSettings(c *vallenctx.Ctx) {
	var b strings.Builder
	b.WriteString("set_default_color|`o\nadd_label_with_icon|big|`wBackground & Theme Settings``|left|32|\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_textbox|`oCustomize your dialog background color and transparency to match your style!``|left|\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_smalltext|`wGrowtopia Theme Palette:``<CR>Choose your favorite theme color:|left|\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_checkicon|theme_0|`!Default``|frame|520||1|\n")
	b.WriteString("add_checkicon|theme_1|`bDark Black``|frame|166||0|\n")
	b.WriteString("add_checkicon|theme_2|`#Neon Pink``|frame|182||0|\n")
	b.WriteString("add_checkicon|theme_3|`pRoyal Purple``|frame|2026||0|\n")
	b.WriteString("add_checkicon|theme_4|`4Vivid Red``|frame|170||0|\n")
	b.WriteString("add_checkicon|theme_5|`eOcean Blue``|frame|180||0|\n")
	b.WriteString("add_checkicon|theme_6|`8Sunset Orange``|frame|172||0|\n")
	b.WriteString("add_checkicon|theme_7|`2Emerald Green``|frame|176||0|\n")
	b.WriteString("add_checkicon|theme_8|`9Golden Yellow``|frame|174||0|\n")
	b.WriteString("add_button_with_icon||END_LIST|noflags|0||\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_smalltext|`wTheme Transparency:``<CR>Adjust transparency of dialog backgrounds:|left|\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_checkicon|themetrans_0|`$0%``|frame|1052||0|\n")
	b.WriteString("add_checkicon|themetrans_1|`$25%``|frame|1052||0|\n")
	b.WriteString("add_checkicon|themetrans_2|`$50%``|frame|1052||1|\n")
	b.WriteString("add_checkicon|themetrans_3|`$75%``|frame|1052||0|\n")
	b.WriteString("add_checkicon|themetrans_4|`$100%``|frame|1052||0|\n")
	b.WriteString("add_button_with_icon||END_LIST|noflags|0||\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_button|apply_theme|`2Apply Theme``|noflags|0|0|\n")
	b.WriteString("add_button|back|Back|noflags|0|0|\n")
	b.WriteString("add_quick_exit|\n")
	b.WriteString("end_dialog|bg_settings|Cancel||\n")

	c.Dialog(b.String())
}

// SendGuildHub menampilkan informasi Guild
func SendGuildHub(c *vallenctx.Ctx) {
	var b strings.Builder
	b.WriteString("set_default_color|`o\nadd_label_with_icon|big|`wGrow Guild Hub``|left|5814|\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_textbox|`oYou are not currently a member of any Guild!``|left|\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_textbox|`wGuild Features:``<CR>`o- Guild Clash & Seasonal Events<CR>- Exclusive Guild Items, Shields & Titles<CR>- Shared Guild Vault & Team World<CR>- Guild Chat Channel|left|\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_smalltext|`2To create a Guild, you will need a `wGuild Key`2 and 100,000 Gems.``|left|\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_button|back|Back|noflags|0|0|\n")
	b.WriteString("add_button||Close|noflags|0|0|\n")
	b.WriteString("add_quick_exit|\n")
	b.WriteString("end_dialog|guild_info|||\n")

	c.Dialog(b.String())
}

// HandleFriendsDialogReturn memproses seluruh tombol aksi dari dialog-dialog Social Portal
func HandleFriendsDialogReturn(c *vallenctx.Ctx, isOnline func(name string) bool, getPlayerWorld func(name string) string) {
	if c.Player == nil {
		return
	}

	btn := c.Button()

	// Navigasi Back
	if btn == "back" {
		SendSocialPortal(c)
		return
	}
	if btn == "back_to_friends" {
		SendFriendsList(c, false, isOnline, getPlayerWorld)
		return
	}
	if btn == "friend_all" {
		SendFriendsList(c, true, isOnline, getPlayerWorld)
		return
	}
	if btn == "friend_online_only" {
		SendFriendsList(c, false, isOnline, getPlayerWorld)
		return
	}
	if btn == "all_friends" {
		SendAllFriendsManage(c)
		return
	}
	if btn == "friends_options" {
		SendFriendsOptions(c)
		return
	}

	// Klik teman tertentu dari Friends List -> Buka detail friend
	if strings.HasPrefix(btn, "friend_") {
		name := strings.TrimPrefix(btn, "friend_")
		SendFriendDetail(c, name, isOnline, getPlayerWorld)
		return
	}

	// Warp to friend's world (menggunakan data yang di-embed dari tombol)
	if btn == "goto_friend" {
		targetWorld := c.Str("target_world")
		if targetWorld != "" && targetWorld != "EXIT" {
			c.Warp(targetWorld)
			return
		}
	}

	// Send message to friend
	if strings.HasPrefix(btn, "msg_") {
		targetName := strings.TrimPrefix(btn, "msg_")
		c.Console("`oType `5/msg %s <your_message>`` in chat to whisper to your friend!``", targetName)
		return
	}

	// Mute / Unmute friend
	if btn == "mute_friend" || btn == "unmute_friend" {
		muteVal := btn == "mute_friend"
		targetName := c.Str("friendID")
		for i := range c.Player.Friends {
			if strings.EqualFold(c.Player.Friends[i].Name, targetName) {
				c.Player.Friends[i].Mute = muteVal
				c.Save()
				if muteVal {
					c.Success("Muted %s.", targetName)
				} else {
					c.Success("Unmuted %s.", targetName)
				}
				SendFriendDetail(c, targetName, isOnline, getPlayerWorld)
				return
			}
		}
	}

	// Block / Unblock trade
	if btn == "block_trade" || btn == "unblock_trade" {
		blockVal := btn == "block_trade"
		targetName := c.Str("friendID")
		for i := range c.Player.Friends {
			if strings.EqualFold(c.Player.Friends[i].Name, targetName) {
				c.Player.Friends[i].Block = blockVal
				c.Save()
				if blockVal {
					c.Success("Trade disabled for %s.", targetName)
				} else {
					c.Success("Trade enabled for %s.", targetName)
				}
				SendFriendDetail(c, targetName, isOnline, getPlayerWorld)
				return
			}
		}
	}

	// Remove friend
	if btn == "remove_friend" {
		targetName := c.Str("friendID")
		if targetName != "" {
			c.Player.RemoveFriend(targetName)
			c.Save()
			c.Success("Removed %s from friends.", targetName)
			SendFriendsList(c, false, isOnline, getPlayerWorld)
			return
		}
	}

	// Batch actions from Manage Friends
	if btn == "batch_remove" {
		removedCount := 0
		for k, v := range c.Values {
			if strings.HasPrefix(k, "cf_") && v == "1" {
				idx := 0
				fmt.Sscanf(strings.TrimPrefix(k, "cf_"), "%d", &idx)
				if idx >= 0 && idx < len(c.Player.Friends) {
					c.Player.RemoveFriend(c.Player.Friends[idx].Name)
					removedCount++
				}
			}
		}
		if removedCount > 0 {
			c.Save()
			c.Success("Removed %d selected friend(s).", removedCount)
		}
		SendFriendsList(c, false, isOnline, getPlayerWorld)
		return
	}

	if btn == "batch_mute" {
		for k, v := range c.Values {
			if strings.HasPrefix(k, "cf_") && v == "1" {
				idx := 0
				fmt.Sscanf(strings.TrimPrefix(k, "cf_"), "%d", &idx)
				if idx >= 0 && idx < len(c.Player.Friends) {
					c.Player.Friends[idx].Mute = !c.Player.Friends[idx].Mute
				}
			}
		}
		c.Save()
		c.Success("Updated mute status for selected friends.")
		SendAllFriendsManage(c)
		return
	}

	if btn == "batch_block" {
		for k, v := range c.Values {
			if strings.HasPrefix(k, "cf_") && v == "1" {
				idx := 0
				fmt.Sscanf(strings.TrimPrefix(k, "cf_"), "%d", &idx)
				if idx >= 0 && idx < len(c.Player.Friends) {
					c.Player.Friends[idx].Block = !c.Player.Friends[idx].Block
				}
			}
		}
		c.Save()
		c.Success("Updated trade block status for selected friends.")
		SendAllFriendsManage(c)
		return
	}

	// Save Friend Options
	if btn == "save_friend_options" {
		c.Player.ShowLocation = c.Str("checkbox_public") == "1"
		c.Player.ShowNotifications = c.Str("checkbox_notifications") == "1"
		c.Save()
		c.Success("Friend privacy options updated.")
		SendFriendsList(c, false, isOnline, getPlayerWorld)
		return
	}

	// Apply Proxy settings
	if btn == "apply_proxy" {
		c.Player.Proxy.Roulette = c.Str("proxy_wheel") == "1"
		c.Player.Proxy.Reme = c.Str("proxy_reme") == "1"
		c.Player.Proxy.FastSpin = c.Str("proxy_spin") == "1"
		c.Save()
		c.Success("Proxy mode settings updated and saved!")
		SendSocialPortal(c)
		return
	}

	// Apply Theme settings
	if btn == "apply_theme" {
		borderColor := "Default"
		bgColor := "0,0,0,150"

		if c.Str("theme_1") == "1" {
			borderColor = "0,0,0,255" // Dark Black
		} else if c.Str("theme_2") == "1" {
			borderColor = "255,0,255,255" // Neon Pink
		} else if c.Str("theme_3") == "1" {
			borderColor = "117,27,107,255" // Royal Purple
		} else if c.Str("theme_4") == "1" {
			borderColor = "255,3,11,255" // Vivid Red
		} else if c.Str("theme_5") == "1" {
			borderColor = "3,36,252,255" // Ocean Blue
		} else if c.Str("theme_6") == "1" {
			borderColor = "181,102,45,255" // Sunset Orange
		} else if c.Str("theme_7") == "1" {
			borderColor = "47,117,33,255" // Emerald Green
		} else if c.Str("theme_8") == "1" {
			borderColor = "245,245,0,255" // Golden Yellow
		}

		alpha := "150"
		if c.Str("themetrans_0") == "1" {
			alpha = "0"
		} else if c.Str("themetrans_1") == "1" {
			alpha = "64"
		} else if c.Str("themetrans_2") == "1" {
			alpha = "150"
		} else if c.Str("themetrans_3") == "1" {
			alpha = "210"
		} else if c.Str("themetrans_4") == "1" {
			alpha = "250"
		}

		if borderColor != "Default" {
			parts := strings.Split(borderColor, ",")
			if len(parts) >= 3 {
				bgColor = fmt.Sprintf("%s,%s,%s,%s", parts[0], parts[1], parts[2], alpha)
			}
		}

		c.Player.BorderColor = borderColor
		c.Player.BgColor = bgColor
		c.Save()
		c.Success("Dialog theme applied! All your dialogs will now use this color.")
		SendSocialPortal(c)
		return
	}
}

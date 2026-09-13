package events

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	vallenctx "gtps/VallenSource/VallenContext"
	dialog "gtps/VallenSource/VallenDialog"
	items "gtps/VallenSource/VallenItems"
	player "gtps/VallenSource/VallenPlayer"
	variant "gtps/VallenSource/VallenVariant"
)

// ─────────────────────────────────────────────────────────────────────────────
// SuperMain Action Dispatcher
// ─────────────────────────────────────────────────────────────────────────────

func HandleSuperMainAction(c *vallenctx.Ctx, action string, onDungeonMenu func()) {
	if c.Player == nil {
		return
	}

	switch action {
	case "showdungeonsui":
		if onDungeonMenu != nil {
			onDungeonMenu()
		}
	case "show_mailbox_ui":
		SendMailboxDialog(c)
	case "show_auction_ui":
		SendAuctionHouseDialog(c)
	case "openPiggyBank":
		SendPiggyBankDialog(c)
	case "dailychallengemenu":
		SendDailyChallengeDialog(c)
	case "show_bingo_ui":
		SendBingoDialog(c)
	case "winterrallymenu":
		SendWinterRallyDialog(c)
	case "leaderboardBtnClicked":
		SendLeaderboardDialog(c)
	case "euphoriaBtnClicked":
		SendEuphoriaDialog(c)
	case "show_fruit_mixer_dialog":
		SendFruitMixerDialog(c)
	default:
		SendSeasonalEventDialog(c, action)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. MAILBOX SYSTEM
// ─────────────────────────────────────────────────────────────────────────────

func SendMailboxDialog(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	c.Player.EnsureMailboxDefaults()

	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wMailbox``", 2204)
	d.AddSpacer("small")

	unreadCount := 0
	for _, m := range c.Player.Mailbox {
		if !m.Claimed {
			unreadCount++
		}
	}

	d.AddTextbox(fmt.Sprintf("You have <span class=\"color_crazy_orange\">%d</span> unread message(s).", unreadCount))
	d.AddSpacer("small")

	for i, m := range c.Player.Mailbox {
		status := "`2[NEW]``"
		if m.Claimed {
			status = "`7[READ]``"
		}
		d.AddTextbox(fmt.Sprintf("%s `w%s`` from %s", status, m.Title, m.Sender))
		d.AddSmallText(fmt.Sprintf("`o%s``", m.Message))
		if m.RewardQty > 0 && !m.Claimed {
			itemInfo := items.GetItem(uint16(m.RewardID))
			itemName := "Item"
			if itemInfo != nil {
				itemName = itemInfo.Name
			}
			d.AddLabelWithIcon("small", fmt.Sprintf("`5Reward: %dx %s``", m.RewardQty, itemName), m.RewardID)
			d.AddButton(fmt.Sprintf("claim_mail_%d", i), fmt.Sprintf("Claim Reward #%d", i+1))
		}
		d.AddSpacer("small")
	}

	if unreadCount > 0 {
		d.AddButton("claim_all_mail", "`2Claim All Rewards``")
		d.AddSpacer("small")
	}

	d.AddQuickExit()
	c.Dialog(d.EndDialog("mailbox_dialog", "Close", ""))
}

func HandleMailboxDialogReturn(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	btn := c.Button()

	if btn == "claim_all_mail" {
		totalClaimed := 0
		for i := range c.Player.Mailbox {
			m := &c.Player.Mailbox[i]
			if !m.Claimed && m.RewardQty > 0 {
				if c.Player.CanAddItems([]player.Item{{ID: m.RewardID, Count: m.RewardQty}}) {
					c.AddItem(m.RewardID, m.RewardQty)
					m.Claimed = true
					totalClaimed++
				}
			}
		}
		if totalClaimed > 0 {
			c.Success("Claimed all rewards from %d mail(s)!", totalClaimed)
			c.SendRawPacket(variant.New("OnAddNotification", "interface/large/mailbox_reddot1.rttex", "All Mail Rewards Claimed!", "audio/secret.wav").Pack(), true)
		} else {
			c.Error("No unclaimed rewards or backpack is full!")
		}
		return
	}

	if strings.HasPrefix(btn, "claim_mail_") {
		idxStr := strings.TrimPrefix(btn, "claim_mail_")
		idx, err := strconv.Atoi(idxStr)
		if err == nil && idx >= 0 && idx < len(c.Player.Mailbox) {
			m := &c.Player.Mailbox[idx]
			if !m.Claimed && m.RewardQty > 0 {
				if c.Player.CanAddItems([]player.Item{{ID: m.RewardID, Count: m.RewardQty}}) {
					c.AddItem(m.RewardID, m.RewardQty)
					m.Claimed = true
					c.Success("Claimed %dx reward from %s!", m.RewardQty, m.Title)
				} else {
					c.Error("Your backpack is full! Free up space first.")
				}
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. WORLD LOCK BANK
// ─────────────────────────────────────────────────────────────────────────────

func SendWLBankDialog(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	inventoryWL := c.Player.GetItemCount(242)

	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wWorld Lock Bank Storage``", 242)
	d.AddSpacer("small")
	d.AddTextbox(fmt.Sprintf("Bank Balance: `2%d World Locks``", c.Player.BankWL))
	d.AddTextbox(fmt.Sprintf("Backpack WLs: `$%d World Locks``", inventoryWL))
	d.AddSpacer("small")
	d.AddLabel("small", "`9--- Quick Deposit ---``")
	d.AddButton("dep_1", "Deposit 1 WL")
	d.AddButton("dep_10", "Deposit 10 WL")
	d.AddButton("dep_50", "Deposit 50 WL")
	d.AddButton("dep_all", "Deposit All WLs")
	d.AddSpacer("small")
	d.AddLabel("small", "`9--- Quick Withdraw ---``")
	d.AddButton("wdr_1", "Withdraw 1 WL")
	d.AddButton("wdr_10", "Withdraw 10 WL")
	d.AddButton("wdr_50", "Withdraw 50 WL")
	d.AddButton("wdr_all", "Withdraw All WLs")
	d.AddSpacer("small")
	d.AddTextInput("custom_amount", "Custom Amount:", "0", 6)
	d.AddButton("custom_dep", "Deposit Amount")
	d.AddButton("custom_wdr", "Withdraw Amount")
	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("wl_bank_dialog", "Close", ""))
}

func HandleWLBankDialogReturn(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	btn := c.Button()
	inventoryWL := c.Player.GetItemCount(242)

	switch btn {
	case "dep_1":
		DepositWL(c, 1)
	case "dep_10":
		DepositWL(c, 10)
	case "dep_50":
		DepositWL(c, 50)
	case "dep_all":
		DepositWL(c, inventoryWL)
	case "wdr_1":
		WithdrawWL(c, 1)
	case "wdr_10":
		WithdrawWL(c, 10)
	case "wdr_50":
		WithdrawWL(c, 50)
	case "wdr_all":
		WithdrawWL(c, c.Player.BankWL)
	case "custom_dep":
		if amt := c.Int("custom_amount"); amt > 0 {
			DepositWL(c, amt)
		}
	case "custom_wdr":
		if amt := c.Int("custom_amount"); amt > 0 {
			WithdrawWL(c, amt)
		}
	}
}

func DepositWL(c *vallenctx.Ctx, amount int) {
	if amount <= 0 {
		return
	}
	if !c.HasItem(242, amount) {
		c.Error("You don't have that many World Locks in your backpack!")
		return
	}
	c.RemoveItem(242, amount)
	c.Player.BankWL += amount
	c.Save()
	c.Success("Deposited `w%d `2World Locks into Bank! Balance: `w%d``", amount, c.Player.BankWL)
}

func WithdrawWL(c *vallenctx.Ctx, amount int) {
	if amount <= 0 {
		return
	}
	if c.Player.BankWL < amount {
		c.Error("You don't have that many World Locks in your bank!")
		return
	}
	if !c.Player.CanAddItems([]player.Item{{ID: 242, Count: amount}}) {
		c.Error("Not enough backpack slots to withdraw!")
		return
	}
	c.Player.BankWL -= amount
	c.AddItem(242, amount)
	c.Success("Withdrew `w%d `2World Locks from Bank! Remaining: `w%d``", amount, c.Player.BankWL)
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. TITLE SELECTION
// ─────────────────────────────────────────────────────────────────────────────

func SendTitlesDialog(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wPlayer Title Customization``", 32)
	d.AddSpacer("small")
	currentTitle := c.Player.Title
	if currentTitle == "" {
		currentTitle = "(None)"
	}
	d.AddTextbox(fmt.Sprintf("Current Title: `^%s``", currentTitle))
	d.AddSpacer("small")

	for i, t := range AvailableTitles {
		selected := (t == currentTitle) || (t == "(None)" && currentTitle == "")
		d.AddRadioButton("title_group", fmt.Sprintf("title_%d", i), t, selected)
	}

	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("titles_dialog", "Cancel", "Apply Title"))
}

func HandleTitlesDialogReturn(c *vallenctx.Ctx) {
	if c.Player == nil || strings.EqualFold(c.Button(), "cancel") {
		return
	}
	for k, v := range c.Values {
		if strings.HasPrefix(k, "title_") && v == "1" {
			idx, err := strconv.Atoi(strings.TrimPrefix(k, "title_"))
			if err == nil && idx >= 0 && idx < len(AvailableTitles) {
				newTitle := AvailableTitles[idx]
				if newTitle == "(None)" {
					c.Player.Title = ""
				} else {
					c.Player.Title = newTitle
				}
				c.Save()
				c.Success("Title updated to: `^%s``!``", newTitle)
				return
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. ONLINE STATUS
// ─────────────────────────────────────────────────────────────────────────────

func SendOnlineStatusDialog(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wSet Online Status``", 32)
	d.AddSpacer("small")
	d.AddRadioButton("status_group", "status_0", "`2Online`` (Available to play)", c.Player.OnlineStatus == 0)
	d.AddRadioButton("status_group", "status_1", "`9Away`` (Taking a break)", c.Player.OnlineStatus == 1)
	d.AddRadioButton("status_group", "status_2", "`4Busy`` (Do not disturb)", c.Player.OnlineStatus == 2)
	d.AddRadioButton("status_group", "status_3", "`pIn Love`` (Spread the love)", c.Player.OnlineStatus == 3)
	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("status_picker_dialog", "Cancel", "Save Status"))
}

func HandleOnlineStatusDialogReturn(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	for i := 0; i <= 3; i++ {
		if c.Str(fmt.Sprintf("status_%d", i)) == "1" {
			c.Player.OnlineStatus = i
			c.Save()
			statusNames := []string{"`2Online``", "`9Away``", "`4Busy``", "`pIn Love``"}
			c.Success("Status set to: %s``", statusNames[i])
			return
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 5. PERSONALIZE PROFILE & NOTEBOOK
// ─────────────────────────────────────────────────────────────────────────────

func SendPersonalizeProfileDialog(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wPersonalize Profile``", 32)
	d.AddSpacer("small")
	d.AddTextInput("profile_bio", "Bio Quote:", c.Player.Bio, 60)
	d.AddTextInput("profile_prefix", "Chat Color Code (e.g. w, 2, 4, c, p):", c.Player.Prefix, 2)
	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("personalize_dialog", "Cancel", "Save Profile"))
}

func HandlePersonalizeProfileDialogReturn(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	bio := c.Str("profile_bio")
	prefix := c.Str("profile_prefix")
	if len(prefix) > 0 {
		c.Player.Prefix = prefix[:1]
	}
	c.Player.Bio = bio
	c.Save()
	c.Success("Profile personalization saved!")
}

func SendNotebookDialog(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wPersonal Notebook``", 32)
	d.AddSpacer("small")
	d.AddTextInput("notes_content", "My Notes:", c.Player.Notebook, 200)
	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("notebook_dialog", "Cancel", "Save Notes"))
}

func HandleNotebookDialogReturn(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	c.Player.Notebook = c.Str("notes_content")
	c.Save()
	c.Success("Notebook updated successfully!")
}

// ─────────────────────────────────────────────────────────────────────────────
// 6. DAILY BONUS, SEED DIARY, ACHIEVEMENTS, GROWMOJIS & MY WORLDS
// ─────────────────────────────────────────────────────────────────────────────

func SendDailyBonusDialog(c *vallenctx.Ctx) {
	if c.Player == nil || c.Config == nil {
		return
	}
	
	// Type assert config (safe because we know it's *config.Config from server)
	type ConfigGetter interface {
		GetDailyBonus() (enabled bool, cooldown int64, gems int, wl int, xp int)
	}
	
	// Extract settings
	enabled := true
	cooldown := int64(86400)
	rewardGems := 5000
	rewardWL := 3
	rewardXP := 50
	
	// Try to get from config if available
	if cfg, ok := c.Config.(ConfigGetter); ok {
		enabled, cooldown, rewardGems, rewardWL, rewardXP = cfg.GetDailyBonus()
	}
	
	if !enabled {
		c.Console("`4Daily Bonus is currently disabled by server.``")
		return
	}
	
	now := time.Now().Unix()
	canClaim := (now - c.Player.LastDailyBonus) >= cooldown

	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wDaily Login Bonus``", 242)
	d.AddSpacer("small")

	if canClaim {
		d.AddTextbox("`2Your Daily Bonus is ready to claim!``")
		d.AddTextbox("Rewards:")
		d.AddSmallText(fmt.Sprintf("• `2+%d Gems``", rewardGems))
		d.AddSmallText(fmt.Sprintf("• `5+%d World Locks``", rewardWL))
		d.AddSmallText(fmt.Sprintf("• `3+%d XP``", rewardXP))
		d.AddSpacer("small")
		d.AddButton("claim_daily_bonus", "`2Claim Reward Now!``")
	} else {
		remSec := cooldown - (now - c.Player.LastDailyBonus)
		hours := remSec / 3600
		mins := (remSec % 3600) / 60
		d.AddTextbox("`4You already claimed your Daily Bonus!``")
		d.AddTextbox(fmt.Sprintf("Next bonus in: `w%d hours, %d minutes.``", hours, mins))
	}
	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("daily_bonus_dialog", "Close", ""))
}

func HandleDailyBonusDialogReturn(c *vallenctx.Ctx) {
	if c.Player == nil || c.Button() != "claim_daily_bonus" || c.Config == nil {
		return
	}
	
	// Type assert config
	type ConfigGetter interface {
		GetDailyBonus() (enabled bool, cooldown int64, gems int, wl int, xp int)
	}
	
	// Get settings
	enabled, cooldown, rewardGems, rewardWL, rewardXP := true, int64(86400), 5000, 3, 50
	if cfg, ok := c.Config.(ConfigGetter); ok {
		enabled, cooldown, rewardGems, rewardWL, rewardXP = cfg.GetDailyBonus()
	}
	
	if !enabled {
		c.Error("Daily Bonus is disabled!")
		return
	}
	
	now := time.Now().Unix()
	if (now - c.Player.LastDailyBonus) < cooldown {
		c.Error("Daily bonus is on cooldown!")
		return
	}

	c.Player.LastDailyBonus = now
	c.Player.Gems += rewardGems
	c.Player.XP += rewardXP
	c.AddItem(242, rewardWL)

	c.SendRawPacket(variant.SetBux(c.Player.Gems, 0).Pack(), true)
	c.Success("Daily Bonus Claimed! +%d Gems, +%d WLs, +%d XP!", rewardGems, rewardWL, rewardXP)
	c.SendRawPacket(variant.New("OnAddNotification", "interface/large/gui_wrench_daily_bonus_active.rttex", "Daily Bonus Claimed!", "audio/secret.wav").Pack(), true)
}

func SendSeedDiaryDialog(c *vallenctx.Ctx) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wSeed Diary & Splicing Guide``", 11)
	d.AddSpacer("small")
	d.AddTextbox("Discover cross-breeding recipes to grow new trees:")
	d.AddSpacer("small")

	recipes := []string{
		"• `wDirt Seed (3) + Rock Seed (11) = Grass Seed (5)``",
		"• `wDirt Seed (3) + Cave Background (15) = Door Seed (7)``",
		"• `wRock Seed (11) + Dirt Seed (3) = Lava Seed (4)``",
		"• `wGrass Seed (5) + Wood Block (9) = Rose Seed (13)``",
		"• `wDoor Seed (7) + Cave Background (15) = Sign Seed (19)``",
		"• `wGlass Pane (17) + Danger Sign (29) = Laser Grid (224)``",
		"• `wPortcullis (24) + Glass Pane (17) = Forcefield (270)``",
	}
	for _, r := range recipes {
		d.AddSmallText(r)
	}
	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("seed_diary_dialog", "Close", ""))
}

func SendAchievementsDialog(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wAchievements (Growtopia Trophy)``", 32)
	d.AddSpacer("small")
	d.AddTextbox(fmt.Sprintf("• `9Level Achiever:`` Current Level %d (Milestone: Level 50)", c.Player.Level))
	d.AddTextbox(fmt.Sprintf("• `9Wealthy Growtopian:`` Current Gems: %d | Bank: %d WL", c.Player.Gems, c.Player.BankWL))
	d.AddTextbox(fmt.Sprintf("• `9Social Butterfly:`` Friends Added: %d", len(c.Player.Friends)))
	d.AddTextbox(fmt.Sprintf("• `9Firefighter:`` Fires Extinguished: %d", c.Player.FiresRemoved))
	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("achievements_dialog", "Close", ""))
}

func SendGrowmojisDialog(c *vallenctx.Ctx) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wGrowmojis Emote Picker``", 32)
	d.AddSpacer("small")
	d.AddTextbox("Type any of these codes into chat to show Growmojis:")
	d.AddSpacer("small")
	emojis := []string{
		"`2:)`` - Smile  |  `2:(`` - Sad  |  `2:D`` - Laugh",
		"`2;)`` - Wink   |  `2:P`` - Tongue |  `2:O`` - Surprised",
		"`2:cry:`` - Cry |  `2:mad:`` - Angry |  `2:love:`` - Heart",
		"`2:wl:`` - World Lock | `2:dl:`` - Diamond Lock | `2:bgl:`` - Blue Gem Lock",
		"`2:cool:`` - Sunglasses | `2:omg:`` - Shock | `2:party:`` - Confetti",
	}
	for _, e := range emojis {
		d.AddSmallText(e)
	}
	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("emojis_dialog", "Close", ""))
}

func SendMyWorldsDialog(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wMy Worlds History``", 242)
	d.AddSpacer("small")
	d.AddTextbox("Recently visited worlds:")
	d.AddSpacer("small")

	hasWorlds := false
	for i, wName := range c.Player.RecentWorlds {
		if wName != "" {
			hasWorlds = true
			d.AddButton(fmt.Sprintf("warp_recent_%d", i), fmt.Sprintf("Warp to: %s", wName))
		}
	}
	if !hasWorlds {
		d.AddSmallText("No recent worlds recorded yet.")
	}
	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("my_worlds_dialog", "Close", ""))
}

func HandleMyWorldsDialogReturn(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	btn := c.Button()
	if strings.HasPrefix(btn, "warp_recent_") {
		idx, err := strconv.Atoi(strings.TrimPrefix(btn, "warp_recent_"))
		if err == nil && idx >= 0 && idx < len(c.Player.RecentWorlds) {
			worldName := c.Player.RecentWorlds[idx]
			if worldName != "" && c.OnJoinWorld != nil {
				c.OnJoinWorld(worldName, "")
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 7. SUPERMAIN HUD DIALOGS
// ─────────────────────────────────────────────────────────────────────────────

func SendAuctionHouseDialog(c *vallenctx.Ctx) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wAuction House Market``", 242)
	d.AddSpacer("small")
	d.AddTextbox("Welcome to the Growtopia Auction House!")
	d.AddTextbox("Browse active listings, place bids, or auction your own rare items.")
	d.AddSpacer("small")
	d.AddLabel("small", "`9Current Auction System Status:`` `2Active``")
	d.AddSmallText("WL Bank Balance is automatically linked with auction bidding.")
	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("auction_house_dialog", "Close", ""))
}

func SendPiggyBankDialog(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wPiggy Bank Savings``", 18)
	d.AddSpacer("small")
	d.AddTextbox(fmt.Sprintf("Saved Gems in Piggy Bank: `$%d Gems``", c.Player.PiggyGems))
	d.AddSpacer("small")
	if c.Player.PiggyGems > 0 {
		d.AddButton("smash_piggy", "`4Smash Piggy Bank & Claim Gems!``")
	} else {
		d.AddSmallText("Your Piggy Bank gets filled automatically while farming and doing daily activities.")
	}
	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("piggy_bank_dialog", "Close", ""))
}

func HandlePiggyBankDialogReturn(c *vallenctx.Ctx) {
	if c.Player == nil || c.Button() != "smash_piggy" || c.Player.PiggyGems <= 0 {
		return
	}
	claimed := c.Player.PiggyGems
	c.Player.Gems += claimed
	c.Player.PiggyGems = 0
	c.Save()
	c.SendRawPacket(variant.SetBux(c.Player.Gems, 0).Pack(), true)
	c.Success("Smashed Piggy Bank and claimed %d Gems!", claimed)
}

func SendDailyChallengeDialog(c *vallenctx.Ctx) {
	if c.Player == nil || c.Config == nil {
		return
	}
	
	// Get settings
	type ConfigGetter interface {
		GetDailyChallenge() (enabled bool, targetPoints int, rewardGems int)
	}
	enabled, targetPoints, rewardGems := true, 50, 10000
	if cfg, ok := c.Config.(ConfigGetter); ok {
		enabled, targetPoints, rewardGems = cfg.GetDailyChallenge()
	}
	
	if !enabled {
		c.Console("`4Daily Challenge is currently disabled by server.``")
		return
	}
	
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wDaily Challenge & Quests``", 23)
	d.AddSpacer("small")
	d.AddTextbox(fmt.Sprintf("Today's Challenge: `wSmash 50 Blocks & Plant 20 Trees``"))
	d.AddTextbox(fmt.Sprintf("Your Progress: `2%d/%d pts``", c.Player.DailyChallengeProgress, targetPoints))
	d.AddSpacer("small")
	if c.Player.DailyChallengeProgress >= targetPoints {
		d.AddButton("claim_daily_challenge", fmt.Sprintf("`2Claim Challenge Trophy & %s Gems!``", formatNumber(rewardGems)))
	} else {
		d.AddSmallText("Keep playing to earn challenge points!")
	}
	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("daily_challenge_dialog", "Close", ""))
}

func HandleDailyChallengeDialogReturn(c *vallenctx.Ctx) {
	if c.Player == nil || c.Button() != "claim_daily_challenge" || c.Config == nil {
		return
	}
	
	type ConfigGetter interface {
		GetDailyChallenge() (enabled bool, targetPoints int, rewardGems int)
	}
	enabled, targetPoints, rewardGems := true, 50, 10000
	if cfg, ok := c.Config.(ConfigGetter); ok {
		enabled, targetPoints, rewardGems = cfg.GetDailyChallenge()
	}
	
	if !enabled || c.Player.DailyChallengeProgress < targetPoints {
		return
	}
	c.Player.Gems += rewardGems
	c.Player.DailyChallengeProgress = 0
	c.Save()
	c.SendRawPacket(variant.SetBux(c.Player.Gems, 0).Pack(), true)
	c.Success("Daily Challenge Completed! Received %s Gems!", formatNumber(rewardGems))
}

func formatNumber(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d", n)
}

func SendBingoDialog(c *vallenctx.Ctx) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wSeasonal Bingo Card``", 32)
	d.AddSpacer("small")
	d.AddTextbox("Complete horizontal, vertical, or diagonal lines to win rare prizes!")
	d.AddSpacer("small")
	d.AddSmallText("1. [X] Log in today")
	d.AddSmallText("2. [X] Visit 3 worlds")
	d.AddSmallText("3. [ ] Harvest 10 trees")
	d.AddSmallText("4. [X] Trade with a friend")
	d.AddSmallText("5. [ ] Perform surgery / play minigame")
	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("bingo_dialog", "Close", ""))
}

func SendWinterRallyDialog(c *vallenctx.Ctx) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wWinter Rally Championship``", 32)
	d.AddSpacer("small")
	d.AddTextbox("Rally checkpoints cleared: `2Stage 3/5``")
	d.AddTextbox("Grand Prize: Winter Rally Sports Car & Trophy!")
	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("winter_rally_dialog", "Close", ""))
}

func SendLeaderboardDialog(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wServer Leaderboard``", 18)
	d.AddSpacer("small")
	d.AddTextbox(fmt.Sprintf("Your Rank Stats: Level `2%d`` | Gems `2%d`` | Bank `2%d WL``", c.Player.Level, c.Player.Gems, c.Player.BankWL))
	d.AddSpacer("small")
	d.AddSmallText("#1. Top Player: Vallen (Level 125, 9,999,999 Gems)")
	d.AddSmallText("#2. Top Guild: Vallen Legion (2,450,000 Points)")
	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("leaderboard_dialog", "Close", ""))
}

func SendEuphoriaDialog(c *vallenctx.Ctx) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wEuphoria Anniversary Party``", 32)
	d.AddSpacer("small")
	d.AddTextbox("Celebrate Growtopia Anniversary with 2x XP and Extra Drop Rate!")
	d.AddTextbox("Active Euphoria Event: `2ENABLED``")
	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("euphoria_dialog", "Close", ""))
}

func SendFruitMixerDialog(c *vallenctx.Ctx) {
	if c.Config == nil {
		return
	}
	
	type ConfigGetter interface {
		GetFruitMixer() (enabled bool, costGems int, rewardID int, rewardCount int)
	}
	enabled, costGems, _, _ := true, 500, 242, 1
	if cfg, ok := c.Config.(ConfigGetter); ok {
		enabled, costGems, _, _ = cfg.GetFruitMixer()
	}
	
	if !enabled {
		c.Console("`4Fruit Mixer is currently disabled by server.``")
		return
	}
	
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wTropical Fruit Mixer (Pineapple Party)``", 32)
	d.AddSpacer("small")
	d.AddTextbox("Mix exotic fruits to create tasty tropical smoothies and get rare prizes!")
	d.AddSpacer("small")
	d.AddButton("mix_fruits", fmt.Sprintf("`2Mix Tropical Fruits (Cost: %d Gems)``", costGems))
	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("fruit_mixer_dialog", "Close", ""))
}

func HandleFruitMixerDialogReturn(c *vallenctx.Ctx) {
	if c.Player == nil || c.Button() != "mix_fruits" || c.Config == nil {
		return
	}
	
	type ConfigGetter interface {
		GetFruitMixer() (enabled bool, costGems int, rewardID int, rewardCount int)
	}
	enabled, costGems, rewardID, rewardCount := true, 500, 242, 1
	if cfg, ok := c.Config.(ConfigGetter); ok {
		enabled, costGems, rewardID, rewardCount = cfg.GetFruitMixer()
	}
	
	if !enabled {
		c.Error("Fruit Mixer is disabled!")
		return
	}
	if c.Player.Gems < costGems {
		c.Error("You need at least %d Gems to mix fruits!", costGems)
		return
	}
	c.Player.Gems -= costGems
	c.AddItem(rewardID, rewardCount)
	c.SendRawPacket(variant.SetBux(c.Player.Gems, 0).Pack(), true)
	itemName := fmt.Sprintf("Item #%d", rewardID)
	if rewardID == 242 {
		itemName = "World Lock"
	}
	c.Success("Mixed delicious Tropical Smoothie! Received +%d %s!", rewardCount, itemName)
}

func SendSeasonalEventDialog(c *vallenctx.Ctx, action string) {
	name, ok := SuperMainActions[action]
	if !ok {
		name = "Seasonal Event"
	}
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", fmt.Sprintf("`w%s``", name), 32)
	d.AddSpacer("small")
	d.AddTextbox(fmt.Sprintf("Welcome to %s! Special holiday quests and bonuses are active.", name))
	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("seasonal_event_dialog", "Close", ""))
}

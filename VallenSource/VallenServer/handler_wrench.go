package server

/*
#include "../../enet/enet.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	dialog "gtps/VallenSource/VallenDialog"
	items "gtps/VallenSource/VallenItems"
	player "gtps/VallenSource/VallenPlayer"
	role "gtps/VallenSource/VallenRole"
	wardrobe "gtps/VallenSource/VallenWardrobe"
)

// ─────────────────────────────────────────────
// Wrench Action Handler (Wrenching Avatar)
// ─────────────────────────────────────────────

// handleWrench menangani paket action|wrench dari client (menggunakan Ctx).
func (s *Server) handleWrench(c *Ctx) {
	if c.Player == nil || c.World == nil {
		return
	}

	targetNetID := c.Int("netid")
	if targetNetID == 0 {
		return
	}

	// Cari target peer di world yang sama
	var targetPeer *C.ENetPeer
	var targetState *peerState

	s.peersMu.RLock()
	for p, st := range s.peers {
		if st != nil && st.currentWorld == c.State.currentWorld && st.netID == targetNetID {
			targetPeer = p
			targetState = st
			break
		}
	}
	s.peersMu.RUnlock()

	if targetState == nil || targetState.p == nil {
		return
	}
	if targetState.netID != c.State.netID && !canReachPosition(c.State, targetState.posX, targetState.posY, 3) {
		return
	}

	// 1. Wrench Diri Sendiri
	if targetState.netID == c.State.netID {
		s.sendSelfWrenchMenu(c)
		return
	}

	// 2. Wrench Player Lain
	s.sendOtherPlayerWrenchMenu(c, targetPeer, targetState)
}

// sendSelfWrenchMenu menampilkan menu profil dan kustomisasi diri sendiri (100% English RGT).
func (s *Server) sendSelfWrenchMenu(c *Ctx) {
	p := c.Player
	r := role.GetRole(p.Role)

	titleStr := ""
	if p.Title != "" {
		titleStr = fmt.Sprintf("add_textbox|`oTitle: `^%s``|\n", p.Title)
	}

	bioStr := ""
	if p.Bio != "" {
		bioStr = fmt.Sprintf("add_textbox|`oBio: `7\"%s\"``|\n", strings.ReplaceAll(p.Bio, "|", ""))
	}

	accountAge := 1
	if p.CreatedAt > 0 {
		days := int((time.Now().Unix() - p.CreatedAt) / 86400)
		if days > 0 {
			accountAge = days
		}
	}

	dialogStr := fmt.Sprintf(
		"set_default_color|`o\n"+
			"add_popup_name|WrenchMenu|\n"+
			"embed_data|netID|%d\n"+
			"add_player_info|%s%s``|%d|%d|%d|\n"+
			"add_achieve|0|\n"+
			"add_label|small|`1Achievements:`` 0/173|left|\n"+
			"add_label|small|`1Account Age:`` %d days|left|\n"+
			"add_spacer|small|\n"+
			"add_button|renew_pvp_license|Get Card Battle License|noflags|0|0|\n"+
			"add_spacer|small|\n"+
			"set_custom_spacing|5|10|\n"+
			"add_custom_button|open_personlize_profile|image:interface/large/gui_wrench_personalize_profile.rttex;image_size:400,260;width:0.19;|\n"+
			"add_custom_button|set_online_status|image:interface/large/gui_wrench_online_status_1green.rttex;image_size:400,260;width:0.19;|\n"+
			"add_custom_button|billboard_edit|image:interface/large/gui_wrench_edit_billboard.rttex;image_size:400,260;width:0.19;|\n"+
			"add_custom_button|wardrobe_customization|image:interface/large/gui_wrench_wardrobe.rttex;image_size:400,260;width:0.19;|\n"+
			"add_custom_button|seed_diary_customization|image:interface/large/gui_wrench_seed_diary.rttex;image_size:400,260;width:0.19;|\n"+
			"add_custom_button|notebook_edit|image:interface/large/gui_wrench_notebook.rttex;image_size:400,260;width:0.19;|\n"+
			"add_custom_button|goals|image:interface/large/gui_wrench_goals_quests.rttex;image_size:400,260;width:0.19;|\n"+
			"add_custom_button|bonus|image:interface/large/gui_wrench_daily_bonus_active.rttex;image_size:400,260;width:0.19;|\n"+
			"add_custom_button|my_worlds|image:interface/large/gui_wrench_my_worlds.rttex;image_size:400,260;width:0.19;|\n"+
			"add_custom_button|alist|image:interface/large/gui_wrench_achievements.rttex;image_size:400,260;width:0.19;|\n"+
			"add_custom_label|(0/173)|target:alist;top:0.72;left:0.5;size:small|\n"+
			"add_custom_button|emojis|image:interface/large/gui_wrench_growmojis.rttex;image_size:400,260;width:0.19;|\n"+
			"add_custom_button|marvelous_missions|image:interface/large/gui_wrench_marvelous_missions.rttex;image_size:400,260;width:0.19;|\n"+
			"add_custom_button|title_edit|image:interface/large/gui_wrench_title.rttex;image_size:400,260;width:0.19;|\n"+
			"add_custom_button|trade_scan|image:interface/large/gui_wrench_trades.rttex;image_size:400,260;width:0.19;|\n"+
			"add_custom_button|pets|image:interface/large/gui_wrench_battle_pets.rttex;image_size:400,260;width:0.19;|\n"+
			"add_custom_button|wrench_customization|image:interface/large/gui_wrench_customization.rttex;image_size:400,260;width:0.19;|\n"+
			"add_custom_button|open_worldlock_storage|image:interface/large/gui_wrench_auction.rttex;image_size:400,260;width:0.19;|\n"+
			"add_custom_break|\n"+
			"add_spacer|small|\n"+
			"set_custom_spacing|0|0|\n"+
			"add_textbox|`oRole: %s%s %s%s`` (AdminLevel: %d)|\n"+
			"%s"+
			"%s"+
			"add_textbox|`oGems: `2%d``  |  Level: `2%d`` (XP: %d)|\n"+
			"add_textbox|`oWorld Lock Bank: `2%d WL``|\n"+
			"add_spacer|small|\n"+
			"add_textbox|`oBackpack slots: `w%d`` slots.|\n"+
			"add_textbox|`oCurrent world: `w%s`` (X: %d, Y: %d)|\n"+
			"add_spacer|small|\n"+
			"add_quick_exit|\n"+
			"end_dialog|popup||Continue|\n",
		c.State.netID,
		r.NameColor, c.Player.GrowID,
		p.Level, p.XP, p.Level*100,
		accountAge,
		r.NameColor, r.Name, r.TagColor, r.Tag, p.AdminLevel,
		titleStr,
		bioStr,
		p.Gems, p.Level, p.XP,
		p.BankWL,
		p.SlotSize,
		c.World.Name, int(c.State.posX/32), int(c.State.posY/32),
	)

	c.Dialog(dialogStr)
}

// sendOtherPlayerWrenchMenu menampilkan profil player lain (100% English RGT).
func (s *Server) sendOtherPlayerWrenchMenu(c *Ctx, targetPeer *C.ENetPeer, targetState *peerState) {
	tp := targetState.p
	tr := role.GetRole(tp.Role)

	titleStr := ""
	if tp.Title != "" {
		titleStr = fmt.Sprintf("add_textbox|`oTitle: `^%s``|\n", tp.Title)
	}

	bioStr := ""
	if tp.Bio != "" {
		bioStr = fmt.Sprintf("add_textbox|`oBio: `7\"%s\"``|\n", strings.ReplaceAll(tp.Bio, "|", ""))
	}

	accountAge := 1
	if tp.CreatedAt > 0 {
		days := int((time.Now().Unix() - tp.CreatedAt) / 86400)
		if days > 0 {
			accountAge = days
		}
	}

	dialogStr := fmt.Sprintf(
		"set_default_color|`o\n"+
			"add_popup_name|WrenchMenu|\n"+
			"embed_data|netID|%d\n"+
			"add_label_with_icon|big|%s%s `2(Level %d)``|left|18|\n"+
			"add_achieve|0|\n"+
			"add_label|small|`1Achievements:`` 0/173|left|\n"+
			"add_label|small|`1Account Age:`` %d days|left|\n"+
			"add_spacer|small|\n"+
			"add_textbox|`oRole: %s%s %s%s``|\n"+
			"%s"+
			"%s"+
			"add_spacer|small|\n"+
			"add_button|trade|`wTrade``|\n"+
			"add_button|sendpm|`wSend Message``|\n"+
			"add_textbox|(No Battle Leash equipped)|\n"+
			"add_textbox|You need a valid license to battle!|\n"+
			"add_button|friend_add|`wAdd as friend``|\n"+
			"add_button|show_clothes|`wView worn clothes``|\n"+
			"add_button|ignore_player|`wIgnore Player``|\n"+
			"add_button|report_player|`wReport Player``|\n",
		targetState.netID,
		tr.NameColor, targetState.growID, tp.Level,
		accountAge,
		tr.NameColor, tr.Name, tr.TagColor, tr.Tag,
		titleStr,
		bioStr,
	)

	isStaff := c.Player.CanAccess(role.LevelEliteGuardian) || s.config.IsOwner(c.Player.GrowID)
	if c.IsWorldOwner() || isStaff {
		dialogStr += "add_button|pull_player|`5Pull``|\n"
		dialogStr += "add_button|kick_player|`4Kick``|\n"
		dialogStr += "add_button|ban_player|`4Ban from World``|\n"
	}

	if isStaff {
		dialogStr += "add_button|admin_grole|`4Manage Role (/grole)``|\n"
	}

	dialogStr += "add_spacer|small|\nadd_quick_exit|\nend_dialog|popup||Continue|\n"

	c.Dialog(dialogStr)
}

// sendPlayerStatusDialog menampilkan pakaian yang sedang dikenakan oleh player target (100% English RGT).
func (s *Server) sendPlayerStatusDialog(c *Ctx, target *player.Player) {
	if target == nil {
		return
	}
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", fmt.Sprintf("`wWorn Clothes of %s``", target.GrowID), 18)
	d.AddSpacer("small")

	wornCount := 0
	for slot := 0; slot < 10; slot++ {
		itemID := target.Clothing[slot]
		slotName := wardrobe.SlotNames[slot]
		if itemID > 0 {
			wornCount++
			itemName := fmt.Sprintf("Item #%d", itemID)
			if itm := items.GetItem(uint16(itemID)); itm != nil && itm.Name != "" {
				itemName = itm.Name
			}
			d.AddLabelWithIcon("small", fmt.Sprintf("`w%s:`` `2%s``", slotName, itemName), itemID)
		} else {
			d.AddSmallText(fmt.Sprintf("`7%s: (None)``", slotName))
		}
	}

	if wornCount == 0 {
		d.AddSpacer("small")
		d.AddSmallText(fmt.Sprintf("`6%s is currently not wearing any clothes.``", target.GrowID))
	}

	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("view_clothes", "", "Continue"))
}

var wrenchPreviewActions = map[string]string{
	"open_personlize_profile":  "Personalize Profile",
	"set_online_status":        "Online Status",
	"wardrobe_customization":   "Wardrobe",
	"seed_diary_customization": "Seed Diary",
	"notebook_edit":            "Notebook",
	"goals":                    "Goals and Quests",
	"bonus":                    "Daily Bonus",
	"my_worlds":                "My Worlds",
	"alist":                    "Achievements",
	"emojis":                   "Growmojis",
	"marvelous_missions":       "Marvelous Missions",
	"title_edit":               "Titles",
	"open_worldlock_storage":   "World Lock Storage",
}

func (s *Server) sendWrenchPreviewDialog(peer *C.ENetPeer, state *peerState, action string) {
	title, ok := wrenchPreviewActions[action]
	if !ok || state == nil || state.p == nil {
		return
	}
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", fmt.Sprintf("`w%s``", title), 32)
	d.AddSpacer("small")
	d.AddTextbox("This feature menu is ready. Its permanent game system will be added next.")
	if action == "my_worlds" {
		d.AddTextbox(fmt.Sprintf("Recent worlds saved: `2%d``", len(state.p.RecentWorlds)))
	}
	d.AddSmallText("No items, gems, or account data were changed.")
	d.AddQuickExit()
	s.sendDialog(peer, d.EndDialog("wrench_preview", "", "Continue"))
}

// ─────────────────────────────────────────────
// Self Wrench Customization & Profiles
// ─────────────────────────────────────────────

// sendBillboardEditDialog menampilkan dialog edit bio/billboard profil pribadi
func (s *Server) sendBillboardEditDialog(c *Ctx) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wEdit Personal Billboard / Bio``", 18)
	d.AddSpacer("small")
	d.AddTextbox("Write a personal message or bio. Other players will see this when they wrench your profile!")
	d.AddSpacer("small")
	d.AddTextInput("bio_text", "Personal Bio:", c.Player.Bio, 128)
	d.AddSpacer("small")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("billboard_edit", "Cancel", "Save Bio"))
}

// handleBillboardEditReturn memproses perubahan bio/billboard pribadi
func (s *Server) handleBillboardEditReturn(c *Ctx) {
	if c.Player == nil {
		return
	}
	btn := c.Button()
	if btn != "" && btn != "Save Bio" {
		return
	}
	bio := strings.TrimSpace(c.Str("bio_text"))
	bio = strings.ReplaceAll(bio, "|", "/")
	if len(bio) > 128 {
		bio = bio[:128]
	}
	c.Player.Bio = bio
	c.Save()
	c.Success("Personal Billboard / Bio updated successfully!")
	s.sendSelfWrenchMenu(c)
}

type WrenchSkin struct {
	ID   int
	Name string
	Icon int
}

var availableWrenchSkins = []WrenchSkin{
	{ID: 32, Name: "Classic Golden Wrench", Icon: 32},
	{ID: 14360, Name: "Prismatic Wrench", Icon: 14360},
	{ID: 14492, Name: "Shiny Wrench", Icon: 14492},
	{ID: 14496, Name: "Wrecked Wrench", Icon: 14496},
	{ID: 14500, Name: "Fresh Wrench", Icon: 14500},
	{ID: 14504, Name: "Beautiful Wrench", Icon: 14504},
	{ID: 14824, Name: "Shocking Wrench", Icon: 14824},
	{ID: 14560, Name: "Musical Wrench", Icon: 14560},
	{ID: 14714, Name: "Runic Wrench", Icon: 14714},
	{ID: 14726, Name: "Mechanical Wrench", Icon: 14726},
	{ID: 15014, Name: "Icy Wrench", Icon: 15014},
	{ID: 9882, Name: "Dark Void Wrench", Icon: 9882},
}

// sendWrenchCustomizationDialog menampilkan dialog kustomisasi skin wrench
func (s *Server) sendWrenchCustomizationDialog(c *Ctx) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wWrench Customization``", 32)
	d.AddSpacer("small")

	currentSkinName := "Classic Golden Wrench"
	currentIcon := 32
	for _, ws := range availableWrenchSkins {
		if ws.ID == c.Player.WrenchStyle {
			currentSkinName = ws.Name
			currentIcon = ws.Icon
			break
		}
	}

	d.AddLabelWithIcon("small", fmt.Sprintf("`oCurrently Equipped: `2%s``", currentSkinName), currentIcon)
	d.AddSpacer("small")
	d.AddTextbox("`oChoose a custom wrench style for your avatar:``")
	d.AddSpacer("small")

	for _, ws := range availableWrenchSkins {
		tag := ""
		if ws.ID == c.Player.WrenchStyle || (c.Player.WrenchStyle == 0 && ws.ID == 32) {
			tag = " `2[EQUIPPED]``"
		}
		d.AddButton(fmt.Sprintf("wskin_%d", ws.ID), fmt.Sprintf("`w%s%s``", ws.Name, tag))
	}

	d.AddSpacer("small")
	d.AddButton("btn_back_self", "Back to Menu")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("wrench_custom_dialog", "", ""))
}

// handleWrenchCustomizationReturn memproses pemilihan skin wrench
func (s *Server) handleWrenchCustomizationReturn(c *Ctx) {
	if c.Player == nil {
		return
	}
	btn := c.Button()
	if btn == "btn_back_self" || btn == "" {
		s.sendSelfWrenchMenu(c)
		return
	}
	if strings.HasPrefix(btn, "wskin_") {
		idStr := strings.TrimPrefix(btn, "wskin_")
		skinID, _ := strconv.Atoi(idStr)
		if skinID > 0 {
			c.Player.WrenchStyle = skinID
			c.Save()
			c.Sound("audio/secret.wav")
			c.Success("Wrench style changed successfully!")
		}
	}
	s.sendWrenchCustomizationDialog(c)
}

// sendTradeScanDialog menampilkan scanner keamanan dan riwayat trade terakhir
func (s *Server) sendTradeScanDialog(c *Ctx) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wTrade Scan & Security Monitor``", 1366)
	d.AddSpacer("small")

	d.AddLabelWithIcon("small", "`2Account Status: `wVERIFIED & SECURE``", 242)
	d.AddSmallText("`oTrust Rating: `2100% (No negative scam reports recorded)``")
	d.AddSmallText("`oTwo-Factor Trade Verification: `2ENABLED``")
	d.AddSpacer("small")

	d.AddTextbox("`wRecent Trade Activity:``")
	if len(c.Player.TradeHistory) == 0 {
		d.AddSmallText("`7No trades on record yet. Transactions will appear here automatically.``")
	} else {
		start := 0
		if len(c.Player.TradeHistory) > 8 {
			start = len(c.Player.TradeHistory) - 8
		}
		for i := len(c.Player.TradeHistory) - 1; i >= start; i-- {
			d.AddSmallText(fmt.Sprintf("`2- %s``", c.Player.TradeHistory[i]))
		}
	}

	d.AddSpacer("small")
	d.AddButton("btn_back_self", "Back")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("trade_scan_dialog", "", ""))
}

// sendBattlePetLicenseDialog menampilkan status Card Battle License & Battle Pets
func (s *Server) sendBattlePetLicenseDialog(c *Ctx) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wCard Battle License & Battle Pets``", 2480)
	d.AddSpacer("small")

	d.AddLabelWithIcon("small", "`2Card Battle License: `wACTIVE (Level 1)``", 2480)
	d.AddSmallText("`oDuel Ranking: `wNovice (0 RP)``")
	d.AddSmallText("`oPVP Battle Passes: `wUNLIMITED``")
	d.AddSpacer("small")

	d.AddTextbox("`wBattle Pet Status:``")
	d.AddSmallText("`7No battle pet currently equipped. Obtain a Battle Leash in Store or Dungeons to summon your pet!``")
	d.AddSpacer("small")

	d.AddButton("btn_back_self", "Back")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("battle_pets_dialog", "", ""))
}

// ─────────────────────────────────────────────
// Other Player Wrench Actions (PM & Report)
// ─────────────────────────────────────────────

// sendPMDialog menampilkan form kirim Private Message ke player lain
func (s *Server) sendPMDialog(c *Ctx, targetNetID int) {
	targetPeer, targetState := s.findPeerByNetID(targetNetID)
	if targetPeer == nil || targetState == nil {
		c.Error("Player is no longer online.")
		return
	}

	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", fmt.Sprintf("`wSend Message to %s``", targetState.growID), 32)
	d.AddSpacer("small")
	d.EmbedData("targetNetID", targetNetID)
	d.AddTextInput("pm_message", "Message:", "", 128)
	d.AddSpacer("small")
	d.AddButton("btn_send_pm", "`2Send Message``")
	d.AddButton("btn_cancel_pm", "Cancel")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("send_pm_dialog", "", ""))
}

// handlePMSendReturn memproses pengiriman Private Message
func (s *Server) handlePMSendReturn(c *Ctx) {
	btn := c.Button()
	if btn != "btn_send_pm" {
		return
	}

	targetNetID := c.Int("targetNetID")
	msg := strings.TrimSpace(c.Str("pm_message"))
	if msg == "" {
		return
	}

	targetPeer, targetState := s.findPeerByNetID(targetNetID)
	if targetPeer == nil || targetState == nil {
		c.Error("Player is no longer online.")
		return
	}

	// Format pesan whisper / PM
	pmForTarget := fmt.Sprintf("`5[PM from %s]: `o%s``", c.Player.GrowID, msg)
	pmForSender := fmt.Sprintf("`5[PM to %s]: `o%s``", targetState.growID, msg)

	s.sendConsole(targetPeer, pmForTarget)
	c.Console(pmForSender)

	s.playWorldSound(c.World.Name, "audio/hub_open.wav", c.State.posX, c.State.posY)
	c.Success("Message sent to %s.", targetState.growID)
}

// sendReportPlayerDialog menampilkan formulir pelaporan player
func (s *Server) sendReportPlayerDialog(c *Ctx, targetNetID int) {
	targetPeer, targetState := s.findPeerByNetID(targetNetID)
	if targetPeer == nil || targetState == nil {
		c.Error("Player is no longer online.")
		return
	}

	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", fmt.Sprintf("`4Report Player: %s``", targetState.growID), 18)
	d.AddSpacer("small")
	d.EmbedData("targetNetID", targetNetID)
	d.AddTextbox("Please select the violation category:")
	d.AddRadioButton("report_category", "cat_scam", "Scamming or Attempted Fraud", true)
	d.AddRadioButton("report_category", "cat_harass", "Harassment, Toxicity, or Vulgar Language", false)
	d.AddRadioButton("report_category", "cat_bot", "Illegal Automation, Cheating, or Botting", false)
	d.AddRadioButton("report_category", "cat_other", "Inappropriate World, Hate Speech, or Other", false)
	d.AddSpacer("small")
	d.AddTextInput("report_details", "Additional Details:", "", 128)
	d.AddSpacer("small")
	d.AddButton("btn_submit_report", "`4Submit Report``")
	d.AddButton("btn_cancel_report", "Cancel")
	d.AddQuickExit()
	c.Dialog(d.EndDialog("report_player_dialog", "", ""))
}

// handleReportPlayerReturn memproses formulir laporan player
func (s *Server) handleReportPlayerReturn(c *Ctx) {
	btn := c.Button()
	if btn != "btn_submit_report" {
		return
	}

	targetNetID := c.Int("targetNetID")
	targetPeer, targetState := s.findPeerByNetID(targetNetID)
	targetName := "Unknown"
	if targetState != nil {
		targetName = targetState.growID
	}
	_ = targetPeer

	category := c.Str("report_category")
	categoryName := "Scamming"
	switch category {
	case "cat_harass":
		categoryName = "Harassment / Toxic"
	case "cat_bot":
		categoryName = "Cheating / Botting"
	case "cat_other":
		categoryName = "Inappropriate Behavior"
	}

	details := strings.TrimSpace(c.Str("report_details"))

	// Broadcast alert ke seluruh Staff online
	alertMsg := fmt.Sprintf("`4[REPORT ALERT] `w%s `oreported `w%s `ofor `4%s``! (World: `w%s``) Details: `7%s``",
		c.Player.GrowID, targetName, categoryName, c.World.Name, details)

	s.peersMu.RLock()
	for p, st := range s.peers {
		if st != nil && st.p != nil && (st.p.AdminLevel >= role.LevelEliteGuardian || s.config.IsOwner(st.growID)) {
			s.sendConsole(p, alertMsg)
		}
	}
	s.peersMu.RUnlock()

	c.Success("Report submitted against %s. Thank you for keeping Vallen safe!", targetName)
}

func (s *Server) sendTradeScaffold(peer *C.ENetPeer, state *peerState, target *player.Player) {
	if state == nil || state.p == nil || target == nil {
		return
	}
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", fmt.Sprintf("`wTrade with %s``", target.GrowID), 32)
	d.AddSpacer("small")
	d.AddTextbox("Trading is not enabled yet. Your items and gems are safe.")
	d.AddQuickExit()
	s.sendDialog(peer, d.EndDialog("trade", "", "Continue"))
}

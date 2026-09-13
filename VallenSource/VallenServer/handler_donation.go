package server

/*
#include "../../enet/enet.h"
*/
import "C"

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	dialog "gtps/VallenSource/VallenDialog"
	items "gtps/VallenSource/VallenItems"
	role "gtps/VallenSource/VallenRole"
	world "gtps/VallenSource/VallenWorld"
)

func isDonationItem(id int) bool {
	return id == 1452 || id == 2814
}

// handleDonationWrench menangani interaksi wrench pada Donation Box
func (s *Server) handleDonationWrench(peer *C.ENetPeer, state *peerState, w *world.World, tileX, tileY int) {
	if state == nil || state.p == nil || w == nil {
		return
	}

	box := w.FindDonationAt(tileX, tileY)
	if box == nil {
		box = &world.DonationBox{
			TileX: tileX,
			TileY: tileY,
		}
		w.SetDonation(*box)
		box = w.FindDonationAt(tileX, tileY)
	}

	canEdit := w.CanEditTile(state.p.UserID, tileX, tileY, state.p.CanAccess(role.LevelEliteGuardian))
	if canEdit {
		s.sendDonationOwnerDialog(peer, state, w, box)
	} else {
		s.sendDonationVisitorDialog(peer, state, w, box)
	}
}

// sendDonationOwnerDialog menampilkan menu pemilik donation box
func (s *Server) sendDonationOwnerDialog(peer *C.ENetPeer, state *peerState, w *world.World, box *world.DonationBox) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wDonation Box (Owner Menu)``", 1452)
	d.AddSpacer("small")
	d.EmbedData("tilex", box.TileX)
	d.EmbedData("tiley", box.TileY)

	if len(box.Donations) == 0 {
		d.AddTextbox("`oNo donations received yet. People can donate items to you here!``")
	} else {
		d.AddTextbox(fmt.Sprintf("`2Received %d donation(s):``", len(box.Donations)))
		d.AddSpacer("small")

		// Tampilkan 10 donasi terbaru
		start := 0
		if len(box.Donations) > 10 {
			start = len(box.Donations) - 10
		}
		for i := len(box.Donations) - 1; i >= start; i-- {
			don := box.Donations[i]
			name := fmt.Sprintf("Item #%d", don.ItemID)
			if def := items.GetItem(uint16(don.ItemID)); def != nil && def.Name != "" {
				name = def.Name
			}
			msgStr := ""
			if don.Message != "" {
				msgStr = fmt.Sprintf(" - `7\"%s\"``", don.Message)
			}
			d.AddLabelWithIcon("small", fmt.Sprintf("`w%s: `2%d %s``%s", don.DonatorName, don.Count, name, msgStr), don.ItemID)
		}

		d.AddSpacer("small")
		d.AddButton("btn_claim_donations", "`2Claim All Donated Items``")
		d.AddButton("btn_clear_log", "`4Clear Donation History``")
	}

	d.AddSpacer("small")
	d.AddButton("btn_close", "Close")
	d.AddQuickExit()
	s.sendDialog(peer, d.EndDialog("donation_owner", "", ""))
}

// sendDonationVisitorDialog menampilkan menu donasi untuk pengunjung
func (s *Server) sendDonationVisitorDialog(peer *C.ENetPeer, state *peerState, w *world.World, box *world.DonationBox) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wDonation Box``", 1452)
	d.AddSpacer("small")
	d.EmbedData("tilex", box.TileX)
	d.EmbedData("tiley", box.TileY)

	d.AddTextbox("`oLeave a gift and a friendly message for the owner of this world!``")
	d.AddSpacer("small")

	d.AddButton("btn_pick_donation", "`w+ Choose Item to Donate``")
	d.AddSpacer("small")
	d.AddButton("btn_close", "Cancel")
	d.AddQuickExit()
	s.sendDialog(peer, d.EndDialog("donation_visitor", "", ""))
}

// handleDonationOwnerReturn memproses aksi pemilik donation box
func (s *Server) handleDonationOwnerReturn(c *Ctx) {
	if c.Player == nil || c.World == nil {
		return
	}

	tileX, tileY := c.Int("tilex"), c.Int("tiley")
	box := c.World.FindDonationAt(tileX, tileY)
	if box == nil {
		return
	}

	btn := c.Button()

	if btn == "btn_claim_donations" {
		if len(box.Donations) == 0 {
			return
		}
		totalClaimed := 0
		for _, don := range box.Donations {
			c.Player.AddItem(don.ItemID, don.Count)
			totalClaimed += don.Count
		}
		box.Donations = nil
		c.Save()
		c.SaveWorld()
		c.SyncInventory()
		c.Sound("audio/trade_complete.wav")
		c.Success("Claimed all donations into your backpack! (%d items)", totalClaimed)
		s.sendDonationOwnerDialog(c.Peer, c.State, c.World, box)
		return
	}

	if btn == "btn_clear_log" {
		box.Donations = nil
		c.SaveWorld()
		c.Success("Donation log cleared.")
		s.sendDonationOwnerDialog(c.Peer, c.State, c.World, box)
		return
	}
}

// sendDonationPickList menampilkan inventory untuk dipilih donasi
func (s *Server) sendDonationPickList(peer *C.ENetPeer, state *peerState, tileX, tileY int) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wSelect Item to Donate``", 1452)
	d.AddSpacer("small")
	d.EmbedData("tilex", tileX)
	d.EmbedData("tiley", tileY)

	count := 0
	for _, itm := range state.p.Inventory {
		if itm.ID <= 0 || itm.Count <= 0 {
			continue
		}
		name := fmt.Sprintf("Item #%d", itm.ID)
		if def := items.GetItem(uint16(itm.ID)); def != nil && def.Name != "" {
			name = def.Name
		}
		d.AddButton(fmt.Sprintf("donate_%d", itm.ID), fmt.Sprintf("`w%s `2(You have %d)``", name, itm.Count))
		count++
	}

	if count == 0 {
		d.AddSmallText("`7No items in backpack to donate!``")
	}

	d.AddSpacer("small")
	d.AddButton("btn_cancel", "Cancel")
	d.AddQuickExit()
	s.sendDialog(peer, d.EndDialog("donation_pick", "", ""))
}

// handleDonationPickReturn memproses pemilihan item donasi
func (s *Server) handleDonationPickReturn(c *Ctx) {
	btn := c.Button()
	tileX, tileY := c.Int("tilex"), c.Int("tiley")

	if strings.HasPrefix(btn, "donate_") {
		idStr := strings.TrimPrefix(btn, "donate_")
		itemID, _ := strconv.Atoi(idStr)
		owned := c.Player.GetItemCount(itemID)
		if itemID > 0 && owned > 0 {
			name := fmt.Sprintf("Item #%d", itemID)
			if def := items.GetItem(uint16(itemID)); def != nil && def.Name != "" {
				name = def.Name
			}
			d := dialog.New().SetDefaultColor("`o")
			d.AddLabelWithIcon("big", fmt.Sprintf("`wDonate %s``", name), itemID)
			d.AddSpacer("small")
			d.EmbedData("tilex", tileX)
			d.EmbedData("tiley", tileY)
			d.EmbedData("don_item_id", itemID)
			d.AddTextInput("don_count", "Amount to Donate:", "1", 5)
			d.AddTextInput("don_msg", "Message / Note (optional):", "Nice world!", 64)
			d.AddSpacer("small")
			d.AddButton("btn_confirm_donate", "`2Donate Now!``")
			d.AddButton("btn_cancel", "Cancel")
			d.AddQuickExit()
			c.Dialog(d.EndDialog("donation_confirm", "", ""))
			return
		}
	}
}

// handleDonationConfirmReturn memproses transfer item donasi
func (s *Server) handleDonationConfirmReturn(c *Ctx) {
	if c.Button() != "btn_confirm_donate" {
		return
	}

	tileX, tileY := c.Int("tilex"), c.Int("tiley")
	box := c.World.FindDonationAt(tileX, tileY)
	if box == nil {
		return
	}

	itemID := c.Int("don_item_id")
	count := c.Int("don_count")
	msg := strings.TrimSpace(c.Str("don_msg"))
	msg = strings.ReplaceAll(msg, "|", "/")
	if len(msg) > 64 {
		msg = msg[:64]
	}

	owned := c.Player.GetItemCount(itemID)
	if itemID <= 0 || count <= 0 || owned < count {
		c.Error("Invalid donation amount!")
		return
	}

	c.Player.RemoveItem(itemID, count)
	record := world.DonationRecord{
		DonatorName: c.Player.GrowID,
		ItemID:      itemID,
		Count:       count,
		Message:     msg,
		DonatedAt:   time.Now().Format("2006-01-02 15:04"),
	}
	box.Donations = append(box.Donations, record)

	c.Save()
	c.SaveWorld()
	c.SyncInventory()

	itemName := fmt.Sprintf("Item #%d", itemID)
	if def := items.GetItem(uint16(itemID)); def != nil && def.Name != "" {
		itemName = def.Name
	}

	c.Sound("audio/object_spawn.wav")
	c.Success("Thank you! You donated %d %s!", count, itemName)
	c.Bubble("`2[Donated %d %s!]``", count, itemName)
}

package server

/*
#include "../../enet/enet.h"
*/
import "C"

import (
	"fmt"
	"strconv"
	"strings"

	"gtps/VallenSource/VallenDialog"
	"gtps/VallenSource/VallenItems"
	"gtps/VallenSource/VallenVariant"
	"gtps/VallenSource/VallenWorld"
)

func isVendingItem(id int) bool {
	return id == 2978 || id == 4424 || id == 9266
}

func isDisplayItem(id int) bool {
	return id == 1422 || id == 2488
}

// handleVendingWrench menangani saat Vending Machine di-wrench
func (s *Server) handleVendingWrench(peer *C.ENetPeer, state *peerState, w *world.World, tileX, tileY int) {
	if state == nil || state.p == nil || w == nil {
		return
	}

	tile := w.GetTile(tileX, tileY)
	if tile == nil || !isVendingItem(tile.Foreground) {
		return
	}

	vend := w.FindVendingAt(tileX, tileY)
	if vend == nil {
		vend = &world.VendingMachine{
			TileX: tileX,
			TileY: tileY,
		}
		w.SetVending(*vend)
		vend = w.FindVendingAt(tileX, tileY)
	}

	canEdit := w.CanEditTile(state.p.UserID, tileX, tileY, state.p.CanAccess(9))
	if canEdit {
		s.sendVendingOwnerDialog(peer, state, w, vend)
	} else {
		s.sendVendingBuyerDialog(peer, state, w, vend)
	}
}

// sendVendingOwnerDialog menampilkan menu kelola vending untuk pemilik
func (s *Server) sendVendingOwnerDialog(peer *C.ENetPeer, state *peerState, w *world.World, vend *world.VendingMachine) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wVending Machine (Owner Menu)``", 2978)
	d.AddSpacer("small")

	d.EmbedData("tilex", strconv.Itoa(vend.TileX))
	d.EmbedData("tiley", strconv.Itoa(vend.TileY))

	if vend.ItemID <= 0 || vend.Count <= 0 {
		d.AddTextbox("`4This machine is currently empty.``")
		d.AddSpacer("small")
		d.AddButton("btn_stock_item", "`w+ Deposit Item from Backpack``")
	} else {
		itemName := fmt.Sprintf("Item #%d", vend.ItemID)
		if def := items.GetItem(uint16(vend.ItemID)); def != nil && def.Name != "" {
			itemName = def.Name
		}
		d.AddLabelWithIcon("small", fmt.Sprintf("`wContains: `2%d %s``", vend.Count, itemName), vend.ItemID)
		d.AddSpacer("small")

		if vend.Price == 0 {
			d.AddTextbox("`ePrice is not set! Machine will not sell until a price is set.``")
		} else if vend.Price > 0 {
			d.AddLabelWithIcon("small", fmt.Sprintf("`wCost: `2%d World Lock(s) per 1 item``", vend.Price), 242)
		} else {
			d.AddLabelWithIcon("small", fmt.Sprintf("`wCost: `21 World Lock per %d items``", -vend.Price), 242)
		}
		d.AddSpacer("small")

		priceVal := strconv.Itoa(vend.Price)
		if vend.Price < 0 {
			priceVal = strconv.Itoa(-vend.Price)
		}
		d.AddTextInput("vend_price", "Price:", priceVal, 5)

		d.AddCheckbox("chk_perlock", "Items per World Lock (Centang jika ingin jual banyak per 1 WL)", vend.Price < 0)
		d.AddSpacer("small")

		d.AddButton("btn_stock_item", "`w+ Deposit More Stock``")
		d.AddButton("btn_empty_vend", "`4Empty Machine (Take all items)``")
	}
	d.AddSpacer("small")

	if vend.LocksEarned > 0 {
		d.AddLabelWithIcon("small", fmt.Sprintf("`2You earned: `w%d World Lock(s)``", vend.LocksEarned), 242)
		d.AddButton("btn_withdraw_locks", fmt.Sprintf("`2Withdraw %d World Lock(s)``", vend.LocksEarned))
		d.AddSpacer("small")
	}

	d.AddButton("btn_update_price", "`2Save Settings``")
	d.AddQuickExit()
	s.sendDialog(peer, d.EndDialog("vending_owner", "Close", ""))
}

// sendVendingBuyerDialog menampilkan menu beli vending untuk pembeli
func (s *Server) sendVendingBuyerDialog(peer *C.ENetPeer, state *peerState, w *world.World, vend *world.VendingMachine) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wVending Machine``", 2978)
	d.AddSpacer("small")

	d.EmbedData("tilex", strconv.Itoa(vend.TileX))
	d.EmbedData("tiley", strconv.Itoa(vend.TileY))

	if vend.ItemID <= 0 || vend.Count <= 0 || vend.Price == 0 {
		d.AddTextbox("`4This machine is currently out of stock or out of order.``")
		d.AddQuickExit()
		s.sendDialog(peer, d.EndDialog("vending_buyer", "Close", ""))
		return
	}

	itemName := fmt.Sprintf("Item #%d", vend.ItemID)
	if def := items.GetItem(uint16(vend.ItemID)); def != nil && def.Name != "" {
		itemName = def.Name
	}

	d.AddLabelWithIcon("small", fmt.Sprintf("`wSelling: `2%s``", itemName), vend.ItemID)
	d.AddSmallText(fmt.Sprintf("`7Stock remaining: `w%d``", vend.Count))
	d.AddSpacer("small")

	if vend.Price > 0 {
		d.AddLabelWithIcon("small", fmt.Sprintf("`wPrice: `2%d World Lock(s) per item``", vend.Price), 242)
	} else {
		d.AddLabelWithIcon("small", fmt.Sprintf("`wPrice: `2%d item(s) per 1 World Lock``", -vend.Price), 242)
	}
	d.AddSpacer("small")

	myWLs := state.p.GetItemCount(242)
	d.AddSmallText(fmt.Sprintf("`oYou have: `w%d World Lock(s)``", myWLs))
	d.AddSpacer("small")

	d.AddTextInput("buy_count", "How many items to buy:", "1", 5)
	d.AddButton("btn_buy_confirm", "`2Buy Now``")
	d.AddButton("btn_close", "Cancel")
	d.AddQuickExit()

	s.sendDialog(peer, d.EndDialog("vending_buyer", "", ""))
}

// handleVendingOwnerReturn memproses input dialog owner
func (s *Server) handleVendingOwnerReturn(peer *C.ENetPeer, state *peerState, values map[string]string) {
	if state == nil || state.p == nil {
		return
	}
	w := s.getOrCreateWorld(state.currentWorld)
	if w == nil {
		return
	}

	tileX, _ := strconv.Atoi(values["tilex"])
	tileY, _ := strconv.Atoi(values["tiley"])
	vend := w.FindVendingAt(tileX, tileY)
	if vend == nil {
		return
	}

	btn := values["buttonClicked"]

	// Withdraw locks
	if btn == "btn_withdraw_locks" {
		if vend.LocksEarned > 0 {
			earned := vend.LocksEarned
			vend.LocksEarned = 0
			state.p.AddItem(242, earned)
			s.db.SavePlayer(state.p)
			s.db.SaveWorld(w)
			s.sendInventory(peer, state)
			s.sendConsole(peer, "`2Withdrew `w%d World Lock(s)`` from vending machine!``", earned)
			s.playWorldSound(w.Name, "audio/trade_complete.wav", float32(tileX*32), float32(tileY*32))
		}
		s.sendVendingOwnerDialog(peer, state, w, vend)
		return
	}

	// Empty machine
	if btn == "btn_empty_vend" {
		if vend.Count > 0 && vend.ItemID > 0 {
			retrieved := vend.Count
			itmID := vend.ItemID
			vend.Count = 0
			vend.ItemID = 0
			vend.Price = 0
			state.p.AddItem(itmID, retrieved)
			s.db.SavePlayer(state.p)
			s.db.SaveWorld(w)
			s.sendInventory(peer, state)

			// Update tile visual agar item melayang hilang
			tile := w.GetTile(tileX, tileY)
			if tile != nil {
				s.sendTileUpdate(w, tileX, tileY, tile)
			}

			s.sendConsole(peer, "`2Retrieved `w%d %s`` from vending machine!``", retrieved, items.GetItem(uint16(itmID)).Name)
		}
		s.sendVendingOwnerDialog(peer, state, w, vend)
		return
	}

	// Stock item dialog
	if btn == "btn_stock_item" {
		s.sendVendingDepositList(peer, state, vend)
		return
	}

	// Save settings / price
	priceStr := values["vend_price"]
	price, _ := strconv.Atoi(priceStr)
	if price < 0 {
		price = 0
	}
	chkPerLock := values["chk_perlock"] == "1"
	if chkPerLock && price > 0 {
		vend.Price = -price
	} else {
		vend.Price = price
	}

	s.db.SaveWorld(w)
	tile := w.GetTile(tileX, tileY)
	if tile != nil {
		s.sendTileUpdate(w, tileX, tileY, tile)
	}
	s.sendConsole(peer, "`2Vending machine settings updated!``")
}

// sendVendingDepositList menampilkan daftar item inventory yang bisa didepositkan
func (s *Server) sendVendingDepositList(peer *C.ENetPeer, state *peerState, vend *world.VendingMachine) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wSelect Item to Deposit``", 2978)
	d.AddSpacer("small")
	d.EmbedData("tilex", strconv.Itoa(vend.TileX))
	d.EmbedData("tiley", strconv.Itoa(vend.TileY))

	count := 0
	for _, itm := range state.p.Inventory {
		if itm.ID <= 0 || itm.Count <= 0 {
			continue
		}
		if itm.ID == 242 || itm.ID == 1796 {
			continue // jangan deposit locks
		}
		// Jika vending sudah punya jenis item, hanya boleh deposit jenis yang sama
		if vend.ItemID > 0 && itm.ID != vend.ItemID {
			continue
		}

		name := fmt.Sprintf("Item #%d", itm.ID)
		if def := items.GetItem(uint16(itm.ID)); def != nil && def.Name != "" {
			name = def.Name
		}
		d.AddButton(fmt.Sprintf("dep_%d", itm.ID), fmt.Sprintf("`w%s `2(You have %d)``", name, itm.Count))
		count++
	}

	if count == 0 {
		d.AddSmallText("`7No matching items available in backpack!``")
	}

	d.AddSpacer("small")
	d.AddButton("btn_back", "Back")
	d.AddQuickExit()
	s.sendDialog(peer, d.EndDialog("vending_deposit", "", ""))
}

// handleVendingDepositReturn memproses pilihan item yang didepositkan
func (s *Server) handleVendingDepositReturn(peer *C.ENetPeer, state *peerState, values map[string]string) {
	w := s.getOrCreateWorld(state.currentWorld)
	if w == nil {
		return
	}

	tileX, _ := strconv.Atoi(values["tilex"])
	tileY, _ := strconv.Atoi(values["tiley"])
	vend := w.FindVendingAt(tileX, tileY)
	if vend == nil {
		return
	}

	btn := values["buttonClicked"]
	if btn == "btn_back" || btn == "" {
		s.sendVendingOwnerDialog(peer, state, w, vend)
		return
	}

	if strings.HasPrefix(btn, "dep_") {
		itemIDStr := strings.TrimPrefix(btn, "dep_")
		itemID, _ := strconv.Atoi(itemIDStr)
		owned := state.p.GetItemCount(itemID)
		if itemID > 0 && owned > 0 {
			// Masukkan seluruh stok yang ada di inventory
			state.p.RemoveItem(itemID, owned)
			vend.ItemID = itemID
			vend.Count += owned
			s.db.SavePlayer(state.p)
			s.db.SaveWorld(w)
			s.sendInventory(peer, state)

			// Update tile visual
			tile := w.GetTile(tileX, tileY)
			if tile != nil {
				s.sendTileUpdate(w, tileX, tileY, tile)
			}

			s.sendConsole(peer, "`2Deposited `w%d %s`` into vending machine!``", owned, items.GetItem(uint16(itemID)).Name)
			s.playWorldSound(w.Name, "audio/object_spawn.wav", float32(tileX*32), float32(tileY*32))
		}
	}

	s.sendVendingOwnerDialog(peer, state, w, vend)
}

// handleVendingBuyerReturn memproses pembelian dari vending
func (s *Server) handleVendingBuyerReturn(peer *C.ENetPeer, state *peerState, values map[string]string) {
	if state == nil || state.p == nil {
		return
	}
	w := s.getOrCreateWorld(state.currentWorld)
	if w == nil {
		return
	}

	tileX, _ := strconv.Atoi(values["tilex"])
	tileY, _ := strconv.Atoi(values["tiley"])
	vend := w.FindVendingAt(tileX, tileY)
	if vend == nil || vend.ItemID <= 0 || vend.Count <= 0 || vend.Price == 0 {
		return
	}

	btn := values["buttonClicked"]
	if btn != "btn_buy_confirm" {
		return
	}

	buyCount, _ := strconv.Atoi(values["buy_count"])
	if buyCount <= 0 {
		s.sendConsole(peer, "`4Invalid amount to buy!``")
		return
	}
	if buyCount > vend.Count {
		s.sendConsole(peer, "`4Not enough stock! Only %d available.``", vend.Count)
		return
	}

	var costWL int
	if vend.Price > 0 {
		costWL = buyCount * vend.Price
	} else {
		itemsPerWL := -vend.Price
		if buyCount%itemsPerWL != 0 {
			s.sendConsole(peer, "`4You must buy in multiples of %d items (1 World Lock's worth)!``", itemsPerWL)
			return
		}
		costWL = buyCount / itemsPerWL
	}

	myWLs := state.p.GetItemCount(242)
	if myWLs < costWL {
		s.sendConsole(peer, "`4You need %d World Lock(s) but you only have %d!``", costWL, myWLs)
		return
	}

	// Eksekusi pembelian
	state.p.RemoveItem(242, costWL)
	vend.LocksEarned += costWL
	vend.Count -= buyCount
	state.p.AddItem(vend.ItemID, buyCount)

	s.db.SavePlayer(state.p)
	s.db.SaveWorld(w)
	s.sendInventory(peer, state)

	// Update tile visual
	tile := w.GetTile(tileX, tileY)
	if tile != nil {
		s.sendTileUpdate(w, tileX, tileY, tile)
	}

	itemName := fmt.Sprintf("Item #%d", vend.ItemID)
	if def := items.GetItem(uint16(vend.ItemID)); def != nil && def.Name != "" {
		itemName = def.Name
	}

	s.sendConsole(peer, "`2Successfully bought `w%d %s`` for `2%d World Lock(s)``!``", buyCount, itemName, costWL)
	s.sendPacket(peer, variant.New("OnTalkBubble", int32(state.netID), fmt.Sprintf("`2[Purchased %d %s!]``", buyCount, itemName)).Pack(), true)
	s.playWorldSound(w.Name, "audio/trade_complete.wav", float32(tileX*32), float32(tileY*32))
}

// handleDisplayWrench menangani saat Display Box / Shelf di-wrench
func (s *Server) handleDisplayWrench(peer *C.ENetPeer, state *peerState, w *world.World, tileX, tileY int) {
	if state == nil || state.p == nil || w == nil {
		return
	}
	tile := w.GetTile(tileX, tileY)
	if tile == nil || !isDisplayItem(tile.Foreground) {
		return
	}

	disp := w.FindDisplayAt(tileX, tileY)
	if disp == nil {
		disp = &world.DisplayBox{
			TileX: tileX,
			TileY: tileY,
		}
		w.SetDisplay(*disp)
		disp = w.FindDisplayAt(tileX, tileY)
	}

	canEdit := w.CanEditTile(state.p.UserID, tileX, tileY, state.p.CanAccess(9))
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", fmt.Sprintf("`w%s``", items.GetItem(uint16(tile.Foreground)).Name), tile.Foreground)
	d.AddSpacer("small")
	d.EmbedData("tilex", strconv.Itoa(tileX))
	d.EmbedData("tiley", strconv.Itoa(tileY))

	if disp.ItemID <= 0 {
		d.AddTextbox("`oThis display is currently empty.``")
		if canEdit {
			d.AddSpacer("small")
			d.AddButton("btn_display_item", "`w+ Put an Item on Display``")
		}
	} else {
		itemName := fmt.Sprintf("Item #%d", disp.ItemID)
		itemRarity := 0
		if def := items.GetItem(uint16(disp.ItemID)); def != nil {
			if def.Name != "" {
				itemName = def.Name
			}
			itemRarity = int(def.Rarity)
		}
		d.AddLabelWithIcon("small", fmt.Sprintf("`wDisplaying: `2%s``", itemName), disp.ItemID)
		d.AddSmallText(fmt.Sprintf("`7Rarity: `w%d``", itemRarity))
		if canEdit {
			d.AddSpacer("small")
			d.AddButton("btn_take_display", "`4Take Item Back``")
		}
	}

	d.AddQuickExit()
	s.sendDialog(peer, d.EndDialog("display_box", "Close", ""))
}

// handleDisplayBoxReturn memproses dialog Display Box
func (s *Server) handleDisplayBoxReturn(peer *C.ENetPeer, state *peerState, values map[string]string) {
	w := s.getOrCreateWorld(state.currentWorld)
	if w == nil {
		return
	}

	tileX, _ := strconv.Atoi(values["tilex"])
	tileY, _ := strconv.Atoi(values["tiley"])
	disp := w.FindDisplayAt(tileX, tileY)
	if disp == nil {
		return
	}

	btn := values["buttonClicked"]
	if btn == "btn_take_display" {
		if disp.ItemID > 0 {
			takeID := disp.ItemID
			disp.ItemID = 0
			state.p.AddItem(takeID, 1)
			s.db.SavePlayer(state.p)
			s.db.SaveWorld(w)
			s.sendInventory(peer, state)

			tile := w.GetTile(tileX, tileY)
			if tile != nil {
				s.sendTileUpdate(w, tileX, tileY, tile)
			}
			s.sendConsole(peer, "`2Took %s from display box.``", items.GetItem(uint16(takeID)).Name)
		}
		s.handleDisplayWrench(peer, state, w, tileX, tileY)
		return
	}

	if btn == "btn_display_item" {
		s.sendDisplayDepositList(peer, state, disp)
		return
	}
}

// sendDisplayDepositList menampilkan inventory untuk dipajang di display box
func (s *Server) sendDisplayDepositList(peer *C.ENetPeer, state *peerState, disp *world.DisplayBox) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wSelect Item to Display``", 1422)
	d.AddSpacer("small")
	d.EmbedData("tilex", strconv.Itoa(disp.TileX))
	d.EmbedData("tiley", strconv.Itoa(disp.TileY))

	for _, itm := range state.p.Inventory {
		if itm.ID <= 0 || itm.Count <= 0 {
			continue
		}
		name := fmt.Sprintf("Item #%d", itm.ID)
		if def := items.GetItem(uint16(itm.ID)); def != nil && def.Name != "" {
			name = def.Name
		}
		d.AddButton(fmt.Sprintf("put_%d", itm.ID), name)
	}

	d.AddQuickExit()
	s.sendDialog(peer, d.EndDialog("display_pick", "Cancel", ""))
}

// handleDisplayPickReturn memproses item yang dipilih untuk dipajang
func (s *Server) handleDisplayPickReturn(peer *C.ENetPeer, state *peerState, values map[string]string) {
	w := s.getOrCreateWorld(state.currentWorld)
	if w == nil {
		return
	}

	tileX, _ := strconv.Atoi(values["tilex"])
	tileY, _ := strconv.Atoi(values["tiley"])
	disp := w.FindDisplayAt(tileX, tileY)
	if disp == nil {
		return
	}

	btn := values["buttonClicked"]
	if strings.HasPrefix(btn, "put_") {
		itemIDStr := strings.TrimPrefix(btn, "put_")
		itemID, _ := strconv.Atoi(itemIDStr)
		if itemID > 0 && state.p.GetItemCount(itemID) > 0 {
			state.p.RemoveItem(itemID, 1)
			disp.ItemID = itemID
			s.db.SavePlayer(state.p)
			s.db.SaveWorld(w)
			s.sendInventory(peer, state)

			tile := w.GetTile(tileX, tileY)
			if tile != nil {
				s.sendTileUpdate(w, tileX, tileY, tile)
			}
			s.sendConsole(peer, "`2Placed %s on display!``", items.GetItem(uint16(itemID)).Name)
			s.playWorldSound(w.Name, "audio/tile_created.wav", float32(tileX*32), float32(tileY*32))
		}
	}
	s.handleDisplayWrench(peer, state, w, tileX, tileY)
}

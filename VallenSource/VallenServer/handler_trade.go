package server

/*
#include "../../enet/enet.h"
*/
import "C"

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"gtps/VallenSource/VallenDialog"
	"gtps/VallenSource/VallenItems"
	"gtps/VallenSource/VallenVariant"
)

// TradeItem mewakili satu item dalam tawaran barter
type TradeItem struct {
	ItemID int
	Count  int
}

// TradeParty menyimpan state satu pemain dalam sesi trade
type TradeParty struct {
	Peer      *C.ENetPeer
	State     *peerState
	Items     []TradeItem
	WL        int
	DL        int
	Accepted  bool
	Confirmed bool
}

// TradeSession mengelola sesi barter antara 2 player
type TradeSession struct {
	WorldName string
	Party1    *TradeParty
	Party2    *TradeParty
	Stage     int // 0 = Offer, 1 = Confirm
}

func (s *Server) getTradeSession(netID int) *TradeSession {
	s.tradesMu.Lock()
	defer s.tradesMu.Unlock()
	return s.trades[netID]
}

func (s *Server) findPeerByNetID(netID int) (*C.ENetPeer, *peerState) {
	s.peersMu.RLock()
	defer s.peersMu.RUnlock()
	for p, st := range s.peers {
		if st != nil && st.netID == netID {
			return p, st
		}
	}
	return nil, nil
}

// handleTradeInvite mengirim undangan barter ke target
func (s *Server) handleTradeInvite(fromPeer *C.ENetPeer, fromState *peerState, targetNetID int) {
	if fromState == nil || fromState.p == nil {
		return
	}

	targetPeer, targetState := s.findPeerByNetID(targetNetID)
	if targetPeer == nil || targetState == nil || targetState.p == nil {
		s.sendConsole(fromPeer, "`4Player is not online or not in this world!``")
		return
	}

	if fromState.currentWorld != targetState.currentWorld {
		s.sendConsole(fromPeer, "`4You must be in the same world to trade!``")
		return
	}

	if fromState.netID == targetState.netID {
		s.sendConsole(fromPeer, "`4You cannot trade with yourself!``")
		return
	}

	// Jarak maksimal 4 block (128 pixel)
	dx := float64(fromState.posX - targetState.posX)
	dy := float64(fromState.posY - targetState.posY)
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist > 160.0 {
		s.sendConsole(fromPeer, "`4You are too far away to trade with %s!``", targetState.growID)
		return
	}

	s.tradesMu.Lock()
	if s.trades[fromState.netID] != nil {
		s.tradesMu.Unlock()
		s.sendConsole(fromPeer, "`4You are already in a trade!``")
		return
	}
	if s.trades[targetState.netID] != nil {
		s.tradesMu.Unlock()
		s.sendConsole(fromPeer, "`4%s is currently busy trading with someone else!``", targetState.growID)
		return
	}
	s.tradesMu.Unlock()

	// Kirim notifikasi ke pengirim
	s.sendConsole(fromPeer, "`oSent trade request to `w%s``...``", targetState.growID)
	s.sendPacket(fromPeer, variant.New("OnTalkBubble", int32(fromState.netID), fmt.Sprintf("`oSent trade request to `w%s``...``", targetState.growID)).Pack(), true)

	// Kirim dialog undangan ke target
	inviteDialog := fmt.Sprintf(
		"set_default_color|`o\n"+
			"add_label_with_icon|big|`wTrade Request``|left|32|\n"+
			"add_spacer|small|\n"+
			"add_textbox|`w%s`` wants to trade with you!|left|\n"+
			"add_spacer|small|\n"+
			"embed_data|targetNetID|%d\n"+
			"add_button|trade_req_accept|`2Accept``|noflags|0|0|\n"+
			"add_button|trade_req_decline|`4Decline``|noflags|0|0|\n"+
			"add_quick_exit|\n"+
			"end_dialog|trade_invite|||\n",
		fromState.growID, fromState.netID,
	)
	s.sendPacket(targetPeer, variant.New("OnDialogRequest", inviteDialog).Pack(), true)
}

// handleTradeInviteResponse memproses respon penerimaan undangan barter
func (s *Server) handleTradeInviteResponse(peer *C.ENetPeer, state *peerState, values map[string]string) {
	targetNetID, _ := strconv.Atoi(values["targetNetID"])
	targetPeer, targetState := s.findPeerByNetID(targetNetID)

	button := values["buttonClicked"]
	if button != "trade_req_accept" {
		// Ditolak / dicancel
		if targetPeer != nil && targetState != nil {
			s.sendConsole(targetPeer, "`w%s `4declined your trade request.``", state.growID)
			s.sendPacket(targetPeer, variant.New("OnTalkBubble", int32(targetState.netID), fmt.Sprintf("`w%s `4declined trade request.``", state.growID)).Pack(), true)
		}
		s.sendConsole(peer, "`oTrade request declined.``")
		return
	}

	if targetPeer == nil || targetState == nil || targetState.currentWorld != state.currentWorld {
		s.sendConsole(peer, "`4The player left the world.``")
		return
	}

	s.tradesMu.Lock()
	if s.trades[state.netID] != nil || s.trades[targetState.netID] != nil {
		s.tradesMu.Unlock()
		s.sendConsole(peer, "`4One of you is already in a trade!``")
		return
	}

	session := &TradeSession{
		WorldName: state.currentWorld,
		Party1: &TradeParty{
			Peer:  targetPeer,
			State: targetState,
		},
		Party2: &TradeParty{
			Peer:  peer,
			State: state,
		},
		Stage: 0,
	}

	s.trades[targetState.netID] = session
	s.trades[state.netID] = session
	s.tradesMu.Unlock()

	s.sendTradeDialog(session)
}

func (s *Server) sendTradeDialog(session *TradeSession) {
	if session == nil {
		return
	}

	s.sendPartyTradeDialog(session.Party1, session.Party2)
	s.sendPartyTradeDialog(session.Party2, session.Party1)
}

func (s *Server) sendPartyTradeDialog(me *TradeParty, other *TradeParty) {
	if me == nil || me.Peer == nil || me.State == nil {
		return
	}

	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", fmt.Sprintf("`wTrading with %s``", other.State.growID), 32)
	d.AddSpacer("small")

	// Status tawaran kita
	d.AddTextbox("`2Your Offer:``")
	if len(me.Items) == 0 && me.WL == 0 && me.DL == 0 {
		d.AddSmallText("`7(No items or locks offered yet)``")
	} else {
		for i, item := range me.Items {
			name := fmt.Sprintf("Item #%d", item.ItemID)
			if itm := items.GetItem(uint16(item.ItemID)); itm != nil && itm.Name != "" {
				name = itm.Name
			}
			d.AddLabelWithIcon("small", fmt.Sprintf("`w%d %s``", item.Count, name), item.ItemID)
			_ = i
		}
		if me.WL > 0 {
			d.AddLabelWithIcon("small", fmt.Sprintf("`w%d World Lock(s)``", me.WL), 242)
		}
		if me.DL > 0 {
			d.AddLabelWithIcon("small", fmt.Sprintf("`w%d Diamond Lock(s)``", me.DL), 1796)
		}
	}
	d.AddSpacer("small")

	// Status tawaran lawan
	d.AddTextbox(fmt.Sprintf("`1%s's Offer:``", other.State.growID))
	if len(other.Items) == 0 && other.WL == 0 && other.DL == 0 {
		d.AddSmallText("`7(Waiting for partner's offer...)``")
	} else {
		for _, item := range other.Items {
			name := fmt.Sprintf("Item #%d", item.ItemID)
			if itm := items.GetItem(uint16(item.ItemID)); itm != nil && itm.Name != "" {
				name = itm.Name
			}
			d.AddLabelWithIcon("small", fmt.Sprintf("`w%d %s``", item.Count, name), item.ItemID)
		}
		if other.WL > 0 {
			d.AddLabelWithIcon("small", fmt.Sprintf("`w%d World Lock(s)``", other.WL), 242)
		}
		if other.DL > 0 {
			d.AddLabelWithIcon("small", fmt.Sprintf("`w%d Diamond Lock(s)``", other.DL), 1796)
		}
	}
	d.AddSpacer("small")

	// Input Locks kita
	d.AddTextInput("trade_wl", "Offer World Locks:", strconv.Itoa(me.WL), 5)
	d.AddTextInput("trade_dl", "Offer Diamond Locks:", strconv.Itoa(me.DL), 5)
	d.AddSpacer("small")

	if len(me.Items) < 4 {
		d.AddButton("btn_add_item", "`w+ Add Item from Backpack``")
	}
	if len(me.Items) > 0 {
		d.AddButton("btn_clear_items", "`4Clear Offered Items``")
	}
	d.AddSpacer("small")

	// Tombol Accept
	if me.Accepted {
		d.AddButton("btn_unaccept", "`eUnlock Offer (Already Accepted)``")
	} else {
		d.AddButton("btn_accept", "`2Accept Offer``")
	}
	d.AddButton("btn_cancel", "`4Cancel Trade``")
	d.AddQuickExit()

	s.sendDialog(me.Peer, d.EndDialog("trade_session", "", ""))
}

// handleTradeSessionReturn menangani input dari jendela trade
func (s *Server) handleTradeSessionReturn(peer *C.ENetPeer, state *peerState, values map[string]string) {
	session := s.getTradeSession(state.netID)
	if session == nil {
		return
	}

	var me, other *TradeParty
	if session.Party1.State.netID == state.netID {
		me = session.Party1
		other = session.Party2
	} else {
		me = session.Party2
		other = session.Party1
	}

	btn := values["buttonClicked"]

	if btn == "btn_cancel" || btn == "" {
		s.cancelTrade(state.netID, fmt.Sprintf("`4Trade was canceled by %s.``", state.growID))
		return
	}

	// Update Locks
	newWL, _ := strconv.Atoi(values["trade_wl"])
	newDL, _ := strconv.Atoi(values["trade_dl"])
	if newWL < 0 {
		newWL = 0
	}
	if newDL < 0 {
		newDL = 0
	}

	// Batasi sesuai inventory
	maxWL := state.p.GetItemCount(242)
	maxDL := state.p.GetItemCount(1796)
	if newWL > maxWL {
		newWL = maxWL
	}
	if newDL > maxDL {
		newDL = maxDL
	}

	lockChanged := me.WL != newWL || me.DL != newDL
	me.WL = newWL
	me.DL = newDL

	if lockChanged {
		// Reset status accept kedua pihak jika tawaran berubah
		me.Accepted = false
		other.Accepted = false
	}

	if btn == "btn_add_item" {
		s.sendTradeAddItemDialog(peer, state, me)
		return
	}

	if btn == "btn_clear_items" {
		me.Items = nil
		me.Accepted = false
		other.Accepted = false
		s.sendTradeDialog(session)
		return
	}

	if btn == "btn_unaccept" {
		me.Accepted = false
		other.Accepted = false
		s.sendTradeDialog(session)
		return
	}

	if btn == "btn_accept" {
		me.Accepted = true
		if me.Accepted && other.Accepted {
			// Masuk ke fase konfirmasi akhir
			session.Stage = 1
			s.sendTradeConfirmDialog(session)
			return
		}
		s.sendTradeDialog(session)
		return
	}

	s.sendTradeDialog(session)
}

// sendTradeAddItemDialog menampilkan inventory untuk dipilih
func (s *Server) sendTradeAddItemDialog(peer *C.ENetPeer, state *peerState, me *TradeParty) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wSelect Item to Offer``", 32)
	d.AddSpacer("small")

	itemCount := 0
	for _, itm := range state.p.Inventory {
		if itm.ID <= 0 || itm.Count <= 0 {
			continue
		}
		// Jangan masukkan lock di slot item biasa jika sudah ada slot input lock
		if itm.ID == 242 || itm.ID == 1796 {
			continue
		}
		// Cek untradeable
		itemDef := items.GetItem(uint16(itm.ID))
		if itemDef != nil && itemDef.Category&items.CatUntradeable != 0 {
			continue
		}

		name := fmt.Sprintf("Item #%d", itm.ID)
		if itemDef != nil && itemDef.Name != "" {
			name = itemDef.Name
		}

		d.AddButton(fmt.Sprintf("pick_%d", itm.ID), fmt.Sprintf("`w%s `2(You have %d)``", name, itm.Count))
		itemCount++
	}

	if itemCount == 0 {
		d.AddSmallText("`7You have no tradeable items in your backpack!``")
	}

	d.AddSpacer("small")
	d.AddButton("btn_back", "Back")
	d.AddQuickExit()
	s.sendDialog(peer, d.EndDialog("trade_add_item", "", ""))
}

// handleTradeAddItemReturn memproses pemilihan item dari inventory
func (s *Server) handleTradeAddItemReturn(peer *C.ENetPeer, state *peerState, values map[string]string) {
	session := s.getTradeSession(state.netID)
	if session == nil {
		return
	}

	btn := values["buttonClicked"]
	if btn == "btn_back" || btn == "" {
		s.sendTradeDialog(session)
		return
	}

	if strings.HasPrefix(btn, "pick_") {
		itemIDStr := strings.TrimPrefix(btn, "pick_")
		itemID, _ := strconv.Atoi(itemIDStr)
		owned := state.p.GetItemCount(itemID)
		if itemID > 0 && owned > 0 {
			// Buka dialog input count
			name := fmt.Sprintf("Item #%d", itemID)
			if itm := items.GetItem(uint16(itemID)); itm != nil && itm.Name != "" {
				name = itm.Name
			}
			cntDialog := fmt.Sprintf(
				"set_default_color|`o\n"+
					"add_label_with_icon|big|`wOffer %s``|left|%d|\n"+
					"add_spacer|small|\n"+
					"add_textbox|How many to offer? (You have %d)|left|\n"+
					"add_text_input|offer_count||%d|5|\n"+
					"embed_data|offer_item_id|%d\n"+
					"add_button|btn_confirm_add|`2Add to Trade``|noflags|0|0|\n"+
					"add_button|btn_cancel_add|Cancel|noflags|0|0|\n"+
					"add_quick_exit|\n"+
					"end_dialog|trade_set_count|||\n",
				name, itemID, owned, owned, itemID,
			)
			s.sendPacket(peer, variant.New("OnDialogRequest", cntDialog).Pack(), true)
			return
		}
	}

	s.sendTradeDialog(session)
}

// handleTradeSetCountReturn memproses jumlah item yang dimasukkan
func (s *Server) handleTradeSetCountReturn(peer *C.ENetPeer, state *peerState, values map[string]string) {
	session := s.getTradeSession(state.netID)
	if session == nil {
		return
	}

	var me, other *TradeParty
	if session.Party1.State.netID == state.netID {
		me = session.Party1
		other = session.Party2
	} else {
		me = session.Party2
		other = session.Party1
	}

	btn := values["buttonClicked"]
	if btn == "btn_cancel_add" || btn == "" {
		s.sendTradeDialog(session)
		return
	}

	itemID, _ := strconv.Atoi(values["offer_item_id"])
	cnt, _ := strconv.Atoi(values["offer_count"])
	owned := state.p.GetItemCount(itemID)

	if itemID > 0 && cnt > 0 && owned > 0 {
		if cnt > owned {
			cnt = owned
		}
		if len(me.Items) < 4 {
			// Cek apakah item sudah ada di list
			found := false
			for i := range me.Items {
				if me.Items[i].ItemID == itemID {
					me.Items[i].Count = cnt
					found = true
					break
				}
			}
			if !found {
				me.Items = append(me.Items, TradeItem{ItemID: itemID, Count: cnt})
			}
			me.Accepted = false
			other.Accepted = false
		}
	}

	s.sendTradeDialog(session)
}

// sendTradeConfirmDialog menampilkan dialog konfirmasi tahap 2
func (s *Server) sendTradeConfirmDialog(session *TradeSession) {
	if session == nil {
		return
	}
	s.sendPartyConfirmDialog(session.Party1, session.Party2)
	s.sendPartyConfirmDialog(session.Party2, session.Party1)
}

func (s *Server) sendPartyConfirmDialog(me *TradeParty, other *TradeParty) {
	d := dialog.New().SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "`wTrade Confirmation (Final Step)``", 1366)
	d.AddSpacer("small")

	d.AddTextbox("`4You will give:``")
	if len(me.Items) == 0 && me.WL == 0 && me.DL == 0 {
		d.AddSmallText("`7(Nothing)``")
	} else {
		for _, itm := range me.Items {
			name := fmt.Sprintf("Item #%d", itm.ItemID)
			if def := items.GetItem(uint16(itm.ItemID)); def != nil && def.Name != "" {
				name = def.Name
			}
			d.AddLabelWithIcon("small", fmt.Sprintf("`4- %d %s``", itm.Count, name), itm.ItemID)
		}
		if me.WL > 0 {
			d.AddLabelWithIcon("small", fmt.Sprintf("`4- %d World Lock(s)``", me.WL), 242)
		}
		if me.DL > 0 {
			d.AddLabelWithIcon("small", fmt.Sprintf("`4- %d Diamond Lock(s)``", me.DL), 1796)
		}
	}
	d.AddSpacer("small")

	d.AddTextbox("`2You will receive:``")
	if len(other.Items) == 0 && other.WL == 0 && other.DL == 0 {
		d.AddSpacer("small")
		d.AddTextbox("`4SCAM WARNING: You are about to trade without receiving anything in return!``")
		d.AddTextbox("`4Once confirmed, this action CANNOT be undone!``")
	} else {
		for _, itm := range other.Items {
			name := fmt.Sprintf("Item #%d", itm.ItemID)
			if def := items.GetItem(uint16(itm.ItemID)); def != nil && def.Name != "" {
				name = def.Name
			}
			d.AddLabelWithIcon("small", fmt.Sprintf("`2+ %d %s``", itm.Count, name), itm.ItemID)
		}
		if other.WL > 0 {
			d.AddLabelWithIcon("small", fmt.Sprintf("`2+ %d World Lock(s)``", other.WL), 242)
		}
		if other.DL > 0 {
			d.AddLabelWithIcon("small", fmt.Sprintf("`2+ %d Diamond Lock(s)``", other.DL), 1796)
		}
	}
	d.AddSpacer("small")

	d.AddButton("btn_final_confirm", "`2Do The Trade!``")
	d.AddButton("btn_final_cancel", "`4Cancel``")
	d.AddQuickExit()

	s.sendDialog(me.Peer, d.EndDialog("trade_confirm", "", ""))
}

// handleTradeConfirmReturn memproses konfirmasi final
func (s *Server) handleTradeConfirmReturn(peer *C.ENetPeer, state *peerState, values map[string]string) {
	session := s.getTradeSession(state.netID)
	if session == nil {
		return
	}

	var me, other *TradeParty
	if session.Party1.State.netID == state.netID {
		me = session.Party1
		other = session.Party2
	} else {
		me = session.Party2
		other = session.Party1
	}

	btn := values["buttonClicked"]
	if btn == "btn_final_cancel" || btn == "" {
		s.cancelTrade(state.netID, fmt.Sprintf("`4Trade was canceled by %s.``", state.growID))
		return
	}

	if btn == "btn_final_confirm" {
		me.Confirmed = true
		if me.Confirmed && other.Confirmed {
			s.executeTrade(session)
			return
		}
		s.sendConsole(peer, "`2You confirmed the trade. Waiting for %s to confirm...``", other.State.growID)
	}
}

// executeTrade mengeksekusi transfer item secara atomik
func (s *Server) executeTrade(session *TradeSession) {
	p1 := session.Party1
	p2 := session.Party2

	// Verifikasi ketersediaan item di inventory P1
	for _, itm := range p1.Items {
		if p1.State.p.GetItemCount(itm.ItemID) < itm.Count {
			s.cancelTrade(p1.State.netID, fmt.Sprintf("`4Trade failed: %s no longer has enough %s!``", p1.State.growID, items.GetItem(uint16(itm.ItemID)).Name))
			return
		}
	}
	if p1.WL > 0 && p1.State.p.GetItemCount(242) < p1.WL {
		s.cancelTrade(p1.State.netID, fmt.Sprintf("`4Trade failed: %s no longer has enough World Locks!``", p1.State.growID))
		return
	}
	if p1.DL > 0 && p1.State.p.GetItemCount(1796) < p1.DL {
		s.cancelTrade(p1.State.netID, fmt.Sprintf("`4Trade failed: %s no longer has enough Diamond Locks!``", p1.State.growID))
		return
	}

	// Verifikasi ketersediaan item di inventory P2
	for _, itm := range p2.Items {
		if p2.State.p.GetItemCount(itm.ItemID) < itm.Count {
			s.cancelTrade(p2.State.netID, fmt.Sprintf("`4Trade failed: %s no longer has enough %s!``", p2.State.growID, items.GetItem(uint16(itm.ItemID)).Name))
			return
		}
	}
	if p2.WL > 0 && p2.State.p.GetItemCount(242) < p2.WL {
		s.cancelTrade(p2.State.netID, fmt.Sprintf("`4Trade failed: %s no longer has enough World Locks!``", p2.State.growID))
		return
	}
	if p2.DL > 0 && p2.State.p.GetItemCount(1796) < p2.DL {
		s.cancelTrade(p2.State.netID, fmt.Sprintf("`4Trade failed: %s no longer has enough Diamond Locks!``", p2.State.growID))
		return
	}

	// 1. Kurangi item & lock dari P1
	for _, itm := range p1.Items {
		p1.State.p.RemoveItem(itm.ItemID, itm.Count)
	}
	if p1.WL > 0 {
		p1.State.p.RemoveItem(242, p1.WL)
	}
	if p1.DL > 0 {
		p1.State.p.RemoveItem(1796, p1.DL)
	}

	// 2. Kurangi item & lock dari P2
	for _, itm := range p2.Items {
		p2.State.p.RemoveItem(itm.ItemID, itm.Count)
	}
	if p2.WL > 0 {
		p2.State.p.RemoveItem(242, p2.WL)
	}
	if p2.DL > 0 {
		p2.State.p.RemoveItem(1796, p2.DL)
	}

	// 3. Tambahkan tawaran P2 ke P1
	for _, itm := range p2.Items {
		p1.State.p.AddItem(itm.ItemID, itm.Count)
	}
	if p2.WL > 0 {
		p1.State.p.AddItem(242, p2.WL)
	}
	if p2.DL > 0 {
		p1.State.p.AddItem(1796, p2.DL)
	}

	// 4. Tambahkan tawaran P1 ke P2
	for _, itm := range p1.Items {
		p2.State.p.AddItem(itm.ItemID, itm.Count)
	}
	if p1.WL > 0 {
		p2.State.p.AddItem(242, p1.WL)
	}
	if p1.DL > 0 {
		p2.State.p.AddItem(1796, p1.DL)
	}

	// Simpan ke database
	s.db.SavePlayer(p1.State.p)
	s.db.SavePlayer(p2.State.p)

	// Update inventory di client
	s.sendInventory(p1.Peer, p1.State)
	s.sendInventory(p2.Peer, p2.State)

	// Catat riwayat barter ke TradeHistory
	nowStr := time.Now().Format("2006-01-02 15:04")
	p1Log := fmt.Sprintf("[%s] Traded with %s", nowStr, p2.State.growID)
	p2Log := fmt.Sprintf("[%s] Traded with %s", nowStr, p1.State.growID)
	p1.State.p.TradeHistory = append(p1.State.p.TradeHistory, p1Log)
	p2.State.p.TradeHistory = append(p2.State.p.TradeHistory, p2Log)

	// Bersihkan sesi trade
	s.tradesMu.Lock()
	delete(s.trades, p1.State.netID)
	delete(s.trades, p2.State.netID)
	s.tradesMu.Unlock()

	// Notifikasi sukses dan audio
	s.sendConsole(p1.Peer, "`2Trade completed successfully with `w%s``!``", p2.State.growID)
	s.sendConsole(p2.Peer, "`2Trade completed successfully with `w%s``!``", p1.State.growID)
	s.playWorldSound(session.WorldName, "audio/trade_complete.wav", p1.State.posX, p1.State.posY)
	s.sendPacket(p1.Peer, variant.New("OnTalkBubble", int32(p1.State.netID), "`2[Trade Successful!]``").Pack(), true)
	s.sendPacket(p2.Peer, variant.New("OnTalkBubble", int32(p2.State.netID), "`2[Trade Successful!]``").Pack(), true)
}

// cancelTrade membatalkan sesi trade secara aman
func (s *Server) cancelTrade(netID int, reason string) {
	s.tradesMu.Lock()
	session := s.trades[netID]
	if session == nil {
		s.tradesMu.Unlock()
		return
	}
	delete(s.trades, session.Party1.State.netID)
	delete(s.trades, session.Party2.State.netID)
	s.tradesMu.Unlock()

	if reason != "" {
		s.sendConsole(session.Party1.Peer, reason)
		s.sendConsole(session.Party2.Peer, reason)
	}
	// Tutup dialog trade
	s.sendPacket(session.Party1.Peer, variant.New("OnForceTradeEnd").Pack(), true)
	s.sendPacket(session.Party2.Peer, variant.New("OnForceTradeEnd").Pack(), true)
}

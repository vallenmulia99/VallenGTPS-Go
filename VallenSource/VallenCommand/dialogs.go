package command

import (
	"fmt"
	"strconv"
	"strings"

	"gtps/VallenSource/VallenConfig"
	dialog "gtps/VallenSource/VallenDialog"
	items "gtps/VallenSource/VallenItems"
	role "gtps/VallenSource/VallenRole"
)

// ─────────────────────────────────────────────
// FIND DIALOG (Item Finder dengan Bypass Seed)
// ─────────────────────────────────────────────

const ItemsPerPage = 12

// FilterItems mencari item berdasarkan query teks (bisa nama atau ID)
// SEED DI-BYPASS (TIDAK AKAN MUNCUL)
func FilterItems(query string) []items.Item {
	if items.DB == nil {
		return nil
	}

	cleanQuery := strings.ToLower(strings.TrimSpace(query))
	var matches []items.Item

	for i := range items.DB.Items {
		item := items.DB.Items[i]
		if item.ID == 0 || item.Name == "" {
			continue
		}

		// BYPASS SEED: Seed tidak akan ikut ke list pencarian
		lowerName := strings.ToLower(item.Name)
		if item.Type == items.TypeSeed || strings.HasSuffix(lowerName, " seed") || strings.HasSuffix(lowerName, " tree") {
			continue
		}

		// Cocokkan dengan ID atau Nama Item
		if cleanQuery == "" {
			matches = append(matches, item)
		} else {
			idStr := fmt.Sprintf("%d", item.ID)
			if idStr == cleanQuery || strings.Contains(lowerName, cleanQuery) {
				matches = append(matches, item)
			}
		}
	}

	return matches
}

// RenderFindDialog menampilkan dialog pencarian item yang lebar, tinggi, dan proporsional
func RenderFindDialog(query string, page int) string {
	matches := FilterItems(query)
	totalMatches := len(matches)

	totalPages := (totalMatches + ItemsPerPage - 1) / ItemsPerPage
	if totalPages == 0 {
		totalPages = 1
	}
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	startIdx := page * ItemsPerPage
	endIdx := startIdx + ItemsPerPage
	if endIdx > totalMatches {
		endIdx = totalMatches
	}

	var pageItems []items.Item
	if startIdx < totalMatches {
		pageItems = matches[startIdx:endIdx]
	}

	escapedQuery := strings.ReplaceAll(query, "|", "")

	var b strings.Builder
	b.WriteString("set_default_color|`o\n")
	b.WriteString("add_label_with_icon|big|`wGrowtopia Item Finder``|left|6016|\n")
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_textbox|`oSearch for blocks, clothing, and items by Name or ID. (Seeds are filtered out)``|left|\n")
	b.WriteString("add_spacer|small|\n")

	fmt.Fprintf(&b, "add_text_input|find_query|Search Item / ID:|%s|35|\n", escapedQuery)
	b.WriteString("add_button|find_search_btn|`2Search Item``|noflags|0|0|\n")
	b.WriteString("add_spacer|small|\n")

	if totalMatches == 0 {
		fmt.Fprintf(&b, "add_textbox|`4No non-seed items found matching '`w%s`4'!``|left|\n", escapedQuery)
	} else {
		fmt.Fprintf(&b, "add_textbox|`oFound `2%d`` items  |  Page `2%d`` of `2%d``|left|\n", totalMatches, page+1, totalPages)
		b.WriteString("add_spacer|small|\n")

		for _, item := range pageItems {
			cleanName := strings.ReplaceAll(item.Name, "|", "")
			fmt.Fprintf(&b, "add_button_with_icon|find_select_%d|`w%s `7(ID: `2%d`7, Rarity: `2%d`7)``|left|%d|\n", item.ID, cleanName, item.ID, item.Rarity, item.ID)
		}
	}

	b.WriteString("add_spacer|small|\n")
	fmt.Fprintf(&b, "embed_data|current_query|%s\n", escapedQuery)
	fmt.Fprintf(&b, "embed_data|current_page|%d\n", page)

	// Tombol Navigasi Halaman
	if page > 0 {
		fmt.Fprintf(&b, "add_button|find_page_%d|`w<< Previous Page``|noflags|0|0|\n", page-1)
	}
	if page < totalPages-1 {
		fmt.Fprintf(&b, "add_button|find_page_%d|`wNext Page >>``|noflags|0|0|\n", page+1)
	}

	b.WriteString("add_quick_exit|\n")
	b.WriteString("end_dialog|find_item_dialog|Close||\n")

	return b.String()
}

// RenderFindGiveDialog menampilkan konfirmasi pengambilan item dengan detail lengkap
func RenderFindGiveDialog(itemID int) string {
	item := items.GetItem(uint16(itemID))
	name := fmt.Sprintf("Item #%d", itemID)
	rarity := 0
	if item != nil {
		if item.Name != "" {
			name = item.Name
		}
		rarity = int(item.Rarity)
	}

	cleanName := strings.ReplaceAll(name, "|", "")
	var b strings.Builder
	b.WriteString("set_default_color|`o\n")
	fmt.Fprintf(&b, "add_label_with_icon|big|`w%s``|left|%d|\n", cleanName, itemID)
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_textbox|`oSelect the quantity of this item you would like to receive into your inventory:``|left|\n")
	b.WriteString("add_spacer|small|\n")
	fmt.Fprintf(&b, "add_textbox|`oItem ID: `2%d``  |  Rarity: `2%d``|left|\n", itemID, rarity)
	b.WriteString("add_spacer|small|\n")
	b.WriteString("add_text_input|count|Quantity (1-200):|200|5|\n")
	b.WriteString("add_spacer|small|\n")
	fmt.Fprintf(&b, "embed_data|itemID|%d\n", itemID)
	b.WriteString("add_quick_exit|\n")
	b.WriteString("end_dialog|find_give_confirm|Cancel|Get Item|\n")

	return b.String()
}

// ─────────────────────────────────────────────
// GROLE DIALOG (Role Management UI)
// ─────────────────────────────────────────────

// RenderRoleDialog menampilkan dialog pemilihan role untuk target player
func RenderRoleDialog(targetGrowID string, targetCurrentAdminLevel int) string {
	currentRole := role.GetRoleByAdminLevel(targetCurrentAdminLevel)
	d := dialog.New()
	d.SetDefaultColor("`o")
	d.AddLabelWithIcon("big", fmt.Sprintf("`wSet Role for `2%s``", targetGrowID), 242)
	d.AddSpacer("small")
	d.AddTextbox(fmt.Sprintf("Current role: %s%s``. Select exactly one new role:", currentRole.NameColor, currentRole.Name))
	d.AddSpacer("small")

	for _, itemRole := range role.RoleList {
		d.AddCheckbox(
			fmt.Sprintf("role_lvl_%d", itemRole.AdminLevel),
			fmt.Sprintf("%s%s `7(Level %d)``", itemRole.NameColor, itemRole.Name, itemRole.AdminLevel),
			false,
		)
	}

	d.AddSpacer("small")
	d.EmbedData("target_growid", targetGrowID)
	d.AddQuickExit()
	return d.EndDialog("grole_edit", "Cancel", "NEXT")
}

// RenderRoleConfirmDialog menampilkan dialog konfirmasi perubahan role
func RenderRoleConfirmDialog(targetGrowID string, newAdminLevel int) string {
	newRole := role.GetRoleByAdminLevel(newAdminLevel)
	d := dialog.New()
	d.SetDefaultColor("`o")
	d.AddLabelWithIcon("big", "Confirm Role Change", 242)
	d.AddSpacer("small")
	d.AddTextbox(fmt.Sprintf("Set `w%s`` to %s%s %s%s``?", targetGrowID, newRole.NameColor, newRole.Name, newRole.TagColor, newRole.Tag))
	d.AddSpacer("small")
	d.AddSmallText("This action immediately updates the player's role and badge.")
	d.AddSpacer("small")
	d.EmbedData("target_growid", targetGrowID)
	d.EmbedData("role_level", newRole.AdminLevel)
	return d.EndDialog("grole_confirm", "Cancel", "CONFIRM")
}

// ReadRoleSelection memproses dialog grole_edit dan mengembalikan target + role yang dipilih
func ReadRoleSelection(senderGrowID string, senderAdminLevel int, cfg *config.Config, params map[string]string) (targetGrowID string, newAdminLevel int, errMsg string) {
	targetGrowID = params["target_growid"]
	if targetGrowID == "" {
		return "", 0, "Target player tidak valid."
	}
	if !canManageRoles(senderGrowID, senderAdminLevel, cfg) {
		return targetGrowID, 0, "Anda tidak memiliki izin untuk menggunakan /grole!"
	}

	chosenLevel := -1
	selectedCount := 0
	for key, value := range params {
		if !strings.HasPrefix(key, "role_lvl_") || value != "1" {
			continue
		}
		level, err := strconv.Atoi(strings.TrimPrefix(key, "role_lvl_"))
		if err != nil {
			continue
		}
		chosenLevel = level
		selectedCount++
	}
	if selectedCount != 1 || !isValidRoleLevel(chosenLevel) {
		return targetGrowID, 0, "Silakan centang tepat satu role terlebih dahulu!"
	}
	if errMsg = ValidateRoleChange(senderGrowID, senderAdminLevel, cfg, chosenLevel); errMsg != "" {
		return targetGrowID, 0, errMsg
	}
	return targetGrowID, chosenLevel, ""
}

// ValidateRoleChange memvalidasi apakah sender bisa memberikan role dengan level tertentu
func ValidateRoleChange(senderGrowID string, senderAdminLevel int, cfg *config.Config, chosenLevel int) string {
	if !canManageRoles(senderGrowID, senderAdminLevel, cfg) {
		return "Anda tidak memiliki izin untuk menggunakan /grole!"
	}
	if !isValidRoleLevel(chosenLevel) {
		return "Role yang dipilih tidak valid."
	}
	isOwner := (cfg != nil && cfg.IsOwner(senderGrowID)) || senderAdminLevel >= role.LevelMonarch
	if !isOwner && chosenLevel > senderAdminLevel {
		return fmt.Sprintf("Anda tidak bisa memberikan role dengan level %d (Level Anda: %d)!", chosenLevel, senderAdminLevel)
	}
	return ""
}

func canManageRoles(growID string, adminLevel int, cfg *config.Config) bool {
	return (cfg != nil && cfg.IsOwner(growID)) || adminLevel >= role.LevelEliteGuardian
}

func isValidRoleLevel(level int) bool {
	for _, itemRole := range role.RoleList {
		if itemRole.AdminLevel == level {
			return true
		}
	}
	return false
}

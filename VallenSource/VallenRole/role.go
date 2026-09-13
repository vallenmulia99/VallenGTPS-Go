package role

import (
	"fmt"
	"strings"
)

// AdminLevel constants sesuai permintaan: 0, 1, 2, 5, 7, 9, 999
const (
	LevelPlayer          = 0   // Player biasa
	LevelSupporter       = 1   // Supporter / VIP
	LevelVip             = LevelSupporter // Alias VIP
	LevelCreator         = 2   // Content Creator / Builder
	LevelEliteGuardian   = 5   // Moderator (mstate 1)
	LevelGrandArchitect  = 7   // Admin (mstate 1, smstate 1)
	LevelShadowSovereign = 9   // Developer / Head Admin
	LevelMonarch         = 999 // Owner / Super Admin
)

// Role defines role attributes, colors, tags, and privileges
type Role struct {
	ID          int    `json:"id"`
	AdminLevel  int    `json:"admin_level"`  // 0, 1, 2, 5, 7, 9, 999
	Name        string `json:"name"`         // Simple name (e.g., "Monarch")
	DisplayName string `json:"display_name"` // Full formatted title
	Tag         string `json:"tag"`          // Overhead suffix tag (e.g., "[Monarch]")
	NameColor   string `json:"name_color"`   // Color code for player name
	TagColor    string `json:"tag_color"`    // Color code for tag
	ChatColor   string `json:"chat_color"`   // Chat text color
	IsModerator bool   `json:"is_moderator"` // mstate flag in OnSpawn (Level >= 5)
	IsDeveloper bool   `json:"is_developer"` // smstate flag in OnSpawn (Level >= 7)
}

// Registry of all server roles, mapped by AdminLevel or sequential ID
var RoleList = []Role{
	{
		ID:          0,
		AdminLevel:  LevelPlayer,
		Name:        "Player",
		DisplayName: "Player",
		Tag:         "",
		NameColor:   "`w",
		TagColor:    "`w",
		ChatColor:   "`w",
		IsModerator: false,
		IsDeveloper: false,
	},
	{
		ID:          1,
		AdminLevel:  LevelSupporter,
		Name:        "Supporter",
		DisplayName: "Supporter",
		Tag:         "[Supporter]",
		NameColor:   "`1", // Aqua / Sky Blue
		TagColor:    "`c", // Light Blue / Cyan
		ChatColor:   "`1",
		IsModerator: false,
		IsDeveloper: false,
	},
	{
		ID:          2,
		AdminLevel:  LevelCreator,
		Name:        "Creator",
		DisplayName: "Creator",
		Tag:         "[Creator]",
		NameColor:   "`6", // Orange
		TagColor:    "`8", // Gold / Yellow
		ChatColor:   "`6",
		IsModerator: false,
		IsDeveloper: false,
	},
	{
		ID:          3,
		AdminLevel:  LevelEliteGuardian,
		Name:        "Elite Guardian",
		DisplayName: "Elite Guardian",
		Tag:         "[Elite Guardian]",
		NameColor:   "`2", // Emerald Green
		TagColor:    "`7", // Silver / Gray
		ChatColor:   "`2",
		IsModerator: true,
		IsDeveloper: false,
	},
	{
		ID:          4,
		AdminLevel:  LevelGrandArchitect,
		Name:        "Grand Architect",
		DisplayName: "Grand Architect",
		Tag:         "[Grand Architect]",
		NameColor:   "`c", // Cyan
		TagColor:    "`1", // Biru Neon / Sky Blue
		ChatColor:   "`c",
		IsModerator: false,
		IsDeveloper: true,
	},
	{
		ID:          5,
		AdminLevel:  LevelShadowSovereign,
		Name:        "Shadow Sovereign",
		DisplayName: "Shadow Sovereign",
		Tag:         "[Shadow Sovereign]",
		NameColor:   "`b", // Hitam (Black)
		TagColor:    "`4", // Merah Dark (Dark Red)
		ChatColor:   "`4",
		IsModerator: false,
		IsDeveloper: true,
	},
	{
		ID:          6,
		AdminLevel:  LevelMonarch,
		Name:        "Monarch",
		DisplayName: "Monarch",
		Tag:         "[Monarch]",
		NameColor:   "`#", // Ungu / Purple
		TagColor:    "`8", // Gold / Yellow
		ChatColor:   "`#",
		IsModerator: false,
		IsDeveloper: true,
	},
}

// GetRole returns the Role info for a given role ID or AdminLevel. Defaults to Player if not found.
func GetRole(val int) Role {
	// 1. Check by sequential ID (0..6)
	if val >= 0 && val < len(RoleList) {
		return RoleList[val]
	}
	// 2. Check by AdminLevel (0, 1, 2, 5, 7, 9, 999)
	for _, r := range RoleList {
		if r.AdminLevel == val {
			return r
		}
	}
	return RoleList[0]
}

// GetRoleByAdminLevel returns role with specific AdminLevel
func GetRoleByAdminLevel(level int) Role {
	for _, r := range RoleList {
		if r.AdminLevel == level {
			return r
		}
	}
	return RoleList[0]
}

// GetRoleByName finds a role by name (case-insensitive) or AdminLevel string
func GetRoleByName(name string) (Role, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, r := range RoleList {
		if strings.ToLower(r.Name) == name ||
			strings.ToLower(strings.ReplaceAll(r.Name, " ", "")) == name ||
			strings.ToLower(strings.ReplaceAll(r.Name, "_", "")) == name {
			return r, true
		}
	}
	// Check by AdminLevel number
	for _, r := range RoleList {
		if fmt.Sprintf("%d", r.AdminLevel) == name || fmt.Sprintf("%d", r.ID) == name {
			return r, true
		}
	}
	return RoleList[0], false
}

// FormatPlayerName formats the overhead player name with role color and trailing tag.
// Example: Vallen with Monarch -> "`#Vallen `8[Monarch]``"
func FormatPlayerName(growID string, roleVal int) string {
	return FormatPlayerNameWithTitle(growID, roleVal, "")
}

// FormatPlayerNameWithTitle formats the overhead name with title prefix/suffix and role tag.
func FormatPlayerNameWithTitle(growID string, roleVal int, title string) string {
	return FormatPlayerNameInWorld(growID, roleVal, title, "", false)
}

// FormatPlayerNameInWorld formats the overhead name dynamically based on world lock ownership.
// If the player is the world owner -> name color turns `2 (Green) like official Growtopia!
// If the player has access -> name color turns `^ (Cyan)!
// Staff members retain their staff colors/tags.
func FormatPlayerNameInWorld(growID string, roleVal int, title string, worldOwnerName string, hasAccess bool) string {
	r := GetRole(roleVal)
	nameColor := r.NameColor

	// If player is a normal player (not staff), apply world lock color
	if roleVal == LevelPlayer {
		if worldOwnerName != "" && strings.EqualFold(growID, worldOwnerName) {
			nameColor = "`2" // Green for World Lock Owner!
		} else if hasAccess {
			nameColor = "`^" // Cyan for Player with Access!
		} else {
			nameColor = "`w" // White for normal player
		}
	}

	titleStr := ""
	if title != "" && title != "(None)" {
		titleStr = fmt.Sprintf(" `^%s``", title)
	}
	if r.Tag != "" {
		return fmt.Sprintf("%s%s%s %s%s``", nameColor, growID, titleStr, r.TagColor, r.Tag)
	}
	return fmt.Sprintf("%s%s%s``", nameColor, growID, titleStr)
}

// FormatChat formats in-game talk bubble and console message with role colors.
func FormatChat(growID string, roleVal int, text string) (bubbleMsg string, consoleMsg string) {
	r := GetRole(roleVal)
	var formattedName string
	if r.Tag != "" {
		formattedName = fmt.Sprintf("%s%s %s%s``", r.NameColor, growID, r.TagColor, r.Tag)
	} else {
		formattedName = fmt.Sprintf("%s%s``", r.NameColor, growID)
	}

	bubbleMsg = fmt.Sprintf("%s: %s", formattedName, text)
	consoleMsg = fmt.Sprintf("%s: `w%s``", formattedName, text)
	return bubbleMsg, consoleMsg
}

// ListAllRoles returns a formatted list of all roles for help/info commands
func ListAllRoles() string {
	var sb strings.Builder
	sb.WriteString("`9SERVER ROLES & ADMIN LEVELS``\n")
	for _, r := range RoleList {
		sb.WriteString(fmt.Sprintf("`w[%d] %s%s %s%s`` `7(AdminLevel: `2%d``)``\n",
			r.ID, r.NameColor, r.Name, r.TagColor, r.Tag, r.AdminLevel))
	}
	return sb.String()
}

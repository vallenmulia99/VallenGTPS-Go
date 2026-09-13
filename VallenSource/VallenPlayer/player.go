package player

import (
	"strings"

	"gtps/VallenSource/VallenRole"
)

const (
	// DefaultSkinColor and DefaultHairColor match the baseline appearance used
	// by the reference server for a freshly-created player.
	// 0xFFFFFFFF = natural hair color tint (no green hair bug)
	DefaultSkinColor uint32 = 2527912447
	DefaultHairColor uint32 = 0xFFFFFFFF
)

type Player struct {
	// Identity
	GrowID     string `json:"growid"`
	Password   string `json:"password"`
	UserID     int    `json:"user_id"`
	Role       int    `json:"role"`        // Role ID (0-6), maps to AdminLevel in VallenRole
	AdminLevel int    `json:"admin_level"` // 0, 1, 2, 5, 7, 9, 999 - determines command permissions
	CreatedAt  int64  `json:"created_at"`

	// Network
	NetID   int    `json:"-"` // Runtime only, don't save
	Prefix  string `json:"prefix"`
	Country string `json:"country"`

	// Appearance
	Clothing    [10]int `json:"clothing"` // [hair, shirt, pants, feet, face, hand, back, head, charm, ances]
	PunchEffect int     `json:"punch_effect"`
	SkinColor   uint32  `json:"skin_color"`
	HairColor   uint32  `json:"hair_color"`

	// State
	State      int     `json:"state"`
	PosX       float32 `json:"pos_x"`
	PosY       float32 `json:"pos_y"`
	RespawnX   float32 `json:"respawn_x"`
	RespawnY   float32 `json:"respawn_y"`
	FacingLeft bool    `json:"facing_left"`
	HP         int     `json:"hp"`

	// Inventory
	SlotSize  int    `json:"slot_size"`
	Inventory []Item `json:"inventory"`
	Favorites []int  `json:"favorites"`

	// Progression
	Gems  int `json:"gems"`
	Level int `json:"level"`
	XP    int `json:"xp"`

	// World Data
	RecentWorlds [6]string   `json:"recent_worlds"`
	MyWorlds     [200]string `json:"my_worlds"`

	// Social
	Friends   []Friend  `json:"friends"`
	Billboard Billboard `json:"billboard"`

	// Misc
	FiresRemoved int `json:"fires_removed"`
	GBCPity      int `json:"gbc_pity"`

	// Extended UI & Features
	BankWL                 int        `json:"bank_wl"`
	Notebook               string     `json:"notebook"`
	Title                  string     `json:"title"`
	OnlineStatus           int        `json:"online_status"` // 0=Online, 1=Away, 2=Busy, 3=InLove
	Bio                    string     `json:"bio"`
	Outfit1                [10]int    `json:"outfit1"`
	Outfit2                [10]int    `json:"outfit2"`
	LastDailyBonus         int64      `json:"last_daily_bonus"`
	PiggyGems              int        `json:"piggy_gems"`
	DailyChallengeProgress int        `json:"daily_challenge_progress"`
	Mailbox                []MailItem `json:"mailbox"`
	TradeHistory           []string      `json:"trade_history"`
	ThemeColor             string        `json:"theme_color"`
	BorderColor            string        `json:"border_color"`
	BgColor                string        `json:"bg_color"`
	WrenchStyle            int           `json:"wrench_style"`
	ShowLocation           bool          `json:"show_location"`
	ShowNotifications      bool          `json:"show_notifications"`
	Proxy                  ProxySettings `json:"proxy"`
}

type ProxySettings struct {
	FastSpin bool `json:"fast_spin"`
	Roulette bool `json:"roulette"`
	Reme     bool `json:"reme"`
}

type MailItem struct {
	ID        int    `json:"id"`
	Sender    string `json:"sender"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	RewardID  int    `json:"reward_id"`
	RewardQty int    `json:"reward_qty"`
	Claimed   bool   `json:"claimed"`
	Timestamp int64  `json:"timestamp"`
}

type Item struct {
	ID    int `json:"id"`
	Count int `json:"count"`
}

type Friend struct {
	Name   string `json:"name"`
	Ignore bool   `json:"ignore"`
	Block  bool   `json:"block"`
	Mute   bool   `json:"mute"`
}

type Billboard struct {
	ItemID  int  `json:"item_id"`
	Show    bool `json:"show"`
	Buying  bool `json:"buying"`
	Price   int  `json:"price"`
	PerItem bool `json:"per_item"`
}

type Cloth struct {
	Hair     int `json:"hair"`
	Shirt    int `json:"shirt"`
	Pants    int `json:"pants"`
	Feet     int `json:"feet"`
	Face     int `json:"face"`
	Hand     int `json:"hand"`
	Back     int `json:"back"`
	Mask     int `json:"mask"`
	Necklace int `json:"necklace"`
	Ances    int `json:"ances"`
}

func New(growID, password string) *Player {
	return &Player{
		GrowID:     growID,
		Password:   password,
		Gems:       0,
		Role:       0, // Default: Player (AdminLevel 0)
		AdminLevel: 0, // Default: Player
		Level:      1,
		XP:         0,
		HP:         10,
		SlotSize:   16,
		Inventory: []Item{
			{ID: 18, Count: 1}, // Fist
			{ID: 32, Count: 1}, // Wrench
		},
		Prefix:    "w",
		SkinColor: DefaultSkinColor,
		HairColor: DefaultHairColor, // Green
		Mailbox: []MailItem{
			{
				ID:        1,
				Sender:    "`6The Growtopia Team``",
				Title:     "Welcome to Growtopia!",
				Message:   "Welcome to the server! Here is a starter gift from the team to begin your journey.",
				RewardID:  242, // World Lock
				RewardQty: 5,
				Claimed:   false,
			},
		},
	}
}

func (p *Player) EnsureMailboxDefaults() {
	if p == nil {
		return
	}
	if len(p.Mailbox) == 0 {
		p.Mailbox = []MailItem{
			{
				ID:        1,
				Sender:    "`6The Growtopia Team``",
				Title:     "Welcome to Growtopia!",
				Message:   "Welcome to the server! Here is a starter gift from the team to begin your journey.",
				RewardID:  242, // World Lock
				RewardQty: 5,
				Claimed:   false,
			},
		}
	}
}

// EnsureAppearanceDefaults repairs player files created before appearance was
// persisted. A zero ARGB color is not a valid visible skin/hair color and
// makes some clients fall back to their white preview avatar.
func (p *Player) EnsureAppearanceDefaults() bool {
	if p == nil {
		return false
	}
	changed := false
	if p.SkinColor == 0 {
		p.SkinColor = DefaultSkinColor
		changed = true
	}
	if p.HairColor == 0 || p.HairColor == 0xFF00FF00 {
		p.HairColor = DefaultHairColor
		changed = true
	}
	return changed
}

// AddItem adds item to inventory (max 200 per slot) and returns any amount
// that could not fit. Multiple stacks of the same item are supported.
func (p *Player) AddItem(id, count int) int {
	if count <= 0 {
		return 0
	}
	remaining := count

	// Fill existing stacks first.
	for i := range p.Inventory {
		if p.Inventory[i].ID != id || p.Inventory[i].Count >= 200 {
			continue
		}
		space := 200 - p.Inventory[i].Count
		added := remaining
		if added > space {
			added = space
		}
		p.Inventory[i].Count += added
		remaining -= added
		if remaining == 0 {
			return 0
		}
	}

	// Create stacks while slots are available.
	for remaining > 0 && len(p.Inventory) < p.SlotSize {
		stack := remaining
		if stack > 200 {
			stack = 200
		}
		p.Inventory = append(p.Inventory, Item{ID: id, Count: stack})
		remaining -= stack
	}
	return remaining
}

// RemoveItem removes item from inventory
func (p *Player) RemoveItem(id, count int) bool {
	if count <= 0 || p.GetItemCount(id) < count {
		return false
	}
	remaining := count
	for i := range p.Inventory {
		if p.Inventory[i].ID != id || remaining == 0 {
			continue
		}
		removed := remaining
		if removed > p.Inventory[i].Count {
			removed = p.Inventory[i].Count
		}
		p.Inventory[i].Count -= removed
		remaining -= removed
	}
	filtered := p.Inventory[:0]
	for _, item := range p.Inventory {
		if item.Count > 0 {
			filtered = append(filtered, item)
		}
	}
	p.Inventory = filtered
	return true
}

// HasItem checks if player has item
func (p *Player) HasItem(id int, count int) bool {
	return count > 0 && p.GetItemCount(id) >= count
}

// GetItemCount returns how many of an item the player has (0 if none)
func (p *Player) GetItemCount(id int) int {
	total := 0
	for _, item := range p.Inventory {
		if item.ID == id && item.Count > 0 {
			total += item.Count
		}
	}
	return total
}

// CanAddItems validates capacity without mutating the real inventory.
func (p *Player) CanAddItems(rewards []Item) bool {
	clone := *p
	clone.Inventory = append([]Item(nil), p.Inventory...)
	for _, reward := range rewards {
		if clone.AddItem(reward.ID, reward.Count) != 0 {
			return false
		}
	}
	return true
}

// SetFriend updates an existing social relationship or adds a new one.
func (p *Player) SetFriend(name string, ignore, block, mute bool) {
	for i := range p.Friends {
		if strings.EqualFold(p.Friends[i].Name, name) {
			p.Friends[i].Ignore, p.Friends[i].Block, p.Friends[i].Mute = ignore, block, mute
			return
		}
	}
	p.Friends = append(p.Friends, Friend{Name: name, Ignore: ignore, Block: block, Mute: mute})
}

// RemoveFriend removes a social relationship by GrowID.
func (p *Player) RemoveFriend(name string) {
	for i := range p.Friends {
		if strings.EqualFold(p.Friends[i].Name, name) {
			p.Friends = append(p.Friends[:i], p.Friends[i+1:]...)
			return
		}
	}
}

// IsIgnoring reports whether this player ignores the supplied GrowID.
func (p *Player) IsIgnoring(name string) bool {
	for _, friend := range p.Friends {
		if strings.EqualFold(friend.Name, name) {
			return friend.Ignore || friend.Block
		}
	}
	return false
}

// GetRoleInfo returns the role info based on Player.Role (sequential ID 0-6)
func (p *Player) GetRoleInfo() (name string, color string, tag string) {
	r := role.GetRole(p.Role)
	return r.Name, r.NameColor, r.Tag
}

// SetRole sets both Role (sequential ID) and AdminLevel based on role.AdminLevel
// This keeps them in sync
func (p *Player) SetRole(roleID int) {
	p.Role = roleID
	p.AdminLevel = role.GetRole(roleID).AdminLevel
}

// CanAccess checks if player's AdminLevel is >= required level
func (p *Player) CanAccess(requiredLevel int) bool {
	return p.AdminLevel >= requiredLevel
}

// GetClothing returns the item ID worn in a specific clothing slot
// Slots: 0=hair, 1=shirt, 2=pants, 3=feet, 4=face, 5=hand, 6=back, 7=head, 8=charm, 9=ances
func (p *Player) GetClothing(slot int) int {
	if slot >= 0 && slot < len(p.Clothing) {
		return p.Clothing[slot]
	}
	return 0
}

// IsEquipped returns whether an item is currently worn in any clothing slot.
func (p *Player) IsEquipped(itemID int) bool {
	if itemID <= 0 {
		return false
	}
	for _, id := range p.Clothing {
		if id == itemID {
			return true
		}
	}
	return false
}

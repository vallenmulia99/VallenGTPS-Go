package world

import (
	"math/rand"
)

// Tile state flags for state[2]
const (
	StateRight   = 0x00
	StateLeft    = 0x20
	StateSpliced = 0x1d // Tree has been spliced
	StateToggle  = 0x40
	StatePublic  = 0x80
)

// Tile state flags for state[3]
const (
	StateWater    = 0x04
	StateGlue     = 0x08
	StateFire     = 0x10
	StateRed      = 0x20
	StateGreen    = 0x40
	StateYellow   = StateRed | StateGreen
	StateBlue     = 0x80
	StateAqua     = StateGreen | StateBlue
	StatePurple   = StateRed | StateBlue
	StateCharcoal = StateRed | StateGreen | StateBlue
	StateVanish   = StateRed | StateYellow | StateGreen | StateAqua | StateBlue | StatePurple | StateCharcoal
)

// ApplyDamage increments hit counter for a tile
// Returns true if block should break
func (t *Tile) ApplyDamage(isForeground bool, requiredHits int) bool {
	if isForeground {
		t.Hits[0]++
		return t.Hits[0] >= uint8(requiredHits)
	}
	t.Hits[1]++
	return t.Hits[1] >= uint8(requiredHits)
}

// ResetHits clears hit counters
func (t *Tile) ResetHits() {
	t.Hits[0] = 0
	t.Hits[1] = 0
}

// SetHits sets hit counter directly (for tool bonuses)
func (t *Tile) SetHits(isForeground bool, hits int) {
	if isForeground {
		t.Hits[0] = uint8(hits)
	} else {
		t.Hits[1] = uint8(hits)
	}
}

// GetHits returns current hit count
func (t *Tile) GetHits(isForeground bool) int {
	if isForeground {
		return int(t.Hits[0])
	}
	return int(t.Hits[1])
}

// ToggleState toggles a state flag using XOR
func (t *Tile) ToggleState(stateIndex int, flag uint8) {
	t.State[stateIndex] ^= flag
}

// SetState sets a state flag using OR
func (t *Tile) SetState(stateIndex int, flag uint8) {
	t.State[stateIndex] |= flag
}

// ClearState clears a state flag using AND NOT
func (t *Tile) ClearState(stateIndex int, flag uint8) {
	t.State[stateIndex] &= ^flag
}

// HasState checks if a state flag is set
func (t *Tile) HasState(stateIndex int, flag uint8) bool {
	return (t.State[stateIndex] & flag) != 0
}

// DropCalculation calculates what to drop when a block breaks
type DropCalculation struct {
	Gems        int
	GemItemID   int // 112 = gem
	Seeds       int
	SeedItemID  int
	Blocks      int
	BlockItemID int
}

// CalculateDrops determines drops based on item rarity and ID
func CalculateDrops(itemID int, rarity int) DropCalculation {
	drop := DropCalculation{
		GemItemID:   112, // Gem item ID
		BlockItemID: itemID,
		SeedItemID:  itemID + 1,
	}

	// Rarity-based gem calculation
	// Higher rarity = more gems
	rarityToGem := 1
	if rarity >= 87 {
		rarityToGem = 22
	} else if rarity >= 68 {
		rarityToGem = 18
	} else if rarity >= 53 {
		rarityToGem = 14
	} else if rarity >= 41 {
		rarityToGem = 11
	} else if rarity >= 36 {
		rarityToGem = 10
	} else if rarity >= 32 {
		rarityToGem = 9
	} else if rarity >= 24 {
		rarityToGem = 5
	}

	// Double chance for farmables (high rarity items)
	dropChanceDivisor := 4
	if rarityToGem > 1 {
		dropChanceDivisor = 1 // farmables have higher drop rate
	}

	// Gem drop (25% base chance, higher for farmables)
	if rand.Intn(dropChanceDivisor+1) == 0 {
		drop.Gems = rand.Intn(rarityToGem) + 1
	}

	// Seed drop (40% chance for even item IDs)
	if itemID%2 == 0 && rand.Intn(100) < 40 {
		drop.Seeds = 1
	} else if rand.Intn(100) < 20 { // 20% chance otherwise
		drop.Blocks = 1
	}

	return drop
}

// CalculateXP returns XP to award based on rarity
func CalculateXP(rarity int) int {
	xp := 1 + (rarity / 5)
	if xp < 1 {
		xp = 1
	}
	return xp
}

// GetRequiredHits returns the number of hits required to break an item
// Falls back to default values if item data not available
func GetRequiredHits(itemID int) int {
	// Default hit values for common blocks
	switch itemID {
	case 2: // Dirt
		return 3
	case 14: // Cave Background
		return 3
	case 10: // Rock
		return 15
	case 4: // Lava
		return 25
	case 8: // Bedrock
		return 999 // Unbreakable
	case 242, 1796, 2408, 4994, 7188: // Locks
		return 6
	default:
		return 4 // Default
	}
}

// IsStrongBlock checks if block is too strong to break
func IsStrongBlock(itemID int) bool {
	return itemID == 8 // Bedrock
}

// IsMainDoor checks if block is main door
func IsMainDoor(itemID int) bool {
	return itemID == 6
}

// ProviderDrops defines what items providers drop
type ProviderDrop struct {
	ItemID int
	MinQty int
	MaxQty int
}

// GetProviderDrop returns what a provider block should drop
func GetProviderDrop(providerID int) *ProviderDrop {
	switch providerID {
	case 1008: // ATM Machine - drops gems (handled specially)
		return nil // Special case: drops random gems 1-100
	case 872: // Chicken
		return &ProviderDrop{ItemID: 874, MinQty: 1, MaxQty: 2} // Egg
	case 866: // Cow
		return &ProviderDrop{ItemID: 868, MinQty: 1, MaxQty: 2} // Milk
	case 1632: // Coffee Maker
		return &ProviderDrop{ItemID: 1634, MinQty: 1, MaxQty: 2} // Coffee
	case 3888: // Sheep
		return &ProviderDrop{ItemID: 3890, MinQty: 1, MaxQty: 2} // Wool
	case 5116: // Tea Set
		return &ProviderDrop{ItemID: 5114, MinQty: 1, MaxQty: 2} // Tea
	case 2798: // Well
		return &ProviderDrop{ItemID: 822, MinQty: 1, MaxQty: 2} // Water Bucket
	case 928: // Science Station - chemical drops (special)
		return nil // Special case: drops random chemicals
	default:
		return nil
	}
}

// GetScienceStationDrop returns random chemical from Science Station
func GetScienceStationDrop() int {
	roll := rand.Intn(100)
	if roll < 6 { // 1/16 chance
		return 918 // P (Purple)
	} else if roll < 18 { // 1/8 chance
		return 920 // B (Blue)
	} else if roll < 35 { // 1/6 chance
		return 924 // Y (Yellow)
	} else if roll < 60 { // 1/4 chance
		return 916 // R (Red)
	}
	return 914 // G (Green) - most common
}

// IsProviderBlock checks if block is a provider
func IsProviderBlock(itemID int) bool {
	switch itemID {
	case 1008, 872, 866, 1632, 3888, 5116, 2798, 928:
		return true
	}
	return false
}

// IsToggleableBlock checks if block can be toggled
func IsToggleableBlock(itemID int) bool {
	// Add toggleable block IDs here based on item type
	// For now, basic implementation
	return false
}

// IsRandomBlock checks if block has random values (Dice, Roshambo)
func IsRandomBlock(itemID int) bool {
	return itemID == 456 || itemID == 1300
}

// GetRandomValue returns random value for random blocks
func GetRandomValue(itemID int) int {
	if itemID == 456 { // Dice
		return rand.Intn(6) // 0-5
	} else if itemID == 1300 { // Roshambo (Rock, Paper, Scissors)
		return rand.Intn(3) + 1 // 1-3
	}
	return 0
}

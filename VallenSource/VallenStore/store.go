// Package store handles the Growtopia store system - item database, tabs, and purchase logic.
package store

import (
	"bufio"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"

	items "gtps/VallenSource/VallenItems"
)

// ─────────────────────────────────────────────
// Store Item Definition
// ─────────────────────────────────────────────

const (
	TabLocks    = 1
	TabItemPack = 2
	TabBigItems = 3
	TabWeather  = 4
	TabToken    = 5

	GrowtokenItemID    = 1486 // Growtoken item ID
	BackpackUpgradeID  = 9412 // Special: not given, increases slot_size
	BaseSlotSize       = 16   // Default slot size
	SlotsPerUpgrade    = 10   // Slots added per upgrade
	MaxUpgrades        = 38   // Max backpack upgrades
)

// Reward is a single {itemID, count} pair
type Reward struct {
	ID    int
	Count int
}

// StoreItem is one purchaseable entry
type StoreItem struct {
	Tab         int
	Btn         string   // button key (e.g. "world_lock")
	Name        string   // display name with color codes
	RTTX        string   // texture path
	Description string   // item description
	Tex1        string   // texture index 1
	Tex2        string   // texture index 2
	Cost        int      // gem cost (positive) or growtoken cost (negative for tab 5)
	Rewards     []Reward // fixed rewards
}

// IsGrowtokenItem returns true when item is in tab 5 (growtoken tab)
func (si *StoreItem) IsGrowtokenItem() bool {
	return si.Tab == TabToken
}

// GrowtokenCost returns the absolute cost in growtokens (tab 5 items have negative cost)
func (si *StoreItem) GrowtokenCost() int {
	if si.Cost < 0 {
		return -si.Cost
	}
	return si.Cost
}

// ─────────────────────────────────────────────
// Store Database
// ─────────────────────────────────────────────

var storeItems []StoreItem

// LoadStore parses resources/store.txt into storeItems slice
func LoadStore(path string) {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("[Store] Cannot open %s: %v — using empty store", path, err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// format: tab|btn|name|rttx|description|tex1|tex2|cost|id:amount[,id:amount...]
		parts := strings.SplitN(line, "|", 9)
		if len(parts) < 8 {
			continue
		}

		tab, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		cost, _ := strconv.Atoi(parts[7])

		si := StoreItem{
			Tab:         tab,
			Btn:         parts[1],
			Name:        parts[2],
			RTTX:        parts[3],
			Description: parts[4],
			Tex1:        parts[5],
			Tex2:        parts[6],
			Cost:        cost,
		}

		// Parse rewards: "242:1" or "12:1,20:1" or ""
		if len(parts) == 9 && parts[8] != "" {
			for _, pair := range strings.Split(parts[8], ",") {
				pair = strings.TrimSpace(pair)
				kv := strings.SplitN(pair, ":", 2)
				if len(kv) != 2 {
					continue
				}
				id, _ := strconv.Atoi(kv[0])
				cnt, _ := strconv.Atoi(kv[1])
				if id > 0 {
					si.Rewards = append(si.Rewards, Reward{id, cnt})
				}
			}
		}

		storeItems = append(storeItems, si)
	}

	log.Printf("[Store] Loaded %d items from %s", len(storeItems), path)
}

// FindItem looks up a store item by its button key
func FindItem(btn string) *StoreItem {
	for i := range storeItems {
		if storeItems[i].Btn == btn {
			return &storeItems[i]
		}
	}
	return nil
}

// GetTabItems returns all items for a given tab
func GetTabItems(tab int) []StoreItem {
	var out []StoreItem
	for _, si := range storeItems {
		if si.Tab == tab {
			out = append(out, si)
		}
	}
	return out
}

// ─────────────────────────────────────────────
// Backpack Upgrade Cost
// ─────────────────────────────────────────────

// UpgradeNumber returns which upgrade number this would be (1-based)
// Formula from C++: No = (slot_size - 16) / 10 + 1
func UpgradeNumber(currentSlotSize int) int {
	return (currentSlotSize-BaseSlotSize)/SlotsPerUpgrade + 1
}

// BackpackCost calculates the gem cost for the next backpack upgrade
// Formula from C++: cost = 100*No*No - 200*No + 200
func BackpackCost(currentSlotSize int) int {
	no := UpgradeNumber(currentSlotSize)
	return 100*no*no - 200*no + 200
}

// CanUpgradeBackpack returns false when already at max upgrades
func CanUpgradeBackpack(currentSlotSize int) bool {
	return UpgradeNumber(currentSlotSize) <= MaxUpgrades
}

// ─────────────────────────────────────────────
// Dynamic Reward Generation
// ─────────────────────────────────────────────

// GenerateRewards builds the actual reward list for special items.
// For fixed-reward items the Rewards slice from StoreItem is returned as-is.
func GenerateRewards(btn string) []Reward {
	switch btn {
	case "basic_splice":
		// 10 Rock Seeds + 10 random rarity-2 seeds
		result := []Reward{{ID: 11, Count: 10}} // Rock Seeds
		rarity2IDs := []int{3567, 2793, 57, 13, 17, 21, 101, 381, 1139}
		for i := 0; i < 10; i++ {
			result = append(result, Reward{ID: rarity2IDs[rand.Intn(len(rarity2IDs))], Count: 1})
		}
		return result

	case "rare_seed":
		// 5 random seeds rarity 13-60
		var pool []int
		for _, it := range items.AllItems() {
			if it.Type == items.TypeSeed && it.Rarity >= 13 && it.Rarity <= 60 {
				pool = append(pool, int(it.ID))
			}
		}
		if len(pool) == 0 {
			return nil
		}
		var result []Reward
		for i := 0; i < 5; i++ {
			result = append(result, Reward{ID: pool[rand.Intn(len(pool))], Count: 1})
		}
		return result

	case "clothes_pack":
		// 3 random clothing items rarity ≤10
		var pool []int
		for _, it := range items.AllItems() {
			if it.Type == items.TypeClothing && it.Rarity <= 10 {
				pool = append(pool, int(it.ID))
			}
		}
		if len(pool) == 0 {
			return nil
		}
		var result []Reward
		for i := 0; i < 3; i++ {
			result = append(result, Reward{ID: pool[rand.Intn(len(pool))], Count: 1})
		}
		return result

	case "rare_clothes_pack":
		// 3 random clothing items rarity 11-60
		var pool []int
		for _, it := range items.AllItems() {
			if it.Type == items.TypeClothing && it.Rarity >= 11 && it.Rarity <= 60 {
				pool = append(pool, int(it.ID))
			}
		}
		if len(pool) == 0 {
			return nil
		}
		var result []Reward
		for i := 0; i < 3; i++ {
			result = append(result, Reward{ID: pool[rand.Intn(len(pool))], Count: 1})
		}
		return result
	}

	return nil // caller uses StoreItem.Rewards for fixed items
}

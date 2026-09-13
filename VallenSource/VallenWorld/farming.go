package world

import (
	"math/rand"
	"time"
)

// PlantTreeWithFruit plants a seed with a specified initial fruit count
func (w *World) PlantTreeWithFruit(x, y, fruitID, fruitCount int, now time.Time) bool {
	if w.GetTile(x, y) == nil {
		return false
	}
	
	// Check if tree already exists at this position
	for _, t := range w.Trees {
		if t.TileX == x && t.TileY == y {
			return false
		}
	}
	
	w.Trees = append(w.Trees, Tree{
		FruitID:    fruitID,
		FruitCount: fruitCount,
		TileX:      x,
		TileY:      y,
		LastPick:   now,
	})
	return true
}

// CanSplice checks if two seeds can be spliced together
// Returns: (canSplice, resultSeedID, splice1Name, splice2Name)
func CanSplice(items interface{}, seed1ID, seed2ID int) (bool, int, string, string) {
	// This will be called from server with items.AllItems()
	// For now, return basic structure
	// The actual splice matching will be done in handler with items database
	return false, 0, "", ""
}

// IsSeedItem checks if an item is a seed (odd ID is generally a seed)
func IsSeedItem(itemID int) bool {
	// Seeds are usually odd IDs (block+1)
	// But we should check item type from items.dat
	return itemID%2 == 1 && itemID > 0
}

// GetFruitFromSeed returns the fruit/block ID from a seed ID
// Seeds are fruit_id + 1
func GetFruitFromSeed(seedID int) int {
	if seedID > 0 {
		return seedID - 1
	}
	return 0
}

// CalculateHarvestAmount returns random fruit count based on tree's fruit capacity
func CalculateHarvestAmount(fruitCapacity int) int {
	if fruitCapacity <= 0 {
		fruitCapacity = 1
	}
	// Random from 1 to fruit*3
	maxFruit := fruitCapacity * 3
	if maxFruit < 1 {
		maxFruit = 1
	}
	return rand.Intn(maxFruit) + 1
}

// SpliceResult holds the result of a splicing operation
type SpliceResult struct {
	Success       bool
	ResultSeedID  int
	Splice1Name   string
	Splice2Name   string
	ResultName    string
	ErrorMessage  string
}

// IsSpliced checks if a tile has been spliced
func (t *Tile) IsSpliced() bool {
	return t.State[2] == StateSpliced
}

// MarkAsSpliced sets the spliced flag on a tile
func (t *Tile) MarkAsSpliced() {
	t.State[2] = StateSpliced
}

// GetTreeAge returns how long a tree has been growing (in seconds)
func (w *World) GetTreeAge(x, y int, now time.Time) int {
	tree, found := w.FindTree(x, y)
	if !found {
		return 0
	}
	elapsed := now.Sub(tree.LastPick)
	return int(elapsed.Seconds())
}

// IsTreeMature checks if a tree is ready to harvest
func (w *World) IsTreeMature(x, y int, growTimeSec uint32, now time.Time) bool {
	tree, found := w.FindTree(x, y)
	if !found {
		return false
	}
	return w.TreeReady(tree, growTimeSec, now)
}

// UpdateTreeTime resets tree growth time (used after splicing)
func (w *World) UpdateTreeTime(x, y int, now time.Time) bool {
	for i := range w.Trees {
		if w.Trees[i].TileX == x && w.Trees[i].TileY == y {
			w.Trees[i].LastPick = now
			return true
		}
	}
	return false
}

// HarvestTreeWithCount harvests a tree and returns specific fruit count
func (w *World) HarvestTreeWithCount(x, y int, growTimeSec uint32, now time.Time) (fruitID int, fruitCount int, seedID int, ready bool) {
	for i, tree := range w.Trees {
		if tree.TileX != x || tree.TileY != y {
			continue
		}
		
		if !w.TreeReady(tree, growTimeSec, now) {
			return 0, 0, 0, false
		}
		
		// Calculate harvest amount
		fruitCount = CalculateHarvestAmount(tree.FruitCount)
		fruitID = tree.FruitID
		seedID = tree.FruitID + 1 // Seed is fruit + 1
		
		// Remove tree
		w.Trees = append(w.Trees[:i], w.Trees[i+1:]...)
		
		return fruitID, fruitCount, seedID, true
	}
	
	return 0, 0, 0, false
}

// SerializeTreeExtraData serializes tree data for tile updates
// Returns elapsed time (not absolute timestamp) and fruit count
func (w *World) SerializeTreeExtraData(x, y int, now time.Time) []byte {
	tree, found := w.FindTree(x, y)
	if !found {
		// Return empty tree data if not found
		data := make([]byte, 5)
		return data
	}
	
	// Calculate elapsed seconds since planting
	elapsed := int32(now.Sub(tree.LastPick).Seconds())
	
	// Format: int32(elapsed_seconds) + uint8(fruit_count)
	data := make([]byte, 5)
	data[0] = byte(elapsed)
	data[1] = byte(elapsed >> 8)
	data[2] = byte(elapsed >> 16)
	data[3] = byte(elapsed >> 24)
	data[4] = byte(tree.FruitCount)
	
	return data
}

package world

import (
	"fmt"
	"time"
)

// IsProtected reports whether a tile is inside ANY lock protection region.
// This is a simple check - for authorization, use CanEditTile instead.
func (w *World) IsProtected(x, y int) bool {
	return w.FindProtectingLock(x, y) != nil
}

// PlantTree records a planted seed and its growth timestamp.
func (w *World) PlantTree(x, y, fruitID int, now time.Time) bool {
	if w.GetTile(x, y) == nil || w.GetTile(x, y).Foreground != 0 {
		return false
	}
	for _, t := range w.Trees {
		if t.TileX == x && t.TileY == y {
			return false
		}
	}
	w.Trees = append(w.Trees, Tree{FruitID: fruitID, TileX: x, TileY: y, LastPick: now})
	return true
}

// TreeReady reports whether a tree has reached its configured growth time.
func (w *World) TreeReady(tree Tree, growTimeSec uint32, now time.Time) bool {
	if growTimeSec == 0 {
		return true
	}
	return !now.Before(tree.LastPick.Add(time.Duration(growTimeSec) * time.Second))
}

// HarvestTree removes a ready tree and returns its fruit payload.
// The caller supplies the item's GrowTimeSec so world state stays independent
// from the items database and remains JSON-compatible.
func (w *World) HarvestTree(x, y int, growTimeSec uint32, now time.Time) (Tree, bool) {
	for i, tree := range w.Trees {
		if tree.TileX != x || tree.TileY != y || !w.TreeReady(tree, growTimeSec, now) {
			continue
		}
		w.Trees = append(w.Trees[:i], w.Trees[i+1:]...)
		return tree, true
	}
	return Tree{}, false
}

// FindTree returns the tree recorded at a tile, if any.
func (w *World) FindTree(x, y int) (Tree, bool) {
	for _, tree := range w.Trees {
		if tree.TileX == x && tree.TileY == y {
			return tree, true
		}
	}
	return Tree{}, false
}

// UpdateTreeFruit records a new fruit count without changing its planting time.
func (w *World) UpdateTreeFruit(x, y, count int) bool {
	for i := range w.Trees {
		if w.Trees[i].TileX == x && w.Trees[i].TileY == y {
			w.Trees[i].FruitCount = count
			return true
		}
	}
	return false
}
// ValidatePlacement memeriksa apakah suatu item dapat ditempatkan pada tile tertentu.
func (w *World) ValidatePlacement(tile *Tile, isClothing bool, itemType uint8) error {
	if isClothing {
		return fmt.Errorf("You can't place clothing as a block!")
	}
	switch itemType {
	case 1: // Consumable
		return fmt.Errorf("You can't place this item as a block!")
	case 0: // Fist
		return fmt.Errorf("You can't place a fist as a block!")
	case 2: // Wrench
		return fmt.Errorf("You can't place a wrench as a block!")
	}

	// Background check (TypeBackground = 18 / 0x12)
	isBackground := itemType == 18
	if isBackground && tile.Background != 0 {
		return fmt.Errorf("Background already occupied.")
	}
	if !isBackground && tile.Foreground != 0 {
		return fmt.Errorf("Foreground already occupied.")
	}
	return nil
}

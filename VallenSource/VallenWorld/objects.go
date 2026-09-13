package world

import "math"

const (
	MaxObjectStack = 200 // Max items per object stack
	GemItemID      = 112 // Gem item ID
)

// FindObjectByUID finds an object by its unique ID
func (w *World) FindObjectByUID(uid int) *DroppedItem {
	for i := range w.DroppedItems {
		if w.DroppedItems[i].UID == uid {
			return &w.DroppedItems[i]
		}
	}
	return nil
}

// FindObjectAtTile finds an object of the same item ID at a tile position
func (w *World) FindObjectAtTile(itemID int, tileX, tileY int) *DroppedItem {
	for i := range w.DroppedItems {
		obj := &w.DroppedItems[i]
		if obj.ItemID == itemID {
			// Check if object is at the same tile (convert pixel to tile)
			objTileX := int(obj.X) / 32
			objTileY := int(obj.Y) / 32
			if objTileX == tileX && objTileY == tileY {
				return obj
			}
		}
	}
	return nil
}

// ObjectChange describes exactly how a world object changed. Clients must
// receive a spawn packet for new objects and an update packet for merged ones.
// Sending a second spawn for an existing UID leaves a ghost drop until map
// reload, so this distinction is protocol-critical.
type ObjectChange struct {
	UID     int
	Created bool
}

// AddObject adds a dropped object while respecting the 200-item object cap.
// It returns every affected object, because a large drop can both merge into
// an existing stack and create one or more additional stacks.
func (w *World) AddObject(itemID, count int, pixelX, pixelY float32) []ObjectChange {
	if itemID <= 0 || count <= 0 {
		return nil
	}

	remaining := count
	changes := make([]ObjectChange, 0, 1)

	// Fill every partially-full stack within 16 pixels (matching GrowTavern C++ WorldInfo.h:2741).
	for i := range w.DroppedItems {
		obj := &w.DroppedItems[i]
		if obj.ItemID != itemID || obj.Count >= MaxObjectStack {
			continue
		}
		if math.Abs(float64(obj.X-pixelX)) <= 16 && math.Abs(float64(obj.Y-pixelY)) <= 16 {
			added := MaxObjectStack - obj.Count
			if added > remaining {
				added = remaining
			}
			obj.Count += added
			remaining -= added
			changes = append(changes, ObjectChange{UID: obj.UID, Created: false})
			if remaining == 0 {
				return changes
			}
		}
	}

	// Create normal-sized stacks for the remainder.
	for remaining > 0 {
		stack := remaining
		if stack > MaxObjectStack {
			stack = MaxObjectStack
		}
		spawnX, spawnY := pixelX, pixelY
		if len(changes) > 0 {
			// Keep independent overflow stacks visually separated but in range.
			spawnX += float32((len(changes)%3)*8)
			spawnY += float32((len(changes)%3)*8)
		}
		uid := w.CreateNewObject(itemID, stack, spawnX, spawnY)
		changes = append(changes, ObjectChange{UID: uid, Created: true})
		remaining -= stack
	}

	return changes
}

// CreateNewObject creates a brand new object with unique UID
func (w *World) CreateNewObject(itemID, count int, pixelX, pixelY float32) int {
	w.LastObjectUID++
	
	newObj := DroppedItem{
		ItemID: itemID,
		Count:  count,
		X:      pixelX,
		Y:      pixelY,
		UID:    w.LastObjectUID,
	}
	
	w.DroppedItems = append(w.DroppedItems, newObj)
	return w.LastObjectUID
}

// AddGems handles gem dropping with intelligent auto-merging:
// Combines nearby gem drops within 17 pixels into standard denominations:
// Yellow (1), Blue (5), Red (10), Green (50), Purple (100), BGL (1000 - itemID 4490).
func (w *World) AddGems(count int, pixelX, pixelY float32) (removedUIDs []int, createdUIDs []int) {
	if count <= 0 {
		return nil, nil
	}

	totalGems := count
	// Find all existing gems within 17 pixels to merge
	var toRemove []int
	for _, obj := range w.DroppedItems {
		if obj.ItemID == GemItemID {
			if math.Abs(float64(obj.X-pixelX)) <= 17 && math.Abs(float64(obj.Y-pixelY)) <= 17 {
				totalGems += obj.Count
				toRemove = append(toRemove, obj.UID)
			}
		}
	}

	// Remove old gem drops
	for _, uid := range toRemove {
		w.RemoveObject(uid)
	}
	removedUIDs = toRemove

	// Breakdown total into standard GT tiers
	bglCount := totalGems / 1000
	rem := totalGems % 1000

	purpleCount := rem / 100
	rem %= 100

	greenCount := rem / 50
	rem %= 50

	redCount := rem / 10
	rem %= 10

	blueCount := rem / 5
	yellowCount := rem % 5

	// Spawn BGLs (item 4490)
	for i := 0; i < bglCount; i++ {
		uid := w.CreateNewObject(4490, 1, pixelX, pixelY)
		createdUIDs = append(createdUIDs, uid)
	}
	// Spawn Purple Gems (100)
	for i := 0; i < purpleCount; i++ {
		uid := w.CreateNewObject(GemItemID, 100, pixelX, pixelY)
		createdUIDs = append(createdUIDs, uid)
	}
	// Spawn Green Gems (50)
	for i := 0; i < greenCount; i++ {
		uid := w.CreateNewObject(GemItemID, 50, pixelX, pixelY)
		createdUIDs = append(createdUIDs, uid)
	}
	// Spawn Red Gems (10)
	for i := 0; i < redCount; i++ {
		uid := w.CreateNewObject(GemItemID, 10, pixelX, pixelY)
		createdUIDs = append(createdUIDs, uid)
	}
	// Spawn Blue Gems (5)
	for i := 0; i < blueCount; i++ {
		uid := w.CreateNewObject(GemItemID, 5, pixelX, pixelY)
		createdUIDs = append(createdUIDs, uid)
	}
	// Spawn Yellow Gems (1)
	for i := 0; i < yellowCount; i++ {
		uid := w.CreateNewObject(GemItemID, 1, pixelX, pixelY)
		createdUIDs = append(createdUIDs, uid)
	}

	return removedUIDs, createdUIDs
}

// RemoveObject removes an object by UID
func (w *World) RemoveObject(uid int) bool {
	for i := range w.DroppedItems {
		if w.DroppedItems[i].UID == uid {
			// Remove from slice
			w.DroppedItems = append(w.DroppedItems[:i], w.DroppedItems[i+1:]...)
			return true
		}
	}
	return false
}

// UpdateObjectCount updates an object's count
func (w *World) UpdateObjectCount(uid int, newCount int) bool {
	obj := w.FindObjectByUID(uid)
	if obj == nil {
		return false
	}
	
	obj.Count = newCount
	
	// Remove if count reaches 0
	if newCount <= 0 {
		w.RemoveObject(uid)
	}
	
	return true
}

// PickupResult holds the result of picking up an object
type PickupResult struct {
	Success       bool
	ItemID        int
	CollectedQty  int
	Overflow      int
	IsGem         bool
	Message       string
}

// PickupObject attempts to pick up an object
// Returns collected amount and overflow amount
func (w *World) PickupObject(uid int, inventoryAddFunc func(itemID, count int) int) *PickupResult {
	obj := w.FindObjectByUID(uid)
	if obj == nil {
		return &PickupResult{Success: false}
	}
	
	result := &PickupResult{
		Success: true,
		ItemID:  obj.ItemID,
		IsGem:   obj.ItemID == GemItemID || obj.ItemID == 4490,
	}
	
	// Special handling for gems
	if result.IsGem {
		if obj.ItemID == 4490 {
			result.CollectedQty = obj.Count * 1000
		} else {
			result.CollectedQty = obj.Count
		}
		result.Overflow = 0
		w.RemoveObject(uid)
		return result
	}
	
	// Try to add to inventory
	overflow := inventoryAddFunc(obj.ItemID, obj.Count)
	result.CollectedQty = obj.Count - overflow
	result.Overflow = overflow
	
	if overflow > 0 {
		// Update object with overflow amount
		obj.Count = overflow
	} else {
		// Fully collected, remove object
		w.RemoveObject(uid)
	}
	
	return result
}

// GetObjectData returns object data for serialization
func (w *World) GetObjectData() []DroppedItem {
	return w.DroppedItems
}

// GetObjectCount returns number of objects in world
func (w *World) GetObjectCount() int {
	return len(w.DroppedItems)
}

// ClearObjects removes all objects from world
func (w *World) ClearObjects() {
	w.DroppedItems = []DroppedItem{}
}

// ObjectPacketData maps directly to PACKET_ITEM_CHANGE_OBJECT header fields.
// Growtopia client uses type 0x0e with specific field semantics:
//   Offset  8 (NetID) : dropper/collector NetID, or -1 for world spawn
//   Offset 12 (UID)   : -1 = new spawn flag, else existing object UID
//   Offset 16 (State) : object UID for spawn packets (so client can track it)
//   Offset 20 (Count) : item count as raw uint32 bits in float32 field
//   Offset 24 (ID)    : item ID for spawn/update, object UID for remove
//   Offset 28 (PosX)  : X position
//   Offset 32 (PosY)  : Y position
type ObjectPacketData struct {
	NetID  int32   // dropper/collector NetID
	UID    int32   // -1 = new spawn, else object UID
	State  int32   // object UID for spawn packets
	Count  float32 // Item count (raw uint32 bitcast into float32)
	ItemID int32   // Item ID (spawn/update) or Object UID (remove)
	X      float32 // X position
	Y      float32 // Y position
}

// BuildSpawnPacketData creates data for spawning new object.
// In Growtopia protocol, NetID must be -1 (0xFFFFFFFF) for spawning dropped items.
// UID at offset 12 is the dropper's NetID (for throw animation) or -1.
// Count at offset 20 MUST be an IEEE-754 float32 (e.g. 10.0f), NOT raw uint32 bitcast!
func BuildSpawnPacketData(obj *DroppedItem, dropperNetID int) ObjectPacketData {
	targetUID := int32(-1)
	if dropperNetID >= 0 {
		targetUID = int32(dropperNetID)
	}
	return ObjectPacketData{
		NetID:  int32(-1),
		UID:    targetUID,
		State:  0,
		Count:  float32(obj.Count),
		ItemID: int32(obj.ItemID),
		X:      obj.X,
		Y:      obj.Y,
	}
}

// BuildUpdatePacketData creates data for updating existing object
func BuildUpdatePacketData(obj *DroppedItem) ObjectPacketData {
	return ObjectPacketData{
		NetID:  int32(-3), // -3 = update existing object
		UID:    int32(obj.UID),
		State:  0,
		Count:  float32(obj.Count),
		ItemID: int32(obj.ItemID),
		X:      obj.X,
		Y:      obj.Y,
	}
}

// BuildRemovePacketData creates data for removing object (pickup)
func BuildRemovePacketData(uid int, playerNetID int) ObjectPacketData {
	return ObjectPacketData{
		NetID:  int32(playerNetID),
		UID:    int32(0),
		State:  0,
		ItemID: int32(uid), // ID field contains the UID when removing
		Count:  0,
		X:      0,
		Y:      0,
	}
}

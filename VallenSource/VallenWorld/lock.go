package world

import "strings"

// Lock item IDs and constants
const (
	ItemSmallLock        = 202   // 5x5 area
	ItemBigLock          = 204   // 11x11 area
	ItemHugeLock         = 206   // 21x21 area
	ItemWorldLock        = 242   // Entire world
	ItemDiamondLock      = 1796  // Entire world
	ItemBlueGemLock      = 7188  // Entire world
	ItemPlatinumGemLock  = 11550 // Entire world
	ItemEmeraldLock      = 11586 // Entire world
	ItemRubyLock         = 11902 // Entire world
	ItemRoyalLock        = 5814  // Entire world
	ItemRoboticLock      = 4802  // Entire world
	ItemDragonLock       = 2408  // Entire world
	ItemLegendaryLock    = 10000 // Entire world
	ItemBuilderLock      = 4994  // 41x41 area

	// Lock state flags
	LockStateDisableMusic       = 0x10
	LockStateDisableMusicRender = 0x20
	LockStateRainbows           = 0x80

	// Tile state flags for lock
	TileStatePublic = 0x80 // state[2]
)

// Lock radius by item type (in tiles)
var LockRadius = map[int]int{
	ItemWorldLock:       200,
	ItemDiamondLock:     200,
	ItemBlueGemLock:     200,
	ItemPlatinumGemLock: 200,
	ItemEmeraldLock:     200,
	ItemRubyLock:        200,
	ItemRoyalLock:       200,
	ItemRoboticLock:     200,
	ItemDragonLock:      200,
	ItemLegendaryLock:   200,
	ItemSmallLock:       2,  // 5x5 (2 tiles radius from center)
	ItemBigLock:         5,  // 11x11
	ItemHugeLock:        10, // 21x21
	ItemBuilderLock:     20, // 41x41
}

// IsWorldLevelLock checks if lock covers the whole world (WL, DL, BGL, Royal, etc)
func IsWorldLevelLock(itemID int) bool {
	switch itemID {
	case ItemWorldLock, ItemDiamondLock, ItemBlueGemLock, ItemPlatinumGemLock,
		ItemEmeraldLock, ItemRubyLock, ItemRoyalLock, ItemRoboticLock,
		ItemDragonLock, ItemLegendaryLock:
		return true
	default:
		return false
	}
}

// IsTileLock checks if lock is a tile-specific area lock (Small, Big, Huge, Builder)
func IsTileLock(itemID int) bool {
	switch itemID {
	case ItemSmallLock, ItemBigLock, ItemHugeLock, ItemBuilderLock:
		return true
	default:
		return false
	}
}

// IsWorldLockItem checks if an item is any type of lock
func IsWorldLockItem(itemID int) bool {
	return IsWorldLevelLock(itemID) || IsTileLock(itemID)
}

// GetLockRadius returns the protection radius for a lock item
func GetLockRadius(itemID int) int {
	if radius, ok := LockRadius[itemID]; ok {
		return radius
	}
	return 0
}

// FindLockAt returns the lock at specific tile position
func (w *World) FindLockAt(x, y int) *Lock {
	for i := range w.Locks {
		if w.Locks[i].TileX == x && w.Locks[i].TileY == y {
			return &w.Locks[i]
		}
	}
	return nil
}

// FindTileLockAt returns a tile-specific lock (Small, Big, Huge, Builder) protecting (x, y)
func (w *World) FindTileLockAt(x, y int) *Lock {
	for i := range w.Locks {
		lock := &w.Locks[i]
		if IsTileLock(lock.ItemID) && IsInLockRange(lock.TileX, lock.TileY, lock.Radius, x, y) {
			return lock
		}
	}
	return nil
}

// GetWorldLock returns the primary world-level lock (World Lock, DL, BGL, Royal, etc)
func (w *World) GetWorldLock() *Lock {
	for i := range w.Locks {
		if IsWorldLevelLock(w.Locks[i].ItemID) {
			return &w.Locks[i]
		}
	}
	return nil
}

// FindProtectingLock returns the lock that protects a specific tile position
func (w *World) FindProtectingLock(x, y int) *Lock {
	// Tile-specific locks take precedence inside their bounds
	if tileLock := w.FindTileLockAt(x, y); tileLock != nil {
		return tileLock
	}
	// Fall back to world lock
	return w.GetWorldLock()
}

// IsInLockRange checks if target tile (tx, ty) is within lock's protection range
func IsInLockRange(lockX, lockY, radius, targetX, targetY int) bool {
	dx := abs(lockX - targetX)
	dy := abs(lockY - targetY)
	return dx <= radius && dy <= radius
}

// CanEditTile checks if a user can modify a tile at position (x, y)
// Bulletproof permission model:
// 1. Staff / Admin always bypass
// 2. Tile lock takes precedence (Small, Big, Huge, Builder)
// 3. World lock protects the entire world if world is owned
// 4. Returns false if locked and user lacks permission
func (w *World) CanEditTile(userID int, x, y int, isAdmin bool) bool {
	if isAdmin {
		return true
	}

	// 1. Check tile-specific lock
	if tileLock := w.FindTileLockAt(x, y); tileLock != nil {
		if tileLock.Owner == userID {
			return true
		}
		for _, allowed := range tileLock.AccessList {
			if allowed != 0 && allowed == userID {
				return true
			}
		}
		if tileLock.IsPublic {
			return true
		}
		return false
	}

	// 2. Check world-level lock (World Lock, DL, BGL, Royal Lock, etc.)
	if w.Owner != 0 || w.HasWorldLock() {
		if w.Owner == userID {
			return true
		}
		for _, allowed := range w.AccessList {
			if allowed != 0 && allowed == userID {
				return true
			}
		}
		if wl := w.GetWorldLock(); wl != nil {
			if wl.Owner == userID {
				return true
			}
			for _, allowed := range wl.AccessList {
				if allowed != 0 && allowed == userID {
					return true
				}
			}
			if wl.IsPublic {
				return true
			}
		}
		if w.IsPublic {
			return true
		}
		// World is locked and player is not owner, not in access list, and not public
		return false
	}

	// No lock exists in world -> free to edit
	return true
}

// AddLock adds a new lock to the world
func (w *World) AddLock(itemID, x, y, owner int) *Lock {
	radius := GetLockRadius(itemID)

	newLock := Lock{
		ItemID:     itemID,
		TileX:      x,
		TileY:      y,
		Owner:      owner,
		AccessList: [20]int{},
		IsPublic:   false,
		LockState:  0,
		Radius:     radius,
	}

	// World level locks become primary world owner
	if IsWorldLevelLock(itemID) && w.Owner == 0 {
		w.Owner = owner
		newLock.AccessList = w.AccessList
		newLock.IsPublic = w.IsPublic
		newLock.LockState = w.LockState
	}

	w.Locks = append(w.Locks, newLock)
	return &w.Locks[len(w.Locks)-1]
}

// RemoveLock removes a lock at position (x, y)
func (w *World) RemoveLock(x, y int) bool {
	for i := range w.Locks {
		if w.Locks[i].TileX == x && w.Locks[i].TileY == y {
			// If this was a World Lock / World-level lock, reset world owner
			if IsWorldLevelLock(w.Locks[i].ItemID) {
				w.Owner = 0
				w.OwnerName = ""
				w.AccessList = [20]int{}
				w.IsPublic = false
				w.LockState = 0
			}

			// Remove lock from slice
			w.Locks = append(w.Locks[:i], w.Locks[i+1:]...)
			return true
		}
	}
	return false
}

// UpdateLockPublic updates the public state of a lock
func (w *World) UpdateLockPublic(x, y int, isPublic bool) bool {
	lock := w.FindLockAt(x, y)
	if lock == nil {
		return false
	}

	lock.IsPublic = isPublic

	if IsWorldLevelLock(lock.ItemID) {
		w.IsPublic = isPublic
	}

	// Update tile state flag
	tile := w.GetTile(x, y)
	if tile != nil {
		if isPublic {
			tile.State[2] |= TileStatePublic
		} else {
			tile.State[2] &= (^uint8(TileStatePublic))
		}
	}

	return true
}

// UpdateLockState updates lock state flags (music, etc)
func (w *World) UpdateLockState(x, y int, lockState int) bool {
	lock := w.FindLockAt(x, y)
	if lock == nil {
		return false
	}

	lock.LockState = lockState

	if IsWorldLevelLock(lock.ItemID) {
		w.LockState = lockState
	}

	return true
}

// AddToAccessList adds a user to lock's access list
func (w *World) AddToAccessList(lockX, lockY int, userID int) bool {
	lock := w.FindLockAt(lockX, lockY)
	if lock == nil {
		return false
	}

	// Find empty slot
	for i := range lock.AccessList {
		if lock.AccessList[i] == 0 {
			lock.AccessList[i] = userID

			if IsWorldLevelLock(lock.ItemID) {
				w.AccessList[i] = userID
			}
			return true
		}
	}

	return false
}

// RemoveFromAccessList removes a user from lock's access list
func (w *World) RemoveFromAccessList(lockX, lockY int, userID int) bool {
	lock := w.FindLockAt(lockX, lockY)
	if lock == nil {
		return false
	}

	for i := range lock.AccessList {
		if lock.AccessList[i] == userID {
			lock.AccessList[i] = 0

			if IsWorldLevelLock(lock.ItemID) {
				w.AccessList[i] = 0
			}
			return true
		}
	}

	return false
}

// HasWorldLock checks if world has any World Lock placed
func (w *World) HasWorldLock() bool {
	if w.Owner != 0 {
		return true
	}
	for i := range w.Locks {
		if IsWorldLevelLock(w.Locks[i].ItemID) {
			return true
		}
	}
	return false
}

// MigrateLegacyWorldLock restores a World Lock record for older saves
func (w *World) MigrateLegacyWorldLock() bool {
	if w.Owner == 0 || w.HasWorldLock() {
		return false
	}

	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			tile := w.GetTile(x, y)
			if tile == nil || !IsWorldLevelLock(tile.Foreground) {
				continue
			}

			w.Locks = append(w.Locks, Lock{
				ItemID:     tile.Foreground,
				TileX:      x,
				TileY:      y,
				Owner:      w.Owner,
				OwnerName:  w.OwnerName,
				AccessList: w.AccessList,
				IsPublic:   w.IsPublic,
				LockState:  w.LockState,
				Radius:     GetLockRadius(tile.Foreground),
			})
			return true
		}
	}

	return false
}

// ResetLock resets lock data (called when lock is broken)
func (w *World) ResetLock() {
	w.Owner = 0
	w.OwnerName = ""
	w.AccessList = [20]int{}
	w.IsPublic = false
	w.LockState = 0
	w.MinimumEntryLevel = 1
	w.Locks = []Lock{}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// HashGrowID computes the FNV-1a based user hash used by Growtopia for Packet 15 and avatar IDs.
func HashGrowID(name string) uint32 {
	clean := strings.ToLower(name)
	var hash uint32 = 0x811c9dc5
	const prime uint32 = 0x1000193
	for i := 0; i < len(clean); i++ {
		hash ^= uint32(clean[i])
		hash *= prime
	}
	return hash / 100
}

// GetLockedTiles returns the tile indices (x + y*worldWidth) protected by this lock.
func (w *World) GetLockedTiles(lock *Lock) []uint16 {
	if lock == nil || IsWorldLevelLock(lock.ItemID) {
		return nil
	}
	var indices []uint16
	for y := lock.TileY - lock.Radius; y <= lock.TileY+lock.Radius; y++ {
		if y < 0 || y >= w.Height {
			continue
		}
		for x := lock.TileX - lock.Radius; x <= lock.TileX+lock.Radius; x++ {
			if x < 0 || x >= w.Width {
				continue
			}
			indices = append(indices, uint16(x+y*w.Width))
		}
	}
	return indices
}

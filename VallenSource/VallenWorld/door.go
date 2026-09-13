package world

import "strings"

// FindDoorAt finds a door at the specified tile position
func (w *World) FindDoorAt(tileX, tileY int) *Door {
	for i := range w.Doors {
		if w.Doors[i].TileX == tileX && w.Doors[i].TileY == tileY {
			return &w.Doors[i]
		}
	}
	return nil
}

// FindDoorByID finds a door by its unique ID
func (w *World) FindDoorByID(doorID string) *Door {
	if doorID == "" {
		return nil
	}
	
	for i := range w.Doors {
		if w.Doors[i].ID == doorID {
			return &w.Doors[i]
		}
	}
	return nil
}

// AddDoor adds a new door to the world
func (w *World) AddDoor(label, dest, id string, tileX, tileY int) {
	door := Door{
		Label: label,
		Dest:  dest,
		ID:    id,
		TileX: tileX,
		TileY: tileY,
	}
	w.Doors = append(w.Doors, door)
}

// RemoveDoorAt removes a door at the specified tile
func (w *World) RemoveDoorAt(tileX, tileY int) bool {
	for i := range w.Doors {
		if w.Doors[i].TileX == tileX && w.Doors[i].TileY == tileY {
			w.Doors = append(w.Doors[:i], w.Doors[i+1:]...)
			return true
		}
	}
	return false
}

// UpdateDoor updates an existing door or creates new one if not found
func (w *World) UpdateDoor(label, dest, id string, tileX, tileY int) {
	door := w.FindDoorAt(tileX, tileY)
	if door != nil {
		door.Label = label
		door.Dest = dest
		door.ID = id
	} else {
		w.AddDoor(label, dest, id, tileX, tileY)
	}
}

// ParseDoorDestination parses door destination format
// Returns worldName, doorID, isValid
// Format: "WORLDNAME:ID" or ":ID" (same world)
func ParseDoorDestination(dest string) (string, string, bool) {
	if dest == "" {
		return "", "", false
	}
	
	parts := strings.SplitN(dest, ":", 2)
	if len(parts) == 1 {
		// No colon, treat as world name only
		return strings.ToUpper(strings.TrimSpace(parts[0])), "", true
	}
	
	// Has colon
	worldName := strings.ToUpper(strings.TrimSpace(parts[0]))
	doorID := strings.TrimSpace(parts[1])
	
	return worldName, doorID, true
}

// ValidateWorldName checks if world name is valid
// Max 24 chars, alphanumeric plus underscore. Official private Dungeon
// instances use names such as DUNGEON__20.
func ValidateWorldName(name string) bool {
	if len(name) == 0 || len(name) > 24 {
		return false
	}
	
	for _, c := range name {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	
	return true
}

// FindSignAt finds a sign at the specified tile position
func (w *World) FindSignAt(tileX, tileY int) *Sign {
	for i := range w.Signs {
		if w.Signs[i].TileX == tileX && w.Signs[i].TileY == tileY {
			return &w.Signs[i]
		}
	}
	return nil
}

// UpdateSign updates an existing sign or creates new one
func (w *World) UpdateSign(label string, tileX, tileY int) {
	sign := w.FindSignAt(tileX, tileY)
	if sign != nil {
		sign.Label = label
	} else {
		w.Signs = append(w.Signs, Sign{
			Label: label,
			TileX: tileX,
			TileY: tileY,
		})
	}
}

// RemoveSignAt removes a sign at the specified tile
func (w *World) RemoveSignAt(tileX, tileY int) bool {
	for i := range w.Signs {
		if w.Signs[i].TileX == tileX && w.Signs[i].TileY == tileY {
			w.Signs = append(w.Signs[:i], w.Signs[i+1:]...)
			return true
		}
	}
	return false
}

// GetCheckpointTile gets the tile at checkpoint position
// Returns tile position from spawn point (rest_pos)
func (w *World) GetCheckpointTile(spawnX, spawnY int) *Tile {
	if spawnX < 0 || spawnX >= w.Width || spawnY < 0 || spawnY >= w.Height {
		return nil
	}
	
	index := spawnY*w.Width + spawnX
	if index >= 0 && index < len(w.Tiles) {
		return &w.Tiles[index]
	}
	
	return nil
}

// ToggleCheckpoint toggles checkpoint state
// Untoggles old checkpoint, sets new checkpoint
func (w *World) ToggleCheckpoint(oldX, oldY, newX, newY int) {
	// Untoggle old checkpoint
	if oldTile := w.GetCheckpointTile(oldX, oldY); oldTile != nil {
		oldTile.ClearState(2, StateToggle) // state[2], S_TOGGLE flag
	}
	
	// Toggle new checkpoint
	if newTile := w.GetCheckpointTile(newX, newY); newTile != nil {
		newTile.SetState(2, StateToggle) // state[2], S_TOGGLE flag
	}
}

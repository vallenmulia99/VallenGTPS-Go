package dungeon

import (
	"fmt"

	world "gtps/VallenSource/VallenWorld"
)

const (
	floorY    = 43
	exitX     = 90
	exitY     = 42
	spawnX    = 10
	spawnY    = 42
	targetID  = 10 // Rock: visible and has a familiar damage animation.
	borderID  = 8  // Bedrock: the dungeon boundary cannot be broken.
)

func objectivesForRoom(room int) map[Target]bool {
	// Each room shifts the trial stones slightly. The fixed layout keeps this
	// initial engine deterministic while combat/mobs are added later.
	offset := (room - 1) * 2
	return map[Target]bool{
		{X: 26 + offset, Y: 42}: true,
		{X: 46, Y: 42}:              true,
		{X: 66 - offset, Y: 42}:     true,
	}
}

// BuildRoom resets a private world into one safe room of a dungeon run.
// No normal-world drops, locks, trees or mutable map state are retained.
func BuildRoom(w *world.World, run *Run) {
	if w == nil || run == nil {
		return
	}
	w.Owner = 0
	w.IsPublic = false
	w.MinimumEntryLevel = 1
	w.Locks = nil
	w.DroppedItems = nil
	w.Trees = nil
	w.Doors = nil
	w.Signs = nil
	w.Weather = 25 // 25 = FIRE_HAZE (DUNGEON WEATHER ID!)
	w.SpawnTileX, w.SpawnTileY = spawnX, spawnY

	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			tile := w.GetTile(x, y)
			*tile = world.Tile{Background: 14}
			if x == 0 || x == w.Width-1 || y >= floorY {
				tile.Foreground = borderID
			}
		}
	}

	for target := range run.Objectives {
		if tile := w.GetTile(target.X, target.Y); tile != nil {
			tile.Foreground = targetID
		}
	}
	if exit := w.GetTile(exitX, exitY); exit != nil {
		exit.Foreground = 6
	}
	w.AddDoor("NEXT ROOM", "", "DUNGEON_NEXT", exitX, exitY)
	w.UpdateSign(fmt.Sprintf("DUNGEON ROOM %d/%d: destroy all Trial Stones, then use the door.", run.Room, MaxRooms), 8, 39)
	if sign := w.GetTile(8, 39); sign != nil {
		sign.Foreground = 20
	}
}

package world

import (
	"encoding/binary"
	"math"
	"math/rand"
	"strings"
	"time"
)

const (
	WorldWidth  = 100
	WorldHeight = 60
)

// Lock represents a lock tile (World/Small/Big/Huge/Builder Lock)
type Lock struct {
	ItemID     int     `json:"item_id"`      // 242=World, 202=Small, 204=Big, 206=Huge, 4994=Builder
	TileX      int     `json:"tile_x"`       // Lock tile position
	TileY      int     `json:"tile_y"`       // Lock tile position
	Owner      int     `json:"owner"`        // user_id of lock owner
	OwnerName  string  `json:"owner_name"`   // GrowID of lock owner
	AccessList [20]int `json:"access_list"`  // user_ids with access to this lock area
	IsPublic   bool    `json:"is_public"`    // Public lock allows anyone to build
	LockState  int     `json:"lock_state"`   // Lock flags (DISABLE_MUSIC, etc)
	Radius     int     `json:"radius"`       // Protection radius in tiles
}

type World struct {
	// Identity
	Name      string `json:"name"`
	Owner     int    `json:"owner"`      // user_id of primary world lock owner (0 = no world lock)
	OwnerName string `json:"owner_name"` // GrowID of primary world lock owner

	// World-level access (for World Lock only)
	AccessList        [20]int `json:"access_list"` // user_ids for world lock access
	IsPublic          bool    `json:"is_public"`
	LockState         int     `json:"lock_state"`
	MinimumEntryLevel int     `json:"minimum_entry_level"`

	// Multiple locks support (tile-specific locks)
	Locks []Lock `json:"locks"` // All locks in world (Small/Big/Huge/Builder/World)

	// Dimensions
	Width  int `json:"width"`
	Height int `json:"height"`

	// State (runtime only)
	Visitors      int `json:"-"`
	NetIDCounter  int `json:"-"`
	LastObjectUID int `json:"last_object_uid"`

	// Tiles
	Tiles []Tile `json:"tiles"` // 1D array: width * height (6000)

	// Objects
	DroppedItems []DroppedItem    `json:"dropped_items"`
	Doors        []Door           `json:"doors"`
	Signs        []Sign           `json:"signs"`
	Trees        []Tree           `json:"trees"`
	Vendings     []VendingMachine `json:"vendings"`
	Displays     []DisplayBox     `json:"displays"`
	Donations    []DonationBox    `json:"donations"`
	BannedUsers  []BannedPlayer   `json:"banned_users"`

	// Jammers & Geiger
	PunchJammerActive  bool `json:"punch_jammer_active"`
	ZombieJammerActive bool `json:"zombie_jammer_active"`
	GeigerX            int  `json:"geiger_x"`
	GeigerY            int  `json:"geiger_y"`

	// Spawn (tile coords)
	SpawnTileX int `json:"spawn_tile_x"`
	SpawnTileY int `json:"spawn_tile_y"`

	Weather     int       `json:"weather"`
	LastVisited time.Time `json:"last_visited"`
}

type Tile struct {
	Foreground int      `json:"fg"`
	Background int      `json:"bg"`
	State      [4]uint8 `json:"state"` // [data1, data2, lock_flags, visual_flags]
	Hits       [2]uint8 `json:"hits"`  // [fg_hits, bg_hits] - punch counter
}

type DroppedItem struct {
	ItemID int     `json:"item_id"`
	Count  int     `json:"count"`
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	UID    int     `json:"uid"`
}

type Door struct {
	Label string `json:"label"` // Display name for door
	Dest  string `json:"dest"`  // Destination "WORLDNAME:ID" or ":ID"
	ID    string `json:"id"`    // Unique ID for targeting this door
	TileX int    `json:"tile_x"`
	TileY int    `json:"tile_y"`
}

type Sign struct {
	Label string `json:"label"`
	TileX int    `json:"tile_x"`
	TileY int    `json:"tile_y"`
}

type Tree struct {
	FruitID    int       `json:"fruit_id"`
	FruitCount int       `json:"fruit_count"`
	LastPick   time.Time `json:"last_pick"`
	TileX      int       `json:"tile_x"`
	TileY      int       `json:"tile_y"`
}

type VendingMachine struct {
	TileX       int `json:"tile_x"`
	TileY       int `json:"tile_y"`
	ItemID      int `json:"item_id"`      // Item yang dijual (0 = kosong)
	Count       int `json:"count"`        // Stok item di mesin
	Price       int `json:"price"`        // Harga (> 0: World Lock per item; < 0: item per World Lock)
	LocksEarned int `json:"locks_earned"` // World Lock hasil penjualan yang tersimpan di mesin
}

type DisplayBox struct {
	TileX  int `json:"tile_x"`
	TileY  int `json:"tile_y"`
	ItemID int `json:"item_id"` // Item yang dipajang
}

// FindVendingAt mencari vending machine di tile (x, y)
func (w *World) FindVendingAt(x, y int) *VendingMachine {
	for i := range w.Vendings {
		if w.Vendings[i].TileX == x && w.Vendings[i].TileY == y {
			return &w.Vendings[i]
		}
	}
	return nil
}

// SetVending menyimpan atau mengupdate vending machine di tile (x, y)
func (w *World) SetVending(v VendingMachine) {
	for i := range w.Vendings {
		if w.Vendings[i].TileX == v.TileX && w.Vendings[i].TileY == v.TileY {
			w.Vendings[i] = v
			return
		}
	}
	w.Vendings = append(w.Vendings, v)
}

// RemoveVending menghapus data vending machine di tile (x, y)
func (w *World) RemoveVending(x, y int) {
	for i := range w.Vendings {
		if w.Vendings[i].TileX == x && w.Vendings[i].TileY == y {
			w.Vendings = append(w.Vendings[:i], w.Vendings[i+1:]...)
			return
		}
	}
}

// FindDisplayAt mencari display box di tile (x, y)
func (w *World) FindDisplayAt(x, y int) *DisplayBox {
	for i := range w.Displays {
		if w.Displays[i].TileX == x && w.Displays[i].TileY == y {
			return &w.Displays[i]
		}
	}
	return nil
}

// SetDisplay menyimpan atau mengupdate display box di tile (x, y)
func (w *World) SetDisplay(d DisplayBox) {
	for i := range w.Displays {
		if w.Displays[i].TileX == d.TileX && w.Displays[i].TileY == d.TileY {
			w.Displays[i] = d
			return
		}
	}
	w.Displays = append(w.Displays, d)
}

// RemoveDisplay menghapus display box di tile (x, y)
func (w *World) RemoveDisplay(x, y int) {
	for i := range w.Displays {
		if w.Displays[i].TileX == x && w.Displays[i].TileY == y {
			w.Displays = append(w.Displays[:i], w.Displays[i+1:]...)
			return
		}
	}
}

type DonationRecord struct {
	DonatorName string `json:"donator_name"`
	ItemID      int    `json:"item_id"`
	Count       int    `json:"count"`
	Message     string `json:"message"`
	DonatedAt   string `json:"donated_at"`
}

type DonationBox struct {
	TileX     int              `json:"tile_x"`
	TileY     int              `json:"tile_y"`
	Donations []DonationRecord `json:"donations"`
}

// FindDonationAt mencari donation box di tile (x, y)
func (w *World) FindDonationAt(x, y int) *DonationBox {
	for i := range w.Donations {
		if w.Donations[i].TileX == x && w.Donations[i].TileY == y {
			return &w.Donations[i]
		}
	}
	return nil
}

// SetDonation menyimpan atau mengupdate donation box di tile (x, y)
func (w *World) SetDonation(d DonationBox) {
	for i := range w.Donations {
		if w.Donations[i].TileX == d.TileX && w.Donations[i].TileY == d.TileY {
			w.Donations[i] = d
			return
		}
	}
	w.Donations = append(w.Donations, d)
}

// RemoveDonation menghapus donation box di tile (x, y)
func (w *World) RemoveDonation(x, y int) {
	for i := range w.Donations {
		if w.Donations[i].TileX == x && w.Donations[i].TileY == y {
			w.Donations = append(w.Donations[:i], w.Donations[i+1:]...)
			return
		}
	}
}

type BannedPlayer struct {
	GrowID      string    `json:"grow_id"`
	BannedUntil time.Time `json:"banned_until"`
}

// IsBanned memeriksa apakah seorang player sedang terkena ban dari world
func (w *World) IsBanned(growID string) bool {
	for i := range w.BannedUsers {
		if strings.EqualFold(w.BannedUsers[i].GrowID, growID) {
			if time.Now().Before(w.BannedUsers[i].BannedUntil) {
				return true
			}
		}
	}
	return false
}

// BanUser menambahkan player ke daftar banned world dengan durasi tertentu
func (w *World) BanUser(growID string, duration time.Duration) {
	until := time.Now().Add(duration)
	for i := range w.BannedUsers {
		if strings.EqualFold(w.BannedUsers[i].GrowID, growID) {
			w.BannedUsers[i].BannedUntil = until
			return
		}
	}
	w.BannedUsers = append(w.BannedUsers, BannedPlayer{GrowID: growID, BannedUntil: until})
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// New generates a new world 100% matching ANALISIS-GTPS-WORK algorithm:
// - Sky: y < 37 (empty, fg: 0, bg: 0)
// - Ground: y >= 37 (bg: 14 cave background)
// - Dirt (fg: 2), Rock (fg: 10, y: 38-49), Lava (fg: 4, y: 51-53), Bedrock (fg: 8, y >= 54)
// - Main Door: y = 36 at random X (2-95) with Bedrock underneath at y = 37
func New(name string) *World {
	width := WorldWidth
	height := WorldHeight
	totalTiles := width * height

	tiles := make([]Tile, totalTiles)

	// Random main door X between 2 and 95 (matching RandomRange(2, cord(0, 60)/100 - 4))
	mainDoorX := rand.Intn(94) + 2
	mainDoorY := 36

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x

			if y < 37 {
				// Sky
				tiles[idx] = Tile{Foreground: 0, Background: 0}
			} else {
				// Underground with cave background
				tiles[idx].Background = 14

				if y >= 38 && y < 50 && rand.Intn(39) <= 1 {
					// Rock (id 10)
					tiles[idx].Foreground = 10
				} else if y > 50 && y < 54 && rand.Intn(9) < 3 {
					// Lava (id 4)
					tiles[idx].Foreground = 4
				} else if y >= 54 {
					// Bedrock bottom layer (id 8)
					tiles[idx].Foreground = 8
				} else {
					// Dirt (id 2)
					tiles[idx].Foreground = 2
				}
			}

			// Main Door at (mainDoorX, 36)
			if x == mainDoorX && y == 36 {
				tiles[idx].Foreground = 6 // Main Door
			} else if x == mainDoorX && y == 37 {
				tiles[idx].Foreground = 8 // Bedrock below Main Door
				tiles[idx].Background = 14
			}
		}
	}

	return &World{
		Name:              name,
		Width:             width,
		Height:            height,
		Tiles:             tiles,
		DroppedItems:      []DroppedItem{},
		Doors:             []Door{{Label: "EXIT", TileX: mainDoorX, TileY: mainDoorY}},
		Signs:             []Sign{},
		Trees:             []Tree{},
		Locks:             []Lock{},
		GeigerX:           rand.Intn(width-6) + 3,
		GeigerY:           rand.Intn(height-15) + 5,
		SpawnTileX:        mainDoorX,
		SpawnTileY:        mainDoorY,
		IsPublic:          true,
		MinimumEntryLevel: 1,
		Weather:           0,
		LastVisited:       time.Now(),
	}
}

// GetTile gets tile at x,y (tile coordinates)
func (w *World) GetTile(x, y int) *Tile {
	if x < 0 || x >= w.Width || y < 0 || y >= w.Height {
		return nil
	}
	return &w.Tiles[y*w.Width+x]
}

// SpawnPixelX returns spawn X in pixels
func (w *World) SpawnPixelX() float32 {
	return float32(w.SpawnTileX) * 32.0
}

// SpawnPixelY returns spawn Y in pixels (top of tile)
func (w *World) SpawnPixelY() float32 {
	return float32(w.SpawnTileY) * 32.0
}

// Serialize builds the PACKET_SEND_MAP_DATA payload
// Format reference: 100% matching ANALISIS-GTPS-WORK world::serialize()
func (w *World) Serialize() []byte {
	buf := []byte{}

	// Header
	// version = 3 enables modern weather stream deserialization in the client
	buf = appendU16(buf, 0x0003)     // version
	buf = appendU32(buf, 0x00000000) // flags

	// World name
	buf = appendU16(buf, uint16(len(w.Name)))
	buf = append(buf, []byte(w.Name)...)

	// Dimensions
	buf = appendU32(buf, uint32(w.Width))
	buf = appendU32(buf, uint32(w.Height))
	buf = appendU16(buf, uint16(len(w.Tiles)))

	// Reserved bytes
	buf = appendU32(buf, 0x00000000)
	buf = appendU16(buf, 0x0000)
	buf = append(buf, 0x00)

	// Tiles
	for i, tile := range w.Tiles {
		buf = appendU16(buf, uint16(tile.Foreground))
		buf = appendU16(buf, uint16(tile.Background))
		buf = append(buf, tile.State[0], tile.State[1], tile.State[2], tile.State[3])

		tileX := i % w.Width
		tileY := i / w.Width

		// Extra data per tile type
		switch tile.Foreground {
		case 6: // Main door / Door / Portal
			buf = append(buf, 0x01) // type: door
			label := ""
			for _, d := range w.Doors {
				if d.TileX == tileX && d.TileY == tileY {
					label = d.Label
					break
				}
			}
			buf = appendU16(buf, uint16(len(label)))
			buf = append(buf, []byte(label)...)
			buf = append(buf, 0x00) // null terminator

		case 20: // Sign
			buf = append(buf, 0x02) // type: sign
			label := ""
			for _, s := range w.Signs {
				if s.TileX == tileX && s.TileY == tileY {
					label = s.Label
					break
				}
			}
			buf = appendU16(buf, uint16(len(label)))
			buf = append(buf, []byte(label)...)
			buf = appendU32(buf, 0xFFFFFFFF)

		case 202, 204, 206, 242, 1796, 2408, 4802, 4994, 5814, 7188, 10000, 11550, 11586, 11902: // All Locks
			buf = append(buf, 0x03) // type: lock
			
			// Find lock at this position
			lock := w.FindLockAt(tileX, tileY)
			ownerID := uint32(w.Owner)
			lockState := uint8(w.LockState)
			var accessList []uint32

			if lock != nil {
				ownerID = uint32(lock.Owner)
				lockState = uint8(lock.LockState)
				for _, uid := range lock.AccessList {
					if uid != 0 {
						accessList = append(accessList, uint32(uid))
					}
				}
			} else {
				for _, uid := range w.AccessList {
					if uid != 0 {
						accessList = append(accessList, uint32(uid))
					}
				}
			}

			buf = append(buf, lockState)
			buf = appendU32(buf, ownerID)
			buf = appendU32(buf, uint32(len(accessList)))
			for _, uid := range accessList {
				buf = appendU32(buf, uid)
			}
		}
	}

	// End of tile data padding
	buf = appendU32(buf, 0x00000000)
	buf = appendU32(buf, 0x00000000)
	buf = appendU32(buf, 0x00000000)

	// Dropped items: item count first, then total_drop_uid (LastObjectUID) - matching GrowTavern C++ WorldInfo.h:15324
	buf = appendU32(buf, uint32(len(w.DroppedItems)))
	buf = appendU32(buf, uint32(w.LastObjectUID))
	for _, obj := range w.DroppedItems {
		buf = appendU16(buf, uint16(obj.ItemID))
		buf = appendF32(buf, obj.X)
		buf = appendF32(buf, obj.Y)
		buf = appendU16(buf, uint16(obj.Count))
		buf = appendU32(buf, uint32(obj.UID))
	}

	// Weather data for map version >= 3:
	// BaseWeather (uint16), CurrentWeather (uint16), WeatherModifier (uint16), WeatherParam (uint16)
	// Growtopia client checks [World+426] == 25 to load DungeonWorldUI.rml
	weatherID := uint16(w.Weather)
	if strings.HasPrefix(w.Name, "DUNGEON") {
		weatherID = 25 // 25 = FIRE_HAZE (DUNGEON WEATHER ID!)
	}
	buf = appendU16(buf, weatherID) // Base Weather
	buf = appendU16(buf, weatherID) // Current Weather
	buf = appendU16(buf, 0x0000)    // Weather Modifier
	buf = appendU16(buf, 0x0000)    // Weather Param

	return buf
}

func (w *World) BuildMapDataPacket() []byte {
	worldData := w.Serialize()

	pkt := make([]byte, 60)
	binary.LittleEndian.PutUint32(pkt[0:], 4)          
	binary.LittleEndian.PutUint32(pkt[4:], 4)          
	binary.LittleEndian.PutUint32(pkt[8:], 0xFFFFFFFF) 
	binary.LittleEndian.PutUint32(pkt[16:], 8)        
	binary.LittleEndian.PutUint32(pkt[56:], uint32(len(worldData)))

	return append(pkt, worldData...)
}

func appendU16(buf []byte, v uint16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, v)
	return append(buf, b...)
}

func appendU32(buf []byte, v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return append(buf, b...)
}

func appendF32(buf []byte, v float32) []byte {
	return appendU32(buf, math.Float32bits(v))
}

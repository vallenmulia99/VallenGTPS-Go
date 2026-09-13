package items

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
)

// Item types (from items.hpp / actionType byte in items.dat)
// Referensi: BaseServer.h getItemCategory() - case number = actionType byte
const (
	TypeFist        = 0x00 // case 0
	TypeWrench      = 0x01 // case 1
	TypeDoor        = 0x02 // case 2
	TypeLock        = 0x03 // case 3
	TypeForeground  = 0x11 // case 17 = Foreground_Block
	TypeBackground  = 0x12 // case 18 = Background_Block
	TypeSeed        = 0x13 // case 19 = Seed
	TypeClothing    = 0x14 // case 20 = Clothing
	TypeConsumable  = 0x08 // consumable
	TypeAura        = 0x6b // aura/ances
)

// Category flags
const (
	CatReturn      = 0x02
	CatPublic      = 0x10
	CatHoliday     = 0x40
	CatUntradeable = 0x80
)

// Clothing slots (matching C++ ClothTypes enum in BaseServer.h)
// 0=Hair, 1=Shirt, 2=Pants, 3=Feet, 4=Face, 5=Hand, 6=Back, 7=Mask, 8=Necklace, 9=Ances
const (
	ClothHair     = 0
	ClothShirt    = 1
	ClothPants    = 2
	ClothFeet     = 3
	ClothFace     = 4
	ClothHand     = 5
	ClothBack     = 6
	ClothMask     = 7
	ClothHair2    = 7 // Legacy alias for Mask
	ClothNecklace = 8
	ClothCharm    = 8 // Legacy alias for Necklace
	ClothAnces    = 9
	ClothNone     = 10
)

type Item struct {
	ID          uint16
	Property    uint8
	Category    uint8
	Type        uint8
	Name        string
	Ingredient  uint32
	Collision   uint8
	Hits        uint8
	HitResetMS  uint32
	ClothSlot   uint8
	Rarity      uint16
	GrowTimeSec uint32
	Description string
	SpliceSeed1 uint16
	SpliceSeed2 uint16
}

// IsClothing reports whether the item is wearable clothing (regular clothing or ances).
func (it *Item) IsClothing() bool {
	if it == nil {
		return false
	}
	return it.Type == TypeClothing || it.Type == TypeAura
}

// IsClothingItem reports whether the given item is wearable clothing.
func IsClothingItem(it *Item) bool {
	return it != nil && (it.Type == TypeClothing || it.Type == TypeAura)
}

type ItemDatabase struct {
	Version uint16
	Items   []Item
	Hash    uint32
	RawData []byte // Full items.dat binary
}

var (
	DB *ItemDatabase
	decryptToken = []byte("PBG892FXX982ABC*")
)

// Load parses items.dat file
func Load(filename string) (*ItemDatabase, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read items.dat: %w", err)
	}

	db := &ItemDatabase{
		RawData: data,
		Hash:    crc32.ChecksumIEEE(data),
	}

	pos := 0

	// Read header
	if len(data) < 6 {
		return nil, fmt.Errorf("file too small")
	}

	db.Version = binary.LittleEndian.Uint16(data[pos:])
	pos += 2

	count := binary.LittleEndian.Uint32(data[pos:])
	pos += 4

	fmt.Printf("Items.dat version: 0x%x (%d items)\n", db.Version, count)

	// Parse items
	for i := uint32(0); i < count; i++ {
		item, newPos, err := parseItem(data, pos, db.Version)
		if err != nil {
			return nil, fmt.Errorf("failed to parse item %d: %w", i, err)
		}
		db.Items = append(db.Items, item)
		pos = newPos
	}

	DB = db
	fmt.Printf("[Items] Parsed %d items successfully! Hash: %d\n", len(db.Items), db.Hash)
	return db, nil
}

func parseItem(data []byte, pos int, version uint16) (Item, int, error) {
	item := Item{}

	// ID (uint32 stored, we use uint16)
	if pos+4 > len(data) {
		return item, pos, fmt.Errorf("unexpected EOF at ID")
	}
	item.ID = uint16(binary.LittleEndian.Uint32(data[pos:]))
	pos += 4

	// Property
	item.Property = data[pos]
	pos++

	// Category
	item.Category = data[pos]
	pos++

	// Type
	item.Type = data[pos]
	pos++

	// Padding
	pos++

	// Name (encrypted)
	nameLen := binary.LittleEndian.Uint16(data[pos:])
	pos += 2

	if pos+int(nameLen) > len(data) {
		return item, pos, fmt.Errorf("unexpected EOF at name")
	}

	encrypted := data[pos : pos+int(nameLen)]
	item.Name = decryptName(encrypted, item.ID)
	pos += int(nameLen)

	// Texture file
	texLen := binary.LittleEndian.Uint16(data[pos:])
	pos += 2
	pos += int(texLen) // Skip texture path

	// Unknown fields
	pos += 4 // uint32
	pos++    // uint8

	// Ingredient
	item.Ingredient = binary.LittleEndian.Uint32(data[pos:])
	pos += 4

	// Padding
	pos += 4

	// Collision
	item.Collision = data[pos]
	pos++

	// Break hits (divide by 6)
	item.Hits = data[pos]
	if item.Hits != 0 {
		item.Hits /= 6
	}
	pos++

	// Hit reset time
	item.HitResetMS = binary.LittleEndian.Uint32(data[pos:])
	pos += 4

	// Clothing slot
	if item.Type == TypeClothing {
		item.ClothSlot = data[pos]
		pos++
	} else {
		pos++ // Skip
	}

	// Aura = ances slot
	if item.Type == TypeAura {
		item.ClothSlot = ClothAnces
	}

	// Rarity
	item.Rarity = binary.LittleEndian.Uint16(data[pos:])
	pos += 2

	// Padding
	pos++

	// Audio path
	audioLen := binary.LittleEndian.Uint16(data[pos:])
	pos += 2
	pos += int(audioLen) // Skip audio

	pos += 4 // uint32

	// Unknown arrays
	pos += 4 // byte[4]

	// Skip variable strings
	for i := 0; i < 4; i++ {
		strLen := binary.LittleEndian.Uint16(data[pos:])
		pos += 2
		pos += int(strLen)
	}

	pos += 16 // byte[16]

	// Grow time
	item.GrowTimeSec = binary.LittleEndian.Uint32(data[pos:])
	pos += 4

	pos += 4 // uint16[2]

	// Skip more strings
	for i := 0; i < 3; i++ {
		strLen := binary.LittleEndian.Uint16(data[pos:])
		pos += 2
		pos += int(strLen)
	}

	pos += 80 // byte[80]

	// Version-dependent fields
	if version >= 0x0b {
		strLen := binary.LittleEndian.Uint16(data[pos:])
		pos += 2
		pos += int(strLen)
	}
	if version >= 0x0c {
		pos += 4 // uint32
		pos += 9 // byte[9]
	}
	if version >= 0x0d {
		pos += 4
	}
	if version >= 0x0e {
		pos += 4
	}
	if version >= 0x0f {
		pos += 25
		strLen := binary.LittleEndian.Uint16(data[pos:])
		pos += 2
		pos += int(strLen)
	}
	if version >= 0x10 {
		strLen := binary.LittleEndian.Uint16(data[pos:])
		pos += 2
		pos += int(strLen)
	}
	if version >= 0x11 {
		pos += 4
	}
	if version >= 0x12 {
		pos += 4
	}
	if version >= 0x13 {
		pos += 9
	}
	if version >= 0x15 {
		pos += 2
	}
	if version >= 0x16 {
		// Item description
		infoLen := binary.LittleEndian.Uint16(data[pos:])
		pos += 2
		if infoLen > 0 && pos+int(infoLen) <= len(data) {
			item.Description = string(data[pos : pos+int(infoLen)])
		}
		pos += int(infoLen)
	}
	if version >= 0x17 {
		// Seed splicing
		item.SpliceSeed1 = binary.LittleEndian.Uint16(data[pos:])
		pos += 2
		item.SpliceSeed2 = binary.LittleEndian.Uint16(data[pos:])
		pos += 2
	}
	if version >= 0x18 {
		pos++
	}
	if version >= 0x19 {
		strLen := binary.LittleEndian.Uint16(data[pos:])
		pos += 2
		if strLen > 6 {
			pos += int(strLen)
		}
		pos += 4
	}
	if version == 0x1a {
		pos++
	}

	return item, pos, nil
}

func decryptName(encrypted []byte, itemID uint16) string {
	decrypted := make([]byte, len(encrypted))
	for i := 0; i < len(encrypted); i++ {
		decrypted[i] = encrypted[i] ^ decryptToken[(i+int(itemID))%16]
	}
	return string(decrypted)
}

// GetItem returns item by ID
func GetItem(id uint16) *Item {
	if DB == nil || int(id) >= len(DB.Items) {
		return nil
	}
	return &DB.Items[id]
}

// GetItemByName returns item by name
func GetItemByName(name string) *Item {
	if DB == nil {
		return nil
	}
	for i := range DB.Items {
		if DB.Items[i].Name == name {
			return &DB.Items[i]
		}
	}
	return nil
}

// FindSpliceResult checks if two seeds can be spliced and returns the result
func FindSpliceResult(seed1ID, seed2ID int) (canSplice bool, resultSeedID int, seed1Name, seed2Name, resultName string) {
	if DB == nil {
		return false, 0, "", "", ""
	}

	// Search through all items for a splice combination match
	for i := range DB.Items {
		item := &DB.Items[i]
		
		// Check if this item is the result of splicing these two seeds
		splice1 := int(item.SpliceSeed1)
		splice2 := int(item.SpliceSeed2)
		
		// Match both orders: (A,B) or (B,A)
		if (splice1 == seed1ID && splice2 == seed2ID) || (splice1 == seed2ID && splice2 == seed1ID) {
			// Found a splice match!
			seed1Item := GetItem(uint16(seed1ID))
			seed2Item := GetItem(uint16(seed2ID))
			
			seed1NameStr := "Unknown"
			seed2NameStr := "Unknown"
			
			if seed1Item != nil {
				seed1NameStr = seed1Item.Name
			}
			if seed2Item != nil {
				seed2NameStr = seed2Item.Name
			}
			
			return true, int(item.ID), seed1NameStr, seed2NameStr, item.Name
		}
	}
	
	return false, 0, "", "", ""
}

// IsSeedType checks if an item is a seed type
func IsSeedType(itemID uint16) bool {
	item := GetItem(itemID)
	if item == nil {
		return false
	}
	return item.Type == TypeSeed
}

// AllItems returns all loaded items (read-only slice)
func AllItems() []Item {
	if DB == nil {
		return nil
	}
	return DB.Items
}

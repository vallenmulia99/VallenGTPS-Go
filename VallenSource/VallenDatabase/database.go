package database

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gtps/VallenSource/VallenPlayer"
	"gtps/VallenSource/VallenWorld"
)

type JSONDatabase struct {
	dataDir string
	players map[string]*player.Player
	worlds  map[string]*world.World
	mu      sync.RWMutex
}

func NewJSONDatabase(dataDir string) (*JSONDatabase, error) {
	db := &JSONDatabase{
		dataDir: dataDir,
		players: make(map[string]*player.Player),
		worlds:  make(map[string]*world.World),
	}

	if err := os.MkdirAll(filepath.Join(dataDir, "players"), 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "worlds"), 0755); err != nil {
		return nil, err
	}

	if err := db.loadAllPlayers(); err != nil {
		return nil, err
	}

	if err := db.loadAllWorlds(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *JSONDatabase) loadAllPlayers() error {
	playersDir := filepath.Join(db.dataDir, "players")
	entries, err := os.ReadDir(playersDir)
	if err != nil {
		return nil 
	}

	seenUserIDs := make(map[int]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}

		filePath := filepath.Join(playersDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read player file %s: %w", entry.Name(), err)
		}

		var p player.Player
		if err := json.Unmarshal(data, &p); err != nil {
			return fmt.Errorf("failed to parse player file %s: %w", entry.Name(), err)
		}
		appearanceMigrated := p.EnsureAppearanceDefaults()

		key := strings.ToUpper(p.GrowID)
		if existing, duplicate := seenUserIDs[p.UserID]; p.UserID > 0 && duplicate {
			log.Printf("[Database] Duplicate user_id %d: %s and %s. Resolve before using lock access lists.", p.UserID, existing, p.GrowID)
		} else if p.UserID > 0 {
			seenUserIDs[p.UserID] = p.GrowID
		}
		db.players[key] = &p
		if appearanceMigrated {
			if err := db.writePlayerFile(&p); err != nil {
				return fmt.Errorf("failed to save migrated appearance for %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

func (db *JSONDatabase) loadAllWorlds() error {
	worldsDir := filepath.Join(db.dataDir, "worlds")
	entries, err := os.ReadDir(worldsDir)
	if err != nil {
		return nil // dir empty / doesn't exist yet
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}

		filePath := filepath.Join(worldsDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read world file %s: %w", entry.Name(), err)
		}

		var w world.World
		if err := json.Unmarshal(data, &w); err != nil {
			return fmt.Errorf("failed to parse world file %s: %w", entry.Name(), err)
		}
		migratedLegacyLock := w.MigrateLegacyWorldLock()

		db.worlds[w.Name] = &w
		if migratedLegacyLock {
			if err := db.writeWorldFile(&w); err != nil {
				return fmt.Errorf("failed to save migrated world lock %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

func (db *JSONDatabase) playerFilePath(growID string) string {
	return filepath.Join(db.dataDir, "players", strings.ToUpper(growID)+".json")
}

func (db *JSONDatabase) worldFilePath(name string) string {
	return filepath.Join(db.dataDir, "worlds", strings.ToUpper(name)+".json")
}

func (db *JSONDatabase) Save() error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	for _, p := range db.players {
		if err := db.writePlayerFile(p); err != nil {
			return err
		}
	}

	for _, w := range db.worlds {
		if err := db.writeWorldFile(w); err != nil {
			return err
		}
	}

	return nil
}

func (db *JSONDatabase) writePlayerFile(p *player.Player) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal player %s: %w", p.GrowID, err)
	}
	return writeFileAtomic(db.playerFilePath(p.GrowID), data)
}

func (db *JSONDatabase) writeWorldFile(w *world.World) error {
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal world %s: %w", w.Name, err)
	}
	return writeFileAtomic(db.worldFilePath(w.Name), data)
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".save-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func (db *JSONDatabase) GetPlayer(growID string) (*player.Player, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	p, ok := db.players[strings.ToUpper(growID)]
	return p, ok
}

func (db *JSONDatabase) SavePlayer(p *player.Player) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.players[strings.ToUpper(p.GrowID)] = p
	if err := db.writePlayerFile(p); err != nil {
		log.Printf("[Database] Failed to save player %s: %v", p.GrowID, err)
	}
}

func (db *JSONDatabase) GetWorld(name string) (*world.World, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	w, ok := db.worlds[name]
	return w, ok
}

func (db *JSONDatabase) SaveWorld(w *world.World) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.worlds[w.Name] = w
	if err := db.writeWorldFile(w); err != nil {
		log.Printf("[Database] Failed to save world %s: %v", w.Name, err)
	}
}

func (db *JSONDatabase) PlayerCount() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.players)
}

func (db *JSONDatabase) NextUserID() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	maxID := 0
	for _, p := range db.players {
		if p != nil && p.UserID > maxID {
			maxID = p.UserID
		}
	}
	return maxID + 1
}

func (db *JSONDatabase) WorldCount() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.worlds)
}

package config

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// Sub-Structs — Setiap bagian setting.json punya struct sendiri
// ─────────────────────────────────────────────────────────────────────────────

type StarterItem struct {
	ID    int `json:"id"`
	Count int `json:"count"`
}

type ServerSetting struct {
	LoginMessage       string `json:"login_message"`
	MaintenanceMode    bool   `json:"maintenance_mode"`
	MaintenanceMessage string `json:"maintenance_message"`
	GameTheme          string `json:"game_theme"`
	ChooseMusic        string `json:"choose_music"`
	ItemsHash          uint32 `json:"items_hash"`
}

type NewPlayerSetting struct {
	StartGems  int           `json:"start_gems"`
	StartItems []StarterItem `json:"start_items"`
}

type WorldSetting struct {
	DefaultWidth    int `json:"default_width"`
	DefaultHeight   int `json:"default_height"`
	MaxDroppedItems int `json:"max_dropped_items"`
}

type DailyBonusSetting struct {
	Enabled         bool  `json:"enabled"`
	CooldownSeconds int64 `json:"cooldown_seconds"`
	RewardGems      int   `json:"reward_gems"`
	RewardWL        int   `json:"reward_wl"`
	RewardXP        int   `json:"reward_xp"`
}

type DailyChallengeSetting struct {
	Enabled      bool `json:"enabled"`
	TargetPoints int  `json:"target_points"`
	RewardGems   int  `json:"reward_gems"`
}

type FruitMixerSetting struct {
	Enabled         bool `json:"enabled"`
	CostGems        int  `json:"cost_gems"`
	RewardItemID    int  `json:"reward_item_id"`
	RewardItemCount int  `json:"reward_item_count"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Setting — Struct utama yang di-parse dari setting.json
// ─────────────────────────────────────────────────────────────────────────────

type Setting struct {
	ServerName         string `json:"server_name"`
	Owners             []string `json:"owners"`
	DefaultRole        int      `json:"default_role"`
	EnableRegistration bool     `json:"enable_registration"`
	MaxPlayers         int      `json:"max_players"`

	Server         ServerSetting         `json:"server"`
	NewPlayer      NewPlayerSetting      `json:"new_player"`
	World          WorldSetting          `json:"world"`
	DailyBonus     DailyBonusSetting     `json:"daily_bonus"`
	DailyChallenge DailyChallengeSetting `json:"daily_challenge"`
	FruitMixer     FruitMixerSetting     `json:"fruit_mixer"`
	Events         map[string]bool       `json:"events"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Config — Gabungan .env (host/port) + setting.json
// ─────────────────────────────────────────────────────────────────────────────

type Config struct {
	Host    string
	Port    int
	Setting Setting
}

// ─────────────────────────────────────────────────────────────────────────────
// Defaults — Nilai default supaya server tetap jalan walau setting.json kosong
// ─────────────────────────────────────────────────────────────────────────────

func defaultSetting() Setting {
	return Setting{
		ServerName:         "VALLEN GTPS",
		Owners:             []string{"VALLEN", "ADMIN"},
		DefaultRole:        0,
		EnableRegistration: true,
		MaxPlayers:         1024,

		Server: ServerSetting{
			LoginMessage:       "Welcome back, `w%s``. No friends are online.",
			MaintenanceMode:    false,
			MaintenanceMessage: "`4Server is under maintenance. Please try again later.``",
			GameTheme:          "growtopia",
			ChooseMusic:        "audio/mp3/about_theme.mp3",
			ItemsHash:          2966867045,
		},

		NewPlayer: NewPlayerSetting{
			StartGems: 1000,
			StartItems: []StarterItem{
				{ID: 242, Count: 50},
				{ID: 2, Count: 200},
				{ID: 10, Count: 50},
				{ID: 14, Count: 50},
			},
		},

		World: WorldSetting{
			DefaultWidth:    100,
			DefaultHeight:   60,
			MaxDroppedItems: 2000,
		},

		DailyBonus: DailyBonusSetting{
			Enabled:         true,
			CooldownSeconds: 86400,
			RewardGems:      5000,
			RewardWL:        3,
			RewardXP:        50,
		},

		DailyChallenge: DailyChallengeSetting{
			Enabled:      true,
			TargetPoints: 50,
			RewardGems:   10000,
		},

		FruitMixer: FruitMixerSetting{
			Enabled:         true,
			CostGems:        500,
			RewardItemID:    242,
			RewardItemCount: 1,
		},

		Events: map[string]bool{
			"showdungeonsui":            true,
			"show_mailbox_ui":           true,
			"show_auction_ui":           true,
			"openPiggyBank":             true,
			"dailychallengemenu":        true,
			"show_bingo_ui":             true,
			"winterrallymenu":           true,
			"leaderboardBtnClicked":     true,
			"euphoriaBtnClicked":        true,
			"show_fruit_mixer_dialog":   true,
			"eventmenu":                 true,
			"openLnySparksPopup":        false,
			"ShowValentinesQuestDialog": false,
			"showegseeventui":           false,
			"openStPatrickPiggyBank":    false,
			"dailyrewardmenu":           false,
			"claimprogressbar":          false,
		},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Load — Baca .env + setting.json, merge dengan defaults
// ─────────────────────────────────────────────────────────────────────────────

func Load() (*Config, error) {
	cfg := &Config{
		Host:    "0.0.0.0",
		Port:    17091,
		Setting: defaultSetting(),
	}

	// ── 1. Baca VallenSetting/env untuk HOST / PORT ──────────────────────────────────
	if file, err := os.Open("VallenSetting/env"); err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "SERVER_HOST":
				cfg.Host = value
			case "SERVER_PORT":
				if port, err := strconv.Atoi(value); err == nil {
					cfg.Port = port
				}
			}
		}
		file.Close()
	}

	// ── 2. Baca setting.json — merge di atas defaults ───────────────────
	settingPath := "VallenSetting/setting.json"
	if data, err := os.ReadFile(settingPath); err == nil {
		var s Setting
		if err := json.Unmarshal(data, &s); err == nil {
			mergeSettings(&cfg.Setting, &s)
		} else {
			log.Printf("[Config] Failed to parse %s: %v — using defaults", settingPath, err)
		}
	} else {
		log.Printf("[Config] %s not found — using defaults", settingPath)
	}

	log.Printf("[Config] Loaded: %s | Events: %d configured | Players: max %d",
		cfg.Setting.ServerName, len(cfg.Setting.Events), cfg.Setting.MaxPlayers)

	return cfg, nil
}

// mergeSettings menimpa defaults dengan nilai dari file JSON.
// Field zero-value di JSON tidak akan menimpa defaults yang sudah benar.
func mergeSettings(dst, src *Setting) {
	// Top-level fields
	if src.ServerName != "" {
		dst.ServerName = src.ServerName
	}
	if len(src.Owners) > 0 {
		dst.Owners = src.Owners
	}
	// DefaultRole 0 is valid, always overwrite
	dst.DefaultRole = src.DefaultRole
	dst.EnableRegistration = src.EnableRegistration
	if src.MaxPlayers > 0 {
		dst.MaxPlayers = src.MaxPlayers
	}

	// Server
	if src.Server.LoginMessage != "" {
		dst.Server.LoginMessage = src.Server.LoginMessage
	}
	dst.Server.MaintenanceMode = src.Server.MaintenanceMode
	if src.Server.MaintenanceMessage != "" {
		dst.Server.MaintenanceMessage = src.Server.MaintenanceMessage
	}
	if src.Server.GameTheme != "" {
		dst.Server.GameTheme = src.Server.GameTheme
	}
	if src.Server.ChooseMusic != "" {
		dst.Server.ChooseMusic = src.Server.ChooseMusic
	}
	if src.Server.ItemsHash != 0 {
		dst.Server.ItemsHash = src.Server.ItemsHash
	}

	// New Player
	if src.NewPlayer.StartGems > 0 {
		dst.NewPlayer.StartGems = src.NewPlayer.StartGems
	}
	if len(src.NewPlayer.StartItems) > 0 {
		dst.NewPlayer.StartItems = src.NewPlayer.StartItems
	}

	// World
	if src.World.DefaultWidth > 0 {
		dst.World.DefaultWidth = src.World.DefaultWidth
	}
	if src.World.DefaultHeight > 0 {
		dst.World.DefaultHeight = src.World.DefaultHeight
	}
	if src.World.MaxDroppedItems > 0 {
		dst.World.MaxDroppedItems = src.World.MaxDroppedItems
	}

	// Daily Bonus
	dst.DailyBonus.Enabled = src.DailyBonus.Enabled
	if src.DailyBonus.CooldownSeconds > 0 {
		dst.DailyBonus.CooldownSeconds = src.DailyBonus.CooldownSeconds
	}
	if src.DailyBonus.RewardGems > 0 {
		dst.DailyBonus.RewardGems = src.DailyBonus.RewardGems
	}
	if src.DailyBonus.RewardWL > 0 {
		dst.DailyBonus.RewardWL = src.DailyBonus.RewardWL
	}
	if src.DailyBonus.RewardXP > 0 {
		dst.DailyBonus.RewardXP = src.DailyBonus.RewardXP
	}

	// Daily Challenge
	dst.DailyChallenge.Enabled = src.DailyChallenge.Enabled
	if src.DailyChallenge.TargetPoints > 0 {
		dst.DailyChallenge.TargetPoints = src.DailyChallenge.TargetPoints
	}
	if src.DailyChallenge.RewardGems > 0 {
		dst.DailyChallenge.RewardGems = src.DailyChallenge.RewardGems
	}

	// Fruit Mixer
	dst.FruitMixer.Enabled = src.FruitMixer.Enabled
	if src.FruitMixer.CostGems > 0 {
		dst.FruitMixer.CostGems = src.FruitMixer.CostGems
	}
	if src.FruitMixer.RewardItemID > 0 {
		dst.FruitMixer.RewardItemID = src.FruitMixer.RewardItemID
	}
	if src.FruitMixer.RewardItemCount > 0 {
		dst.FruitMixer.RewardItemCount = src.FruitMixer.RewardItemCount
	}

	// Events — merge individual keys, keep defaults for missing keys
	if len(src.Events) > 0 {
		for key, val := range src.Events {
			dst.Events[key] = val
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper Methods
// ─────────────────────────────────────────────────────────────────────────────

// IsOwner mengecek apakah GrowID termasuk owner di setting.json.
func (c *Config) IsOwner(growID string) bool {
	growIDUpper := strings.ToUpper(strings.TrimSpace(growID))
	for _, owner := range c.Setting.Owners {
		if strings.ToUpper(strings.TrimSpace(owner)) == growIDUpper {
			return true
		}
	}
	return false
}

// IsEventEnabled mengecek apakah event button aktif di setting.json.
// Default true jika key tidak ada di map (backward compatible).
func (c *Config) IsEventEnabled(action string) bool {
	if val, ok := c.Setting.Events[action]; ok {
		return val
	}
	return true // default on jika belum di-setting
}

// ─────────────────────────────────────────────────────────────────────────────
// Getter Methods untuk diakses via interface{} dari package lain (avoid import cycle)
// ─────────────────────────────────────────────────────────────────────────────

// GetDailyBonus returns daily bonus settings
func (c *Config) GetDailyBonus() (enabled bool, cooldown int64, gems int, wl int, xp int) {
	return c.Setting.DailyBonus.Enabled,
		c.Setting.DailyBonus.CooldownSeconds,
		c.Setting.DailyBonus.RewardGems,
		c.Setting.DailyBonus.RewardWL,
		c.Setting.DailyBonus.RewardXP
}

// GetDailyChallenge returns daily challenge settings
func (c *Config) GetDailyChallenge() (enabled bool, targetPoints int, rewardGems int) {
	return c.Setting.DailyChallenge.Enabled,
		c.Setting.DailyChallenge.TargetPoints,
		c.Setting.DailyChallenge.RewardGems
}

// GetFruitMixer returns fruit mixer settings
func (c *Config) GetFruitMixer() (enabled bool, costGems int, rewardID int, rewardCount int) {
	return c.Setting.FruitMixer.Enabled,
		c.Setting.FruitMixer.CostGems,
		c.Setting.FruitMixer.RewardItemID,
		c.Setting.FruitMixer.RewardItemCount
}


// ActiveEventButtons mengembalikan list event actions yang enabled.
func (c *Config) ActiveEventButtons() []string {
	var active []string
	for action, enabled := range c.Setting.Events {
		if enabled {
			active = append(active, action)
		}
	}
	return active
}

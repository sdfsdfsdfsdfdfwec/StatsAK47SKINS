package main

import (
	"database/sql"
	"encoding/json"
	"time"
)

type Player struct {
	SteamID   string    `json:"steamid"`
	Name      string    `json:"name"`
	LastSeen  time.Time `json:"last_seen"`
}

type PlayerStats struct {
	SteamID       string    `json:"steamid"`
	Position      int       `json:"position"`
	TotalValue    float64   `json:"total_value"`
	SkinCount     int       `json:"skin_count"`
	SnapshotTime  time.Time `json:"snapshot_time"`
}

type PlayerSkin struct {
	SteamID    string        `json:"steamid"`
	SkinName   string        `json:"skin_name"`
	FirstSeen  time.Time     `json:"first_seen"`
	LastSeen   time.Time     `json:"last_seen"`
	Status     string        `json:"status"`
	Price      float64       `json:"price"`
	ItemFloat  string        `json:"item_float"`
}

type SkinEvent struct {
	SteamID     string          `json:"steamid"`
	SkinName    string          `json:"skin_name"`
	EventType   string          `json:"event_type"`
	DetectedAt  time.Time       `json:"detected_at"`
	Details     json.RawMessage `json:"details"`
}

type NouveauRougeAlert struct {
	SteamID       string    `json:"steamid"`
	PlayerName    string    `json:"player_name"`
	DetectedAt    time.Time `json:"detected_at"`
	ServerInfo    string    `json:"server_info"`
	StorageStatus string    `json:"storage_status"`
}

type StorageItem struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Image       string  `json:"image"`
	FloatValue  string  `json:"item_float"`
	RarityColor string  `json:"rarity_color"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	SellAt      *string `json:"sell_at"`
	UnlockAt    *string `json:"unlock_at"`
}

type ProfileStorage []StorageItem

type LeaderboardPlayer struct {
	Position    int       `json:"position"`
	SteamID     string    `json:"steamid"`
	Name        string    `json:"name"`
	Avatar      string    `json:"avatar_medium"`
	IsVip       bool      `json:"isVip"`
	TotalValue  float64   `json:"total"`
	SkinCount   int       `json:"count"`
	SnapshotTime time.Time `json:"-"`
	Skins       []struct {
		Name        string `json:"name"`
		Image       string `json:"image"`
		RarityColor string `json:"rarity_color"`
	} `json:"skins"`
}

type LeaderboardResponse struct {
	Total   int                `json:"total"`
	Players []LeaderboardPlayer `json:"players"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type OverallStats struct {
	TotalPlayers    int `json:"total_players"`
	TotalSnapshots  int `json:"total_snapshots"`
	TotalAlerts     int `json:"total_alerts"`
	TotalSkins      int `json:"total_skins"`
	TotalEvents     int `json:"total_events"`
}

type NullableString struct {
	sql.NullString
}

func (ns NullableString) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ns.String)
}

type NullableFloat struct {
	sql.NullFloat64
}

func (nf NullableFloat) MarshalJSON() ([]byte, error) {
	if !nf.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(nf.Float64)
}

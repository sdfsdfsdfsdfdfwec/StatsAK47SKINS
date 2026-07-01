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

type LeaderboardEntry struct {
	Page   int `json:"page"`
	Limit  int `json:"limit"`
	Total  int `json:"total"`
}

type StorageItem struct {
	Hash        string  `json:"hash"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Rarity      string  `json:"rarity"`
	Price       float64 `json:"price"`
	FloatValue  string  `json:"float_value"`
	Status      string  `json:"status"`
	Tradable    bool    `json:"tradable"`
	MarketHash  string  `json:"market_hash_name"`
}

type ProfileStorage struct {
	SteamID     string        `json:"steamid"`
	Items       []StorageItem `json:"storage"`
	TotalItems  int           `json:"total_items"`
	TotalValue  float64       `json:"total_value"`
}

type LeaderboardPlayer struct {
	SteamID    string  `json:"steamid"`
	Name       string  `json:"name"`
	Position   int     `json:"position"`
	TotalValue float64 `json:"total_value"`
	SkinCount  int     `json:"skin_count"`
}

type LeaderboardResponse struct {
	Data       []LeaderboardPlayer `json:"data"`
	Pagination struct {
		CurrentPage int `json:"current_page"`
		TotalPages  int `json:"total_pages"`
		TotalItems  int `json:"total_items"`
	} `json:"pagination"`
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

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func Connect(dataSourceName string) (*sql.DB, error) {
	var err error
	db, err = sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Database connection established")
	return db, nil
}

func InitDB(database *sql.DB) error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS players (
			steamid TEXT PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			last_seen TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS player_stats (
			steamid TEXT NOT NULL,
			position INT NOT NULL DEFAULT 0,
			total_value FLOAT8 NOT NULL DEFAULT 0,
			skin_count INT NOT NULL DEFAULT 0,
			snapshot_time TIMESTAMP NOT NULL DEFAULT NOW(),
			PRIMARY KEY (steamid, snapshot_time)
		)`,
		`CREATE TABLE IF NOT EXISTS player_skins (
			steamid TEXT NOT NULL,
			skin_name TEXT NOT NULL,
			first_seen TIMESTAMP NOT NULL DEFAULT NOW(),
			last_seen TIMESTAMP NOT NULL DEFAULT NOW(),
			status TEXT NOT NULL DEFAULT '',
			price FLOAT8 NOT NULL DEFAULT 0,
			item_float TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (steamid, skin_name)
		)`,
		`CREATE TABLE IF NOT EXISTS skin_events (
			id SERIAL PRIMARY KEY,
			steamid TEXT NOT NULL,
			skin_name TEXT NOT NULL,
			event_type TEXT NOT NULL,
			detected_at TIMESTAMP NOT NULL DEFAULT NOW(),
			details JSONB
		)`,
		`CREATE TABLE IF NOT EXISTS nouveau_rouge_alerts (
			id SERIAL PRIMARY KEY,
			steamid TEXT NOT NULL,
			player_name TEXT NOT NULL DEFAULT '',
			detected_at TIMESTAMP NOT NULL DEFAULT NOW(),
			server_info TEXT NOT NULL DEFAULT '',
			storage_status TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_player_stats_steamid ON player_stats(steamid)`,
		`CREATE INDEX IF NOT EXISTS idx_player_stats_time ON player_stats(snapshot_time)`,
		`CREATE INDEX IF NOT EXISTS idx_skin_events_steamid ON skin_events(steamid)`,
		`CREATE INDEX IF NOT EXISTS idx_skin_events_type ON skin_events(event_type)`,
		`CREATE INDEX IF NOT EXISTS idx_skin_events_time ON skin_events(detected_at)`,
		`CREATE INDEX IF NOT EXISTS idx_player_skins_steamid ON player_skins(steamid)`,
		`CREATE INDEX IF NOT EXISTS idx_nouveau_rouge_steamid ON nouveau_rouge_alerts(steamid)`,
		`CREATE INDEX IF NOT EXISTS idx_nouveau_rouge_time ON nouveau_rouge_alerts(detected_at)`,
	}

	for _, query := range tables {
		if _, err := database.Exec(query); err != nil {
			return fmt.Errorf("failed to execute migration: %s: %w", query[:50], err)
		}
	}

	log.Println("Database migrations completed")
	return nil
}

func UpsertPlayer(database *sql.DB, p Player) error {
	query := `
		INSERT INTO players (steamid, name, last_seen)
		VALUES ($1, $2, $3)
		ON CONFLICT (steamid) DO UPDATE SET
			name = EXCLUDED.name,
			last_seen = EXCLUDED.last_seen
	`
	_, err := database.Exec(query, p.SteamID, p.Name, p.LastSeen)
	if err != nil {
		return fmt.Errorf("failed to upsert player %s: %w", p.SteamID, err)
	}
	return nil
}

func InsertPlayerStats(database *sql.DB, ps PlayerStats) error {
	query := `
		INSERT INTO player_stats (steamid, position, total_value, skin_count, snapshot_time)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (steamid, snapshot_time) DO UPDATE SET
			position = EXCLUDED.position,
			total_value = EXCLUDED.total_value,
			skin_count = EXCLUDED.skin_count
	`
	_, err := database.Exec(query, ps.SteamID, ps.Position, ps.TotalValue, ps.SkinCount, ps.SnapshotTime)
	if err != nil {
		return fmt.Errorf("failed to insert player stats for %s: %w", ps.SteamID, err)
	}
	return nil
}

func UpsertPlayerSkin(database *sql.DB, ps PlayerSkin) error {
	query := `
		INSERT INTO player_skins (steamid, skin_name, first_seen, last_seen, status, price, item_float)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (steamid, skin_name) DO UPDATE SET
			last_seen = EXCLUDED.last_seen,
			status = EXCLUDED.status,
			price = EXCLUDED.price,
			item_float = EXCLUDED.item_float
	`
	_, err := database.Exec(query, ps.SteamID, ps.SkinName, ps.FirstSeen, ps.LastSeen, ps.Status, ps.Price, ps.ItemFloat)
	if err != nil {
		return fmt.Errorf("failed to upsert player skin %s/%s: %w", ps.SteamID, ps.SkinName, err)
	}
	return nil
}

func InsertSkinEvent(database *sql.DB, se SkinEvent) error {
	query := `
		INSERT INTO skin_events (steamid, skin_name, event_type, detected_at, details)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := database.Exec(query, se.SteamID, se.SkinName, se.EventType, se.DetectedAt, se.Details)
	if err != nil {
		return fmt.Errorf("failed to insert skin event: %w", err)
	}
	return nil
}

func InsertNouveauRougeAlert(database *sql.DB, a NouveauRougeAlert) error {
	query := `
		INSERT INTO nouveau_rouge_alerts (steamid, player_name, detected_at, server_info, storage_status)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := database.Exec(query, a.SteamID, a.PlayerName, a.DetectedAt, a.ServerInfo, a.StorageStatus)
	if err != nil {
		return fmt.Errorf("failed to insert nouveau rouge alert: %w", err)
	}
	return nil
}

func GetAllPlayers(database *sql.DB, offset, limit int) ([]Player, int, error) {
	countQuery := `SELECT COUNT(*) FROM players`
	var total int
	if err := database.QueryRow(countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count players: %w", err)
	}

	query := `
		SELECT steamid, name, last_seen
		FROM players
		ORDER BY last_seen DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := database.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query players: %w", err)
	}
	defer rows.Close()

	var players []Player
	for rows.Next() {
		var p Player
		if err := rows.Scan(&p.SteamID, &p.Name, &p.LastSeen); err != nil {
			return nil, 0, fmt.Errorf("failed to scan player: %w", err)
		}
		players = append(players, p)
	}

	return players, total, nil
}

func GetPlayerBySteamID(database *sql.DB, steamID string) (*Player, error) {
	query := `SELECT steamid, name, last_seen FROM players WHERE steamid = $1`
	var p Player
	err := database.QueryRow(query, steamID).Scan(&p.SteamID, &p.Name, &p.LastSeen)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get player %s: %w", steamID, err)
	}
	return &p, nil
}

func GetPlayerLatestStats(database *sql.DB, steamID string) (*PlayerStats, error) {
	query := `
		SELECT steamid, position, total_value, skin_count, snapshot_time
		FROM player_stats
		WHERE steamid = $1
		ORDER BY snapshot_time DESC
		LIMIT 1
	`
	var ps PlayerStats
	err := database.QueryRow(query, steamID).Scan(
		&ps.SteamID, &ps.Position, &ps.TotalValue, &ps.SkinCount, &ps.SnapshotTime,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest stats for %s: %w", steamID, err)
	}
	return &ps, nil
}

func GetPlayerStatsHistory(database *sql.DB, steamID string, limit int) ([]PlayerStats, error) {
	query := `
		SELECT steamid, position, total_value, skin_count, snapshot_time
		FROM player_stats
		WHERE steamid = $1
		ORDER BY snapshot_time DESC
		LIMIT $2
	`
	rows, err := database.Query(query, steamID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats history for %s: %w", steamID, err)
	}
	defer rows.Close()

	var stats []PlayerStats
	for rows.Next() {
		var ps PlayerStats
		if err := rows.Scan(&ps.SteamID, &ps.Position, &ps.TotalValue, &ps.SkinCount, &ps.SnapshotTime); err != nil {
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}
		stats = append(stats, ps)
	}

	return stats, nil
}

func GetPlayerSkins(database *sql.DB, steamID string) ([]PlayerSkin, error) {
	query := `
		SELECT steamid, skin_name, first_seen, last_seen, status, price, item_float
		FROM player_skins
		WHERE steamid = $1
		ORDER BY price DESC
	`
	rows, err := database.Query(query, steamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get skins for %s: %w", steamID, err)
	}
	defer rows.Close()

	var skins []PlayerSkin
	for rows.Next() {
		var ps PlayerSkin
		if err := rows.Scan(&ps.SteamID, &ps.SkinName, &ps.FirstSeen, &ps.LastSeen, &ps.Status, &ps.Price, &ps.ItemFloat); err != nil {
			return nil, fmt.Errorf("failed to scan skin: %w", err)
		}
		skins = append(skins, ps)
	}

	return skins, nil
}

func GetPlayerSkinEvents(database *sql.DB, steamID string, limit int) ([]SkinEvent, error) {
	query := `
		SELECT steamid, skin_name, event_type, detected_at, details
		FROM skin_events
		WHERE steamid = $1
		ORDER BY detected_at DESC
		LIMIT $2
	`
	rows, err := database.Query(query, steamID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get skin events for %s: %w", steamID, err)
	}
	defer rows.Close()

	var events []SkinEvent
	for rows.Next() {
		var se SkinEvent
		var details []byte
		if err := rows.Scan(&se.SteamID, &se.SkinName, &se.EventType, &se.DetectedAt, &details); err != nil {
			return nil, fmt.Errorf("failed to scan skin event: %w", err)
		}
		if details != nil {
			se.Details = json.RawMessage(details)
		}
		events = append(events, se)
	}

	return events, nil
}

func GetAllNouveauRougeAlerts(database *sql.DB, limit int) ([]NouveauRougeAlert, error) {
	query := `
		SELECT steamid, player_name, detected_at, server_info, storage_status
		FROM nouveau_rouge_alerts
		ORDER BY detected_at DESC
		LIMIT $1
	`
	rows, err := database.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get nouveau rouge alerts: %w", err)
	}
	defer rows.Close()

	var alerts []NouveauRougeAlert
	for rows.Next() {
		var a NouveauRougeAlert
		if err := rows.Scan(&a.SteamID, &a.PlayerName, &a.DetectedAt, &a.ServerInfo, &a.StorageStatus); err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}
		alerts = append(alerts, a)
	}

	return alerts, nil
}

func GetNouveauRougeAlertsBySteamID(database *sql.DB, steamID string) ([]NouveauRougeAlert, error) {
	query := `
		SELECT steamid, player_name, detected_at, server_info, storage_status
		FROM nouveau_rouge_alerts
		WHERE steamid = $1
		ORDER BY detected_at DESC
	`
	rows, err := database.Query(query, steamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get alerts for %s: %w", steamID, err)
	}
	defer rows.Close()

	var alerts []NouveauRougeAlert
	for rows.Next() {
		var a NouveauRougeAlert
		if err := rows.Scan(&a.SteamID, &a.PlayerName, &a.DetectedAt, &a.ServerInfo, &a.StorageStatus); err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}
		alerts = append(alerts, a)
	}

	return alerts, nil
}

func GetOverallStats(database *sql.DB) (*OverallStats, error) {
	stats := &OverallStats{}

	err := database.QueryRow(`SELECT COUNT(*) FROM players`).Scan(&stats.TotalPlayers)
	if err != nil {
		return nil, fmt.Errorf("failed to count players: %w", err)
	}

	err = database.QueryRow(`SELECT COUNT(*) FROM player_stats`).Scan(&stats.TotalSnapshots)
	if err != nil {
		return nil, fmt.Errorf("failed to count snapshots: %w", err)
	}

	err = database.QueryRow(`SELECT COUNT(*) FROM nouveau_rouge_alerts`).Scan(&stats.TotalAlerts)
	if err != nil {
		return nil, fmt.Errorf("failed to count alerts: %w", err)
	}

	err = database.QueryRow(`SELECT COUNT(*) FROM player_skins`).Scan(&stats.TotalSkins)
	if err != nil {
		return nil, fmt.Errorf("failed to count skins: %w", err)
	}

	err = database.QueryRow(`SELECT COUNT(*) FROM skin_events`).Scan(&stats.TotalEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to count events: %w", err)
	}

	return stats, nil
}

func HasNouveauRougeAlert(database *sql.DB, steamID string) (bool, error) {
	var count int
	err := database.QueryRow(
		`SELECT COUNT(*) FROM nouveau_rouge_alerts WHERE steamid = $1`, steamID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check alerts: %w", err)
	}
	return count > 0, nil
}

func GetExistingSkins(database *sql.DB, steamID string) (map[string]PlayerSkin, error) {
	query := `
		SELECT steamid, skin_name, first_seen, last_seen, status, price, item_float
		FROM player_skins
		WHERE steamid = $1
	`
	rows, err := database.Query(query, steamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing skins for %s: %w", steamID, err)
	}
	defer rows.Close()

	skins := make(map[string]PlayerSkin)
	for rows.Next() {
		var ps PlayerSkin
		if err := rows.Scan(&ps.SteamID, &ps.SkinName, &ps.FirstSeen, &ps.LastSeen, &ps.Status, &ps.Price, &ps.ItemFloat); err != nil {
			return nil, fmt.Errorf("failed to scan existing skin: %w", err)
		}
		skins[ps.SkinName] = ps
	}

	return skins, nil
}

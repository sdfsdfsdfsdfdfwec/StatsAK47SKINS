package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	apiBaseURL       = "https://api.fearproject.ru"
	leaderboardPath  = "/leaderboard/drops"
	storagePath      = "/profile/%s/storage"
	requestDelay     = 150 * time.Millisecond
	maxPages         = 214
	quickPages       = 20
	itemsPerPage     = 10
	nouveauRougeName = "AK-47 | Nouveau Rouge"
	fastInterval     = 15 * time.Minute
	fullInterval     = 1 * time.Hour
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func StartBackgroundScraper(database *sql.DB) {
	log.Println("Starting background scraper...")

	scrapeQuickCycle(database)

	go func() {
		fastTicker := time.NewTicker(fastInterval)
		defer fastTicker.Stop()
		for range fastTicker.C {
			scrapeQuickCycle(database)
		}
	}()

	go func() {
		fullTicker := time.NewTicker(fullInterval)
		defer fullTicker.Stop()
		for range fullTicker.C {
			scrapeFullCycle(database)
		}
	}()
}

func scrapeQuickCycle(database *sql.DB) {
	log.Println("[QUICK] Starting quick scrape cycle (top 200 players)...")

	for page := 1; page <= quickPages; page++ {
		players, err := fetchLeaderboardPage(page)
		if err != nil {
			log.Printf("[QUICK] Error fetching page %d: %v", page, err)
			time.Sleep(requestDelay)
			continue
		}
		if len(players) == 0 {
			break
		}

		for i := range players {
			if err := savePlayerStats(database, &players[i]); err != nil {
				log.Printf("[QUICK] Error saving stats for %s: %v", players[i].SteamID, err)
			}
			time.Sleep(50 * time.Millisecond)
		}

		storageBatch := players
		if len(storageBatch) > 50 {
			storageBatch = storageBatch[:50]
		}
		for i := range storageBatch {
			if err := processPlayerStorage(database, &storageBatch[i]); err != nil {
				log.Printf("[QUICK] Error processing storage for %s: %v", storageBatch[i].SteamID, err)
			}
			time.Sleep(requestDelay)
		}

		time.Sleep(requestDelay)
	}

	log.Println("[QUICK] Quick scrape cycle completed")
}

func scrapeFullCycle(database *sql.DB) {
	log.Println("[FULL] Starting full scrape cycle (all pages)...")

	for page := 1; page <= maxPages; page++ {
		players, err := fetchLeaderboardPage(page)
		if err != nil {
			log.Printf("[FULL] Error fetching page %d: %v", page, err)
			time.Sleep(requestDelay)
			continue
		}
		if len(players) == 0 {
			break
		}

		for i := range players {
			if err := savePlayerStats(database, &players[i]); err != nil {
				log.Printf("[FULL] Error saving stats for %s: %v", players[i].SteamID, err)
			}
			time.Sleep(50 * time.Millisecond)
		}

		time.Sleep(requestDelay)
	}

	log.Println("[FULL] Scrape cycle completed")
}

func savePlayerStats(database *sql.DB, lp *LeaderboardPlayer) error {
	now := time.Now()

	player := Player{
		SteamID:  lp.SteamID,
		Name:     lp.Name,
		LastSeen: now,
	}
	if err := UpsertPlayer(database, player); err != nil {
		return fmt.Errorf("upsert player: %w", err)
	}

	stats := PlayerStats{
		SteamID:      lp.SteamID,
		Position:     lp.Position,
		TotalValue:   lp.TotalValue,
		SkinCount:    lp.SkinCount,
		SnapshotTime: now,
	}
	if err := InsertPlayerStats(database, stats); err != nil {
		return fmt.Errorf("insert stats: %w", err)
	}

	return nil
}

func processPlayerStorage(database *sql.DB, lp *LeaderboardPlayer) error {
	now := time.Now()

	if err := savePlayerStats(database, lp); err != nil {
		return err
	}

	storage, err := fetchPlayerStorage(lp.SteamID)
	if err != nil {
		log.Printf("Warning: failed to fetch storage for %s: %v", lp.SteamID, err)
		return nil
	}

	existingSkins, err := GetExistingSkins(database, lp.SteamID)
	if err != nil {
		return fmt.Errorf("get existing skins: %w", err)
	}

	currentSkins := make(map[string]StorageItem, len(storage))
	for _, item := range storage {
		currentSkins[item.Name] = item
	}

	for _, item := range storage {
		ps := PlayerSkin{
			SteamID:   lp.SteamID,
			SkinName:  item.Name,
			FirstSeen: now,
			LastSeen:  now,
			Status:    item.Status,
			Price:     item.Price,
			ItemFloat: item.FloatValue,
		}
		if existing, ok := existingSkins[item.Name]; ok {
			ps.FirstSeen = existing.FirstSeen
		}
		if err := UpsertPlayerSkin(database, ps); err != nil {
			log.Printf("Warning: upsert skin %s for %s: %v", item.Name, lp.SteamID, err)
			continue
		}

		if _, existed := existingSkins[item.Name]; !existed {
			details, _ := json.Marshal(map[string]interface{}{
				"action": "added", "price": item.Price, "float": item.FloatValue, "status": item.Status,
			})
			InsertSkinEvent(database, SkinEvent{
				SteamID: lp.SteamID, SkinName: item.Name, EventType: "skin_added", DetectedAt: now, Details: details,
			})
		}
	}

	for skinName := range existingSkins {
		if _, exists := currentSkins[skinName]; !exists {
			InsertSkinEvent(database, SkinEvent{
				SteamID: lp.SteamID, SkinName: skinName, EventType: "skin_removed", DetectedAt: now,
			})
			database.Exec(`DELETE FROM player_skins WHERE steamid = $1 AND skin_name = $2`, lp.SteamID, skinName)
		}
	}

	checkNouveauRouge(database, lp.SteamID, lp.Name, storage)

	return nil
}

func fetchLeaderboardPage(page int) ([]LeaderboardPlayer, error) {
	url := fmt.Sprintf("%s%s?page=%d&limit=%d", apiBaseURL, leaderboardPath, page, itemsPerPage)

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var leaderboardResp LeaderboardResponse
	if err := json.Unmarshal(body, &leaderboardResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return leaderboardResp.Players, nil
}

func fetchPlayerStorage(steamID string) (ProfileStorage, error) {
	url := fmt.Sprintf("%s%s", apiBaseURL, fmt.Sprintf(storagePath, steamID))

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var storage ProfileStorage
	if err := json.Unmarshal(body, &storage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal storage: %w", err)
	}

	return storage, nil
}

func checkNouveauRouge(database *sql.DB, steamID, playerName string, storage ProfileStorage) {
	for _, item := range storage {
		if item.Name == nouveauRougeName {
			log.Printf("Nouveau Rouge detected for player %s (%s)!", playerName, steamID)

			hasAlert, err := HasNouveauRougeAlert(database, steamID)
			if err != nil {
				log.Printf("Warning: failed to check existing alerts: %v", err)
				continue
			}

			if !hasAlert {
				alert := NouveauRougeAlert{
					SteamID:       steamID,
					PlayerName:    playerName,
					DetectedAt:    time.Now(),
					ServerInfo:    "fearproject.ru",
					StorageStatus: item.Status,
				}
				if err := InsertNouveauRougeAlert(database, alert); err != nil {
					log.Printf("Warning: failed to insert nouveau rouge alert: %v", err)
				} else {
					log.Printf("Nouveau Rouge alert created for %s (%s)", playerName, steamID)
				}
			}
			break
		}
	}
}

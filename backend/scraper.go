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
	apiBaseURL        = "https://api.fearproject.ru"
	leaderboardPath   = "/leaderboard/drops"
	storagePath       = "/profile/%s/storage"
	requestDelay      = 200 * time.Millisecond
	maxPages          = 214
	itemsPerPage      = 10
	nouveauRougeName  = "AK-47 | Nouveau Rouge"
	scraperInterval   = 5 * time.Minute
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func StartBackgroundScraper(database *sql.DB) {
	log.Println("Starting background scraper...")
	scrapeAllPages(database)

	ticker := time.NewTicker(scraperInterval)
	defer ticker.Stop()

	for range ticker.C {
		scrapeAllPages(database)
	}
}

func scrapeAllPages(database *sql.DB) {
	log.Println("Starting full scrape cycle...")

	for page := 1; page <= maxPages; page++ {
		players, err := fetchLeaderboardPage(page)
		if err != nil {
			log.Printf("Error fetching page %d: %v", page, err)
			time.Sleep(requestDelay)
			continue
		}

		if len(players) == 0 {
			log.Printf("No players on page %d, stopping pagination", page)
			break
		}

		log.Printf("Page %d: found %d players", page, len(players))

		for _, lp := range players {
			if err := processPlayer(database, lp); err != nil {
				log.Printf("Error processing player %s (%s): %v", lp.SteamID, lp.Name, err)
				continue
			}
			time.Sleep(requestDelay)
		}

		time.Sleep(requestDelay)
	}

	log.Println("Scrape cycle completed")
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

	return leaderboardResp.Data, nil
}

func processPlayer(database *sql.DB, lp LeaderboardPlayer) error {
	now := time.Now()

	player := Player{
		SteamID:  lp.SteamID,
		Name:     lp.Name,
		LastSeen: now,
	}

	if err := UpsertPlayer(database, player); err != nil {
		return fmt.Errorf("failed to upsert player: %w", err)
	}

	stats := PlayerStats{
		SteamID:      lp.SteamID,
		Position:     lp.Position,
		TotalValue:   lp.TotalValue,
		SkinCount:    lp.SkinCount,
		SnapshotTime: now,
	}

	if err := InsertPlayerStats(database, stats); err != nil {
		return fmt.Errorf("failed to insert stats: %w", err)
	}

	storage, err := fetchPlayerStorage(lp.SteamID)
	if err != nil {
		log.Printf("Warning: failed to fetch storage for %s: %v", lp.SteamID, err)
		return nil
	}

	existingSkins, err := GetExistingSkins(database, lp.SteamID)
	if err != nil {
		return fmt.Errorf("failed to get existing skins: %w", err)
	}

	currentSkins := make(map[string]StorageItem)
	for _, item := range storage.Items {
		currentSkins[item.Name] = item
	}

	for _, item := range storage.Items {
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
			log.Printf("Warning: failed to upsert skin %s for %s: %v", item.Name, lp.SteamID, err)
			continue
		}

		if _, existed := existingSkins[item.Name]; !existed {
			details, _ := json.Marshal(map[string]interface{}{
				"action":  "added",
				"price":   item.Price,
				"float":   item.FloatValue,
				"status":  item.Status,
			})
			event := SkinEvent{
				SteamID:    lp.SteamID,
				SkinName:   item.Name,
				EventType:  "skin_added",
				DetectedAt: now,
				Details:    details,
			}
			if err := InsertSkinEvent(database, event); err != nil {
				log.Printf("Warning: failed to insert skin event: %v", err)
			}
		}
	}

	for skinName := range existingSkins {
		if _, exists := currentSkins[skinName]; !exists {
			details, _ := json.Marshal(map[string]interface{}{
				"action": "removed",
			})
			event := SkinEvent{
				SteamID:    lp.SteamID,
				SkinName:   skinName,
				EventType:  "skin_removed",
				DetectedAt: now,
				Details:    details,
			}
			if err := InsertSkinEvent(database, event); err != nil {
				log.Printf("Warning: failed to insert skin removal event: %v", err)
			}

			_, err := database.Exec(
				`DELETE FROM player_skins WHERE steamid = $1 AND skin_name = $2`,
				lp.SteamID, skinName,
			)
			if err != nil {
				log.Printf("Warning: failed to delete removed skin: %v", err)
			}
		}
	}

	checkNouveauRouge(database, lp.SteamID, lp.Name, storage)

	return nil
}

func fetchPlayerStorage(steamID string) (*ProfileStorage, error) {
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

	storage.SteamID = steamID
	return &storage, nil
}

func checkNouveauRouge(database *sql.DB, steamID, playerName string, storage *ProfileStorage) {
	for _, item := range storage.Items {
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
					ServerInfo:    fmt.Sprintf("fearproject.ru"),
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

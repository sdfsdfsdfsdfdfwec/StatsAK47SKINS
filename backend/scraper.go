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
	battlepassPath   = "/battlepass/rewards"
	requestDelay     = 100 * time.Millisecond
	itemsPerPage     = 10
	nouveauRougeName = "AK-47 | Nouveau Rouge"
	nouveauRougeCode = "ak47"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

type BattlepassReward struct {
	ID          int              `json:"id"`
	Code        string           `json:"code"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Cost        int              `json:"cost"`
	OrderIndex  int              `json:"order_index"`
	LineIndex   int              `json:"line_index"`
	State       string           `json:"state"`
	Remaining   *int             `json:"remaining"`
	Branch      []BattlepassItem `json:"branch"`
}

type BattlepassItem struct {
	ID          int    `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Cost        int    `json:"cost"`
	OrderIndex  int    `json:"order_index"`
	LineIndex   int    `json:"line_index"`
	State       string `json:"state"`
	Remaining   *int   `json:"remaining"`
}

var lastNouveauRougeRemaining *int

func StartBackgroundScraper(database *sql.DB) {
	log.Println("Starting background scraper...")

	go runLeaderboardLoop(database)
	go runBattlepassLoop(database)
}

func runLeaderboardLoop(database *sql.DB) {
	scrapeLeaderboardQuick(database)

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		scrapeLeaderboardQuick(database)
	}
}

func runBattlepassLoop(database *sql.DB) {
	checkBattlepass(database)

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		checkBattlepass(database)
	}
}

func scrapeLeaderboardQuick(database *sql.DB) {
	log.Println("[LEADERBOARD] Quick scrape...")

	for page := 1; page <= 30; page++ {
		players, err := fetchLeaderboardPage(page)
		if err != nil {
			log.Printf("[LEADERBOARD] Error page %d: %v", page, err)
			time.Sleep(requestDelay)
			continue
		}
		if len(players) == 0 {
			break
		}

		now := time.Now()
		for i := range players {
			lp := &players[i]
			player := Player{
				SteamID:  lp.SteamID,
				Name:     lp.Name,
				LastSeen: now,
			}
			if err := UpsertPlayer(database, player); err != nil {
				log.Printf("[LEADERBOARD] Upsert player %s: %v", lp.SteamID, err)
			}

			stats := PlayerStats{
				SteamID:      lp.SteamID,
				Position:     lp.Position,
				TotalValue:   lp.TotalValue,
				SkinCount:    lp.SkinCount,
				SnapshotTime: now,
			}
			if err := InsertPlayerStats(database, stats); err != nil {
				log.Printf("[LEADERBOARD] Insert stats %s: %v", lp.SteamID, err)
			}
			time.Sleep(30 * time.Millisecond)
		}

		time.Sleep(requestDelay)
	}

	log.Println("[LEADERBOARD] Quick scrape done")
}

func checkBattlepass(database *sql.DB) {
	log.Println("[BATTLEPASS] Checking rewards...")

	rewards, err := fetchBattlepassRewards()
	if err != nil {
		log.Printf("[BATTLEPASS] Error: %v", err)
		return
	}

	nouveauRouge := findNouveauRougeReward(rewards)
	if nouveauRouge == nil {
		log.Println("[BATTLEPASS] Nouveau Rouge not found in rewards")
		return
	}

	remaining := nouveauRouge.Remaining
	if remaining == nil {
		log.Println("[BATTLEPASS] Nouveau Rouge has no remaining count")
		return
	}

	log.Printf("[BATTLEPASS] Nouveau Rouge remaining: %d", *remaining)

	if lastNouveauRougeRemaining != nil && *remaining != *lastNouveauRougeRemaining {
		diff := *lastNouveauRougeRemaining - *remaining
		log.Printf("[BATTLEPASS] ALERT! Nouveau Rouge remaining changed: %d -> %d (claimed: %d)",
			*lastNouveauRougeRemaining, *remaining, diff)

		for i := 0; i < diff; i++ {
			go findNewNouveauRougeOwner(database)
		}
	}

	lastNouveauRougeRemaining = remaining
}

func fetchBattlepassRewards() ([]BattlepassReward, error) {
	url := fmt.Sprintf("%s%s", apiBaseURL, battlepassPath)

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
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rewards []BattlepassReward
	if err := json.Unmarshal(body, &rewards); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	return rewards, nil
}

func findNouveauRougeReward(rewards []BattlepassReward) *BattlepassItem {
	for i := range rewards {
		for j := range rewards[i].Branch {
			item := &rewards[i].Branch[j]
			if item.Code == nouveauRougeCode {
				return item
			}
		}
	}
	return nil
}

func findNewNouveauRougeOwner(database *sql.DB) {
	log.Println("[BATTLEPASS] Scanning leaderboard to find Nouveau Rouge owner...")

	for page := 1; page <= 30; page++ {
		players, err := fetchLeaderboardPage(page)
		if err != nil {
			log.Printf("[BATTLEPASS] Error fetching page %d: %v", page, err)
			time.Sleep(requestDelay)
			continue
		}
		if len(players) == 0 {
			break
		}

		for i := range players {
			lp := &players[i]
			storage, err := fetchPlayerStorage(lp.SteamID)
			if err != nil {
				time.Sleep(requestDelay)
				continue
			}

			for _, item := range storage {
				if item.Name == nouveauRougeName {
					hasAlert, _ := HasNouveauRougeAlert(database, lp.SteamID)
					if !hasAlert {
						alert := NouveauRougeAlert{
							SteamID:       lp.SteamID,
							PlayerName:    lp.Name,
							DetectedAt:    time.Now(),
							ServerInfo:    "fearproject.ru",
							StorageStatus: item.Status,
						}
						if err := InsertNouveauRougeAlert(database, alert); err != nil {
							log.Printf("[BATTLEPASS] Error inserting alert: %v", err)
						} else {
							log.Printf("[BATTLEPASS] FOUND! Nouveau Rouge owner: %s (%s)", lp.Name, lp.SteamID)
						}
					}
				}
			}
			time.Sleep(requestDelay)
		}
		time.Sleep(requestDelay)
	}
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

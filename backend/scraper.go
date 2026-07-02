package main

import (
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
	requestDelay     = 120 * time.Millisecond
	itemsPerPage     = 10
	nouveauRougeName = "AK-47 | Nouveau Rouge"
	nouveauRougeCode = "ak47"
	maxPages         = 300
	storageBatchSize = 50
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

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

var lastNouveauRougeRemaining = intPtr(20)

func intPtr(v int) *int { return &v }

func StartBackgroundScraper() {
	log.Println("Starting background scraper...")

	go runLeaderboardLoop()
	go runBattlepassLoop()
}

func runLeaderboardLoop() {
	time.Sleep(3 * time.Second)
	for {
		scrapeAllLeaderboardPages()
		time.Sleep(20 * time.Second)
	}
}

func runBattlepassLoop() {
	time.Sleep(10 * time.Second)
	for {
		checkBattlepass()
		time.Sleep(60 * time.Second)
	}
}

func scrapeAllLeaderboardPages() {
	log.Println("[SCRAPER] Starting full leaderboard scrape...")

	allPlayers := make([]LeaderboardPlayer, 0, 2200)

	for page := 1; page <= maxPages; page++ {
		players, err := fetchLeaderboardPage(page)
		if err != nil {
			log.Printf("[SCRAPER] Error page %d: %v", page, err)
			time.Sleep(requestDelay)
			continue
		}
		if len(players) == 0 {
			log.Printf("[SCRAPER] No players on page %d, stopping", page)
			break
		}

		allPlayers = append(allPlayers, players...)

		now := time.Now()
		for i := range players {
			lp := &players[i]
			store.UpsertPlayer(Player{
				SteamID:  lp.SteamID,
				Name:     lp.Name,
				LastSeen: now,
			})
			store.AddStats(PlayerStats{
				SteamID:      lp.SteamID,
				Position:     lp.Position,
				TotalValue:   lp.TotalValue,
				SkinCount:    lp.SkinCount,
				SnapshotTime: now,
			})
		}

		if page%20 == 0 {
			log.Printf("[SCRAPER] Progress: page %d, total players so far: %d", page, len(allPlayers))
		}
		time.Sleep(requestDelay)
	}

	store.SetLeaderboard(allPlayers)
	log.Printf("[SCRAPER] Leaderboard scrape done. Total: %d players", len(allPlayers))

	go scanStorageForNouveauRouge(allPlayers)
}

func scanStorageForNouveauRouge(players []LeaderboardPlayer) {
	log.Printf("[SCRAPER] Scanning storage for Nouveau Rouge (%d players)...", len(players))

	for i := range players {
		lp := &players[i]
		storage, err := fetchPlayerStorage(lp.SteamID)
		if err != nil {
			time.Sleep(requestDelay)
			continue
		}

		for _, item := range storage {
			if item.Name == nouveauRougeName {
				if !store.HasAlert(lp.SteamID) {
					alert := NouveauRougeAlert{
						SteamID:       lp.SteamID,
						PlayerName:    lp.Name,
						DetectedAt:    time.Now(),
						ServerInfo:    "fearproject.ru",
						StorageStatus: item.Status,
					}
					store.AddAlert(alert)
					log.Printf("[SCRAPER] *** NOUVEAU ROUGE: %s (%s) ***", lp.Name, lp.SteamID)
				}
			}
		}

		if (i+1)%100 == 0 {
			log.Printf("[SCRAPER] Storage scan progress: %d/%d", i+1, len(players))
		}
		time.Sleep(requestDelay)
	}

	log.Printf("[SCRAPER] Storage scan complete")
}

func checkBattlepass() {
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
		log.Printf("[BATTLEPASS] !!! ALERT: remaining changed %d -> %d (claimed: %d) !!!",
			*lastNouveauRougeRemaining, *remaining, diff)

		players := store.GetLeaderboard(3000)
		for i := range players {
			lp := &players[i]
			go checkPlayerForNouveauRouge(lp)
		}
	}

	lastNouveauRougeRemaining = remaining
}

func checkPlayerForNouveauRouge(lp *LeaderboardPlayer) {
	storage, err := fetchPlayerStorage(lp.SteamID)
	if err != nil {
		return
	}

	for _, item := range storage {
		if item.Name == nouveauRougeName {
			if !store.HasAlert(lp.SteamID) {
				alert := NouveauRougeAlert{
					SteamID:       lp.SteamID,
					PlayerName:    lp.Name,
					DetectedAt:    time.Now(),
					ServerInfo:    "fearproject.ru",
					StorageStatus: item.Status,
				}
				store.AddAlert(alert)
				log.Printf("[BATTLEPASS] *** OWNER FOUND: %s (%s) ***", lp.Name, lp.SteamID)
			}
		}
	}
}

func fetchLeaderboardPage(page int) ([]LeaderboardPlayer, error) {
	url := fmt.Sprintf("%s%s?page=%d&limit=%d", apiBaseURL, leaderboardPath, page, itemsPerPage)

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var r LeaderboardResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}

	return r.Players, nil
}

func fetchPlayerStorage(steamID string) (ProfileStorage, error) {
	url := fmt.Sprintf("%s%s", apiBaseURL, fmt.Sprintf(storagePath, steamID))

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var storage ProfileStorage
	if err := json.Unmarshal(body, &storage); err != nil {
		return nil, err
	}

	return storage, nil
}

func fetchBattlepassRewards() ([]BattlepassReward, error) {
	url := fmt.Sprintf("%s%s", apiBaseURL, battlepassPath)

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rewards []BattlepassReward
	if err := json.Unmarshal(body, &rewards); err != nil {
		return nil, err
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

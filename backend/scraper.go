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
	apiBaseURL      = "https://api.fearproject.ru"
	leaderboardPath = "/leaderboard/drops"
	storagePath     = "/profile/%s/storage"
	battlepassPath  = "/battlepass/rewards"
	requestDelay    = 120 * time.Millisecond
	itemsPerPage    = 10
	nouveauRougeName = "AK-47 | Nouveau Rouge"
	nouveauRougeCode = "ak47"
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

var lastNouveauRougeRemaining *int

func StartBackgroundScraper() {
	log.Println("Starting background scraper...")

	go runLeaderboardLoop()
	go runBattlepassLoop()
}

func runLeaderboardLoop() {
	time.Sleep(5 * time.Second)
	for {
		scrapeLeaderboard()
	}
}

func runBattlepassLoop() {
	time.Sleep(10 * time.Second)
	for {
		checkBattlepass()
		time.Sleep(60 * time.Second)
	}
}

func scrapeLeaderboard() {
	log.Println("[SCRAPER] Starting leaderboard scrape...")

	allPlayers := make([]LeaderboardPlayer, 0, 300)

	for page := 1; page <= 30; page++ {
		players, err := fetchLeaderboardPage(page)
		if err != nil {
			log.Printf("[SCRAPER] Error page %d: %v", page, err)
			time.Sleep(requestDelay)
			continue
		}
		if len(players) == 0 {
			break
		}

		allPlayers = append(allPlayers, players...)
		log.Printf("[SCRAPER] Page %d: %d players", page, len(players))

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

			time.Sleep(50 * time.Millisecond)
		}

		time.Sleep(requestDelay)
	}

	store.SetLeaderboard(allPlayers)
	log.Printf("[SCRAPER] Done. Total players: %d", len(allPlayers))

	for i := range allPlayers {
		lp := &allPlayers[i]
		go fetchAndStoreSkins(lp)
	}
}

func fetchAndStoreSkins(lp *LeaderboardPlayer) {
	storage, err := fetchPlayerStorage(lp.SteamID)
	if err != nil {
		return
	}

	skins := make([]PlayerSkin, 0, len(storage))
	now := time.Now()

	existingSkins := store.GetPlayerSkins(lp.SteamID)
	existingMap := make(map[string]bool)
	for _, s := range existingSkins {
		existingMap[s.SkinName] = true
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
		skins = append(skins, ps)

		if !existingMap[item.Name] {
			store.AddEvent(SkinEvent{
				SteamID:    lp.SteamID,
				SkinName:   item.Name,
				EventType:  "skin_added",
				DetectedAt: now,
			})
		}

		if item.Name == nouveauRougeName {
			if !store.HasAlert(lp.SteamID) {
				alert := NouveauRougeAlert{
					SteamID:       lp.SteamID,
					PlayerName:    lp.Name,
					DetectedAt:    now,
					ServerInfo:    "fearproject.ru",
					StorageStatus: item.Status,
				}
				store.AddAlert(alert)
				log.Printf("[SCRAPER] *** NOUVEAU ROUGE FOUND: %s (%s) ***", lp.Name, lp.SteamID)
			}
		}
	}

	store.SetSkins(lp.SteamID, skins)
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

		for i := 0; i < diff; i++ {
			go findNewNouveauRougeOwner()
		}
	}

	lastNouveauRougeRemaining = remaining
}

func findNewNouveauRougeOwner() {
	log.Println("[BATTLEPASS] Scanning leaderboard for new Nouveau Rouge owner...")

	for page := 1; page <= 30; page++ {
		players, err := fetchLeaderboardPage(page)
		if err != nil {
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
			time.Sleep(requestDelay)
		}
		time.Sleep(requestDelay)
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

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryStore struct {
	mu sync.RWMutex

	players   map[string]*Player
	stats     map[string][]PlayerStats
	skins     map[string][]PlayerSkin
	events    []SkinEvent
	alerts    []NouveauRougeAlert

	leaderboard []LeaderboardPlayer
	lastUpdated time.Time
}

var store = &MemoryStore{
	players: make(map[string]*Player),
	stats:   make(map[string][]PlayerStats),
	skins:   make(map[string][]PlayerSkin),
}

func (m *MemoryStore) UpsertPlayer(p Player) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.players[p.SteamID] = &p
}

func (m *MemoryStore) AddStats(ps PlayerStats) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats[ps.SteamID] = append(m.stats[ps.SteamID], ps)
	if len(m.stats[ps.SteamID]) > 288 {
		m.stats[ps.SteamID] = m.stats[ps.SteamID][len(m.stats[ps.SteamID])-288:]
	}
}

func (m *MemoryStore) SetSkins(steamID string, skins []PlayerSkin) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.skins[steamID] = skins
}

func (m *MemoryStore) AddEvent(e SkinEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, e)
	if len(m.events) > 10000 {
		m.events = m.events[len(m.events)-5000:]
	}
}

func (m *MemoryStore) AddAlert(a NouveauRougeAlert) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.alerts {
		if existing.SteamID == a.SteamID {
			return
		}
	}
	m.alerts = append(m.alerts, a)
}

func (m *MemoryStore) SetLeaderboard(players []LeaderboardPlayer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.leaderboard = players
	m.lastUpdated = time.Now()
}

func (m *MemoryStore) GetPlayers(page, limit int) ([]Player, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []Player
	for _, p := range m.players {
		list = append(list, *p)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].LastSeen.After(list[j].LastSeen)
	})

	total := len(list)
	start := (page - 1) * limit
	if start >= total {
		return []Player{}, total
	}
	end := start + limit
	if end > total {
		end = total
	}

	return list[start:end], total
}

func (m *MemoryStore) GetPlayer(steamID string) (*Player, []PlayerStats) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.players[steamID]
	if !ok {
		return nil, nil
	}

	st := m.stats[steamID]
	return p, st
}

func (m *MemoryStore) GetPlayerStats(steamID string) []PlayerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats[steamID]
}

func (m *MemoryStore) GetPlayerSkins(steamID string) []PlayerSkin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.skins[steamID]
}

func (m *MemoryStore) GetSkinEvents(steamID string, limit int) []SkinEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []SkinEvent
	for i := len(m.events) - 1; i >= 0; i-- {
		if m.events[i].SteamID == steamID {
			result = append(result, m.events[i])
			if len(result) >= limit {
				break
			}
		}
	}
	return result
}

func (m *MemoryStore) GetAlerts(limit int) []NouveauRougeAlert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]NouveauRougeAlert, len(m.alerts))
	copy(result, m.alerts)

	sort.Slice(result, func(i, j int) bool {
		return result[i].DetectedAt.After(result[j].DetectedAt)
	})

	if len(result) > limit {
		result = result[:limit]
	}
	return result
}

func (m *MemoryStore) GetAlertsByPlayer(steamID string) []NouveauRougeAlert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []NouveauRougeAlert
	for _, a := range m.alerts {
		if a.SteamID == steamID {
			result = append(result, a)
		}
	}
	return result
}

func (m *MemoryStore) HasAlert(steamID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, a := range m.alerts {
		if a.SteamID == steamID {
			return true
		}
	}
	return false
}

func (m *MemoryStore) GetLeaderboard(limit int) []LeaderboardPlayer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]LeaderboardPlayer, len(m.leaderboard))
	copy(result, m.leaderboard)

	sort.Slice(result, func(i, j int) bool {
		if result[i].Position == 0 {
			return false
		}
		if result[j].Position == 0 {
			return true
		}
		return result[i].Position < result[j].Position
	})

	if len(result) > limit {
		result = result[:limit]
	}
	return result
}

func (m *MemoryStore) GetStats() OverallStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return OverallStats{
		TotalPlayers:   len(m.players),
		TotalSnapshots: m.countSnapshots(),
		TotalAlerts:    len(m.alerts),
		TotalSkins:     m.countSkins(),
		TotalEvents:    len(m.events),
	}
}

func (m *MemoryStore) countSnapshots() int {
	total := 0
	for _, v := range m.stats {
		total += len(v)
	}
	return total
}

func (m *MemoryStore) countSkins() int {
	total := 0
	for _, v := range m.skins {
		total += len(v)
	}
	return total
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, APIResponse{
		Success: false,
		Error:   message,
	})
}

func handleGetPlayers(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	if v, err := fmt.Sscanf(pageStr, "%d", &page); err != nil || v == 0 {
		page = 1
	}
	limit := 50
	if v, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || v == 0 || limit > 100 {
		limit = 50
	}
	if page < 1 {
		page = 1
	}

	players, total := store.GetPlayers(page, limit)
	totalPages := (total + limit - 1) / limit

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"players": players,
			"pagination": map[string]interface{}{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": totalPages,
			},
		},
	})
}

func handleGetPlayer(w http.ResponseWriter, r *http.Request) {
	steamID := r.URL.Path[len("/api/players/"):]
	if idx := strings.Index(steamID, "/"); idx != -1 {
		steamID = steamID[:idx]
	}

	player, st := store.GetPlayer(steamID)
	if player == nil {
		writeError(w, http.StatusNotFound, "Player not found")
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"player": player,
			"stats":  st,
		},
	})
}

func handleGetPlayerStats(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/api/players/"):]
	steamID := strings.TrimSuffix(path, "/stats")

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"steamid": steamID,
			"stats":   store.GetPlayerStats(steamID),
		},
	})
}

func handleGetPlayerSkins(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/api/players/"):]
	steamID := strings.TrimSuffix(path, "/skins")

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"steamid": steamID,
			"skins":   store.GetPlayerSkins(steamID),
		},
	})
}

func handleGetPlayerSkinChanges(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/api/players/"):]
	steamID := strings.TrimSuffix(path, "/skin-changes")

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"steamid": steamID,
			"events":  store.GetSkinEvents(steamID, 100),
		},
	})
}

func handleGetSkinTracker(w http.ResponseWriter, r *http.Request) {
	alerts := store.GetAlerts(100)
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"alerts": alerts,
			"total":  len(alerts),
		},
	})
}

func handleGetSkinTrackerByPlayer(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/api/skin-tracker/"):]
	steamID := path

	alerts := store.GetAlertsByPlayer(steamID)
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"steamid": steamID,
			"alerts":  alerts,
			"total":   len(alerts),
		},
	})
}

func handleGetLeaderboard(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if v, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || v == 0 || limit > 500 {
		limit = 50
	}

	players := store.GetLeaderboard(limit)
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"players": players,
			"total":   len(players),
		},
	})
}

func handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats := store.GetStats()
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
	})
}

func SetupRouter(database interface{}) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/players/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		path := strings.TrimPrefix(r.URL.Path, "/api/players/")

		if path == "" || path == "/" {
			handleGetPlayers(w, r)
			return
		}

		parts := strings.SplitN(path, "/", 2)
		steamID := parts[0]
		_ = steamID

		if len(parts) > 1 {
			suffix := parts[1]
			switch suffix {
			case "stats":
				handleGetPlayerStats(w, r)
				return
			case "skins":
				handleGetPlayerSkins(w, r)
				return
			case "skin-changes":
				handleGetPlayerSkinChanges(w, r)
				return
			}
		}

		handleGetPlayer(w, r)
	})

	mux.HandleFunc("/api/players", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		handleGetPlayers(w, r)
	})

	mux.HandleFunc("/api/skin-tracker/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		path := strings.TrimPrefix(r.URL.Path, "/api/skin-tracker/")
		if path != "" {
			handleGetSkinTrackerByPlayer(w, r)
			return
		}
		handleGetSkinTracker(w, r)
	})

	mux.HandleFunc("/api/skin-tracker", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		handleGetSkinTracker(w, r)
	})

	mux.HandleFunc("/api/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		handleGetLeaderboard(w, r)
	})

	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		handleGetStats(w, r)
	})

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		writeJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    map[string]string{"status": "healthy"},
		})
	})

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./static"
	}

	if _, err := os.Stat(staticDir); err == nil {
		mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" && !strings.HasPrefix(r.URL.Path, "/api/") {
				http.ServeFile(w, r, staticDir+"/index.html")
				return
			}
			if r.URL.Path == "/" {
				http.ServeFile(w, r, staticDir+"/index.html")
				return
			}
			http.NotFound(w, r)
		})
	}

	return mux
}

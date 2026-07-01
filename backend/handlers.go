package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type Handlers struct {
	db *sql.DB
}

func NewHandlers(database *sql.DB) *Handlers {
	return &Handlers{db: database}
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

func (h *Handlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	err := h.db.Ping()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, APIResponse{
			Success: false,
			Error:   "Database connection failed",
		})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]string{
			"status": "healthy",
		},
	})
}

func (h *Handlers) GetPlayersHandler(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 50
	}

	offset := (page - 1) * limit

	players, total, err := GetAllPlayers(h.db, offset, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch players")
		return
	}

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

func (h *Handlers) GetPlayerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	steamID := vars["steamid"]

	player, err := GetPlayerBySteamID(h.db, steamID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch player")
		return
	}

	if player == nil {
		writeError(w, http.StatusNotFound, "Player not found")
		return
	}

	stats, err := GetPlayerLatestStats(h.db, steamID)
	if err != nil {
		log.Printf("Warning: failed to get stats for %s: %v", steamID, err)
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"player": player,
			"stats":  stats,
		},
	})
}

func (h *Handlers) GetPlayerStatsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	steamID := vars["steamid"]

	limitStr := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 500 {
		limit = 50
	}

	stats, err := GetPlayerStatsHistory(h.db, steamID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch stats history")
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"steamid": steamID,
			"stats":   stats,
		},
	})
}

func (h *Handlers) GetPlayerSkinsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	steamID := vars["steamid"]

	skins, err := GetPlayerSkins(h.db, steamID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch skins")
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"steamid": steamID,
			"skins":   skins,
		},
	})
}

func (h *Handlers) GetPlayerSkinChangesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	steamID := vars["steamid"]

	limitStr := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 500 {
		limit = 50
	}

	events, err := GetPlayerSkinEvents(h.db, steamID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch skin changes")
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"steamid": steamID,
			"events":  events,
		},
	})
}

func (h *Handlers) GetSkinTrackerHandler(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 500 {
		limit = 100
	}

	alerts, err := GetAllNouveauRougeAlerts(h.db, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch alerts")
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"alerts": alerts,
			"total":  len(alerts),
		},
	})
}

func (h *Handlers) GetSkinTrackerByPlayerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	steamID := vars["steamid"]

	alerts, err := GetNouveauRougeAlertsBySteamID(h.db, steamID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch alerts")
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"steamid": steamID,
			"alerts":  alerts,
			"total":   len(alerts),
		},
	})
}

func (h *Handlers) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := GetOverallStats(h.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch stats")
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
	})
}

func SetupRouter(database *sql.DB) *mux.Router {
	h := NewHandlers(database)

	r := mux.NewRouter()

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	r.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			writeJSON(w, http.StatusServiceUnavailable, APIResponse{
				Success: false,
				Error:   "Database not connected",
			})
			return
		}
		h.HealthHandler(w, r)
	}).Methods("GET")

	r.HandleFunc("/api/players", func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			writeJSON(w, http.StatusServiceUnavailable, APIResponse{Success: false, Error: "Database not connected"})
			return
		}
		h.GetPlayersHandler(w, r)
	}).Methods("GET")

	r.HandleFunc("/api/players/{steamid}", func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			writeJSON(w, http.StatusServiceUnavailable, APIResponse{Success: false, Error: "Database not connected"})
			return
		}
		h.GetPlayerHandler(w, r)
	}).Methods("GET")

	r.HandleFunc("/api/players/{steamid}/stats", func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			writeJSON(w, http.StatusServiceUnavailable, APIResponse{Success: false, Error: "Database not connected"})
			return
		}
		h.GetPlayerStatsHandler(w, r)
	}).Methods("GET")

	r.HandleFunc("/api/players/{steamid}/skins", func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			writeJSON(w, http.StatusServiceUnavailable, APIResponse{Success: false, Error: "Database not connected"})
			return
		}
		h.GetPlayerSkinsHandler(w, r)
	}).Methods("GET")

	r.HandleFunc("/api/players/{steamid}/skin-changes", func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			writeJSON(w, http.StatusServiceUnavailable, APIResponse{Success: false, Error: "Database not connected"})
			return
		}
		h.GetPlayerSkinChangesHandler(w, r)
	}).Methods("GET")

	r.HandleFunc("/api/skin-tracker", func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			writeJSON(w, http.StatusServiceUnavailable, APIResponse{Success: false, Error: "Database not connected"})
			return
		}
		h.GetSkinTrackerHandler(w, r)
	}).Methods("GET")

	r.HandleFunc("/api/skin-tracker/{steamid}", func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			writeJSON(w, http.StatusServiceUnavailable, APIResponse{Success: false, Error: "Database not connected"})
			return
		}
		h.GetSkinTrackerByPlayerHandler(w, r)
	}).Methods("GET")

	r.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			writeJSON(w, http.StatusOK, APIResponse{
				Success: true,
				Data:    OverallStats{},
			})
			return
		}
		h.GetStatsHandler(w, r)
	}).Methods("GET")

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./static"
	}

	if _, err := os.Stat(staticDir); err == nil {
		r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

		r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := filepath.Join(staticDir, r.URL.Path)
			if _, err := os.Stat(path); os.IsNotExist(err) || strings.HasSuffix(r.URL.Path, "/") {
				http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
				return
			}
			http.FileServer(http.Dir(staticDir)).ServeHTTP(w, r)
		})
	}

	return r
}

func init() {
	_ = fmt.Sprintf
}

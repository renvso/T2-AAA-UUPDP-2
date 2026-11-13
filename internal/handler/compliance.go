package handler

import (
	"aaa_bekend/internal/db"
	"aaa_bekend/internal/lmodel"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func GetMyData(w http.ResponseWriter, r *http.Request) {
	user, _ := lmodel.UserFromContext(r.Context())

	profile, err := db.GetUserByID(r.Context(), user.ID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch user data", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(profile)
}

type UpdateMyDataReq struct {
	Username string `json:"username"`
}

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{3,50}$`)

func UpdateMyData(w http.ResponseWriter, r *http.Request) {
	user, _ := lmodel.UserFromContext(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req UpdateMyDataReq
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if !usernamePattern.MatchString(req.Username) {
		http.Error(w, "Username must be 3-50 characters (letters, numbers, '.', '_', '-')", http.StatusBadRequest)
		return
	}

	if err := db.UpdateMyData(r.Context(), user.ID, req.Username); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, "Username already taken", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to update data", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message":  "Data updated successfully",
		"username": req.Username,
	})
}

func GetMyAuditLog(w http.ResponseWriter, r *http.Request) {
	user, _ := lmodel.UserFromContext(r.Context())

	logs, err := db.GetUserAuditLogs(r.Context(), user.ID)
    if err != nil {
        http.Error(w, "Failed to fetch audit logs", http.StatusInternalServerError)
        return
    }

    if logs == nil {
        logs = []lmodel.AuditLog{}
    }

    w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user":  user.Username,
		"total": len(logs),
		"logs":  logs,
	})
}

func ExportMyData(w http.ResponseWriter, r *http.Request) {
	user, _ := lmodel.UserFromContext(r.Context())

	profile, err := db.GetUserByID(r.Context(), user.ID)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            http.Error(w, "User not found", http.StatusNotFound)
            return
        }
        http.Error(w, "Failed to fetch user data", http.StatusInternalServerError)
        return
    }

    projects, err := db.GetUserProjects(r.Context(), user.ID)
    if err != nil {
        http.Error(w, "Failed to fetch user projects", http.StatusInternalServerError)
        return
    }

    logs, err := db.GetUserAuditLogs(r.Context(), user.ID)
    if err != nil {
        http.Error(w, "Failed to fetch audit logs", http.StatusInternalServerError)
        return
    }

    if projects == nil {
        projects = []lmodel.Project{}
    }

    if logs == nil {
        logs = []lmodel.AuditLog{}
    }

	export := map[string]interface{}{
		"exported_at": time.Now(),
		"profile":     profile,
		"projects":    projects,
		"audit_logs":  logs,
	}

	w.Header().Set("Content-Disposition", "attachment; filename=my_data_export.json")
	json.NewEncoder(w).Encode(export)
}
func DeleteMyAccount(w http.ResponseWriter, r *http.Request) {
	user, _ := lmodel.UserFromContext(r.Context())

	err := db.DeleteUserAccount(r.Context(), user.ID)
    if err != nil {
        http.Error(w, "Failed to delete account", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Your account and all data have been permanently deleted",
	})
}

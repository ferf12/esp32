package handler

import (
	"backend/database"
	"backend/models"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
)

// Elenca le camere disponibili nel database
func OttieniCam(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		err, camere := database.ListaCam(db)

		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		rispondiJson(w, http.StatusOK, camere)
	}
}

// Registra una nuova camera
func AggiungiCam(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var camera models.Camera

		err := json.NewDecoder(r.Body).Decode(&camera)

		if err != nil {
			http.Error(w, "errore", http.StatusBadRequest)
			return
		}

		nome := strings.TrimSpace(camera.Nome_cam)

		if nome == "" {
			rispondiJson(w, http.StatusBadRequest, models.EsitoModifica{
				Errore: "Inserisci il nome della camera",
			})
			return
		}

		err, id := database.AggiungiCam(db, nome)

		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		rispondiJson(w, http.StatusCreated, models.Camera{
			Id:       int(id),
			Nome_cam: nome,
		})
	}
}

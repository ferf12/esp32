package handler

import (
	"backend/database"
	"backend/models"
	"database/sql"
	"encoding/json"
	"net/http"
)

func AggiungiOspite(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var ospite models.OspiteEvento

		err := json.NewDecoder(r.Body).Decode(&ospite)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		err, aggiunto := database.AggiungiOspiteEvento(db, ospite)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.EsitoAggiunta{Aggiunto: aggiunto})
	}
}

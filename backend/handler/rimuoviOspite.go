package handler

import (
	"backend/database"
	"backend/models"
	"database/sql"
	"encoding/json"
	"net/http"
)

func RimuoviOspite(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var rimozione models.RimozioneOspite

		err := json.NewDecoder(r.Body).Decode(&rimozione)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		err = database.RimuoviOspiteEvento(db, rimozione.IdEvento, rimozione.IdOspite)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

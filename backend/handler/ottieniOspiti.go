package handler

import (
	"backend/database"
	"database/sql"
	"encoding/json"
	"net/http"
)

func OttieniOspiti(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		err, inv := database.ListaOspiti(db)

		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		err = json.NewEncoder(w).Encode(inv)

		if err != nil {
			http.Error(w, "errore", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)

	}
}

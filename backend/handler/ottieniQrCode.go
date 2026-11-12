package handler

import (
	"backend/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

func OttieniQrcode(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var idevento int

		err := json.NewDecoder(r.Body).Decode(&idevento)
		if err != nil {
			http.Error(w, "database error 1", http.StatusInternalServerError)
			return
		}

		err, qrcodes := database.GetQrcode(db, idevento)

		if err != nil {
			fmt.Println("ERRORE GetQrcode:", err)
			http.Error(w, "database error 2", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(qrcodes)

	}
}

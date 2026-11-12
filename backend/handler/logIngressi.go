package handler

import (
	"backend/database"
	"backend/models"
	"database/sql"
	"encoding/json"
	"net/http"
)

const LIMITE_LOG_INGRESSI = 50

func LogIngressi(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var richiesta models.RichiestaPresenze

		err := json.NewDecoder(r.Body).Decode(&richiesta)

		if err != nil {
			http.Error(w, "errore", http.StatusBadRequest)
			return
		}

		if richiesta.IdEvento <= 0 {
			rispondiJson(w, http.StatusBadRequest, models.EsitoModifica{
				Errore: "ID evento mancante",
			})
			return
		}

		err, logs := database.LogIngressi(db, richiesta.IdEvento, LIMITE_LOG_INGRESSI)

		if err != nil {
			http.Error(w, "errore database", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		err = json.NewEncoder(w).Encode(logs)

		if err != nil {
			http.Error(w, "errore", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)

	}
}

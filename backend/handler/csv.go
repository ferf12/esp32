package handler

import (
	"backend/database"
	"backend/models"
	"database/sql"
	"encoding/json"
	"net/http"
)

// Crea un evento con i suoi ospiti
func CsvPost(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		var CreazEvento models.CreazioneEvento

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		err := json.NewDecoder(r.Body).Decode(&CreazEvento)

		if err != nil {
			http.Error(w, "errore", http.StatusBadRequest)
			return
		}

		CreazEvento.Evento = normalizzaEvento(CreazEvento.Evento)

		errore := validaEvento(CreazEvento.Evento)

		if errore != "" {
			rispondiJson(w, http.StatusBadRequest, models.EsitoModifica{
				Errore: errore,
			})
			return
		}

		err, id := database.AggiungiEvento(db, CreazEvento)

		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(id)
	}
}

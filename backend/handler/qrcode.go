package handler

import (
	"backend/database"
	"backend/models"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

// Controlla il qrcode, verifica la camera, aggiunge la presenza e registra l'ingresso
func Qrcode(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		idCam, err := strconv.Atoi(r.Header.Get("id"))

		if err != nil || idCam <= 0 {
			rispondiJson(w, http.StatusBadRequest, models.EsitoModifica{
				Errore: "header 'id' mancante o non valido",
			})
			return
		}

		var stringqr string
		err = json.NewDecoder(r.Body).Decode(&stringqr)

		if err != nil {
			http.Error(w, "errore", http.StatusBadRequest)
			return
		}

		appartiene, err := database.QrAppartenenteCam(db, stringqr, idCam)

		if err != nil {
			http.Error(w, "errore database", http.StatusInternalServerError)
			return
		}

		if !appartiene {
			rispondiJson(w, http.StatusNotFound, models.EsitoModifica{
				Errore: "QR code non associato a questa camera",
			})
			return
		}

		err, ingresso := database.RegistraIngresso(db, stringqr)

		if errors.Is(err, database.ErrQrNonRiconosciuto) {
			rispondiJson(w, http.StatusNotFound, models.EsitoModifica{
				Errore: "QR code non riconosciuto",
			})
			return
		}

		if err != nil {
			http.Error(w, "errore database", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		err = json.NewEncoder(w).Encode(ingresso)

		if err != nil {
			http.Error(w, "errore", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)

	}

}

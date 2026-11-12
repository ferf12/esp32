package handler

import (
	"backend/database"
	"backend/models"
	"database/sql"
	"encoding/json"
	"net/http"
)

// Modifica nome, date e orari di un evento
func ModificaEvento(db *sql.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var modifica models.ModificaEvento

		err := json.NewDecoder(r.Body).Decode(&modifica)

		if err != nil {
			http.Error(w, "errore", http.StatusBadRequest)
			return
		}

		if modifica.Id <= 0 {
			rispondiJson(w, http.StatusBadRequest, models.EsitoModifica{
				Errore: "ID evento mancante",
			})
			return
		}

		modifica.Evento = normalizzaEvento(modifica.Evento)

		errore := validaEvento(modifica.Evento)

		if errore != "" {
			rispondiJson(w, http.StatusBadRequest, models.EsitoModifica{
				Errore: errore,
			})
			return
		}

		err, precedente := database.DettaglioEvento(db, modifica.Id)

		if err != nil {
			rispondiJson(w, http.StatusNotFound, models.EsitoModifica{
				Errore: "Evento non trovato",
			})
			return
		}

		modifiche := confrontaEvento(precedente, modifica.Evento)

		if len(modifiche) > 0 {
			var aggiornato bool

			err, aggiornato = database.ModificaEvento(db, modifica)

			if err != nil {
				rispondiJson(w, http.StatusInternalServerError, models.EsitoModifica{
					Errore: "Errore durante il salvataggio dell'evento",
				})
				return
			}

			if !aggiornato {
				rispondiJson(w, http.StatusNotFound, models.EsitoModifica{
					Errore: "Evento non trovato",
				})
				return
			}
		}

		rispondiJson(w, http.StatusOK, models.EsitoModifica{
			Modificato: len(modifiche) > 0,
			Modifiche:  modifiche,
		})
	}
}

func confrontaEvento(precedente models.Evento, nuovo models.Evento) []models.CampoModificato {

	modifiche := make([]models.CampoModificato, 0)

	confronta := func(etichetta string, prima string, dopo string) {
		if prima != dopo {
			modifiche = append(modifiche, models.CampoModificato{
				Etichetta: etichetta,
				Prima:     prima,
				Dopo:      dopo,
			})
		}
	}

	confronta("Nome evento", precedente.Nome_evento, nuovo.Nome_evento)
	confronta("Data inizio", precedente.Data_inizio, nuovo.Data_inizio)
	confronta("Data fine", precedente.Data_fine, nuovo.Data_fine)
	confronta("Ora inizio", precedente.Ora_inizio, nuovo.Ora_inizio)
	confronta("Ora fine", precedente.Ora_fine, nuovo.Ora_fine)

	return modifiche
}

func rispondiJson(w http.ResponseWriter, stato int, corpo interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(stato)
	json.NewEncoder(w).Encode(corpo)
}

package handler

import (
	"backend/database"
	"backend/models"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var statoDispositivi = map[int]models.StatoDispositivo{}
var statoDispositiviMutex = sync.Mutex{}

func scriviStatoDispositivo(stato models.StatoDispositivo) {
	statoDispositiviMutex.Lock()
	defer statoDispositiviMutex.Unlock()

	statoDispositivi[stato.IdEvento] = stato
}

func leggiStatoDispositivo(idEvento int) (models.StatoDispositivo, bool) {
	statoDispositiviMutex.Lock()
	defer statoDispositiviMutex.Unlock()

	stato, trovato := statoDispositivi[idEvento]

	return stato, trovato
}

func AggiornaStatoDispositivo(db *sql.DB) http.HandlerFunc {
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

		err, idEvento := database.EventoAttivoDaCam(db, idCam)

		if errors.Is(err, database.ErrEventoNonTrovato) {
			rispondiJson(w, http.StatusNotFound, models.EsitoModifica{
				Errore: "nessun evento attivo per la camera " + strconv.Itoa(idCam),
			})
			return
		}

		if err != nil {
			http.Error(w, "errore database", http.StatusInternalServerError)
			return
		}

		var stato models.StatoDispositivo

		err = json.NewDecoder(r.Body).Decode(&stato)

		if err != nil {
			http.Error(w, "errore", http.StatusBadRequest)
			return
		}

		stato.IdEvento = idEvento
		stato.Aggiornato = time.Now().Format("2006-01-02 15:04:05")

		scriviStatoDispositivo(stato)

		rispondiJson(w, http.StatusOK, stato)
	}
}

func StatoDispositivo(db *sql.DB) http.HandlerFunc {
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

		stato, trovato := leggiStatoDispositivo(richiesta.IdEvento)

		if !trovato {
			rispondiJson(w, http.StatusOK, models.StatoDispositivo{
				IdEvento: richiesta.IdEvento,
			})
			return
		}

		rispondiJson(w, http.StatusOK, stato)
	}
}

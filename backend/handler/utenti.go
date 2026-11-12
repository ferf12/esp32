package handler

import (
	"backend/database"
	"backend/middleware"
	"backend/models"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// POST /login: verifica le credenziali e restituisce la coppia di token
func Login(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var richiesta models.RichiestaLogin

		err := json.NewDecoder(r.Body).Decode(&richiesta)

		if err != nil {
			http.Error(w, "errore", http.StatusBadRequest)
			return
		}

		email := strings.TrimSpace(strings.ToLower(richiesta.Email))

		hash, err := database.VerificaUtente(db, email)

		// Email inesistente e password sbagliata danno lo stesso messaggio
		if errors.Is(err, database.ErrUtenteNonTrovato) {
			rispondiJson(w, http.StatusUnauthorized, models.EsitoModifica{
				Errore: "email o password non corretti",
			})
			return
		}

		if err != nil {
			http.Error(w, "errore database", http.StatusInternalServerError)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(richiesta.Password))

		if err != nil {
			rispondiJson(w, http.StatusUnauthorized, models.EsitoModifica{
				Errore: "email o password non corretti",
			})
			return
		}

		risposta, err := emettiToken(email)

		if err != nil {
			http.Error(w, "errore nel creare i token", http.StatusInternalServerError)
			return
		}

		rispondiJson(w, http.StatusOK, risposta)
	}
}

// POST /refresh: scambia un refresh token valido con un access token nuovo
func Refresh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var richiesta models.RichiestaRefresh

		err := json.NewDecoder(r.Body).Decode(&richiesta)

		if err != nil || richiesta.RefreshToken == "" {
			rispondiJson(w, http.StatusBadRequest, models.EsitoModifica{
				Errore: "refresh token mancante",
			})
			return
		}

		email, ok := middleware.EmailRefreshToken(richiesta.RefreshToken)

		if !ok {
			rispondiJson(w, http.StatusUnauthorized, models.EsitoModifica{
				Errore: "refresh token non valido o scaduto",
			})
			return
		}

		risposta, err := emettiToken(email)

		if err != nil {
			http.Error(w, "errore nel creare i token", http.StatusInternalServerError)
			return
		}

		rispondiJson(w, http.StatusOK, risposta)
	}
}

// POST /logout: revoca il refresh token e chiude la sessione
func Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var richiesta models.RichiestaRefresh

		err := json.NewDecoder(r.Body).Decode(&richiesta)

		if err != nil || richiesta.RefreshToken == "" {
			rispondiJson(w, http.StatusBadRequest, models.EsitoModifica{
				Errore: "refresh token mancante",
			})
			return
		}

		middleware.RevocaRefreshToken(richiesta.RefreshToken)

		rispondiJson(w, http.StatusOK, models.EsitoModifica{})
	}
}

// Crea la coppia di token per un utente autenticato
func emettiToken(email string) (models.RispostaLogin, error) {

	access, err := middleware.NuovoAccessToken(email)

	if err != nil {
		return models.RispostaLogin{}, err
	}

	refresh, err := middleware.NuovoRefreshToken(email)

	if err != nil {
		return models.RispostaLogin{}, err
	}

	return models.RispostaLogin{
		AccessToken:  access,
		RefreshToken: refresh,
		Email:        email,
		Scadenza:     time.Now().Add(middleware.DurataAccessToken).Format("15:04:05"),
	}, nil
}

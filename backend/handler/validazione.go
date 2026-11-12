package handler

import (
	"backend/models"
	"errors"
	"strings"
	"time"
)

const layoutData = "2006-01-02"

func normalizzaEvento(evento models.Evento) models.Evento {
	evento.Nome_evento = strings.TrimSpace(evento.Nome_evento)
	return evento
}

// Accetta sia HH:MM sia HH:MM:SS
func parseOra(ora string) (time.Time, error) {
	ora = strings.TrimSpace(ora)

	for _, layout := range []string{"15:04", "15:04:05"} {
		if valore, err := time.Parse(layout, ora); err == nil {
			return valore, nil
		}
	}

	return time.Time{}, errors.New("ora non valida: " + ora)
}

// Controlla i dati di un evento, restituisce "" se sono coerenti
func validaEvento(evento models.Evento) string {

	if evento.Nome_evento == "" {
		return "Inserisci il nome dell'evento"
	}

	if len(evento.Nome_evento) > 100 {
		return "Il nome dell'evento supera i 100 caratteri"
	}

	inizio, err := time.Parse(layoutData, evento.Data_inizio)
	if err != nil {
		return "La data di inizio non è valida"
	}

	fine, err := time.Parse(layoutData, evento.Data_fine)
	if err != nil {
		return "La data di fine non è valida"
	}

	oraInizio, err := parseOra(evento.Ora_inizio)
	if err != nil {
		return "L'ora di inizio non è valida"
	}

	oraFine, err := parseOra(evento.Ora_fine)
	if err != nil {
		return "L'ora di fine non è valida"
	}

	if fine.Before(inizio) {
		return "La data di fine non può essere precedente alla data di inizio"
	}

	if fine.Equal(inizio) && !oraFine.After(oraInizio) {
		return "Quando inizio e fine coincidono, l'ora di fine deve essere successiva all'ora di inizio"
	}

	return ""
}

package database

import (
	"backend/models"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var ErrQrNonRiconosciuto = errors.New("QR code non riconosciuto")
var ErrEventoNonTrovato = errors.New("evento non trovato")

func codiceQr(idEvento int, idOspite int64) string {
	return strconv.Itoa(idEvento) + ":" + strconv.FormatInt(idOspite, 10)
}

func mysql() {

}

func Connect() (*sql.DB, error) {
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	dsn := user + ":" + password +
		"@tcp(" + host + ":" + port + ")/" + dbName

	db, err := sql.Open("mysql", dsn)

	if err != nil {
		return nil, err
	}

	for i := 0; i < 10; i++ {
		err = db.Ping()

		if err == nil {
			return db, nil
		}

		fmt.Println("Database non ancora disponibile, nuovo tentativo...")

		time.Sleep(2 * time.Second)
	}

	db.Close()

	return nil, err
}

func InserisciInvitato(db *sql.DB, invitati models.CreazioneEvento, idev int64) error {

	query := `
        INSERT INTO ospiti (nome, cognome, email)
        VALUES (?, ?, ?)
    `

	queryEventoOspite := `
		INSERT INTO eventi_ospiti (id_evento, id_ospite, qr_code)
		VALUES (?, ?, ?)
	`

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	var idOspiti []int64

	for _, invitato := range invitati.Invitati {

		result, err := tx.Exec(
			query,
			invitato.Nome,
			invitato.Cognome,
			invitato.Email,
		)
		if err != nil {
			tx.Rollback()
			return err
		}

		idOspite, err := result.LastInsertId()
		if err != nil {
			tx.Rollback()
			return err
		}

		idOspiti = append(idOspiti, idOspite)
	}

	for _, idOspite := range idOspiti {

		qr_code := codiceQr(int(idev), idOspite)

		_, err = tx.Exec(
			queryEventoOspite,
			idev,
			idOspite,
			qr_code,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}
func AggiungiEvento(db *sql.DB, evento models.CreazioneEvento) (error, int64) {

	query := `
        INSERT INTO eventi (nome_evento, inizio, fine, ora_inizio, ora_fine, camera_id)
        VALUES (?, ?, ?, ?, ?, ?)
    `
	result, err := db.Exec(
		query,
		evento.Evento.Nome_evento,
		evento.Evento.Data_inizio,
		evento.Evento.Data_fine,
		evento.Evento.Ora_inizio,
		evento.Evento.Ora_fine,
		evento.Evento.Camera_id,
	)
	if err != nil {
		return err, 0
	}

	eventoID, err := result.LastInsertId()
	if err != nil {
		return err, 0
	}

	err = InserisciInvitato(db, evento, eventoID)
	if err != nil {
		return err, 0
	}

	return nil, eventoID

}

func DettaglioEvento(db *sql.DB, idEvento int) (error, models.Evento) {

	evento := models.Evento{}

	query := `
		SELECT nome_evento, inizio, fine, ora_inizio, ora_fine
		FROM eventi
		WHERE id = ?`

	err := db.QueryRow(
		query,
		idEvento,
	).Scan(
		&evento.Nome_evento,
		&evento.Data_inizio,
		&evento.Data_fine,
		&evento.Ora_inizio,
		&evento.Ora_fine,
	)

	if err != nil {
		return err, evento
	}

	return nil, evento
}

func ModificaEvento(db *sql.DB, modifica models.ModificaEvento) (error, bool) {

	query := `
		UPDATE eventi
		SET nome_evento = ?, inizio = ?, fine = ?, ora_inizio = ?, ora_fine = ?
		WHERE id = ?
	`

	result, err := db.Exec(
		query,
		modifica.Evento.Nome_evento,
		modifica.Evento.Data_inizio,
		modifica.Evento.Data_fine,
		modifica.Evento.Ora_inizio,
		modifica.Evento.Ora_fine,
		modifica.Id,
	)

	if err != nil {
		return err, false
	}

	righe, err := result.RowsAffected()

	if err != nil {
		return err, false
	}

	return nil, righe > 0
}

func EventoInCorso(db *sql.DB, idEvento int) (error, bool) {

	var giornoInizio, giornoFine, oraInizio, oraFine []byte

	err := db.QueryRow(
		`SELECT inizio, fine, ora_inizio, ora_fine FROM eventi WHERE id = ?`,
		idEvento,
	).Scan(
		&giornoInizio,
		&giornoFine,
		&oraInizio,
		&oraFine,
	)

	if err == sql.ErrNoRows {
		return ErrEventoNonTrovato, false
	}

	if err != nil {
		return err, false
	}

	inizio, err := momentoEvento(giornoInizio, oraInizio)

	if err != nil {
		return err, false
	}

	fine, err := momentoEvento(giornoFine, oraFine)

	if err != nil {
		return err, false
	}

	ora := time.Now()

	return nil, !ora.Before(inizio) && ora.Before(fine)
}

// Ricava l'id evento dal qrcode e verifica che l'evento sia assegnato alla camera indicata
func QrAppartenenteCam(db *sql.DB, qrCode string, idCam int) (bool, error) {

	separatore := strings.Index(qrCode, ":")

	if separatore <= 0 {
		return false, nil
	}

	idEvento, err := strconv.Atoi(qrCode[:separatore])

	if err != nil {
		return false, nil
	}

	var totale int

	err = db.QueryRow(
		`SELECT COUNT(*) FROM eventi WHERE id = ? AND camera_id = ?`,
		idEvento,
		idCam,
	).Scan(&totale)

	if err != nil {
		return false, err
	}

	return totale > 0, nil
}

// Cerca, tra gli eventi assegnati alla camera, quello attivo in base a date e orari
func EventoAttivoDaCam(db *sql.DB, idCam int) (error, int) {

	rows, err := db.Query(
		`SELECT id, inizio, fine, ora_inizio, ora_fine FROM eventi WHERE camera_id = ?`,
		idCam,
	)

	if err != nil {
		return err, 0
	}
	defer rows.Close()

	ora := time.Now()

	for rows.Next() {

		var id int
		var giornoInizio, giornoFine, oraInizio, oraFine []byte

		err := rows.Scan(
			&id,
			&giornoInizio,
			&giornoFine,
			&oraInizio,
			&oraFine,
		)
		if err != nil {
			return err, 0
		}

		inizio, errInizio := momentoEvento(giornoInizio, oraInizio)
		fine, errFine := momentoEvento(giornoFine, oraFine)

		if errInizio != nil || errFine != nil {
			continue
		}

		if !ora.Before(inizio) && ora.Before(fine) {
			return nil, id
		}
	}

	if err := rows.Err(); err != nil {
		return err, 0
	}

	return ErrEventoNonTrovato, 0
}

// Elenca le camere registrate nel database
func ListaCam(db *sql.DB) (error, []models.Camera) {

	camere := make([]models.Camera, 0)

	rows, err := db.Query(`SELECT id, nome_cam FROM qrcode_cam ORDER BY nome_cam`)

	if err != nil {
		return err, nil
	}
	defer rows.Close()

	for rows.Next() {

		var camera models.Camera

		err := rows.Scan(
			&camera.Id,
			&camera.Nome_cam,
		)
		if err != nil {
			return err, nil
		}

		camere = append(camere, camera)
	}

	return rows.Err(), camere
}

// Registra una nuova camera e ne restituisce l'id
func AggiungiCam(db *sql.DB, nome string) (error, int64) {

	result, err := db.Exec(
		`INSERT INTO qrcode_cam (nome_cam) VALUES (?)`,
		nome,
	)
	if err != nil {
		return err, 0
	}

	id, err := result.LastInsertId()

	if err != nil {
		return err, 0
	}

	return nil, id
}

func momentoEvento(giorno []byte, ora []byte) (time.Time, error) {

	data, err := time.Parse("2006-01-02", string(giorno))

	if err != nil {
		return time.Time{}, err
	}

	orario, err := time.Parse("15:04:05", string(ora))

	if err != nil {
		return time.Time{}, err
	}

	return time.Date(
		data.Year(), data.Month(), data.Day(),
		orario.Hour(), orario.Minute(), orario.Second(),
		0, time.Local,
	), nil
}

func StatisticheEvento(db *sql.DB, idEvento int) (error, models.EsitoPresenze) {

	statistiche := models.EsitoPresenze{
		IdEvento: idEvento,
	}

	query := `
		SELECT COALESCE(SUM(presenza = TRUE), 0), COUNT(*)
		FROM eventi_ospiti
		WHERE id_evento = ?`

	err := db.QueryRow(
		query,
		idEvento,
	).Scan(
		&statistiche.Presenti,
		&statistiche.Totali,
	)

	if err != nil {
		return err, statistiche
	}

	statistiche.Mancanti = statistiche.Totali - statistiche.Presenti

	return nil, statistiche
}

func AggiungiInv(db *sql.DB, qrCode string) error {

	query := `
		UPDATE eventi_ospiti
		SET presenza = TRUE
		WHERE qr_code = ?
	`

	_, err := db.Exec(
		query,
		qrCode,
	)
	if err != nil {
		return err
	}

	return nil
}

func RegistraIngresso(db *sql.DB, qrCode string) (error, models.LogIngresso) {

	ingresso := models.LogIngresso{}

	var presenza sql.NullBool

	err := db.QueryRow(
		`SELECT id_evento, id_ospite, presenza FROM eventi_ospiti WHERE qr_code = ?`,
		qrCode,
	).Scan(
		&ingresso.IdEvento,
		&ingresso.IdOspite,
		&presenza,
	)

	if err == sql.ErrNoRows {
		return ErrQrNonRiconosciuto, ingresso
	}

	if err != nil {
		return err, ingresso
	}

	ingresso.GiaPresente = presenza.Valid && presenza.Bool

	err = db.QueryRow(
		`SELECT nome, cognome FROM ospiti WHERE id = ?`,
		ingresso.IdOspite,
	).Scan(
		&ingresso.Nome,
		&ingresso.Cognome,
	)

	if err != nil {
		return err, ingresso
	}

	if ingresso.GiaPresente {
		ingresso.DataIngresso = primoIngressoOspite(db, ingresso.IdEvento, ingresso.IdOspite)
		return nil, ingresso
	}

	var idLog int64

	idLog, err = InserisciLogIngresso(db, ingresso.IdEvento, ingresso.IdOspite)

	if err != nil {
		return err, ingresso
	}

	ingresso.Id = int(idLog)
	ingresso.DataIngresso = primoIngressoOspite(db, ingresso.IdEvento, ingresso.IdOspite)

	err = AggiungiInv(db, qrCode)

	return err, ingresso
}

func InserisciLogIngresso(db *sql.DB, idEvento int, idOspite int) (int64, error) {

	res, err := db.Exec(
		`
		INSERT INTO eventi_log (id_evento, id_ospite, data_ingresso)
		VALUES (?, ?, NOW())
		`,
		idEvento,
		idOspite,
	)

	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func primoIngressoOspite(db *sql.DB, idEvento int, idOspite int) string {

	var data string

	db.QueryRow(
		`
		SELECT DATE_FORMAT(data_ingresso, '%Y-%m-%d %H:%i:%s')
		FROM eventi_log
		WHERE id_evento = ? AND id_ospite = ?
		ORDER BY data_ingresso ASC
		LIMIT 1
		`,
		idEvento,
		idOspite,
	).Scan(&data)

	return data
}

func LogIngressi(db *sql.DB, idEvento int, limite int) (error, []models.LogIngresso) {

	logs := make([]models.LogIngresso, 0)

	query := `
		SELECT l.id, l.id_evento, l.id_ospite, o.nome, o.cognome,
			DATE_FORMAT(l.data_ingresso, '%Y-%m-%d %H:%i:%s')
		FROM eventi_log l
		JOIN ospiti o ON o.id = l.id_ospite
		WHERE l.id_evento = ?
		ORDER BY l.data_ingresso DESC, l.id DESC
		LIMIT ?
	`

	rows, err := db.Query(
		query,
		idEvento,
		limite,
	)

	if err != nil {
		return err, nil
	}
	defer rows.Close()

	for rows.Next() {
		var log models.LogIngresso

		err := rows.Scan(
			&log.Id,
			&log.IdEvento,
			&log.IdOspite,
			&log.Nome,
			&log.Cognome,
			&log.DataIngresso,
		)

		if err != nil {
			return err, nil
		}

		logs = append(logs, log)
	}

	return rows.Err(), logs
}

func AggiungiOspiteEvento(db *sql.DB, ospite models.OspiteEvento) (error, bool) {

	var idOspite int64

	err := db.QueryRow(
		`SELECT id FROM ospiti WHERE nome = ? AND cognome = ? LIMIT 1`,
		ospite.Nome,
		ospite.Cognome,
	).Scan(&idOspite)

	if err == sql.ErrNoRows {
		res, inserErrore := db.Exec(
			`INSERT INTO ospiti (nome, cognome, email) VALUES (?, ?, ?)`,
			ospite.Nome,
			ospite.Cognome,
			ospite.Email,
		)
		if inserErrore != nil {
			return inserErrore, false
		}
		idOspite, inserErrore = res.LastInsertId()
		if inserErrore != nil {
			return inserErrore, false
		}
	} else if err != nil {
		return err, false
	}

	var giaPresente int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM eventi_ospiti WHERE id_evento = ? AND id_ospite = ?`,
		ospite.IdEvento,
		idOspite,
	).Scan(&giaPresente)
	if err != nil {
		return err, false
	}
	if giaPresente > 0 {
		return nil, false
	}

	qrCode := codiceQr(ospite.IdEvento, idOspite)

	_, err = db.Exec(
		`INSERT INTO eventi_ospiti (id_evento, id_ospite, qr_code) VALUES (?, ?, ?)`,
		ospite.IdEvento,
		idOspite,
		qrCode,
	)
	if err != nil {
		return err, false
	}

	return nil, true
}

func RimuoviOspiteEvento(db *sql.DB, idEvento int, idOspite int) error {

	_, err := db.Exec(
		`DELETE FROM eventi_ospiti WHERE id_evento = ? AND id_ospite = ?`,
		idEvento,
		idOspite,
	)
	if err != nil {
		return err
	}

	return nil
}

func ListaOspiti(db *sql.DB) (error, []models.Invitato) {

	ospiti := make([]models.Invitato, 0)

	rows, err := db.Query(`SELECT nome, cognome, COALESCE(MAX(email), '')
	FROM ospiti
	GROUP BY nome, cognome;`)
	if err != nil {
		return err, nil
	}
	defer rows.Close()

	var inv models.Invitato

	for rows.Next() {
		err := rows.Scan(
			&inv.Nome,
			&inv.Cognome,
			&inv.Email,
		)
		if err != nil {
			return err, nil
		}
		ospiti = append(ospiti, inv)
	}

	err = rows.Err()

	if err != nil {
		return err, nil
	}

	return nil, ospiti

}

func ListaOspitiEvento(db *sql.DB, idEvento int64) (error, []models.Invitato) {

	ospiti := make([]models.Invitato, 0)

	rows, err := db.Query(`
		SELECT ospiti.nome, ospiti.cognome, COALESCE(ospiti.email, '')
		FROM eventi_ospiti
		JOIN ospiti ON eventi_ospiti.id_ospite = ospiti.id
		WHERE eventi_ospiti.id_evento = ?`, idEvento)
	if err != nil {
		return err, nil
	}
	defer rows.Close()

	var inv models.Invitato

	for rows.Next() {
		err := rows.Scan(
			&inv.Nome,
			&inv.Cognome,
			&inv.Email,
		)
		if err != nil {
			return err, nil
		}
		ospiti = append(ospiti, inv)
	}

	err = rows.Err()

	if err != nil {
		return err, nil
	}

	return nil, ospiti
}

func ListaEventi(db *sql.DB) (error, []models.EventoConOspiti) {

	eventi := make([]models.EventoConOspiti, 0)

	rows, err := db.Query(`
		SELECT id, nome_evento, inizio, fine, ora_inizio, ora_fine
		FROM eventi
		ORDER BY id`)
	if err != nil {
		return err, nil
	}
	defer rows.Close()

	for rows.Next() {

		var evento models.EventoConOspiti
		var idEvento int64

		err := rows.Scan(
			&idEvento,
			&evento.Nome_evento,
			&evento.Data_inizio,
			&evento.Data_fine,
			&evento.Ora_inizio,
			&evento.Ora_fine,
		)
		if err != nil {
			return err, nil
		}

		evento.Id = int(idEvento)

		err, evento.Invitati = ListaOspitiEvento(db, idEvento)
		if err != nil {
			return err, nil
		}

		eventi = append(eventi, evento)
	}

	err = rows.Err()

	if err != nil {
		return err, nil
	}

	return nil, eventi
}

func GetQrcode(db *sql.DB, id int) (error, []models.Ospinvito) {
	var invOspiti []models.Ospinvito
	var invOsp models.Ospinvito

	query := `SELECT eventi_ospiti.qr_code, ospiti.nome, ospiti.cognome, ospiti.email
		FROM eventi_ospiti
		JOIN ospiti ON eventi_ospiti.id_ospite = ospiti.id
		WHERE id_evento = ?`

	rows, err := db.Query(query, id)
	if err != nil {
		return fmt.Errorf("errore query: %w", err), nil
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(
			&invOsp.Invito,
			&invOsp.Nome,
			&invOsp.Cognome,
			&invOsp.Email,
		)
		if err != nil {
			return fmt.Errorf("errore scan: %w", err), nil
		}

		invOspiti = append(invOspiti, invOsp)
	}

	err = rows.Err()
	if err != nil {
		return fmt.Errorf("errore rows: %w", err), nil
	}

	return nil, invOspiti
}

var ErrUtenteNonTrovato = errors.New("utente non trovato")

// Cerca l'utente per email e restituisce l'hash della password
func VerificaUtente(db *sql.DB, email string) (string, error) {

	query := `SELECT password FROM utenti WHERE email = ?`

	var hash string

	err := db.QueryRow(query, email).Scan(&hash)

	if err == sql.ErrNoRows {
		return "", ErrUtenteNonTrovato
	}

	if err != nil {
		return "", err
	}

	return hash, nil
}

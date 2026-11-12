package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Durata dei token emessi dal login
const (
	DurataAccessToken  = 15 * time.Minute
	DurataRefreshToken = 7 * 24 * time.Hour
)

var ErrTokenNonValido = errors.New("token non valido")

// Chiave di firma degli access token, letta da JWT_SECRET
var segreto []byte

// Refresh token emessi e non ancora revocati
type sessione struct {
	email    string
	scadenza time.Time
}

var sessioni = map[string]sessione{}
var sessioniMutex sync.Mutex

// Deve essere chiamata una volta sola, all'avvio del server
func InizializzaSegreto() error {

	segreto = []byte(os.Getenv("JWT_SECRET"))

	if len(segreto) == 0 {
		return errors.New("variabile d'ambiente JWT_SECRET mancante")
	}

	if len(segreto) < 32 {
		return errors.New("JWT_SECRET deve essere lungo almeno 32 caratteri")
	}

	return nil
}

// Crea l'access token, valido per DurataAccessToken
func NuovoAccessToken(email string) (string, error) {

	ora := time.Now()

	jti := make([]byte, 12)

	_, err := rand.Read(jti)

	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": email,
		"iat": ora.Unix(),
		"exp": ora.Add(DurataAccessToken).Unix(),
		"jti": hex.EncodeToString(jti),
	})

	return token.SignedString(segreto)
}

// Verifica la firma e la scadenza di un access token, restituiscendo l'email
func VerificaAccessToken(token string) (string, error) {

	verificato, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenNonValido
		}
		return segreto, nil
	})

	if err != nil {
		return "", err
	}

	if !verificato.Valid {
		return "", ErrTokenNonValido
	}

	claims, ok := verificato.Claims.(jwt.MapClaims)

	if !ok {
		return "", ErrTokenNonValido
	}

	email, ok := claims["sub"].(string)

	if !ok || email == "" {
		return "", ErrTokenNonValido
	}

	return email, nil
}

// Emette un refresh token casuale e lo registra fra quelli validi
func NuovoRefreshToken(email string) (string, error) {

	grezzo := make([]byte, 32)

	_, err := rand.Read(grezzo)

	if err != nil {
		return "", err
	}

	token := hex.EncodeToString(grezzo)

	sessioniMutex.Lock()
	defer sessioniMutex.Unlock()

	pulisciSessioniScadute()

	sessioni[token] = sessione{
		email:    email,
		scadenza: time.Now().Add(DurataRefreshToken),
	}

	return token, nil
}

// Dato un refresh token valido, produce un access token nuovo
func RinnovaAccessToken(refresh string) (string, error) {

	sessioniMutex.Lock()
	registrata, ok := sessioni[refresh]
	sessioniMutex.Unlock()

	if !ok || time.Now().After(registrata.scadenza) {
		return "", ErrTokenNonValido
	}

	return NuovoAccessToken(registrata.email)
}

// Invalida un refresh token, usato dal logout
func RevocaRefreshToken(refresh string) {

	sessioniMutex.Lock()
	defer sessioniMutex.Unlock()

	delete(sessioni, refresh)
}

// Email associata a un refresh token, per il logout
func EmailRefreshToken(refresh string) (string, bool) {

	sessioniMutex.Lock()
	defer sessioniMutex.Unlock()

	registrata, ok := sessioni[refresh]

	if !ok || time.Now().After(registrata.scadenza) {
		return "", false
	}

	return registrata.email, true
}

// Chiamata tenendo il blocco: libera le sessioni scadute
func pulisciSessioniScadute() {

	ora := time.Now()

	for token, registrata := range sessioni {
		if ora.After(registrata.scadenza) {
			delete(sessioni, token)
		}
	}
}

package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

func middleware() {

}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inizio := time.Now()

		fmt.Println(r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		durata := time.Since(inizio)
		fmt.Println("durata: ", durata)
	})
}

// originsConsentiti := []string{
// 	"https://",
// }

// origin := r.Header.Get("Origin")
// if slices.Contains(originsConsentiti, origin) {
// 	w.Header().Set("Access-Control-Allow-Origin", origin)
// 	w.Header().Set("Access-Control-Allow-Credentials", "true")
// 	w.Header().Set("Vary", "Origin")
// }

func Cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set(
			"Access-Control-Allow-Methods",
			"GET, POST, PUT, DELETE, OPTIONS",
		)

		w.Header().Set(
			"Access-Control-Allow-Headers",
			"Content-Type, Authorization",
		)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)

	})
}

// Accetta solo l'access token JWT emesso dal login
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		token := r.Header.Get("Authorization")

		if !strings.HasPrefix(token, "Bearer ") {
			http.Error(w, "unauthorized request", http.StatusUnauthorized)
			return
		}

		_, err := VerificaAccessToken(strings.TrimPrefix(token, "Bearer "))

		if err != nil {
			http.Error(w, "unauthorized request", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

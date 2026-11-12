package handler

import (
	"backend/models"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"
)

func InviaInviti() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var emailInviti []models.EmailInvito

		err := json.NewDecoder(r.Body).Decode(&emailInviti)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		host := os.Getenv("SMTP_HOST")
		port := os.Getenv("SMTP_PORT")
		user := os.Getenv("SMTP_USER")
		pass := os.Getenv("SMTP_PASS")
		from := os.Getenv("MAIL_FROM")

		if host == "" || user == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.EsitoEmail{
				Errore: "SMTP non configurato: imposta SMTP_HOST, SMTP_USER e SMTP_PASS (per Gmail usa una App Password).",
			})
			return
		}
		if port == "" {
			port = "587"
		}
		if from == "" {
			from = user
		}

		esito := models.EsitoEmail{}

		for _, invito := range emailInviti {
			if invito.Email == "" {
				esito.Falliti++
				continue
			}

			msg, err := costruisciMessaggio(from, invito)
			if err != nil {
				esito.Falliti++
				continue
			}

			err = inviaSmtp(host, port, user, pass, from, invito.Email, msg)
			if err != nil {
				fmt.Println("errore invio mail a", invito.Email, ":", err)
				if esito.Errore == "" {
					messaggio := err.Error()
					if len(messaggio) > 250 {
						messaggio = messaggio[:250] + "..."
					}
					esito.Errore = messaggio
				}
				esito.Falliti++
				continue
			}
			esito.Inviati++
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(esito)
	}
}

func costruisciMessaggio(from string, invito models.EmailInvito) ([]byte, error) {
	boundary := "qrcodeboundary"
	mittente := from
	destinatario := invito.Email

	var sb strings.Builder

	sb.WriteString("From: " + mittente + "\r\n")
	sb.WriteString("To: " + destinatario + "\r\n")
	sb.WriteString("Subject: " + invito.Oggetto + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")

	qrBase64 := estraiBase64(invito.QrCode)

	if qrBase64 == "" {
		sb.WriteString("Content-Type: text/html; charset=utf-8\r\n\r\n")
		sb.WriteString(invito.Corpo)
		return []byte(sb.String()), nil
	}

	sb.WriteString("Content-Type: multipart/related; boundary=\"" + boundary + "\"\r\n\r\n")

	sb.WriteString("--" + boundary + "\r\n")
	sb.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	sb.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	sb.WriteString(invito.Corpo)
	sb.WriteString("\r\n")

	sb.WriteString("--" + boundary + "\r\n")
	sb.WriteString("Content-Type: image/png; name=\"qrcode.png\"\r\n")
	sb.WriteString("Content-Transfer-Encoding: base64\r\n")
	sb.WriteString("Content-ID: <qrcode>\r\n")
	sb.WriteString("Content-Disposition: inline; filename=\"qrcode.png\"\r\n\r\n")
	sb.WriteString(formattaBase64(qrBase64))
	sb.WriteString("\r\n--" + boundary + "--")

	return []byte(sb.String()), nil
}

func estraiBase64(qrCode string) string {
	if qrCode == "" {
		return ""
	}
	if idx := strings.Index(qrCode, "base64,"); idx >= 0 {
		return qrCode[idx+len("base64,"):]
	}
	return qrCode
}

func formattaBase64(dati string) string {
	var sb strings.Builder
	for i := 0; i < len(dati); i += 76 {
		fine := i + 76
		if fine > len(dati) {
			fine = len(dati)
		}
		sb.WriteString(dati[i:fine])
		sb.WriteString("\r\n")
	}
	return sb.String()
}

func inviaSmtp(host, port, user, pass, from string, to string, msg []byte) error {
	addr := net.JoinHostPort(host, port)

	tlsConfig := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true,
	}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	auth := smtp.PlainAuth("", user, pass, host)
	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()

}

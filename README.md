# Gestione Eventi con QR (ESP32-CAM)

Sistema per gestire eventi con inviti via email e controllo degli ingressi tramite QR code letto da una
fotocamera ESP32. Comprende un backend Go, un frontend web statico, un database MySQL e il firmware
dell'ESP32-CAM.

Per la documentazione completa (installazione dettagliata, schema del database, spiegazione di ogni
file) vedi **[DOCUMENTAZIONE.md](DOCUMENTAZIONE.md)**.

## Architettura

```
Frontend (web/)  <--- HTTP/JSON + JWT --->  Backend Go (backend/, porta 8080)  <--->  MySQL 8.4 (Docker)
                                                        ^
                                                        | HTTP (header "id" = camera)
                                                  ESP32-CAM (espcam/)
```

- **Backend Go**: API REST, autenticazione JWT, eventi/ospiti, inviti email (SMTP), presenze, stato dispositivi.
- **Frontend statico**: HTML + JS puro (login, gestione eventi, dashboard).
- **MySQL 8.4**: unico database `evento`, inizializzato da `backend/init.sql`.
- **ESP32-CAM**: decodifica i QR con `quirc` e li invia al backend; pubblica periodicamente il suo stato.

## Struttura del progetto

```
backend/    API in Go (handler, models, middleware, database, init.sql, compose.yaml, Dockerfile)
web/        frontend statico (html/, css/, js/)
espcam/     firmware ESP32 (ESP-IDF, main/, components/classifier/)
```

## Requisiti

- Docker Desktop (con `docker compose`)
- Browser moderno (Chrome/Edge/Firefox)
- ESP-IDF >= 5.0 (solo per compilare/flashare il firmware)
- Go 1.26.5 (solo se vuoi compilare il backend senza Docker o generare hash bcrypt)

## Avvio rapido

### 1. Backend + database

```bash
cd backend
copy .env.example .env      # Windows; su Linux/macOS: cp .env.example .env
# Apri .env e imposta: MYSQL_ROOT_PASSWORD, DB_PASSWORD (uguale), SMTP_*, MAIL_FROM, JWT_SECRET
docker compose up -d --build
```

Il backend risponde su `http://localhost:8080`. Al primo avvio MySQL crea DB, tabelle e la camera
di default `1 = espcam32`.

### 2. Utente di login

Il login richiede un utente nella tabella `utenti` con password **hashata bcrypt**. Procedura completa
e comando per generare l'hash in [DOCUMENTAZIONE.md §4.5](DOCUMENTAZIONE.md).

Crea il tuo utente con la tua email e una password a scelta (**non usare credenziali reali nei file versionati**).

### 3. Frontend

Il frontend è statico: servilo via HTTP (consigliato) e apri `login.html`. Ad esempio:

```bash
cd web/html
python -m http.server 5500
# poi apri http://localhost:5500/login.html
```

Il file `web/js/api.js` ricava da solo l'host del backend dall'indirizzo della pagina, quindi funziona
anche cambiando rete/IP senza modifiche.

### 4. Firmware ESP32-CAM

```bash
cd espcam/main
copy impostazioni_locali.h.example impostazioni_locali.h   # imposta Wi-Fi, BACKEND_HOST, ID_CAM
cd ..
idf.py build
idf.py -p COMx flash monitor     # Windows; su Linux/macOS usare /dev/ttyUSB0
```

`BACKEND_HOST` deve essere l'IP del PC dove gira il backend, `ID_CAM` deve coincidere con `qrcode_cam.id`
nel database.

## Configurazione: cosa cambiare sempre

1. `backend/.env`: password MySQL (`MYSQL_ROOT_PASSWORD` = `DB_PASSWORD`), credenziali SMTP
   (Gmail richiede una **App Password** con 2FA attiva), `JWT_SECRET` (>= 32 caratteri).
2. Utente di login in `utenti` (email + hash bcrypt).
3. `espcam/main/impostazioni_locali.h`: `WIFI_SSID`, `WIFI_PASSWORD`, `BACKEND_HOST`, `BACKEND_PORT`, `ID_CAM`.

I file `.env` e `impostazioni_locali.h` non sono versionati: usa i rispettivi `.example`.

## Test rapidi

```bash
cd backend
go build ./...     # compila il backend
go vet ./...       # analisi statica
go test ./...      # test (attualmente nessuno)
```

## Note utili

- Reset completo del database: `docker compose down -v` poi `docker compose up -d --build`.
- Se l'invio email fallisce con `535 BadCredentials`: rigenera l'App Password Gmail e aggiorna
  `SMTP_PASS`, poi `docker compose up -d --force-recreate backend`.
- Log del backend: `docker compose logs -f backend` (deve comparire `SERVER AVVIATO`).

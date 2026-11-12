# Documentazione tecnica — Gestione Eventi con QR (ESP32-CAM)

Sistema per la gestione di eventi con inviti via email e controllo degli ingressi tramite QR code letto da una camera ESP32.

---

## 1. Panoramica

Il progetto è composto da quattro parti che comunicano tra loro via HTTP.

```
+----------------+        HTTP (API REST)        +---------------------+
|   Frontend     |  <------------------------->  |   Backend Go        |
|  (web/ html+js)|      token JWT Bearer          |   (porta 8080)      |
+----------------+                                +----------+----------+
                                                             |
                                                     driver MySQL (TCP)
                                                             |
                                                  +----------v----------+
                                                  |  MySQL 8.4 (Docker) |
                                                  |  DB "evento"        |
                                                  +---------------------+

+----------------+        HTTP POST /qr                     ^
|  ESP32-CAM     |  -------------------->  Backend Go  -------+
|  (espcam/)     |  POST /aggiornaStatoDispositivo
+----------------+  (header "id" = id camera)
```

- **Backend Go** (`backend/`): espone un'API REST sulla porta 8080. Gestisce autenticazione (JWT), eventi, ospiti, inviti email (SMTP), presenze e stato dei dispositivi.
- **Frontend statico** (`web/`): pagine HTML + JS puro (nessun framework) che chiamano l'API. Tre pagine: login, lista/gestione eventi, dashboard evento.
- **Firmware ESP32-CAM** (`espcam/`): acquisisce frame dalla fotocamera, decodifica i QR con la libreria `quirc`, invia il codice letto al backend e periodicamente pubblica il proprio stato (RAM, Wi-Fi, IP).
- **MySQL 8.4** in Docker: unico database, inizializzato da `backend/init.sql`.

L'unico servizio con cui l'ESP32 parla è il backend Go; il frontend parla solo col backend (che a sua volta parla col DB).

---

## 2. Stack e versioni necessarie

| Componente | Versione | Note |
|---|---|---|
| Go | **1.26.5** | Definita in `backend/go.mod` e nel `Dockerfile` (immagine builder). Coerente tra i due file. |
| Docker Engine | recente (>= 20.10) | Per container MySQL e backend. |
| Docker Compose | v2 (`docker compose`) | Il file è `backend/compose.yaml`. |
| MySQL | **8.4** | Immagine Docker `mysql:8.4`, non serve installarlo sul PC. |
| ESP-IDF | **>= 5.0** (testato con 5.2) | Per compilare/flashare il firmware. |
| Modulo hardware | ESP32-CAM (pinout AI-Thinker) + microSD | La SD è opzionale (serve solo al classificatore). |
| Browser | Chrome/Edge/Firefox recente | Il CSS usa `:has()` e il JS usa `fetch`, `Promise`, `URLSearchParams`. |

Dipendenze Go (da `backend/go.mod`):
- `github.com/go-sql-driver/mysql v1.10.0` — driver MySQL.
- `github.com/golang-jwt/jwt/v5 v5.3.1` — creazione/verifica token JWT.
- `golang.org/x/crypto v0.57.0` — hash bcrypt delle password.

Dipendenze firmware (da `espcam/main/idf_component.yml`):
- `espressif/esp32-camera == 2.1.7`
- `espressif/quirc == 1.2.0`
- `idf >= 5.0` (componente locale `classifier`).

---

## 3. Flussi principali (end-to-end)

### 3.1 Login e sessione
1. L'utente apre `login.html` e inserisce email/password.
2. Il frontend chiama `POST /login`. Il backend confronta la password con l'hash bcrypt in `utenti`.
3. Se corretto, restituisce un **access token** JWT (durata 15 min) e un **refresh token** (durata 7 giorni).
4. Il frontend tiene l'access token **solo in memoria** e il refresh token in `localStorage`.
5. Ogni chiamata protetta invia `Authorization: Bearer <accessToken>`. Alla scadenza, il frontend chiama `POST /refresh` per ottenere un nuovo access token senza rifare il login.
6. `POST /logout` revoca il refresh token sul server.

### 3.2 Creazione evento
1. Dal pulsante `+` si apre il form "Nuovo evento": nome, date, orari, **camera** (tendina popolata da `POST /ottieniCam`, con possibilità di aggiungerne una nuova con `POST /aggiungiCam`).
2. Si aggiungono ospiti a mano, da un CSV, o scegliendoli da quelli già in DB.
3. `POST /csv` salva l'evento (`eventi`) e, per ogni invitato, una riga in `ospiti` e una in `eventi_ospiti` con il QR code generato dal backend nel formato `idEvento:idOspite`.

### 3.3 Invito via email
1. Il frontend chiede i QR con `POST /ottieniQrCode` (passando l'id evento).
2. Genera le immagini QR (libreria `qrcode.min.js`) e costruisce per ogni invitato un messaggio HTML con il QR inline.
3. `POST /inviaInviti` invia le email via SMTP (Gmail, con App Password).

### 3.4 Lettura QR all'ingresso
1. L'ESP32-CAM inquadra un QR, lo decodifica e fa `POST /qr` con corpo JSON `"idEvento:idOspite"` e header `id: <idCamera>`.
2. Il backend verifica che l'evento del QR sia assegnato a quella camera (`QrAppartenenteCam`); se no risponde 404.
3. Se valido, registra l'ingresso (`eventi_log`), marca la presenza e risponde con i dati dell'ospite.

### 3.5 Stato dispositivo e dashboard
1. L'ESP32 ogni 10 s fa `POST /aggiornaStatoDispositivo` con header `id: <idCamera>` e corpo `{ram, wifi, ip}`.
2. Il backend trova l'**evento attivo** di quella camera (`EventoAttivoDaCam`) e salva lo stato in memoria, associandolo all'id evento.
3. La dashboard (`dashboard.html?id=<evento>`) mostra presenze, ultimi ingressi e stato del dispositivo, aggiornandosi ogni 10 s.

---

## 4. Configurazione su un nuovo PC (passo-passo)

### 4.1 Prerequisiti
- Installare **Docker Desktop** (include `docker` e `docker compose`).
- Installare **ESP-IDF** (solo se devi compilare/flashare il firmware); su Windows usare l'installer ufficiale o il tool `idf.py`.
- Un browser moderno.
- (Opzionale) **Go 1.26.5** se vuoi compilare il backend senza Docker o generare hash bcrypt.

### 4.2 Ottenere il codice
```
git clone <url-del-repo>
cd esp32
```

### 4.3 Configurare il backend — file `backend/.env`

Il file `backend/.env` **non è versionato** (è in `.gitignore`). Va creato copiando il modello:

```
cd backend
copy .env.example .env      # Windows PowerShell
# oppure: cp .env.example .env
```

Valori da impostare (con riferimento a `backend/.env.example`):

| Variabile | Cosa metterci | Note |
|---|---|---|
| `MYSQL_ROOT_PASSWORD` | La password di root di MySQL | Scegli una password tua. |
| `DB_USER` | `root` | Lascia `root`. |
| `DB_PASSWORD` | **La stessa** di `MYSQL_ROOT_PASSWORD` | Il backend si connette come root: i due valori devono coincidere. |
| `SMTP_HOST` | `smtp.gmail.com` | Server email. |
| `SMTP_PORT` | `465` | Porta SMTP con TLS implicito (usata dal codice). |
| `SMTP_USER` | La tua email mittente | Es. `tuo-indirizzo@gmail.com`. |
| `SMTP_PASS` | **App Password** Gmail (16 caratteri) | NON la password normale dell'account. Vedi nota sotto. |
| `MAIL_FROM` | Uguale a `SMTP_USER` | Mittente mostrato. |
| `JWT_SECRET` | Stringa casuale **>= 32 caratteri** | Il server non parte se manca o è più corta. Cambiarla invalida tutte le sessioni. |

> **App Password Gmail**: serve che l'account Gmail abbia attivato la *verifica in 2 passaggi*. Poi in
> *Sicurezza → Password per le app* si genera una password di 16 caratteri da usare come `SMTP_PASS`.
> Se Gmail risponde `535 BadCredentials`, l'App Password è sbagliata, revocata o manca la 2FA.

Il file `backend/compose.yaml` non contiene segreti: legge tutte le variabili dal `.env`.

### 4.4 Avviare il backend (Docker)

```
docker compose up -d --build
```

- Crea due container: `mysql` (immagine `mysql:8.4`) e `backend` (build dal `Dockerfile`).
- Al primo avvio MySQL esegue `backend/init.sql` (crea DB, tabelle e la camera di default `1 = espcam32`).
- Il backend aspetta che MySQL sia "healthy" (vedi `depends_on`) e riprova la connessione.
- Log del backend:
```
docker compose logs -f backend
```
Deve comparire `SERVER AVVIATO`.

> **Importante sul volume dati**: il `compose.yaml` non define un volume nominato per MySQL, quindi Docker crea un *volume anonimo*. Un `docker compose down` **non cancella** i dati; per azzerare davvero il DB serve `docker compose down -v`.

### 4.5 Database: utente di login e camere

Il login richiede un utente nella tabella `utenti`, con password **hashata bcrypt** (non in chiaro).

**a) Entrare nella console MySQL**
```
docker exec -it backend-mysql-1 mysql -uroot -p
```
Inserire `MYSQL_ROOT_PASSWORD` quando richiesto.

**b) Generare l'hash bcrypt di una password** — crea un file temporaneo `backend/hash/main.go`:
```go
package main

import (
    "fmt"

    "golang.org/x/crypto/bcrypt"
)

// Stampa l'hash bcrypt della password indicata
func main() {
    hash, _ := bcrypt.GenerateFromPassword([]byte("la-tua-password"), bcrypt.DefaultCost)
    fmt.Println(string(hash))
}
```
Eseguilo con Go installato (`go run ./hash` nella cartella `backend`) oppure in un container:
```
docker run --rm -v "%cd%/backend":/app -w /app golang:1.26.5 go run ./hash
```
(Su Linux/macOS usare `"$PWD/backend"`.)

**c) Inserire l'utente** (sostituire email e hash):
```sql
USE evento;
INSERT INTO utenti (email, password) VALUES ('tua-email@example.com', '$2a$10$....hash....');
```

> L'utente va creato con la tua email e una password a scelta (hashata bcrypt).
> Non inserire credenziali reali nella documentazione o in altri file versionati.

**d) Camere**: `init.sql` inserisce `id=1, nome_cam='espcam32'`. Le altre camere si aggiungono
dall'interfaccia (tendina "Aggiungi camera" nella creazione evento). Ogni ESP32 deve avere `ID_CAM`
uguale all'id della camera nel DB.

### 4.6 Servire il frontend

Il frontend è statico e va servito via HTTP (consigliato), per esempio dalla cartella `web/html`:
```
cd web/html
python -m http.server 5500
```
Poi aprire `http://localhost:5500/login.html`.

- `web/js/api.js` ricava l'indirizzo del backend dall'host della pagina: `http://<host-pagina>:8080`.
  Quindi, se servi il frontend dallo **stesso PC** su cui gira il backend, funziona senza modifiche,
  anche cambiando rete/IP.
- Aprendo i file con doppio clic (`file://`) l'host è vuoto e viene usato `localhost:8080`.
- Se apri le pagine **da un altro dispositivo** della LAN, serve l'IP del PC dove gira il backend
  (l'host della pagina sarà quell'IP) e il backend deve essere raggiungibile sulla porta 8080.

### 4.7 Configurare e flashare il firmware ESP32

La configurazione locale è in `espcam/main/impostazioni_locali.h`, **non versionato**. Crearlo dall'esempio:
```
cd espcam/main
copy impostazioni_locali.h.example impostazioni_locali.h
```

Contenuto da adattare:

| Define | Significato |
|---|---|
| `WIFI_SSID` | Nome della rete Wi-Fi a cui si collega l'ESP. |
| `WIFI_PASSWORD` | Password della rete. |
| `BACKEND_HOST` | IP del PC sulla LAN dove gira il backend (es. `192.168.1.6`). |
| `BACKEND_PORT` | `8080`. |
| `ID_CAM` | Id della camera di questo dispositivo, **uguale** a `qrcode_cam.id` nel DB. |

Compilare e flashare (ambiente ESP-IDF attivo):
```
cd espcam
idf.py build
idf.py -p COMx flash monitor     # Windows; su Linux /dev/ttyUSB0
```

> La SD card (opzionale) va formattata FAT e contiene il file `qrclass.txt` con regole
> `<regex> <file.png>`; serve solo alla classificazione visiva dei QR.

### 4.8 Cambiare rete o PC in futuro
- **Frontend**: nessuna modifica, ricava da solo l'host della pagina.
- **Firmware**: vanno aggiornati `WIFI_SSID`, `WIFI_PASSWORD`, `BACKEND_HOST` in
  `impostazioni_locali.h` e riflashato. L'ESP **non** si adatta da solo.

### 4.9 Checklist dei valori da cambiare su un nuovo PC
1. `backend/.env`: `MYSQL_ROOT_PASSWORD`, `DB_PASSWORD` (uguali), `SMTP_*`, `MAIL_FROM`, `JWT_SECRET`.
2. Utente di login in `utenti` (email + hash bcrypt) — cambiare la password di default.
3. `espcam/main/impostazioni_locali.h`: Wi-Fi, `BACKEND_HOST`, `ID_CAM`.
4. Verificare che l'App Password Gmail sia valida e la 2FA attiva.

---

## 5. Database — schema (`backend/init.sql`)

Database: **`evento`**.

| Tabella | Colonne | Descrizione |
|---|---|---|
| `qrcode_cam` | `id` (PK, AI), `nome_cam` VARCHAR(30) | Elenco delle camere/dispositivi. |
| `eventi` | `id` (PK, AI), `nome_evento`, `inizio` DATE, `fine` DATE, `ora_inizio` TIME, `ora_fine` TIME, `camera_id` FK→`qrcode_cam.id` | Eventi; ogni evento è assegnato a **una** camera. |
| `ospiti` | `id` (PK, AI), `nome`, `cognome`, `email` | Anagrafica ospiti (riutilizzabile fra eventi). |
| `eventi_ospiti` | `id` (PK, AI), `id_evento` FK, `id_ospite` FK, `presenza` TINYINT(1), `qr_code` VARCHAR(50) | Iscrizione ospite→evento con stato presenza e codice QR. |
| `eventi_log` | `id` (PK, AI), `id_evento` FK, `id_ospite` FK, `log`, `data_ingresso` DATETIME | Registro degli ingressi. Indice su `(id_evento, data_ingresso)`. |
| `utenti` | `id` (PK, AI), `email` UNIQUE, `password` | Utenti del pannello (password bcrypt). |

**Relazioni**: `eventi.camera_id → qrcode_cam.id`; `eventi_ospiti.id_evento → eventi.id`;
`eventi_ospiti.id_ospite → ospiti.id`; `eventi_log` referenzia `eventi` e `ospiti`.

**Codice QR**: formato `"<idEvento>:<idOspite>"`, generato dal backend (`codiceQr` in `mysql.go`).

**Seed**: `INSERT INTO qrcode_cam (id, nome_cam) VALUES (1, 'espcam32');`

> Nota: un `eventi_ospiti` ha una sola `presenza` per coppia evento/ospite; `eventi_log` può avere
> più righe per lo stesso ospite (più letture). L'ingresso valido è il **primo** (`primoIngressoOspite`).

---

## 6. Backend Go (`backend/`)

Struttura pacchetti:
```
backend/
├── main.go                 # avvio, registrazione rotte
├── models/dati.go          # struct dei dati scambiati (JSON)
├── database/mysql.go       # connessione e tutte le query
├── middleware/
│   ├── middleware.go       # Logger, Cors, Auth
│   └── sessioni.go         # JWT e refresh token
├── handler/                # un file per endpoint/gruppo di endpoint
├── init.sql                # schema + seed
├── Dockerfile              # build multi-stage
├── compose.yaml            # mysql + backend
└── .env / .env.example     # configurazione
```

### 6.1 `main.go`
Registra tutte le rotte e avvia il server su `:8080`.
- `InizializzaSegreto()`: legge `JWT_SECRET`, fallisce se manca o è < 32 caratteri.
- `database.Connect()`: apre il pool MySQL e ritenta il `Ping` fino a 10 volte (attende che MySQL parta).
- Per ogni endpoint costruisce la catena di middleware (es. `Cors(Logger(Auth(handler)))`) — l'ordine conta.
- **Rotte pubbliche** (login/refresh/logout): solo `Logger` + `Cors`.
- **Rotte del dispositivo** (`/qr`, `/aggiornaStatoDispositivo`): solo `Logger`, **senza** `Auth` né `Cors`
  (sono chiamate server-to-server dell'ESP, non dal browser). L'identità è l'header `id`.
- Tutte le altre rotte: `Cors(Logger(Auth(handler)))`.

### 6.2 `middleware/middleware.go`
- `Logger(next)`: stampa metodo+path e durata della richiesta.
- `Cors(next)`: imposta `Access-Control-Allow-Origin` riflettendo l'`Origin` della richiesta, consente
  i metodi GET/POST/PUT/DELETE/OPTIONS e gli header `Content-Type, Authorization`. Risponde 204 alle
  richieste di preflight `OPTIONS`.
- `Auth(next)`: richiede header `Authorization: Bearer <token>`, verifica l'access token con
  `VerificaAccessToken`; se manca o non è valido risponde 401.

### 6.3 `middleware/sessioni.go`
- Costanti: `DurataAccessToken = 15m`, `DurataRefreshToken = 7 giorni`.
- `InizializzaSegreto()`: carica `JWT_SECRET` (min 32 caratteri).
- `NuovoAccessToken(email)`: crea un JWT HS256 con claim `sub` (email), `iat`, `exp`, `jti` casuale.
- `VerificaAccessToken(token)`: verifica firma e scadenza, controlla che il metodo sia HMAC, restituisce l'email.
- Refresh token: `NuovoRefreshToken` genera 32 byte casuali (hex) e li salva in una **mappa in memoria**
  con scadenza. `RinnovaAccessToken` scambia un refresh valido con un nuovo access token.
  `RevocaRefreshToken`/`EmailRefreshToken` gestiscono logout e lookup. `pulisciSessioniScadute` rimuove
  i token scaduti.

> I refresh token e lo stato dispositivo vivono **in memoria**: si perdono al riavvio del backend e non
> sono condivisi fra più repliche.

### 6.4 `models/dati.go`
Struct usate per il (de)serializzazionamento JSON. Le più importanti:
- `Evento` (con `Camera_id`), `Camera{Id, Nome_cam}`, `CreazioneEvento{Evento, Invitati}`, `ModificaEvento`.
- `Invitato`, `OspiteEvento`, `RimozioneOspite`, `LogIngresso`, `StatoDispositivo`.
- `EmailInvito{Email, Nome, Oggetto, Corpo, QrCode}`, `EsitoEmail{Inviati, Falliti, Errore}`.
- Autenticazione: `RichiestaLogin`, `RichiestaRefresh`, `RispostaLogin{AccessToken, RefreshToken, Email, Scadenza}`.
- `CampoModificato`/`EsitoModifica` per il diff delle modifiche evento.

### 6.5 `database/mysql.go`
Contiene **tutte** le query. Blocchi principali:
- `Connect()`: costruisce il DSN da `DB_USER/PASSWORD/HOST/PORT/NAME` e ritenta la connessione.
- `codiceQr(idEvento, idOspite)`: compone la stringa `"idEvento:idOspite"`.
- Eventi: `AggiungiEvento` (insert evento + invitati in transazione), `InserisciInvitato`,
  `DettaglioEvento`, `ModificaEvento`, `ListaEventi` (con ospiti annidati), `EventoInCorso`,
  `momentoEvento` (parsing data+ora MySQL).
- Camera/QR: `QrAppartenenteCam` (verifica che l'evento del QR sia assegnato alla camera),
  `EventoAttivoDaCam` (trova l'evento attivo della camera in base a data/ora correnti),
  `ListaCam`, `AggiungiCam`.
- Ingressi: `RegistraIngresso` (legge evento/ospite dal QR, evita doppi ingressi, scrive `eventi_log`),
  `AggiungiInv` (marca presenza), `InserisciLogIngresso`, `primoIngressoOspite`, `LogIngressi`.
- Ospiti: `ListaOspiti` (distinti per nome/cognome), `ListaOspitiEvento`, `AggiungiOspiteEvento`
  (riusa ospite esistente o lo crea), `RimuoviOspiteEvento`.
- QR: `GetQrcode(idEvento)` per l'invio degli inviti.
- Presenze: `StatisticheEvento` (presenti/totali/mancanti).
- Utenti: `VerificaUtente(email)` restituisce l'hash bcrypt.
- Errori sentinella: `ErrQrNonRiconosciuto`, `ErrEventoNonTrovato`, `ErrUtenteNonTrovato`.

### 6.6 `handler/utenti.go`
- `Login`: decodifica `RichiestaLogin`, normalizza l'email (trim+lowercase), cerca l'hash,
  confronta con `bcrypt.CompareHashAndPassword`. Email inesistente e password errata danno lo stesso
  messaggio "email o password non corretti" (401).
- `Refresh`: scambia un refresh token valido con una nuova coppia tramite `EmailRefreshToken`.
- `Logout`: revoca il refresh token.
- `emettiToken(email)`: crea access + refresh token e la risposta con l'orario di scadenza.

### 6.7 `handler/csv.go` — `CsvPost`
Crea un evento: decodifica `CreazioneEvento`, normalizza/valida, chiama `database.AggiungiEvento`,
risponde 201 con l'id dell'evento.

### 6.8 `handler/validazione.go`
- `normalizzaEvento`: trim del nome.
- `parseOra`: accetta `HH:MM` e `HH:MM:SS`.
- `validaEvento`: controlli su nome (obbligatorio, max 100), date e orari validi, coerenza
  data/ora di fine rispetto all'inizio.

### 6.9 `handler/ottieniEventi.go` — `OttieniEventi`
Restituisce la lista completa eventi con i relativi invitati (`ListaEventi`). Risponde 201
(anche se è una lettura).

### 6.10 `handler/modificaEvento.go` — `ModificaEvento`
Modifica nome/date/orari di un evento. Confronta il vecchio col nuovo (`confrontaEvento`) e restituisce
l'elenco dei campi cambiati (`EsitoModifica`), usato poi per l'email di avviso.
**Qui è definito anche l'helper condiviso `rispondiJson(w, stato, corpo)`**, usato da molti handler.

### 6.11 `handler/aggiungiOspite.go` — `AggiungiOspite`
Aggiunge un ospite a un evento (`AggiungiOspiteEvento`); risponde con `EsitoAggiunta{Aggiunto}` (false
se era già presente).

### 6.12 `handler/rimuoviOspite.go` — `RimuoviOspite`
Rimuove un ospite da un evento (`RimuoviOspiteEvento`).

### 6.13 `handler/ottieniOspiti.go` — `OttieniOspiti`
Elenca gli ospiti distinti in anagrafica (`ListaOspiti`).

### 6.14 `handler/ottieniQrCode.go` — `OttieniQrcode`
Riceve l'id evento e restituisce i QR code degli invitati (`GetQrcode`) da inviare nelle email.

### 6.15 `handler/presenti.go` — `Presenze`
Statistiche di presenza per un evento (`StatisticheEvento`): presenti/totali/mancanti.

### 6.16 `handler/logIngressi.go` — `LogIngressi`
Ultimi ingressi di un evento (limite `LIMITE_LOG_INGRESSI = 50`).

### 6.17 `handler/inviaInviti.go` — `InviaInviti`
- Legge la configurazione SMTP da env (`SMTP_HOST/PORT/USER/PASS`, `MAIL_FROM`).
- `costruisciMessaggio`: compone un'email HTML; se c'è un QR, crea un `multipart/related` con
  l'immagine inline (`Content-ID: <qrcode>`, referenziata nel corpo come `cid:qrcode`).
- `inviaSmtp`: apre una connessione **TLS implicito** (porta 465) con `tls.Dial`, autentica con
  `smtp.PlainAuth` e invia. `InsecureSkipVerify: true` (accetta il certificato senza verifica).
- Restituisce `EsitoEmail{Inviati, Falliti, Errore}` (il primo errore viene incluso, troncato a 250 caratteri).

### 6.18 `handler/qrcode.go` — `Qrcode` (`/qr`)
Endpoint del dispositivo. Legge l'id camera dall'header `id`, decodifica il corpo (stringa QR),
verifica `QrAppartenenteCam`, poi `RegistraIngresso`. 404 se il QR non appartiene alla camera o non è
riconosciuto; altrimenti risponde con i dati dell'ingresso.

### 6.19 `handler/statoDispositivo.go`
Due endpoint e uno stato in memoria (`map[int]StatoDispositivo` protetta da mutex):
- `AggiornaStatoDispositivo` (`/aggiornaStatoDispositivo`): header `id` = camera, trova l'evento attivo
  (`EventoAttivoDaCam`), salva lo stato con timestamp. 404 se nessun evento attivo per la camera.
- `StatoDispositivo` (`/statoDispositivo`): dato un `IdEvento`, restituisce lo stato memorizzato
  (o un oggetto vuoto se assente).

### 6.20 `handler/camere.go`
- `OttieniCam`: elenca le camere (`ListaCam`).
- `AggiungiCam`: valida il nome (non vuoto) e crea la camera (`AggiungiCam`), risponde 201 con la camera.

### 6.21 `Dockerfile`
Build multi-stage: stage `golang:1.26.5` compila un binario statico (`CGO_ENABLED=0`, linux/amd64);
stage finale `alpine` con `ca-certificates` e `tzdata`, copia il binario ed espone la 8080.

### 6.22 `compose.yaml`
Due servizi:
- `mysql`: `mysql:8.4`, `TZ=Europe/Rome`, `MYSQL_ROOT_PASSWORD` dal `.env`, DB `evento`, monta `init.sql`
  in `/docker-entrypoint-initdb.d/`, healthcheck con `mysqladmin ping`.
- `backend`: build locale, porta `8080:8080`, tutte le variabili dall'`.env`, `depends_on: mysql` con
  `condition: service_healthy`.

---

## 7. Frontend Web (`web/`)

```
web/
├── html/
│   ├── login.html      # pagina di accesso
│   ├── index.html      # lista/gestione eventi
│   └── dashboard.html  # presenze e stato dispositivo di un evento
├── css/style.css       # foglio di stile unico
└── js/
    ├── api.js          # URL API, sessione/token, helper HTTP
    ├── login.js        # logica del form di login
    ├── script.js       # logica della pagina eventi
    ├── dashboard.js    # logica della dashboard
    └── qrcode.min.js   # libreria QR (davidshimjs/qrcodejs)
```

### 7.1 `js/api.js` (condiviso, caricato da tutte le pagine)
- **Costanti URL**: `BASE_URL = "http://" + (window.location.hostname || "localhost") + ":8080"` e tutte
  le `API_URL_*` derivate. Questo rende il frontend indipendente dall'IP fisso.
- **Chiavi e pagine**: `PAGINA_LOGIN`, `PAGINA_HOME`, `CHIAVE_REFRESH`, `CHIAVE_DESTINAZIONE`.
- **Gestione sessione**: `leggiRefreshToken`/`salvaRefreshToken`/`cancellaRefreshToken` (localStorage);
  `accessToken` vive solo in memoria; `rinnovoInCorso` evita refresh concorrenti.
- `richiediSessione()`: usata all'avvio delle pagine protette, rimanda al login se manca il refresh token.
- `rinnovaAccessToken()`: chiama `/refresh` e aggiorna i token; in caso di errore cancella la sessione.
- `accessTokenDisponibile()`: riusa l'access token o lo rinnova.
- `login(email, password)` / `logout()`: gestiscono la coppia di token.
- `postJson(url, corpo, rinnovoGiaTentato)`: helper per le chiamate protette. Allega
  `Authorization: Bearer`, e se riceve **401** prova **una volta** a rinnovare e ripetere, poi rimanda al login.
- `leggiRisposta`: estrae JSON/errore dalle risposte.
- `etichettaIdEvento(id)`: crea lo span `#id` mostrato accanto ai nomi evento.
- Alla fine aggancia il pulsante "Esci" (`#bottoneLogout`) se presente nella pagina.

### 7.2 `js/login.js` + `html/login.html`
- Se esiste già un refresh token, salta il login e vai alla pagina ricordata (`CHIAVE_DESTINAZIONE`) o all'home.
- Sul submit: valida i campi, chiama `login()`, poi reindirizza; in caso di errore mostra il messaggio.
- `login.html`: form con email/password e area errore; carica `api.js?v=4` e `login.js?v=1`.

### 7.3 `js/script.js` + `html/index.html` (pagina eventi)
La pagina ha più "schermate" che vengono mostrate/nascoste via `hidden`. Blocchi principali di `script.js`:
- **Guardia sessione** in testa (`richiediSessione`).
- **Utenze form**: `chiudiFormModifica` di tutti i riferimenti DOM.
- **Utility**: `chiaveOspite` (nome|cognome), `emailValida`, `escHtml`, `dataOggiLocale`, `oraAdessoLocale`,
  `oraMinuti`, `eventoInCorso`, `ordinaEventi` (eventi in corso in cima), `validaDateEOra`.
- **Creazione evento**: `aggiornaStatoConferma` abilita/disabilita "Conferma", gestione vincoli di date/ora,
  `bottoneConferma` che raccoglie gli invitati selezionati e chiama `postJson(API_URL_EVENTO, ...)` con
  `Camera_id`. Alla riuscita mostra la schermata inviti.
- **Camere**: `caricaCam` (popola la tendina da `/ottieniCam`), `bottoneAggiungiCam` (`/aggiungiCam`,
  aggiunge l'opzione e la seleziona).
- **Ospiti**: caricamento da `/ottieniOspiti`, aggiunta manuale, import da CSV (`estraiOspitiDaCsv`,
  rileva delimitatore `,`/`;` e la colonna email), deduplicazione.
- **QR e inviti**: `generaQrImg`, `qrDataUrl` (canvas→dataURL), `preparaInviti` (filtra email valide,
  costruisce corpo via `costruisciCorpoInvito`), `inviaInviti` (chiama `/inviaInviti`).
- **Dettaglio evento**: `mostraDettagli`, `aggiornaListaInvitati`, rimozione ospite, `caricaQrCodesEvento`.
- **Modifica evento**: apertura/chiusura form, validazione, `postJson(API_URL_MODIFICA_EVENTO, ...)` e
  `avvisaInvitati` che invia le email di avviso con le modifiche (stesso QR).
- `index.html`: contiene tutte le sezioni/form/schermate e carica `qrcode.min.js`, `api.js?v=4`,
  `script.js?v=14`.

### 7.4 `js/dashboard.js` + `html/dashboard.html`
- Legge l'id evento dal query string (`?id=`). Senza id mostra un avviso.
- `caricaTitoloEvento`, `caricaPresenze` (`/presenti`), `caricaIngressi` (`/logIngressi`, data formattata
  in italiano), `caricaStatoDispositivo` (`/statoDispositivo`).
- Aggiorna automaticamente ogni `INTERVALLO_aggiORNAMENTO = 10000` ms e col pulsante "Aggiorna".
- `dashboard.html`: due sezioni (presenze/registro a sinistra, dispositivo a destra) e pulsante Esci.

### 7.5 `css/style.css`
Foglio unico per tutte le pagine. Blocchi principali:
- **Variabili `:root`** (palette colori, raggi, ombre, larghezza colonna) e reset.
- **Base** (`body`, `#pagina`, titoli), **Carte** (`.carta`), **Pulsanti** (`.bottone-*`), pulsanti fissi
  `#bottoneLogout` e `#aggiungiEvento`.
- **Form** (input, checkbox, `.nota`, `.errore`, editor modifica).
- **Login** (sfondo scuro centrato).
- **Lista eventi** (`.id-evento`, badge "In corso" con animazione `pulsa`).
- **Dettaglio evento**, lista invitati, ospiti disponibili, griglia QR (`.blocco-qr`).
- **Nuovo evento** (griglia ospiti, `.aggiungi-manuali`).
- **Dashboard** (griglia a due colonne, contatori, registri, scheda dispositivo).
- **Accessibilità**: rispetta `prefers-reduced-motion`.

### 7.6 `js/qrcode.min.js`
Libreria di terze parti (qrcodejs) usata per generare i QR nel browser: `generaQrImg` e `qrDataUrl`
creano un `QRCode` e ne leggono il canvas.

---

## 8. Firmware ESP32-CAM (`espcam/`)

```
espcam/
├── CMakeLists.txt                 # progetto ESP-IDF "qrcode_demo"
├── partitions.csv                 # partizioni (nvs, factory 2M, storage fat 1M)
├── components/classifier/         # componente locale: classificatore regex
├── main/
│   ├── CMakeLists.txt             # sorgenti e dipendenze del componente main
│   ├── idf_component.yml          # dipendenze dal component registry
│   ├── qrcode_demo_main.c         # logica principale: camera, Wi-Fi, QR, HTTP
│   ├── stato_dispositivo.c/.h     # invio periodico dello stato
│   └── impostazioni_locali.h      # config locale (NON versionata)
└── README.md
```

### 8.1 `main/qrcode_demo_main.c`
- **Include e define**: pin del LED (`GPIO 4`), dimensioni immagine (240x240), URL di POST
  `http://BACKEND_HOST:BACKEND_PORT/qr`, bit del gruppo eventi Wi-Fi/IP, dimensioni coda QR.
- **`app_main`**: inizializza NVS (richiesto dal Wi-Fi) e crea `main_task`.
- **`main_task`**: monta la microSD (se presente) e carica il classificatore; avvia la Wi-Fi (`wifi_init_sta`);
  configura LED e camera (pinout AI-Thinker, formato RGB565, 240x240, PSRAM); crea le code e le task
  `processing_task` e `http_task`; avvia `stato_dispositivo_avvia`; poi entra nel ciclo che acquisisce
  frame e li passa alla task di elaborazione.
- **`processing_task`**: prende un frame, lo converte in scala di grigi (`rgb565_to_grayscale_buf`),
  usa `quirc` per individuare e decodificare i QR (`quirc_extract`, `quirc_decode`), lampeggia il LED,
  classifica il contenuto e accoda il payload per la task HTTP. Un cooldown (`CAPTURE_COOLDOWN_US`)
  limita la frequenza di scatti.
- **Wi-Fi**: `wifi_event_handler` gestisce connessione, disconnessione (riconnessione automatica) e
  l'ottenimento dell'IP; `wifi_init_sta` configura la modalità stazione con `WIFI_SSID`/`WIFI_PASSWORD`.
- **`http_post_qrcode`**: costruisce il corpo JSON (stringa tra virgolette, con escape), imposta gli
  header `Content-Type` e `id` (id camera), esegue la POST e mappa l'esito in `QR_INVIATO`/`QR_RIFIUTATO`/
  `QR_SENZA_RETE`.
- **`http_task`**: sposta i QR dalla coda a una lista FIFO (deduplicando), attende che Wi-Fi **e** IP
  siano pronti, invia il QR più vecchio. Se il server è irraggiungibile ritenta (LED a doppio lampeggio);
  se il server rifiuta, il QR viene scartato.
- **Conversione RGB565→grigi**: `rgb565_to_grayscale` e `rgb565_to_grayscale_buf`.

### 8.2 `main/stato_dispositivo.c` / `.h`
- `INTERVALLO_STATO_MS = 10000` (in `.h`).
- `componi_corpo`: crea il JSON `{ram, wifi, ip}` (RAM libera/minima, RSSI Wi-Fi, IP locale).
- `invia_stato`: POST a `/aggiornaStatoDispositivo` con header `id = ID_CAM`; gestisce la risposta
  (200 ok, 404 nessun evento attivo, 400 header mancante).
- `stato_task`: ciclo periodico che chiama `invia_stato`.
- `stato_dispositivo_avvia` avvia la task; `stato_dispositivo_id_cam` restituisce l'id (usato anche dai QR).

### 8.3 `main/impostazioni_locali.h` (non versionato) / `.example`
Contiene `WIFI_SSID`, `WIFI_PASSWORD`, `BACKEND_HOST`, `BACKEND_PORT`, `ID_CAM`. L'esempio è
`impostazioni_locali.h.example`.

### 8.4 `components/classifier/`
Componente locale opzionale.
- `classifier_init(config_path)`: legge `qrclass.txt` (righe `regex filename`), compila le regex e le
  mette in una lista (`sys/queue.h`).
- `classifier_get_pic_from_qrcode_data(text)`: restituisce il primo filename la cui regex matcha il
  contenuto del QR, o `NULL`.
- `qrcode_classifier.h`: dichiarazioni delle due funzioni.

### 8.5 File di build
- `main/CMakeLists.txt`: registra i sorgenti e le dipendenze (`esp_timer`, `vfs`, `fatfs`, `sdmmc`,
  `esp_driver_gpio`, `classifier`, `esp32-camera`, `quirc`, `esp_wifi`, `esp_netif`, `esp_event`,
  `nvs_flash`, `esp_http_client`, `esp_system`); compila in `-O3`.
- `main/idf_component.yml`: versioni di `esp32-camera` e `quirc`, e `idf >= 5.0`.
- `partitions.csv`: `nvs` (16K), `factory` app (2M), `storage` fat (1M).
- `CMakeLists.txt` (root): imposta il progetto ESP-IDF `qrcode_demo`.

---

## 9. Riferimento API HTTP

Tutte le rotte sono `POST`. Salvo dove indicato richiedono `Authorization: Bearer <accessToken>`
(il middleware `Auth` risponde 401 se manca/errato).

| Endpoint | Auth | Corpo (esempio) | Risposta |
|---|---|---|---|
| `/login` | no | `{"Email","Password"}` | `{AccessToken, RefreshToken, Email, Scadenza}` |
| `/refresh` | no | `{"RefreshToken"}` | nuova coppia token |
| `/logout` | no | `{"RefreshToken"}` | `{}` |
| `/csv` | sì | `{Evento:{...camera_id...}, Invitati:[...]}` | id evento (201) |
| `/eventi` | sì | `{}` | lista eventi con invitati |
| `/modificaEvento` | sì | `{Id, Evento:{...}}` | `{Modificato, Modifiche[]}` |
| `/ottieniOspiti` | sì | `{}` | lista ospiti |
| `/aggiungiOspite` | sì | `{IdEvento, Nome, Cognome, Email}` | `{Aggiunto}` |
| `/rimuoviOspite` | sì | `{IdEvento, IdOspite}` | 200 |
| `/ottieniQrCode` | sì | `<idEvento>` | lista QR invitati |
| `/inviaInviti` | sì | `[{Email,Nome,Oggetto,Corpo,QrCode}]` | `{Inviati, Falliti, Errore}` |
| `/presenti` | sì | `{IdEvento}` | `{Presenti, Totali, Mancanti}` |
| `/logIngressi` | sì | `{IdEvento}` | lista ultimi ingressi |
| `/statoDispositivo` | sì | `{IdEvento}` | stato dispositivo salvato |
| `/ottieniCam` | sì | `{}` | lista camere |
| `/aggiungiCam` | sì | `{Nome_cam}` | camera creata (201) |
| `/qr` | no* | `"<idEvento>:<idOspite>"` | dati ingresso; header `id` = camera |
| `/aggiornaStatoDispositivo` | no* | `{ram,wifi,ip}` | stato salvato; header `id` = camera |

\* Rotte del dispositivo: nessun token, l'identità è l'header `id` (id camera) e non passano dal CORS.

---

## 10. Sicurezza e note operative

- **Segreti fuori dal repo**: `.env`, `impostazioni_locali.h` e `.gitignore` escludono credenziali.
  Versionati solo i `.example`. Sui nuovi PC copiare gli esempio e inserire i propri valori.
- **Password di default**: crea il tuo utente (email + hash bcrypt) e non usare credenziali reali nei file versionati.
- **App Password Gmail**: se l'invio email fallisce con `535 BadCredentials`, rigenerare l'App Password
  (serve la 2FA) e aggiornare `SMTP_PASS`; poi `docker compose up -d --force-recreate backend`.
- **Access token**: breve durata (15 min), in memoria. **Refresh token**: in `localStorage`, revocabile.
  Cambiando `JWT_SECRET` tutte le sessioni decadono.
- **Sessioni e stato dispositivo in memoria**: si azzerano al riavvio del backend; non scalano
  orizzontalmente (più repliche non condividono i token).
- **TLS email**: `InsecureSkipVerify: true` accetta qualsiasi certificato server per l'invio SMTP.
- **CORS**: consente qualsiasi `Origin` (riflesso). Le rotte del dispositivo non hanno CORS.
- **Codici di stato**: diversi endpoint di sola lettura rispondono 201 anziché 200; alcuni handler
  impostano `WriteHeader` dopo aver già scritto il corpo (inefficace). Non impatta l'uso normale.
- **Volume MySQL**: non è un volume nominato; `docker compose down` conserva i dati, `down -v` li cancella.
- **Reset del DB**: per ripartire da zero con dati puliti usare `docker compose down -v` e poi
  `docker compose up -d --build` (ricrea tabelle e camera `1 = espcam32`).

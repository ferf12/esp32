# AGENTS.md

## Commenti nel codice

Regola vincolante per tutti i file che scrivi o modifichi:

- I commenti devono essere scritti **esclusivamente in italiano**.
- I commenti devono essere usati **solo come titolo per ogni chunk di codice**: un breve titolo che identifica lo scopo del blocco di codice che segue.
- Non aggiungere commenti che spiegano il comportamento riga per riga, commenti di debug, TODO annotati, commenti su righe singole dentro la logica, né commenti ridondanti rispetto al codice.
- Formato consigliato: una riga di titolo breve sopra il blocco, senza punto finale se non serve.

Esempio corretto:

```go
// Recupera gli ospiti registrati dal database
func ottieniOspiti() {
    // ...
}
```

Esempi da evitare:

```go
// fmt.Println stampa il valore
fmt.Println(x) // stampa x
// questo ciclo scende da i a 0
for i := n; i > 0; i-- {
}
```

## Struttura del progetto

- `backend/` — API in Go (handler, models, middleware, database).
- `espcam/` — firmware ESP32 (ESP-IDF, componenti in `components/`).
- `web/` — frontend statico (`html/`, `css/`, `js/`).

## Build e test

- Backend: `go build ./...` e `go test ./...` dalla cartella `backend`.
- Firmware: build con ESP-IDF (`idf.py build`) dalla cartella `espcam`.
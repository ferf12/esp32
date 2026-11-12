// Backend sullo stesso host da cui viene servita la pagina, porta 8080
const BASE_URL = "http://" + (window.location.hostname || "localhost") + ":8080";
const API_URL_OSPITI = BASE_URL + "/ottieniOspiti";
const API_URL_EVENTO = BASE_URL + "/csv";
const API_URL_EVENTI = BASE_URL + "/eventi";
const API_URL_QRCODE = BASE_URL + "/ottieniQrCode";
const API_URL_AGGIUNGI_OSPITE = BASE_URL + "/aggiungiOspite";
const API_URL_RIMUOVI_OSPITE = BASE_URL + "/rimuoviOspite";
const API_URL_INVIA_INVITI = BASE_URL + "/inviaInviti";
const API_URL_MODIFICA_EVENTO = BASE_URL + "/modificaEvento";
const API_URL_CAM = BASE_URL + "/ottieniCam";
const API_URL_AGGIUNGI_CAM = BASE_URL + "/aggiungiCam";
const API_URL_PRESENZE = BASE_URL + "/presenti";
const API_URL_LOG_INGRESSI = BASE_URL + "/logIngressi";
const API_URL_STATO_DISPOSITIVO = BASE_URL + "/statoDispositivo";
const API_URL_LOGIN = BASE_URL + "/login";
const API_URL_REFRESH = BASE_URL + "/refresh";
const API_URL_LOGOUT = BASE_URL + "/logout";

const PAGINA_LOGIN = "login.html";
const PAGINA_HOME = "index.html";
const CHIAVE_REFRESH = "sessione_refresh";
const CHIAVE_DESTINAZIONE = "sessione_destinazione";

// Access token: vive solo in memoria e sparisce chiudendo il browser
let accessToken = null;

// Refresh token: vive in localStorage ed e l'unico che permette di rientrare
// Impedisce richieste di rinnovo contemporanee, la seconda aspetta la prima
let rinnovoInCorso = null;

function leggiRefreshToken() {
    return localStorage.getItem(CHIAVE_REFRESH);
}

function salvaRefreshToken(token) {
    localStorage.setItem(CHIAVE_REFRESH, token);
}

function cancellaRefreshToken() {
    localStorage.removeItem(CHIAVE_REFRESH);
}

// Ricorda la pagina da cui si arriva al login, per tornare indietro poi
function ricordaDestinazione() {
    const pagina = window.location.pathname.split("/").pop();

    if (pagina && pagina !== PAGINA_LOGIN) {
        sessionStorage.setItem(CHIAVE_DESTINAZIONE, pagina + window.location.search);
    }
}

function vaiAlLogin() {
    ricordaDestinazione();
    cancellaRefreshToken();
    accessToken = null;
    window.location.replace(PAGINA_LOGIN);
}

// Controlla che ci sia una sessione, usato all'avvio di ogni pagina protetta
function richiediSessione() {
    if (!leggiRefreshToken()) {
        vaiAlLogin();
        return false;
    }
    return true;
}

// Chiede un access token nuovo usando il refresh token
function rinnovaAccessToken() {
    const refresh = leggiRefreshToken();

    if (!refresh) {
        return Promise.resolve(false);
    }

    if (rinnovoInCorso) {
        return rinnovoInCorso;
    }

    rinnovoInCorso = fetch(API_URL_REFRESH, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ RefreshToken: refresh })
    }).then(response => {
        if (!response.ok) {
            cancellaRefreshToken();
            return false;
        }
        return response.json();
    }).then(esito => {
        if (!esito) {
            return false;
        }
        accessToken = esito.AccessToken;
        salvaRefreshToken(esito.RefreshToken);
        return true;
    }).catch(() => {
        cancellaRefreshToken();
        return false;
    }).then(esito => {
        rinnovoInCorso = null;
        return esito;
    });

    return rinnovoInCorso;
}

// Access token pronto, riusato se c'e gia' altrimenti rinnovato
function accessTokenDisponibile() {
    if (accessToken) {
        return Promise.resolve(accessToken);
    }

    return rinnovaAccessToken().then(ottenuto => {
        if (!ottenuto) {
            vaiAlLogin();
            return null;
        }
        return accessToken;
    });
}

// Login: scambia email e password con una coppia di token
function login(email, password) {
    return fetch(API_URL_LOGIN, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ Email: email, Password: password })
    }).then(response => {
        return response.text().then(testo => {
            let esito = null;
            if (testo) {
                try {
                    esito = JSON.parse(testo);
                } catch (ignorato) {
                    esito = null;
                }
            }
            if (!response.ok) {
                const messaggio = esito && esito.Errore
                    ? esito.Errore
                    : "Errore nella richiesta di login (" + response.status + ")";
                throw new Error(messaggio);
            }
            return esito;
        });
    }).then(esito => {
        accessToken = esito.AccessToken;
        salvaRefreshToken(esito.RefreshToken);
        return esito;
    });
}

// Logout: revoca il refresh token sul server e cancella i token locali
function logout() {
    const refresh = leggiRefreshToken();

    accessToken = null;
    cancellaRefreshToken();

    if (!refresh) {
        window.location.replace(PAGINA_LOGIN);
        return Promise.resolve();
    }

    return fetch(API_URL_LOGOUT, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ RefreshToken: refresh })
    }).catch(() => {
    }).then(() => {
        window.location.replace(PAGINA_LOGIN);
    });
}

function leggiRisposta(response, url) {
    return response.text().then(testo => {
        if (!response.ok) {
            let messaggio = "Errore nella richiesta a " + url + " (" + response.status + ")";
            if (testo) {
                try {
                    const errore = JSON.parse(testo);
                    messaggio = errore.Errore || messaggio + ": " + testo.trim();
                } catch (ignorato) {
                    messaggio += ": " + testo.trim();
                }
            }
            throw new Error(messaggio);
        }
        if (!testo) {
            return null;
        }
        return JSON.parse(testo);
    });
}

// Chiamata protetta: allega l'access token, lo rinnova una volta e poi rimanda al login
function postJson(url, corpo, rinnovoGiaTentato) {
    return accessTokenDisponibile().then(token => {
        if (!token) {
            throw new Error("Sessione scaduta, accedi di nuovo.");
        }

        return fetch(url, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                "Authorization": "Bearer " + token
            },
            body: JSON.stringify(corpo)
        }).then(response => {
            if (response.status === 401) {
                if (rinnovoGiaTentato) {
                    vaiAlLogin();
                    throw new Error("Sessione scaduta, accedi di nuovo.");
                }
                accessToken = null;
                return rinnovaAccessToken().then(ottenuto => {
                    if (!ottenuto) {
                        vaiAlLogin();
                        throw new Error("Sessione scaduta, accedi di nuovo.");
                    }
                    return postJson(url, corpo, true);
                });
            }
            return leggiRisposta(response, url);
        });
    });
}

// Etichetta con l'id dell'evento, da mostrare accanto al nome
function etichettaIdEvento(id) {
    const etichetta = document.createElement("span");
    etichetta.className = "id-evento";
    etichetta.textContent = "#" + id;
    etichetta.title = "Id evento";
    return etichetta;
}

// Collega il pulsante "Esci" se la pagina ne ha uno
const bottoneLogout = document.getElementById("bottoneLogout");
if (bottoneLogout) {
    bottoneLogout.addEventListener("click", function () {
        logout();
    });
}

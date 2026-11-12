// Senza una sessione valida si viene rimandati alla pagina di login
if (!richiediSessione()) {
    throw new Error("Accesso richiesto");
}

const parametri = new URLSearchParams(window.location.search);
const eventoId = parametri.get("id") ? Number(parametri.get("id")) : null;

const INTERVALLO_aggiORNAMENTO = 10000;

const titoloEvento = document.getElementById("titoloEvento");
const statoPresenze = document.getElementById("statoPresenze");
const numeroPresenti = document.getElementById("presenti");
const numeroTotali = document.getElementById("totali");
const numeroMancanti = document.getElementById("mancanti");
const bottoneAggiorna = document.getElementById("aggiornaPresenze");
const statoIngressi = document.getElementById("statoIngressi");
const listaIngressi = document.getElementById("registriIngresso");
const statoDispositivo = document.getElementById("statoDispositivo");
const dispositivoRam = document.getElementById("dispositivoRam");
const dispositivoWifi = document.getElementById("dispositivoWifi");
const dispositivoIp = document.getElementById("dispositivoIp");

function dataInItaliano(data) {
    if (!data) {
        return "";
    }
    const parti = data.split(" ");
    if (parti.length !== 2) {
        return data;
    }
    const [anno, mese, giorno] = parti[0].split("-");
    return giorno + "/" + mese + "/" + anno + " alle " + parti[1].slice(0, 5);
}

function caricaTitoloEvento() {
    if (!eventoId) {
        return Promise.resolve();
    }

    return postJson(API_URL_EVENTI, {})
    .then(eventi => {
        const evento = eventi.find(e => e.Id === eventoId);
        if (evento) {
            titoloEvento.textContent = "";
            titoloEvento.append(evento.Nome_evento, etichettaIdEvento(evento.Id));
        }
    })
    .catch(error => {
        console.error("Errore nel caricamento dell'evento:", error);
    });
}

function caricaPresenze() {
    if (!eventoId) {
        statoPresenze.textContent = "Nessun evento selezionato: torna alla lista e clicca su un evento in corso.";
        bottoneAggiorna.disabled = true;
        return;
    }

    postJson(API_URL_PRESENZE, { IdEvento: eventoId })
    .then(statistiche => {
        numeroPresenti.textContent = statistiche.Presenti;
        numeroTotali.textContent = statistiche.Totali;
        numeroMancanti.textContent = statistiche.Mancanti;
        statoPresenze.textContent = "Aggiornato alle " + new Date().toLocaleTimeString();
    })
    .catch(error => {
        console.error("Errore nel caricamento delle presenze:", error);
        statoPresenze.textContent = error.message || "Errore nel caricamento delle presenze.";
    });
}

function caricaIngressi() {
    if (!eventoId) {
        return;
    }

    postJson(API_URL_LOG_INGRESSI, { IdEvento: eventoId })
    .then(logs => {
        listaIngressi.textContent = "";
        if (!logs.length) {
            statoIngressi.textContent = "Nessun ingresso registrato per ora.";
            return;
        }
        statoIngressi.textContent = "Ultimi " + logs.length + " ingressi";
        logs.forEach(log => {
            const voce = document.createElement("li");
            voce.textContent = log.Nome + " " + log.Cognome + " si è unito " + dataInItaliano(log.DataIngresso);
            listaIngressi.appendChild(voce);
        });
    })
    .catch(error => {
        console.error("Errore nel caricamento degli ingressi:", error);
        statoIngressi.textContent = error.message || "Errore nel caricamento degli ingressi.";
    });
}

function caricaStatoDispositivo() {
    if (!eventoId) {
        return;
    }

    postJson(API_URL_STATO_DISPOSITIVO, { IdEvento: eventoId })
    .then(stato => {
        if (!stato.Aggiornato) {
            statoDispositivo.textContent = "In attesa dell'ESPcam...";
            dispositivoRam.textContent = "-";
            dispositivoWifi.textContent = "-";
            dispositivoIp.textContent = "-";
            return;
        }
        statoDispositivo.textContent = "Aggiornato " + dataInItaliano(stato.Aggiornato);
        dispositivoRam.textContent = stato.Ram || "-";
        dispositivoWifi.textContent = stato.Wifi || "-";
        dispositivoIp.textContent = stato.Ip || "-";
    })
    .catch(error => {
        console.error("Errore nel caricamento dello stato del dispositivo:", error);
        statoDispositivo.textContent = error.message || "Errore nel caricamento dello stato del dispositivo.";
    });
}

bottoneAggiorna.addEventListener("click", () => {
    caricaPresenze();
    caricaIngressi();
    caricaStatoDispositivo();
});

caricaTitoloEvento();
caricaPresenze();
caricaIngressi();
caricaStatoDispositivo();

setInterval(() => {
    caricaPresenze();
    caricaIngressi();
    caricaStatoDispositivo();
}, INTERVALLO_aggiORNAMENTO);

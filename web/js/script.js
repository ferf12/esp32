// Senza una sessione valida si viene rimandati alla pagina di login
if (!richiediSessione()) {
    throw new Error("Accesso richiesto");
}

const bottoneOspiti = document.getElementById("aggiungiOspiti");
const bottoneConferma = document.getElementById("confermaEvento");
const listaOspiti = document.getElementById("listaOspiti");
const righeOspiti = document.getElementById("righeOspiti");
const bottoneAggiungiManuale = document.getElementById("bottoneAggiungiManuale");
const nomeManuale = document.getElementById("nomeManuale");
const cognomeManuale = document.getElementById("cognomeManuale");
const emailManuale = document.getElementById("emailManuale");
const bottoneCsv = document.getElementById("bottoneCsv");
const fileCsv = document.getElementById("fileCsv");
const esitoCsv = document.getElementById("esitoCsv");
const aggiungiEvento = document.getElementById("aggiungiEvento");
const listaEventi = document.getElementById("listaEventi");
const contenitoreEventi = document.getElementById("contenitoreEventi");
const dettagliEvento = document.getElementById("dettagliEvento");
const schermataInviti = document.getElementById("schermataInviti");
const formOspiteEvento = document.getElementById("formOspiteEvento");
const esitoInviti = document.getElementById("esitoInviti");
const bottoneCsvDettaglio = document.getElementById("bottoneCsvDettaglio");
const fileCsvDettaglio = document.getElementById("fileCsvDettaglio");
const invitaOspiti = document.getElementById("invitaOspiti");
const listaOspitiDisponibili = document.getElementById("listaOspitiDisponibili");
const bottoneModificaEvento = document.getElementById("modificaEvento");
const formModificaEvento = document.getElementById("formModificaEvento");
const bottoneSalvaModifica = document.getElementById("salvaModifica");
const erroreModifica = document.getElementById("erroreModifica");
const erroreEvento = document.getElementById("erroreEvento");
const esitoModifica = document.getElementById("esitoModifica");
const camEvento = document.getElementById("camEvento");
const nomeNuovaCam = document.getElementById("nomeNuovaCam");
const bottoneAggiungiCam = document.getElementById("bottoneAggiungiCam");

let ospitiDisponibili = [];
let eventoCorrente = null;
let eventoAppenaCreato = null;

const CAMPI_EVENTO_MODIFICA = ["modificaNome", "modificaDataInizio", "modificaDataFine", "modificaOraInizio", "modificaOraFine"];

function chiaveOspite(invitato) {
    return (invitato.Nome + "|" + invitato.Cognome).toLowerCase().trim();
}

function emailValida(email) {
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

function escHtml(testo) {
    return String(testo == null ? "" : testo)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;");
}

function dataOggiLocale() {
    const adesso = new Date();
    const mese = String(adesso.getMonth() + 1).padStart(2, "0");
    const giorno = String(adesso.getDate()).padStart(2, "0");
    return adesso.getFullYear() + "-" + mese + "-" + giorno;
}

function oraAdessoLocale() {
    const adesso = new Date();
    return String(adesso.getHours()).padStart(2, "0") + ":" + String(adesso.getMinutes()).padStart(2, "0");
}

function descrizionePeriodo(evento) {
    return evento.Data_inizio === evento.Data_fine
        ? "il " + evento.Data_inizio
        : "dal " + evento.Data_inizio + " al " + evento.Data_fine;
}

// Dal DB gli orari arrivano come HH:MM:SS, dal form come HH:MM
function oraMinuti(ora) {
    return String(ora || "").slice(0, 5);
}

function eventoInCorso(evento) {
    const oggi = dataOggiLocale();
    const adesso = oraAdessoLocale();
    const oraInizio = oraMinuti(evento.Ora_inizio);
    const oraFine = oraMinuti(evento.Ora_fine);

    const iniziato = evento.Data_inizio < oggi ||
        (evento.Data_inizio === oggi && oraInizio <= adesso);

    const nonFinito = evento.Data_fine > oggi ||
        (evento.Data_fine === oggi && oraFine > adesso);

    return iniziato && nonFinito;
}

// Gli eventi in corso stanno in cima, gli altri restano nell'ordine di arrivo
function ordinaEventi(eventi) {
    return eventi.slice().sort((a, b) => {
        return (eventoInCorso(a) ? 0 : 1) - (eventoInCorso(b) ? 0 : 1);
    });
}

// Controllo orari, il confronto testuale coincide con quello cronologico
function validaDateEOra(evento) {
    if (!evento.Data_inizio || !evento.Data_fine || !evento.Ora_inizio || !evento.Ora_fine) {
        return "Compila data e ora di inizio e fine";
    }

    if (evento.Data_fine < evento.Data_inizio) {
        return "La data di fine non può essere precedente alla data di inizio";
    }

    if (evento.Data_fine === evento.Data_inizio && evento.Ora_fine <= evento.Ora_inizio) {
        return "Quando inizio e fine coincidono, l'ora di fine deve essere successiva all'ora di inizio";
    }

    return "";
}

function leggiEventoModifica() {
    return {
        Nome_evento: document.getElementById("modificaNome").value.trim(),
        Data_inizio: document.getElementById("modificaDataInizio").value,
        Data_fine: document.getElementById("modificaDataFine").value,
        Ora_inizio: document.getElementById("modificaOraInizio").value,
        Ora_fine: document.getElementById("modificaOraFine").value
    };
}

function aggiornaStatoConferma() {
    const nome = document.getElementById("nomeEvento").value.trim();
    const dataInizio = document.getElementById("dataInizio").value;
    const dataFine = document.getElementById("dataFine").value;
    const oraInizio = document.getElementById("oraInizio").value;
    const oraFine = document.getElementById("oraFine").value;

    const selezionati = document.querySelectorAll("#righeOspiti input[type='checkbox']:checked").length;

    const conteggio = document.getElementById("conteggioOspiti");
    if (conteggio) {
        conteggio.textContent = selezionati === 0
            ? "Nessun ospite selezionato"
            : selezionati + (selezionati === 1 ? " ospite selezionato" : " ospiti selezionati");
    }

    const errore = validaDateEOra({
        Data_inizio: dataInizio,
        Data_fine: dataFine,
        Ora_inizio: oraInizio,
        Ora_fine: oraFine
    });

    const campiCompleti = dataInizio && dataFine && oraInizio && oraFine && camEvento.value;

    if (erroreEvento) {
        erroreEvento.textContent = campiCompleti ? errore : "";
    }

    bottoneConferma.disabled = !(nome && campiCompleti) || errore !== "";
}

["nomeEvento", "dataInizio", "dataFine", "oraInizio", "oraFine"].forEach(id => {
    document.getElementById(id).addEventListener("input", aggiornaStatoConferma);
    document.getElementById(id).addEventListener("change", aggiornaStatoConferma);
});

camEvento.addEventListener("change", aggiornaStatoConferma);

// Carica le camere dal database e riempie la tendina
function caricaCam() {
    postJson(API_URL_CAM, {})
    .then(camere => {
        camEvento.innerHTML = "";

        const vuota = document.createElement("option");
        vuota.value = "";
        vuota.textContent = "Seleziona una camera";
        camEvento.appendChild(vuota);

        camere.forEach(camera => {
            const opzione = document.createElement("option");
            opzione.value = camera.Id;
            opzione.textContent = camera.Nome_cam;
            camEvento.appendChild(opzione);
        });

        aggiornaStatoConferma();
    })
    .catch(error => {
        console.error("Errore nel caricamento delle camere:", error);
    });
}

bottoneAggiungiCam.addEventListener("click", () => {
    const nome = nomeNuovaCam.value.trim();

    if (!nome) {
        alert("Inserisci il nome della camera");
        return;
    }

    postJson(API_URL_AGGIUNGI_CAM, { Nome_cam: nome })
    .then(camera => {
        nomeNuovaCam.value = "";

        const opzione = document.createElement("option");
        opzione.value = camera.Id;
        opzione.textContent = camera.Nome_cam;
        camEvento.appendChild(opzione);
        camEvento.value = camera.Id;
        aggiornaStatoConferma();
    })
    .catch(error => {
        console.error("Errore nell'aggiunta della camera:", error);
        alert("Errore nell'aggiunta della camera");
    });
});

const oggi = dataOggiLocale();
document.getElementById("dataInizio").min = oggi;

document.getElementById("dataInizio").addEventListener("change", function() {
    document.getElementById("dataFine").min = this.value;
    aggiornaVincoliOra();
});

document.getElementById("dataFine").addEventListener("change", function() {
    aggiornaVincoliOra();
});

document.getElementById("oraInizio").addEventListener("change", function() {
    aggiornaVincoliOra();
});

document.getElementById("oraFine").addEventListener("change", function() {
    aggiornaVincoliOra();
});

function aggiornaVincoliOra() {
    const dataInizio = document.getElementById("dataInizio").value;
    const dataFine = document.getElementById("dataFine").value;
    const oraInizio = document.getElementById("oraInizio").value;
    const oraFine = document.getElementById("oraFine").value;
    const adesso = oraAdessoLocale();

    if (dataInizio === oggi) {
        document.getElementById("oraInizio").min = adesso;
    } else {
        document.getElementById("oraInizio").min = "";
    }

    if (dataFine === dataInizio && dataInizio !== "") {
        document.getElementById("oraFine").min = oraInizio || adesso;
        document.getElementById("oraInizio").max = oraFine || "23:59";
    } else {
        document.getElementById("oraFine").min = "";
        document.getElementById("oraInizio").max = "";
    }
}

// Gli stessi vincoli del form di creazione, ma senza bloccare gli eventi iniziati
function aggiornaVincoliOraModifica() {
    const dataInizio = document.getElementById("modificaDataInizio").value;
    const dataFine = document.getElementById("modificaDataFine").value;
    const oraInizio = document.getElementById("modificaOraInizio").value;
    const oraFine = document.getElementById("modificaOraFine").value;

    document.getElementById("modificaDataFine").min = dataInizio;

    if (dataFine === dataInizio && dataInizio !== "") {
        document.getElementById("modificaOraFine").min = oraInizio;
        document.getElementById("modificaOraInizio").max = oraFine || "23:59";
    } else {
        document.getElementById("modificaOraFine").min = "";
        document.getElementById("modificaOraInizio").max = "";
    }

    aggiornaStatoModifica();
}

function aggiornaStatoModifica() {
    const evento = leggiEventoModifica();
    const errore = validaDateEOra(evento);

    if (errore === "" && !evento.Nome_evento) {
        errore = "Inserisci il nome dell'evento";
    }

    erroreModifica.textContent = errore;
    bottoneSalvaModifica.disabled = errore !== "";
}

function creaRigaOspite(invitato, selezionato) {
    const riga = document.createElement("div");

    const spunta = document.createElement("input");
    spunta.type = "checkbox";
    spunta.dataset.nome = invitato.Nome;
    spunta.dataset.cognome = invitato.Cognome;
    spunta.dataset.email = invitato.Email || "";
    spunta.checked = !!selezionato;
    spunta.addEventListener("change", aggiornaStatoConferma);

    const nome = document.createElement("span");
    nome.textContent = invitato.Nome + " " + invitato.Cognome;

    riga.appendChild(spunta);
    riga.appendChild(nome);

    if (invitato.Email) {
        const mail = document.createElement("span");
        mail.className = "mail-ospite";
        mail.textContent = invitato.Email;
        riga.appendChild(mail);
    }

    return riga;
}

function integraOspiti(nuoviOspiti) {
    const chiavi = new Set(ospitiDisponibili.map(chiaveOspite));
    nuoviOspiti.forEach(invitato => {
        if (!chiavi.has(chiaveOspite(invitato))) {
            ospitiDisponibili.push(invitato);
            chiavi.add(chiaveOspite(invitato));
        }
    });
}

function aggiungiOspite(invitato, selezionato) {
    const chiavi = new Set(ospitiDisponibili.map(chiaveOspite));
    const chiave = chiaveOspite(invitato);

    if (chiavi.has(chiave)) {
        righeOspiti.querySelectorAll("input[type='checkbox']").forEach(spunta => {
            if (chiaveOspite({ Nome: spunta.dataset.nome, Cognome: spunta.dataset.cognome }) === chiave && selezionato) {
                spunta.checked = true;
            }
        });
        return false;
    }

    ospitiDisponibili.push(invitato);
    righeOspiti.appendChild(creaRigaOspite(invitato, selezionato));
    return true;
}

bottoneOspiti.addEventListener("click", () => {
    postJson(API_URL_OSPITI, {})
    .then(ospiti => {
        integraOspiti(ospiti);
        mostraOspiti();
    })
    .catch(error => {
        console.error("Errore nella richiesta:", error);
    });
});

function mostraOspiti() {
    const selezionate = new Set();
    righeOspiti.querySelectorAll("input[type='checkbox']:checked").forEach(spunta => {
        selezionate.add(chiaveOspite({ Nome: spunta.dataset.nome, Cognome: spunta.dataset.cognome }));
    });

    righeOspiti.innerHTML = "";
    listaOspiti.hidden = false;

    ospitiDisponibili.forEach(invitato => {
        righeOspiti.appendChild(creaRigaOspite(invitato, selezionate.has(chiaveOspite(invitato))));
    });

    aggiornaStatoConferma();
}

bottoneAggiungiManuale.addEventListener("click", () => {
    const nome = nomeManuale.value.trim();
    const cognome = cognomeManuale.value.trim();
    const email = emailManuale.value.trim();

    if (!nome) {
        alert("Inserisci il nome della persona");
        return;
    }

    if (email && !emailValida(email)) {
        alert("Email non valida: " + email);
        return;
    }

    aggiungiOspite({ Nome: nome, Cognome: cognome, Email: email }, true);

    nomeManuale.value = "";
    cognomeManuale.value = "";
    emailManuale.value = "";
    aggiornaStatoConferma();
});

bottoneCsv.addEventListener("click", () => {
    fileCsv.click();
});

fileCsv.addEventListener("change", () => {
    const file = fileCsv.files[0];
    if (!file) {
        return;
    }

    const reader = new FileReader();

    reader.onload = () => {
        const ospitiCsv = estraiOspitiDaCsv(reader.result);
        let aggiunti = 0;
        let nonValide = 0;

        ospitiCsv.forEach(invitato => {
            if (invitato.Email && !emailValida(invitato.Email)) {
                nonValide++;
            }
            if (aggiungiOspite(invitato, false)) {
                aggiunti++;
            }
        });

        let messaggio;
        if (aggiunti === 0) {
            messaggio = "Nessun ospite nuovo trovato nel CSV.";
        } else {
            messaggio = aggiunti + " ospiti importati dal CSV" +
                (ospitiCsv.length - aggiunti > 0 ? " (" + (ospitiCsv.length - aggiunti) + " già presenti)" : "");
        }

        if (nonValide > 0) {
            messaggio = messaggio + " Attenzione: " + nonValide + " email non valide saranno bloccate alla conferma.";
        }

        esitoCsv.textContent = messaggio;

        aggiornaStatoConferma();
        fileCsv.value = "";
    };

    reader.readAsText(file);
});

function pulisciCampo(campo) {
    return campo.replace(/^["'\s]+/, "").replace(/["'\s]+$/, "").trim();
}

function indiceColonnaEmail(intestazioni) {
    const sinonimi = ["email", "e-mail", "e_mail", "mail", "indirizzo email"];
    return intestazioni.findIndex(intestazione => sinonimi.includes(intestazione));
}

function estraiOspitiDaCsv(testo) {
    const righe = testo.replace(/\r/g, "").split("\n")
        .filter(riga => riga.trim() !== "");

    if (righe.length === 0) {
        return [];
    }

    const delimitatore = (righe[0].split(";").length > righe[0].split(",").length) ? ";" : ",";
    const ospiti = [];

    const provaHeader = righe[0].split(delimitatore).map(pulisciCampo).map(campo => campo.toLowerCase());
    const eHeader = provaHeader.some(campo => ["nome", "cognome", "email", "e-mail", "e_mail", "mail", "name"].includes(campo));
    const indiceEmail = eHeader ? indiceColonnaEmail(provaHeader) : -1;

    righe.forEach((riga, indice) => {
        const campi = riga.split(delimitatore).map(pulisciCampo);

        if (indice === 0 && eHeader) {
            return;
        }

        const nome = campi[0];
        const cognome = campi[1] || "";
        const email = (indiceEmail >= 0 && campi[indiceEmail]) ? campi[indiceEmail] : "";

        if (nome) {
            ospiti.push({ Nome: nome, Cognome: cognome, Email: email });
        }
    });

    return ospiti;
}

bottoneConferma.addEventListener("click", () => {

    if (bottoneConferma.disabled) {
        return;
    }

    const invitati = [];
    const emailErrate = [];

    righeOspiti.querySelectorAll("input[type='checkbox']").forEach(spunta => {
        if (spunta.checked) {
            const email = spunta.dataset.email || "";
            if (email && !emailValida(email)) {
                emailErrate.push(spunta.dataset.nome + " " + spunta.dataset.cognome + " <" + email + ">");
            }
            invitati.push({
                Nome: spunta.dataset.nome,
                Cognome: spunta.dataset.cognome,
                Email: email
            });
        }
    });

    if (emailErrate.length > 0) {
        alert("Impossibile creare l'evento: email non valide:\n\n" + emailErrate.join("\n"));
        return;
    }

    const evento = {
        Evento: {
            Nome_evento: document.getElementById("nomeEvento").value,
            Data_inizio: document.getElementById("dataInizio").value,
            Data_fine: document.getElementById("dataFine").value,
            Ora_inizio: document.getElementById("oraInizio").value,
            Ora_fine: document.getElementById("oraFine").value,
            Camera_id: Number(camEvento.value)
        },
        Invitati: invitati
    };

    postJson(API_URL_EVENTO, evento)
    .then(idEvento => {
        alert("evento creato");

        eventoAppenaCreato = {
            Id: idEvento,
            Nome_evento: evento.Evento.Nome_evento,
            Data_inizio: evento.Evento.Data_inizio,
            Data_fine: evento.Evento.Data_fine,
            Ora_inizio: evento.Evento.Ora_inizio,
            Ora_fine: evento.Evento.Ora_fine
        };

        document.getElementById("formEvento").reset();
        righeOspiti.innerHTML = "";
        ospitiDisponibili = [];
        nomeManuale.value = "";
        cognomeManuale.value = "";
        emailManuale.value = "";
        esitoCsv.textContent = "";
        fileCsv.value = "";
        listaOspiti.hidden = true;

        mostraSchermataInviti();
    })
    .catch(error => {
        console.error("Errore nella richiesta:", error);
        alert("Errore nella creazione dell'evento");
    });
});

function mostraSchermataInviti() {
    document.getElementById("formEvento").hidden = true;
    listaOspiti.hidden = true;
    bottoneConferma.hidden = true;
    listaEventi.hidden = true;
    dettagliEvento.hidden = true;
    aggiungiEvento.hidden = true;
    esitoInviti.textContent = "";
    schermataInviti.hidden = false;
}

document.getElementById("tornaLista").addEventListener("click", () => {
    schermataInviti.hidden = true;
    eventoAppenaCreato = null;
    caricaEventi();
});

function generaQrImg(testo, dimensione) {
    const contenitore = document.createElement("div");
    contenitore.style.display = "inline-block";
    new QRCode(contenitore, {
        text: testo,
        width: dimensione,
        height: dimensione,
        correctLevel: QRCode.CorrectLevel.M
    });
    return contenitore;
}

function costruisciCorpoInvito(ospite, evento) {
    return "" +
        "<div style='font-family: Arial, sans-serif; color: #2c3e50;'>" +
        "<p>Ciao " + escHtml(ospite.Nome) + ", sei stato invitato a <b>" + escHtml(evento.Nome_evento) + "</b>, " +
        descrizionePeriodo(evento) + ", dalle " + escHtml(evento.Ora_inizio) + " alle " + escHtml(evento.Ora_fine) + ".</p>" +
        "<p>Ecco il tuo QR code da mostrare all'ingresso:</p>" +
        "<img src='cid:qrcode' alt='QR Code' style='width: 200px; height: 200px;'>" +
        "</div>";
}

function costruisciCorpoModifica(ospite, evento, modifiche) {
    const righe = modifiche
        .map(campo =>
            "<li><b>" + escHtml(campo.Etichetta) + "</b>: " + escHtml(campo.Prima) +
            " &rarr; " + escHtml(campo.Dopo) + "</li>")
        .join("");

    return "" +
        "<div style='font-family: Arial, sans-serif; color: #2c3e50;'>" +
        "<p>Ciao " + escHtml(ospite.Nome) + ", l'evento <b>" + escHtml(evento.Nome_evento) + "</b> " +
        "a cui sei invitato ha subito delle modifiche:</p>" +
        "<ul>" + righe + "</ul>" +
        "<p>Le nuove date e orari sono: " + descrizionePeriodo(evento) +
        ", dalle " + escHtml(evento.Ora_inizio) + " alle " + escHtml(evento.Ora_fine) + ".</p>" +
        "<p>Il tuo QR code non cambia e resta valido per l'ingresso:</p>" +
        "<img src='cid:qrcode' alt='QR Code' style='width: 200px; height: 200px;'>" +
        "</div>";
}

function preparaInviti(qrcodes, evento, oggetto, costruisciCorpo) {
    return Promise.all(
        (qrcodes || [])
            .filter(qr => qr.Email && emailValida(qr.Email))
            .map(qr =>
                qrDataUrl(qr.Invito).then(dataUrl => ({
                    Email: qr.Email,
                    Nome: qr.Nome,
                    Oggetto: oggetto,
                    Corpo: costruisciCorpo(qr, evento),
                    QrCode: dataUrl
                }))
            )
    );
}

document.getElementById("inviaInviti").addEventListener("click", () => {
    if (!eventoAppenaCreato) {
        return;
    }

    const bottoneInvia = document.getElementById("inviaInviti");
    bottoneInvia.disabled = true;
    esitoInviti.textContent = "Invio in corso...";

    postJson(API_URL_QRCODE, eventoAppenaCreato.Id)
    .then(qrcodes => preparaInviti(
        qrcodes,
        eventoAppenaCreato,
        "Invito: " + eventoAppenaCreato.Nome_evento,
        costruisciCorpoInvito
    ))

    .then(inviti => {
        if (inviti.length === 0) {
            esitoInviti.textContent = "Nessun invitato con una email valida.";
            bottoneInvia.disabled = true;
            return null;
        }
        return postJson(API_URL_INVIA_INVITI, inviti);
    })
    .then(esito => {
        if (esito) {
            if (esito.Errore) {
                esitoInviti.textContent = "Errore nell'invio degli inviti: " + esito.Errore;
                bottoneInvia.disabled = false;
                return;
            }
            esitoInviti.textContent = "Inviti inviati: " + esito.Inviati +
                (esito.Falliti > 0 ? ", falliti: " + esito.Falliti : "");
            bottoneInvia.disabled = true;
        }
    })
    .catch(error => {
        console.error("Errore nell'invio degli inviti:", error);
        esitoInviti.textContent = error.message || "Errore nell'invio degli inviti.";
        bottoneInvia.disabled = false;
    });
});

function qrDataUrl(testo) {
    return new Promise((resolve, reject) => {
        const contenitore = document.createElement("div");
        new QRCode(contenitore, {
            text: testo,
            width: 200,
            height: 200,
            correctLevel: QRCode.CorrectLevel.M
        });
        setTimeout(() => {
            const canvas = contenitore.querySelector("canvas");
            if (canvas) {
                resolve(canvas.toDataURL("image/png"));
            } else {
                reject(new Error("Impossibile generare il QR code"));
            }
        }, 0);
    });
}

function caricaEventi() {

    postJson(API_URL_EVENTI, {})
    .then(eventi => {
        mostraListaEventi(eventi);
    })
    .catch(error => {
        console.error("Errore nella richiesta:", error);
    });
}

function nascondiTutteLeSchermate() {
    document.getElementById("formEvento").hidden = true;
    listaOspiti.hidden = true;
    bottoneConferma.hidden = true;
    listaEventi.hidden = true;
    dettagliEvento.hidden = true;
    schermataInviti.hidden = true;
    aggiungiEvento.hidden = true;
}

function mostraListaEventi(eventi) {

    nascondiTutteLeSchermate();
    listaEventi.hidden = false;
    aggiungiEvento.hidden = false;

    contenitoreEventi.innerHTML = "";

    ordinaEventi(eventi).forEach(evento => {

        const inCorso = eventoInCorso(evento);

        const blocco = document.createElement("button");
        blocco.type = "button";

        const nome = document.createElement("h3");
        nome.textContent = evento.Nome_evento;
        nome.appendChild(etichettaIdEvento(evento.Id));

        if (inCorso) {
            nome.classList.add("in-corso");
            nome.title = "Evento in corso";
        }

        const periodo = document.createElement("span");
        periodo.textContent = "Dal " + evento.Data_inizio + " al " + evento.Data_fine;

        blocco.appendChild(nome);
        blocco.appendChild(periodo);

        blocco.addEventListener("click", () => {
            if (inCorso) {
                window.location.href = "dashboard.html?id=" + evento.Id;
                return;
            }
            esitoModifica.textContent = "";
            mostraDettagli(evento);
        });

        contenitoreEventi.appendChild(blocco);
    });
}

function aggiornaListaInvitati(evento) {
    const lista = document.getElementById("dettaglioInvitati");
    lista.innerHTML = "";

    evento.Invitati.forEach(invitato => {
        const li = document.createElement("li");

        const testo = document.createElement("span");
        testo.textContent = invitato.Nome + " " + invitato.Cognome +
            (invitato.Email ? " (" + invitato.Email + ")" : "");

        const bottoneElimina = document.createElement("button");
        bottoneElimina.type = "button";
        bottoneElimina.className = "bottone bottone-rosso bottone-piccolo";
        bottoneElimina.textContent = "Elimina";
        bottoneElimina.addEventListener("click", () => {
            rimuoviOspiteDaEvento(invitato);
        });

        li.appendChild(testo);
        li.appendChild(bottoneElimina);
        lista.appendChild(li);
    });
}

function caricaQrCodesEvento(idEvento) {
    const contenitore = document.getElementById("qrcodesEvento");
    contenitore.innerHTML = "<p>Caricamento QR code...</p>";

    if (idEvento === undefined || idEvento === null) {
        contenitore.innerHTML = "<p>Impossibile caricare i QR code: ID evento mancante. Aggiorna il backend (docker compose up -d --build).</p>";
        return;
    }

    postJson(API_URL_QRCODE, idEvento)
    .then(qrcodes => {
        contenitore.innerHTML = "";

        if (!qrcodes || qrcodes.length === 0) {
            contenitore.innerHTML = "<p>Nessun QR code disponibile.</p>";
            return;
        }

        qrcodes.forEach(qr => {
            const blocco = document.createElement("div");
            blocco.className = "blocco-qr";

            const nome = document.createElement("p");
            nome.textContent = qr.Nome + " " + qr.Cognome;

            blocco.appendChild(nome);
            blocco.appendChild(generaQrImg(qr.Invito, 120));

            contenitore.appendChild(blocco);
        });
    })
    .catch(error => {
        console.error("Errore nel caricamento dei QR code:", error);
        contenitore.innerHTML = "<p>Errore nel caricamento dei QR code.</p>";
    });
}

document.getElementById("mostraFormOspite").addEventListener("click", () => {
    formOspiteEvento.hidden = !formOspiteEvento.hidden;
    if (!formOspiteEvento.hidden) {
        document.getElementById("nomeOspiteEvento").focus();
    }
});

document.getElementById("annullaOspiteEvento").addEventListener("click", () => {
    formOspiteEvento.hidden = true;
    formOspiteEvento.reset();
});

document.getElementById("confermaOspiteEvento").addEventListener("click", () => {
    if (!eventoCorrente) {
        return;
    }

    const nome = document.getElementById("nomeOspiteEvento").value.trim();
    const cognome = document.getElementById("cognomeOspiteEvento").value.trim();
    const email = document.getElementById("emailOspiteEvento").value.trim();

    if (!nome || !cognome) {
        alert("Inserisci nome e cognome della persona");
        return;
    }

    if (email && !emailValida(email)) {
        alert("Email non valida: " + email);
        return;
    }

    postJson(API_URL_AGGIUNGI_OSPITE, {
        IdEvento: eventoCorrente.Id,
        Nome: nome,
        Cognome: cognome,
        Email: email
    })
    .then(esito => {
        if (!esito.Aggiunto) {
            alert("Questo ospite è già stato invitato a questo evento.");
            return;
        }
        formOspiteEvento.hidden = true;
        formOspiteEvento.reset();
        ricaricaEventoCorrente();
    })
    .catch(error => {
        console.error("Errore nell'aggiunta dell'ospite:", error);
        alert("Errore nell'aggiunta dell'ospite");
    });
});

function rimuoviOspiteDaEvento(invitato) {
    if (!eventoCorrente) {
        return;
    }

    if (!confirm("Eliminare " + invitato.Nome + " " + invitato.Cognome + " dall'evento?")) {
        return;
    }

    postJson(API_URL_RIMUOVI_OSPITE, {
        IdEvento: eventoCorrente.Id,
        IdOspite: invitato.Id
    })
    .then(() => {
        ricaricaEventoCorrente();
    })
    .catch(error => {
        console.error("Errore nella rimozione dell'ospite:", error);
        alert("Errore nella rimozione dell'ospite");
    });
}

function mostraDettagli(evento) {
    eventoCorrente = evento;
    nascondiTutteLeSchermate();
    dettagliEvento.hidden = false;
    const titoloDettaglio = document.getElementById("dettaglioNome");
    titoloDettaglio.textContent = "";
    titoloDettaglio.append(evento.Nome_evento, etichettaIdEvento(evento.Id));
    document.getElementById("dettaglioInizio").textContent = evento.Data_inizio;
    document.getElementById("dettaglioFine").textContent = evento.Data_fine;
    document.getElementById("dettaglioOraInizio").textContent = evento.Ora_inizio;
    document.getElementById("dettaglioOraFine").textContent = evento.Ora_fine;
    formOspiteEvento.hidden = true;
    bottoneModificaEvento.hidden = eventoInCorso(evento);
    chiudiFormModifica();
    aggiornaListaInvitati(evento);
    caricaQrCodesEvento(evento.Id);
    caricaOspitiDisponibili();
}

function apriFormModifica() {
    if (!eventoCorrente) {
        return;
    }

    formModificaEvento.reset();
    document.getElementById("modificaNome").value = eventoCorrente.Nome_evento;
    document.getElementById("modificaDataInizio").value = eventoCorrente.Data_inizio;
    document.getElementById("modificaDataFine").value = eventoCorrente.Data_fine;
    document.getElementById("modificaOraInizio").value = eventoCorrente.Ora_inizio;
    document.getElementById("modificaOraFine").value = eventoCorrente.Ora_fine;

    esitoModifica.textContent = "";

    aggiornaVincoliOraModifica();

    formModificaEvento.hidden = false;
    document.getElementById("modificaNome").focus();
}

function chiudiFormModifica() {
    formModificaEvento.reset();
    formModificaEvento.hidden = true;
    erroreModifica.textContent = "";
    bottoneSalvaModifica.disabled = false;
}

// Avvisa gli invitati già presenti mantenendo lo stesso QR code
function avvisaInvitati(modifiche, evento) {
    return postJson(API_URL_QRCODE, evento.Id)
    .then(qrcodes => preparaInviti(
        qrcodes,
        evento,
        "Modifica evento: " + evento.Nome_evento,
        (ospite, evt) => costruisciCorpoModifica(ospite, evt, modifiche)
    ))
    .then(inviti => {
        if (inviti.length === 0) {
            return "Nessun invitato con una email valida da avvisare.";
        }
        return postJson(API_URL_INVIA_INVITI, inviti)
        .then(esito => {
            if (esito.Errore) {
                return "Attenzione all'invio delle email di avviso: " + esito.Errore;
            }
            return "Email di avviso inviate a " + esito.Inviati + " invitati" +
                (esito.Falliti > 0 ? ", " + esito.Falliti + " fallite" : "") + ".";
        });
    });
}

bottoneModificaEvento.addEventListener("click", () => {
    if (formModificaEvento.hidden) {
        apriFormModifica();
    } else {
        chiudiFormModifica();
    }
});

document.getElementById("annullaModifica").addEventListener("click", () => {
    chiudiFormModifica();
});

CAMPI_EVENTO_MODIFICA.forEach(id => {
    document.getElementById(id).addEventListener("input", aggiornaStatoModifica);
    document.getElementById(id).addEventListener("change", aggiornaStatoModifica);
});

["modificaDataInizio", "modificaDataFine", "modificaOraInizio", "modificaOraFine"].forEach(id => {
    document.getElementById(id).addEventListener("change", aggiornaVincoliOraModifica);
});

bottoneSalvaModifica.addEventListener("click", () => {
    if (!eventoCorrente) {
        return;
    }

    const dati = leggiEventoModifica();
    const errore = validaDateEOra(dati);

    if (errore !== "") {
        erroreModifica.textContent = errore;
        return;
    }

    const eventoAggiornato = {
        Id: eventoCorrente.Id,
        Nome_evento: dati.Nome_evento,
        Data_inizio: dati.Data_inizio,
        Data_fine: dati.Data_fine,
        Ora_inizio: dati.Ora_inizio,
        Ora_fine: dati.Ora_fine
    };

    const numeroInvitati = eventoCorrente.Invitati ? eventoCorrente.Invitati.length : 0;

    const domanda = numeroInvitati > 0
        ? "Salvare le modifiche e avvisare " + numeroInvitati + " invitati via email?"
        : "Salvare le modifiche?";

    if (!confirm(domanda)) {
        return;
    }

    bottoneSalvaModifica.disabled = true;
    esitoModifica.textContent = "Salvataggio in corso...";

    let salvato = false;

    postJson(API_URL_MODIFICA_EVENTO, {
        Id: eventoAggiornato.Id,
        Evento: {
            Nome_evento: eventoAggiornato.Nome_evento,
            Data_inizio: eventoAggiornato.Data_inizio,
            Data_fine: eventoAggiornato.Data_fine,
            Ora_inizio: eventoAggiornato.Ora_inizio,
            Ora_fine: eventoAggiornato.Ora_fine
        }
    })
    .then(esito => {
        if (!esito.Modificato) {
            esitoModifica.textContent = "Nessuna modifica da salvare.";
            return null;
        }

        salvato = true;
        esitoModifica.textContent = "Modifiche salvate, invio delle email in corso...";

        return avvisaInvitati(esito.Modifiche, eventoAggiornato);
    })
    .then(messaggio => {
        if (!salvato) {
            bottoneSalvaModifica.disabled = false;
            return;
        }

        esitoModifica.textContent = "Modifiche salvate. " + messaggio;
        ricaricaEventoCorrente();
    })
    .catch(error => {
        console.error("Errore nel salvataggio delle modifiche:", error);
        esitoModifica.textContent = error.message || "Errore nel salvataggio delle modifiche.";
        bottoneSalvaModifica.disabled = false;
    });
});

function caricaOspitiDisponibili() {
    postJson(API_URL_OSPITI, {})
    .then(ospiti => {
        listaOspitiDisponibili.innerHTML = "";
        ospiti.forEach(ospite => {
            const div = document.createElement("div");
            div.style.marginBottom = "5px";
            const spunta = document.createElement("input");
            spunta.type = "checkbox";
            spunta.dataset.id = ospite.Id;
            spunta.dataset.nome = ospite.Nome;
            spunta.dataset.cognome = ospite.Cognome;
            spunta.dataset.email = ospite.Email || "";
            const label = document.createElement("span");
            label.textContent = ospite.Nome + " " + ospite.Cognome + (ospite.Email ? " (" + ospite.Email + ")" : "");
            const bottoneAggiungi = document.createElement("button");
            bottoneAggiungi.type = "button";
            bottoneAggiungi.className = "bottone bottone-primario bottone-piccolo";
            bottoneAggiungi.textContent = "Aggiungi";
            bottoneAggiungi.addEventListener("click", () => {
                aggiungiOspiteDaDB(spunta);
            });
            div.appendChild(spunta);
            div.appendChild(label);
            div.appendChild(bottoneAggiungi);
            listaOspitiDisponibili.appendChild(div);
        });
    })
    .catch(error => {
        console.error("Errore nel caricamento degli ospiti:", error);
    });
}

function aggiungiOspiteDaDB(spunta) {
    postJson(API_URL_AGGIUNGI_OSPITE, {
        IdEvento: eventoCorrente.Id,
        Nome: spunta.dataset.nome,
        Cognome: spunta.dataset.cognome,
        Email: spunta.dataset.email
    })
    .then(esito => {
        if (!esito.Aggiunto) {
            alert("Questo ospite è già stato invitato a questo evento.");
            return;
        }
        ricaricaDettaglio();
    })
    .catch(error => {
        console.error("Errore nell'aggiunta dell'ospite:", error);
        alert("Errore nell'aggiunta dell'ospite");
    });
}

function ricaricaDettaglio() {
    postJson(API_URL_EVENTI, {})
    .then(eventi => {
        const aggiornato = eventi.find(e => e.Id === eventoCorrente.Id);
        if (aggiornato) mostraDettagli(aggiornato);
    })
    .catch(error => {
        console.error("Errore nel ricaricamento del dettaglio:", error);
    });
}

bottoneCsvDettaglio.addEventListener("click", () => {
    fileCsvDettaglio.click();
});

fileCsvDettaglio.addEventListener("change", () => {
    const file = fileCsvDettaglio.files[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
        const ospitiCsv = estraiOspitiDaCsv(reader.result);
        let aggiunti = 0;
        ospitiCsv.forEach(invitato => {
            postJson(API_URL_AGGIUNGI_OSPITE, {
                IdEvento: eventoCorrente.Id,
                Nome: invitato.Nome,
                Cognome: invitato.Cognome,
                Email: invitato.Email
            })
            .then(esito => {
                if (esito.Aggiunto) { aggiunti++; ricaricaDettaglio(); }
            })
            .catch(() => {});
        });
        if (aggiunti > 0) alert(aggiunti + " ospiti aggiunti dal CSV.");
        fileCsvDettaglio.value = "";
    };
    reader.readAsText(file);
});

invitaOspiti.addEventListener("click", () => {
    if (!eventoCorrente) return;
    postJson(API_URL_QRCODE, eventoCorrente.Id)
    .then(qrcodes => preparaInviti(
        qrcodes,
        eventoCorrente,
        "Invito: " + eventoCorrente.Nome_evento,
        costruisciCorpoInvito
    ))
    .then(inviti => {
        if (inviti.length === 0) { alert("Nessun ospite con email valida."); return null; }
        return postJson(API_URL_INVIA_INVITI, inviti);
    })
    .then(esito => {
        if (!esito) return;
        if (esito.Errore) { alert("Errore: " + esito.Errore); }
        else { alert("Inviti inviati con successo!"); }
    })
    .catch(error => {
        console.error("Errore nell'invio degli inviti:", error);
        alert("Errore nell'invio degli inviti: " + error.message);
    });
});

function ricaricaEventoCorrente() {
    postJson(API_URL_EVENTI, {})
    .then(eventi => {
        const aggiornato = eventi.find(e => e.Id === eventoCorrente.Id);
        if (aggiornato) mostraDettagli(aggiornato);
        else caricaEventi();
    })
    .catch(error => {
        console.error("Errore nel ricaricamento dell'evento:", error);
    });
}

document.getElementById("indietro").addEventListener("click", () => {
    eventoCorrente = null;
    esitoModifica.textContent = "";
    caricaEventi();
});

aggiungiEvento.addEventListener("click", () => {
    document.getElementById("formEvento").reset();
    righeOspiti.innerHTML = "";
    ospitiDisponibili = [];
    nomeManuale.value = "";
    cognomeManuale.value = "";
    emailManuale.value = "";
    nomeNuovaCam.value = "";
    esitoCsv.textContent = "";
    fileCsv.value = "";
    nascondiTutteLeSchermate();
    document.getElementById("formEvento").hidden = false;
    listaOspiti.hidden = false;
    bottoneConferma.hidden = false;
    caricaCam();
    aggiornaStatoConferma();
});

caricaEventi();

package models

type Invitato struct {
	Id      int
	Nome    string
	Cognome string
	Email   string
}
type Evento struct {
	Nome_evento string
	Data_inizio string
	Data_fine   string
	Ora_inizio  string
	Ora_fine    string
	Camera_id   int
}

type Camera struct {
	Id       int
	Nome_cam string
}

type EventoConOspiti struct {
	Id          int
	Nome_evento string
	Data_inizio string
	Data_fine   string
	Ora_inizio  string
	Ora_fine    string
	Invitati    []Invitato
}

type CreazioneEvento struct {
	Evento   Evento
	Invitati []Invitato
}

type ModificaEvento struct {
	Id     int
	Evento Evento
}

type CampoModificato struct {
	Etichetta string
	Prima     string
	Dopo      string
}

type EsitoModifica struct {
	Modificato bool
	Modifiche  []CampoModificato
	Errore     string
}

type Ospinvito struct {
	Id      int
	Nome    string
	Cognome string
	Email   string
	Invito  string
}

type OspiteEvento struct {
	IdEvento int
	Nome     string
	Cognome  string
	Email    string
}

type RimozioneOspite struct {
	IdEvento int
	IdOspite int
}

type EmailInvito struct {
	Email   string
	Nome    string
	Oggetto string
	Corpo   string
	QrCode  string
}

type EsitoEmail struct {
	Inviati int
	Falliti int
	Errore  string
}

type EsitoAggiunta struct {
	Aggiunto bool
}

type RichiestaPresenze struct {
	IdEvento int
}

type EsitoPresenze struct {
	IdEvento int
	Presenti int
	Totali   int
	Mancanti int
}

type LogIngresso struct {
	Id           int
	IdEvento     int
	IdOspite     int
	Nome         string
	Cognome      string
	DataIngresso string
	GiaPresente  bool
}

type StatoDispositivo struct {
	IdEvento   int
	Ram        string
	Wifi       string
	Ip         string
	Aggiornato string
}

// Credenziali inviate dal frontend per il login
type RichiestaLogin struct {
	Email    string
	Password string
}

// Token di rinnovo per ottenere un nuovo access token scaduto
type RichiestaRefresh struct {
	RefreshToken string
}

// Risposta del login: contiene il token di breve durata e quello di rinnovo
type RispostaLogin struct {
	AccessToken  string
	RefreshToken string
	Email        string
	Scadenza     string
}

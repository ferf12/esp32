package main

import (
	"backend/database"
	"backend/handler"
	"backend/middleware"
	"fmt"
	"net/http"
)

func main() {

	err := middleware.InizializzaSegreto()

	if err != nil {
		fmt.Println("errore configurazione token:", err)
		return
	}

	db, err := database.Connect()
	if err != nil {
		fmt.Println("errore database:", err)
		return
	}
	defer db.Close()

	handlerPost := handler.CsvPost(db)

	auth := middleware.Auth(handlerPost)
	log := middleware.Logger(auth)
	cors := middleware.Cors(log)

	handlerPres := handler.Presenze(db)

	authpres := middleware.Auth(handlerPres)
	logpres := middleware.Logger(authpres)
	corspres := middleware.Cors(logpres)

	handlerQrcode := handler.Qrcode(db)

	logqr := middleware.Logger(handlerQrcode)

	handlerOttieniQrCode := handler.OttieniQrcode(db)

	authOttQR := middleware.Auth(handlerOttieniQrCode)
	logOttQR := middleware.Logger(authOttQR)
	corsOQR := middleware.Cors(logOttQR)

	handlerOttOspiti := handler.OttieniOspiti(db)

	authOttOspiti := middleware.Auth(handlerOttOspiti)
	logOttOspiti := middleware.Logger(authOttOspiti)
	corsOttOspiti := middleware.Cors(logOttOspiti)

	handlerAggiungiOspite := handler.AggiungiOspite(db)

	authAggiungiOspite := middleware.Auth(handlerAggiungiOspite)
	logAggiungiOspite := middleware.Logger(authAggiungiOspite)
	corsAggiungiOspite := middleware.Cors(logAggiungiOspite)

	handlerRimuoviOspite := handler.RimuoviOspite(db)

	authRimuoviOspite := middleware.Auth(handlerRimuoviOspite)
	logRimuoviOspite := middleware.Logger(authRimuoviOspite)
	corsRimuoviOspite := middleware.Cors(logRimuoviOspite)

	handlerInviaInviti := handler.InviaInviti()

	authInviaInviti := middleware.Auth(handlerInviaInviti)
	logInviaInviti := middleware.Logger(authInviaInviti)
	corsInviaInviti := middleware.Cors(logInviaInviti)

	handlerOttEventi := handler.OttieniEventi(db)

	authOttEventi := middleware.Auth(handlerOttEventi)
	logOttEventi := middleware.Logger(authOttEventi)
	corsOttEventi := middleware.Cors(logOttEventi)

	handlerModificaEvento := handler.ModificaEvento(db)

	authModificaEvento := middleware.Auth(handlerModificaEvento)
	logModificaEvento := middleware.Logger(authModificaEvento)
	corsModificaEvento := middleware.Cors(logModificaEvento)

	handlerLogIngressi := handler.LogIngressi(db)

	authLogIngressi := middleware.Auth(handlerLogIngressi)
	logLogIngressi := middleware.Logger(authLogIngressi)
	corsLogIngressi := middleware.Cors(logLogIngressi)

	handlerAggiornaStato := handler.AggiornaStatoDispositivo(db)

	logAggiornaStato := middleware.Logger(handlerAggiornaStato)

	handlerStatoDispositivo := handler.StatoDispositivo(db)

	authStatoDispositivo := middleware.Auth(handlerStatoDispositivo)
	logStatoDispositivo := middleware.Logger(authStatoDispositivo)
	corsStatoDispositivo := middleware.Cors(logStatoDispositivo)

	handlerOttieniCam := handler.OttieniCam(db)

	authOttieniCam := middleware.Auth(handlerOttieniCam)
	logOttieniCam := middleware.Logger(authOttieniCam)
	corsOttieniCam := middleware.Cors(logOttieniCam)

	handlerAggiungiCam := handler.AggiungiCam(db)

	authAggiungiCam := middleware.Auth(handlerAggiungiCam)
	logAggiungiCam := middleware.Logger(authAggiungiCam)
	corsAggiungiCam := middleware.Cors(logAggiungiCam)

	// Rotte protette da Auth, che richiede l'access token emesso dal login
	http.Handle("/qr", logqr)
	http.Handle("/csv", cors)
	http.Handle("/presenti", corspres)
	http.Handle("/logIngressi", corsLogIngressi)
	http.Handle("/aggiornaStatoDispositivo", logAggiornaStato)
	http.Handle("/statoDispositivo", corsStatoDispositivo)
	http.Handle("/ottieniOspiti", corsOttOspiti)
	http.Handle("/eventi", corsOttEventi)
	http.Handle("/ottieniQrCode", corsOQR)
	http.Handle("/aggiungiOspite", corsAggiungiOspite)
	http.Handle("/rimuoviOspite", corsRimuoviOspite)
	http.Handle("/inviaInviti", corsInviaInviti)
	http.Handle("/modificaEvento", corsModificaEvento)
	http.Handle("/ottieniCam", corsOttieniCam)
	http.Handle("/aggiungiCam", corsAggiungiCam)

	// Login e rinnovo token: non passano da Auth, sono loro a produrre i token
	http.Handle("/login", middleware.Logger(middleware.Cors(handler.Login(db))))
	http.Handle("/refresh", middleware.Logger(middleware.Cors(handler.Refresh())))
	http.Handle("/logout", middleware.Logger(middleware.Cors(handler.Logout())))

	fmt.Println("SERVER AVVIATO")

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("errore server:", err)
		return
	}
}

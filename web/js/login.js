const formLogin = document.getElementById("formLogin");
const emailLogin = document.getElementById("emailLogin");
const passwordLogin = document.getElementById("passwordLogin");
const bottoneLogin = document.getElementById("bottoneLogin");
const erroreLogin = document.getElementById("erroreLogin");

// Se c'e' gia' una sessione si salta il login e si va alla destinazione
if (leggiRefreshToken()) {
    const destinazione = sessionStorage.getItem(CHIAVE_DESTINAZIONE);
    sessionStorage.removeItem(CHIAVE_DESTINAZIONE);
    window.location.replace(destinazione || PAGINA_HOME);
}

formLogin.addEventListener("submit", function (event) {
    event.preventDefault();

    erroreLogin.hidden = true;
    bottoneLogin.disabled = true;
    bottoneLogin.textContent = "Accesso in corso...";

    const email = emailLogin.value.trim();
    const password = passwordLogin.value;

    if (!email || !password) {
        erroreLogin.textContent = "Inserisci email e password.";
        erroreLogin.hidden = false;
        bottoneLogin.disabled = false;
        bottoneLogin.textContent = "Accedi";
        return;
    }

    login(email, password).then(() => {
        const destinazione = sessionStorage.getItem(CHIAVE_DESTINAZIONE);
        sessionStorage.removeItem(CHIAVE_DESTINAZIONE);
        window.location.replace(destinazione || PAGINA_HOME);
    }).catch(errore => {
        erroreLogin.textContent = errore.message;
        erroreLogin.hidden = false;
        bottoneLogin.disabled = false;
        bottoneLogin.textContent = "Accedi";
    });
});
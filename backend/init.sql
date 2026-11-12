CREATE DATABASE IF NOT EXISTS evento;

USE evento;

CREATE TABLE qrcode_cam (
    id INT AUTO_INCREMENT PRIMARY KEY,
    nome_cam VARCHAR(30)
);

CREATE TABLE eventi (
    id INT AUTO_INCREMENT PRIMARY KEY,
    nome_evento VARCHAR(100),
    inizio DATE,
    fine DATE,
    ora_inizio TIME,
    ora_fine TIME,
    camera_id INT,
    FOREIGN KEY (camera_id) REFERENCES qrcode_cam(id)
);

CREATE TABLE ospiti (
    id INT AUTO_INCREMENT PRIMARY KEY,
    nome VARCHAR(50),
    cognome VARCHAR(50),
    email VARCHAR(255)
);

CREATE TABLE eventi_ospiti (
    id INT AUTO_INCREMENT PRIMARY KEY,
    id_evento INT,
    id_ospite INT,
    presenza TINYINT(1),
    qr_code VARCHAR(50),
    FOREIGN KEY (id_evento) REFERENCES eventi(id),
    FOREIGN KEY (id_ospite) REFERENCES ospiti(id)
);

CREATE TABLE eventi_log (
    id INT AUTO_INCREMENT PRIMARY KEY,
    id_evento INT,
    id_ospite INT,
    log VARCHAR(100),
    data_ingresso DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (id_evento) REFERENCES eventi(id),
    FOREIGN KEY (id_ospite) REFERENCES ospiti(id)
);

CREATE INDEX idx_eventi_log_ingressi ON eventi_log (id_evento, data_ingresso);

CREATE TABLE utenti (
    id INT AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL
);

INSERT INTO qrcode_cam (id, nome_cam) VALUES (1, 'espcam32');

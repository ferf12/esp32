/*
 * Invio periodico dello stato del dispositivo al backend, con l'id camera nell'header "id"
 *
 * SPDX-License-Identifier: Apache-2.0
 */

#pragma once

// Ogni quanti millisecondi vengono inviati RAM, WiFi e IP al backend
#define INTERVALLO_STATO_MS 10000

// Avvia la task FreeRTOS che invia periodicamente lo stato del dispositivo
void stato_dispositivo_avvia(void);

// Id della camera gestita da questa ESPcam, usato anche per i codici QR
int stato_dispositivo_id_cam(void);

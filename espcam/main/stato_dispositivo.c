/*
 * Invio periodico dello stato del dispositivo al backend
 *
 * SPDX-License-Identifier: Apache-2.0
 */

#include <stdbool.h>
#include <stdio.h>
#include <string.h>

#include "esp_err.h"
#include "esp_http_client.h"
#include "esp_log.h"
#include "esp_netif.h"
#include "esp_system.h"
#include "esp_wifi.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"

#include "stato_dispositivo.h"
#include "impostazioni_locali.h"

static const char *TAG = "stato-dispositivo";

#define URL_STATO "http://" BACKEND_HOST ":" BACKEND_PORT "/aggiornaStatoDispositivo"
#define CORPO_MAX 256
#define HTTP_TIMEOUT_MS 5000

// Id della camera di questa ESPcam, inviato nell'header "id"
static int s_id_cam = ID_CAM;

// Legge l'IP della interfaccia Wi-Fi, NULL se non ancora disponibile
static const char *leggi_ip(void)
{
    static char ip[16];

    esp_netif_ip_info_t info;
    esp_netif_t *netif = esp_netif_get_handle_from_ifkey("WIFI_STA_DEF");

    if (netif == NULL || esp_netif_get_ip_info(netif, &info) != ESP_OK || info.ip.addr == 0) {
        return NULL;
    }

    snprintf(ip, sizeof(ip), IPSTR, IP2STR(&info.ip));
    return ip;
}

// Compone il JSON con lo stato corrente del dispositivo
static bool componi_corpo(char *corpo, size_t len)
{
    wifi_ap_record_t ap;
    if (esp_wifi_sta_get_ap_info(&ap) != ESP_OK) {
        return false;
    }

    const char *ip = leggi_ip();

    // Riporta anche il minimo mai raggiunto per capire se il firmware chiede piu memoria del previsto
    unsigned int liberi_kb = esp_get_free_heap_size() / 1024;
    unsigned int minimi_kb = esp_get_minimum_free_heap_size() / 1024;

    int scritti = snprintf(corpo, len,
                           "{\"ram\":\"%u KB liberi (min %u KB)\","
                           "\"wifi\":\"%d dBm\","
                           "\"ip\":\"%s\"}",
                           liberi_kb, minimi_kb, ap.rssi, ip != NULL ? ip : "non disponibile");

    return scritti > 0 && (size_t) scritti < len;
}

// Invia lo stato del dispositivo al backend
static esp_err_t invia_stato(void)
{
    char corpo[CORPO_MAX];

    if (!componi_corpo(corpo, sizeof(corpo))) {
        ESP_LOGW(TAG, "Wi-Fi non associato, invio dello stato rimandato");
        return ESP_FAIL;
    }

    esp_http_client_config_t config = {
        .url = URL_STATO,
        .method = HTTP_METHOD_POST,
        .timeout_ms = HTTP_TIMEOUT_MS,
    };

    esp_http_client_handle_t client = esp_http_client_init(&config);
    if (client == NULL) {
        return ESP_FAIL;
    }

    char header_id[16];
    snprintf(header_id, sizeof(header_id), "%d", s_id_cam);

    esp_http_client_set_header(client, "Content-Type", "application/json");
    esp_http_client_set_header(client, "id", header_id);
    esp_http_client_set_post_field(client, corpo, strlen(corpo));

    esp_err_t err = esp_http_client_perform(client);

    if (err == ESP_OK) {
        int status = esp_http_client_get_status_code(client);
        if (status == 200) {
            ESP_LOGI(TAG, "Stato inviato per la camera %d: %s", s_id_cam, corpo);
        } else {
            // Il backend risponde 404 se nessun evento della camera e attivo, 400 se manca l'header "id"
            ESP_LOGE(TAG, "Stato rifiutato dal backend (HTTP %d)", status);
        }
        err = (status == 200) ? ESP_OK : ESP_FAIL;
    } else {
        ESP_LOGE(TAG, "Invio dello stato fallito: %s", esp_err_to_name(err));
    }

    esp_http_client_cleanup(client);
    return err;
}

// Task periodica di invio dello stato
static void stato_task(void *arg)
{
    ESP_LOGI(TAG, "Invio dello stato ogni %d ms per la camera %d", INTERVALLO_STATO_MS, s_id_cam);

    while (true) {
        invia_stato();
        vTaskDelay(pdMS_TO_TICKS(INTERVALLO_STATO_MS));
    }
}

void stato_dispositivo_avvia(void)
{
    xTaskCreatePinnedToCore(&stato_task, "stato", 4096, NULL, 1, NULL, 0);
}

int stato_dispositivo_id_cam(void)
{
    return s_id_cam;
}

/*
 * SPDX-FileCopyrightText: 2022-2023 Espressif Systems (Shanghai) CO LTD
 *
 * SPDX-License-Identifier: Apache-2.0
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/param.h>
#include "esp_err.h"
#include "esp_log.h"
#include "esp_timer.h"
#include "esp_vfs_fat.h"
#include "driver/gpio.h"
#include "driver/sdmmc_host.h"
#include "sdmmc_cmd.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "freertos/queue.h"
#include "freertos/event_groups.h"
#include "sdkconfig.h"
#include "esp_wifi.h"
#include "esp_event.h"
#include "esp_netif.h"
#include "nvs_flash.h"
#include "esp_http_client.h"
#include "quirc.h"
#include "quirc_internal.h"
#include "esp_camera.h"
#include "qrcode_classifier.h"
#include "stato_dispositivo.h"
#include "impostazioni_locali.h"

static const char *TAG = "qrcode-demo";

// Dimensione immagine elaborata da quirc, uguale a quella della camera
#define IMG_WIDTH 240
#define IMG_HEIGHT 240
#define CAM_FRAME_SIZE FRAMESIZE_240X240

// LED flash integrato, libero solo con la microSD montata in modalita 1 bit
#define CAM_FLASH_LED_GPIO GPIO_NUM_4

#define SD_MOUNT_POINT "/sdcard"

// QR codes via HTTP POST, host e porta condivisi con l'invio periodico dello stato
#define HTTP_POST_URL "http://" BACKEND_HOST ":" BACKEND_PORT "/qr"

// Bit che segnalano la connessione Wi-Fi e l'ottenimento dell'IP
#define WIFI_CONNECTED_BIT BIT0
#define WIFI_GOT_IP_BIT    BIT1

// Dimensione massima di un payload QR e profondita della coda
#define QR_PAYLOAD_MAX 1024
#define QR_QUEUE_LEN   10

// Lampeggio breve del LED dopo una decodifica riuscita
#define LED_ON_DELAY_US (500 * 1000)
static int64_t s_led_on_until;

// Attesa minima fra due scatti: scatto, timer, scatto
#define CAPTURE_COOLDOWN_US (2 * 1000 * 1000)
static int64_t s_last_capture_us;

// Doppio lampeggio del LED quando una richiesta HTTP fallisce
#define ERROR_BLINK_ON_US  (150 * 1000)
#define ERROR_BLINK_DURATION_US (4 * ERROR_BLINK_ON_US)
static void led_blink_error(void);

// Payload passato dalla task di elaborazione alla task HTTP
typedef struct {
    char payload[QR_PAYLOAD_MAX];
} qr_payload_t;

// Nodo della lista FIFO dei QR decodificati in attesa di invio
typedef struct qr_node {
    char *payload;
    struct qr_node *next;
} qr_node_t;

// Esito dell'invio di un QR code, distingue la risposta del server dalla richiesta mai arrivata
typedef enum {
    QR_INVIATO,     // il server ha accettato il codice
    QR_RIFIUTATO,   // il server ha risposto con un codice diverso da 2xx
    QR_SENZA_RETE,  // nessuna risposta: il codice va ritentato
} esito_qr_t;

static QueueHandle_t s_qr_queue;
static EventGroupHandle_t s_wifi_event_group;
static qr_node_t *s_qr_head;
static qr_node_t *s_qr_tail;

static uint8_t rgb565_to_grayscale(const uint8_t *img);
static void rgb565_to_grayscale_buf(const uint8_t *src, uint8_t *dst, int qr_width, int qr_height);
static void processing_task(void *arg);
static void http_task(void *arg);
static void wifi_init_sta(void);
static esito_qr_t http_post_qrcode(const char *payload);
static void main_task(void *arg);

// Punto d'ingresso dell'applicazione
void app_main(void)
{
    // Inizializza NVS, richiesto dal driver Wi-Fi
    esp_err_t ret = nvs_flash_init();
    if (ret == ESP_ERR_NVS_NO_FREE_PAGES || ret == ESP_ERR_NVS_NEW_VERSION_FOUND) {
        ESP_ERROR_CHECK(nvs_flash_erase());
        ret = nvs_flash_init();
    }
    ESP_ERROR_CHECK(ret);

    xTaskCreatePinnedToCore(&main_task, "main", 8192, NULL, 5, NULL, 0);
}

// Task principale: inizializza SD e camera, avvia le task e gira nel ciclo principale
static void main_task(void *arg)
{
    // Monta la SD, se presente, e carica il classificatore in modalita 1 bit per tenere libero GPIO 4
    sdmmc_host_t host = SDMMC_HOST_DEFAULT();
    host.slot = SDMMC_HOST_SLOT_1;
    host.max_freq_khz = SDMMC_FREQ_HIGHSPEED;
    sdmmc_slot_config_t slot_config = SDMMC_SLOT_CONFIG_DEFAULT();
    slot_config.width = 1;
    slot_config.clk = GPIO_NUM_14;
    slot_config.cmd = GPIO_NUM_15;
    slot_config.d0 = GPIO_NUM_2;
    slot_config.d1 = GPIO_NUM_NC;
    slot_config.d2 = GPIO_NUM_NC;
    slot_config.d3 = GPIO_NUM_NC;
    slot_config.flags = SDMMC_SLOT_FLAG_INTERNAL_PULLUP;

    sdmmc_card_t *card;
    esp_vfs_fat_sdmmc_mount_config_t mount_config = {
        .format_if_mount_failed = false,
        .max_files = 5,
        .allocation_unit_size = 8 * 1024,
    };
    esp_err_t err = esp_vfs_fat_sdmmc_mount(SD_MOUNT_POINT, &host, &slot_config, &mount_config, &card);
    if (err == ESP_OK) {
        classifier_init(SD_MOUNT_POINT "/qrclass.txt");
    } else {
        ESP_LOGW(TAG, "No SD card, no classifiers loaded");
    }

    // Avvia la stazione Wi-Fi, i bit del gruppo eventi segnalano link e IP
    wifi_init_sta();

    // Configura il LED integrato
    gpio_config_t io_conf = {
        .pin_bit_mask = 1ULL << CAM_FLASH_LED_GPIO,
        .mode = GPIO_MODE_OUTPUT,
        .pull_up_en = GPIO_PULLUP_DISABLE,
        .pull_down_en = GPIO_PULLDOWN_DISABLE,
        .intr_type = GPIO_INTR_DISABLE,
    };
    gpio_config(&io_conf);
    gpio_set_level(CAM_FLASH_LED_GPIO, 0);

    // Inizializza la camera con il pinout AI-Thinker ESP32-CAM
    camera_config_t camera_config = {
        .pin_pwdn     = 32,
        .pin_reset    = -1,
        .pin_xclk     = GPIO_NUM_0,
        .pin_sccb_sda = GPIO_NUM_26,
        .pin_sccb_scl = GPIO_NUM_27,
        .pin_d7       = GPIO_NUM_35,
        .pin_d6       = GPIO_NUM_34,
        .pin_d5       = GPIO_NUM_39,
        .pin_d4       = GPIO_NUM_36,
        .pin_d3       = GPIO_NUM_21,
        .pin_d2       = GPIO_NUM_19,
        .pin_d1       = GPIO_NUM_18,
        .pin_d0       = GPIO_NUM_5,
        .pin_vsync    = GPIO_NUM_25,
        .pin_href     = GPIO_NUM_23,
        .pin_pclk     = GPIO_NUM_22,
        .xclk_freq_hz = 20000000,
        .ledc_timer   = LEDC_TIMER_0,
        .ledc_channel = LEDC_CHANNEL_0,
        .pixel_format = PIXFORMAT_RGB565,
        .frame_size   = CAM_FRAME_SIZE,
        .jpeg_quality = 12,
        .fb_count     = 1,
        .fb_location  = CAMERA_FB_IN_PSRAM,
        .grab_mode    = CAMERA_GRAB_WHEN_EMPTY,
    };
    ESP_ERROR_CHECK(esp_camera_init(&camera_config));
    sensor_t *s = esp_camera_sensor_get();
    s->set_vflip(s, 1);
    ESP_LOGI(TAG, "Camera Init done");

    // Coda dei frame della camera verso la task di elaborazione
    QueueHandle_t processing_queue = xQueueCreate(1, sizeof(camera_fb_t *));
    assert(processing_queue);

    // Coda dei QR decodificati dalla task di elaborazione alla task HTTP
    s_qr_queue = xQueueCreate(QR_QUEUE_LEN, sizeof(qr_payload_t));
    assert(s_qr_queue);

    // Task che rileva e decodifica i QR code
    xTaskCreatePinnedToCore(&processing_task, "processing", 35000, processing_queue, 1, NULL, 0);
    ESP_LOGI(TAG, "Processing task started");

    // Task che invia i QR decodificati al server
    xTaskCreatePinnedToCore(&http_task, "http", 8192, s_qr_queue, 2, NULL, 0);
    ESP_LOGI(TAG, "HTTP task started");

    // Task separata che invia RAM, Wi-Fi e IP al backend ogni 10 secondi
    stato_dispositivo_avvia();

    // Ciclo principale: acquisisce i frame dalla camera
    while (1) {
        camera_fb_t *pic = esp_camera_fb_get();
        if (pic == NULL) {
            ESP_LOGE(TAG, "Get frame failed");
            continue;
        }

        // Spegne il LED a lampeggio terminato
        if (esp_timer_get_time() >= s_led_on_until) {
            gpio_set_level(CAM_FLASH_LED_GPIO, 0);
        }

        // Invia il frame alla task di elaborazione, scartandolo se e occupata
        int res = xQueueSend(processing_queue, &pic, pdMS_TO_TICKS(10));
        if (res == pdFAIL) {
            esp_camera_fb_return(pic);
        }
    }
}

// Task di elaborazione: prende un frame dalla coda e ne decodifica i QR code
static void processing_task(void *arg)
{
    struct quirc *qr = quirc_new();
    assert(qr);

    int qr_width = IMG_WIDTH;
    int qr_height = IMG_HEIGHT;
    if (quirc_resize(qr, qr_width, qr_height) < 0) {
        ESP_LOGE(TAG, "Failed to allocate QR buffer");
        return;
    }

    QueueHandle_t input_queue = (QueueHandle_t) arg;
    int frame = 0;
    ESP_LOGI(TAG, "Processing task ready");
    while (true) {
        camera_fb_t *pic;

        // Prende il prossimo frame dalla coda
        int res = xQueueReceive(input_queue, &pic, portMAX_DELAY);
        assert(res == pdPASS);

        // Durante il timer di cooldown i frame vengono scartati senza decodifica
        if (esp_timer_get_time() - s_last_capture_us < CAPTURE_COOLDOWN_US) {
            esp_camera_fb_return(pic);
            continue;
        }

        uint8_t *qr_buf = quirc_begin(qr, NULL, NULL);

// Converte il frame in scala di grigi, quirc accetta solo immagini in grigi
        rgb565_to_grayscale_buf(pic->buf, qr_buf, qr_width, qr_height);

        // Restituisce subito il buffer al driver per evitare errori DMA
        esp_camera_fb_return(pic);

        // Elabora il frame e individua gli angoli del QR code
        quirc_end(qr);
        ++frame;
        int count = quirc_count(qr);
        quirc_decode_error_t err = QUIRC_ERROR_DATA_UNDERFLOW;

        // Tenta la decodifica di ogni QR rilevato
        for (int i = 0; i < count; i++) {
            struct quirc_code code = {};
            struct quirc_data qr_data = {};
            // Estrae i dati binari del QR code
            quirc_extract(qr, i, &code);
            quirc_flip(&code);

            // Decodifica i dati grezzi con correzione degli errori
            err = quirc_decode(&code, &qr_data);
            if (err != 0) {
                continue;
            }
            // Scatto riuscito: riavvia il timer di cooldown
            s_last_capture_us = esp_timer_get_time();

            // Segnala la decodifica con un lampeggio del LED
            gpio_set_level(CAM_FLASH_LED_GPIO, 1);
            s_led_on_until = esp_timer_get_time() + LED_ON_DELAY_US;

            ESP_LOGI(TAG, "QR code: %d bytes: '%s'", qr_data.payload_len, qr_data.payload);

// Classificazione secondo le regole di qrclass.txt
            const char *filename = classifier_get_pic_from_qrcode_data((const char *) qr_data.payload);
            if (filename == NULL) {
                ESP_LOGI(TAG, "Classifier returned NULL");
            } else {
                ESP_LOGI(TAG, "Classified as '%s'", filename);
            }

            // Invia il payload decodificato alla task HTTP tramite la coda
            qr_payload_t item = {};
            strlcpy(item.payload, (const char *) qr_data.payload, sizeof(item.payload));
            if (xQueueSend(s_qr_queue, &item, 0) != pdPASS) {
                ESP_LOGW(TAG, "QR queue full, dropping payload");
            }
        }
        }
    }

// Handler degli eventi Wi-Fi e IP
static void wifi_event_handler(void *arg, esp_event_base_t event_base,
                               int32_t event_id, void *event_data)
{
    if (event_base == WIFI_EVENT && event_id == WIFI_EVENT_STA_START) {
        esp_wifi_connect();
    } else if (event_base == WIFI_EVENT && event_id == WIFI_EVENT_STA_CONNECTED) {
        ESP_LOGI(TAG, "Wi-Fi connected");
        xEventGroupSetBits(s_wifi_event_group, WIFI_CONNECTED_BIT);
    } else if (event_base == WIFI_EVENT && event_id == WIFI_EVENT_STA_DISCONNECTED) {
        ESP_LOGW(TAG, "Wi-Fi disconnected, reconnecting...");
        xEventGroupClearBits(s_wifi_event_group, WIFI_CONNECTED_BIT | WIFI_GOT_IP_BIT);
        esp_wifi_connect();
    } else if (event_base == IP_EVENT && event_id == IP_EVENT_STA_GOT_IP) {
        ip_event_got_ip_t *event = (ip_event_got_ip_t *) event_data;
        ESP_LOGI(TAG, "Got IP: " IPSTR, IP2STR(&event->ip_info.ip));
        xEventGroupSetBits(s_wifi_event_group, WIFI_GOT_IP_BIT);
    }
}

// Inizializza il driver Wi-Fi come stazione e registra i handler degli eventi
static void wifi_init_sta(void)
{
    s_wifi_event_group = xEventGroupCreate();
    assert(s_wifi_event_group);

    ESP_ERROR_CHECK(esp_netif_init());
    ESP_ERROR_CHECK(esp_event_loop_create_default());
    esp_netif_create_default_wifi_sta();

    wifi_init_config_t cfg = WIFI_INIT_CONFIG_DEFAULT();
    ESP_ERROR_CHECK(esp_wifi_init(&cfg));

    esp_event_handler_instance_t instance_any_id;
    esp_event_handler_instance_t instance_got_ip;
    ESP_ERROR_CHECK(esp_event_handler_instance_register(WIFI_EVENT,
                                                        ESP_EVENT_ANY_ID,
                                                        &wifi_event_handler,
                                                        NULL,
                                                        &instance_any_id));
    ESP_ERROR_CHECK(esp_event_handler_instance_register(IP_EVENT,
                                                        IP_EVENT_STA_GOT_IP,
                                                        &wifi_event_handler,
                                                        NULL,
                                                        &instance_got_ip));

    wifi_config_t wifi_config = {
        .sta = {
            .ssid = WIFI_SSID,
            .password = WIFI_PASSWORD,
            .threshold.authmode = WIFI_AUTH_WPA2_PSK,
        },
    };
    ESP_ERROR_CHECK(esp_wifi_set_mode(WIFI_MODE_STA));
    ESP_ERROR_CHECK(esp_wifi_set_config(WIFI_IF_STA, &wifi_config));
    ESP_ERROR_CHECK(esp_wifi_start());

    ESP_LOGI(TAG, "Wi-Fi station started, connecting to '%s'", WIFI_SSID);
}

// Buffer per il corpo della risposta HTTP
#define HTTP_RESP_MAX 2048

typedef struct {
    char *data;
    size_t len;
    size_t cap;
} http_response_t;

// Raccoglie in un unico buffer il corpo della risposta ricevuta a pezzi
static esp_err_t http_response_event_handler(esp_http_client_event_t *evt)
{
    if (evt->event_id == HTTP_EVENT_ON_DATA) {
        http_response_t *resp = (http_response_t *) evt->user_data;
        if (resp != NULL && resp->len + evt->data_len < resp->cap) {
            memcpy(resp->data + resp->len, evt->data, evt->data_len);
            resp->len += evt->data_len;
            resp->data[resp->len] = '\0';
        }
    }
    return ESP_OK;
}

// Invia un payload QR al server con HTTP POST e ne restituisce l'esito
static esito_qr_t http_post_qrcode(const char *payload)
{
    http_response_t response = {
        .data = calloc(HTTP_RESP_MAX, 1),
        .len = 0,
        .cap = HTTP_RESP_MAX,
    };
    if (response.data == NULL) {
        return QR_SENZA_RETE;
    }

    esp_http_client_config_t config = {
        .url = HTTP_POST_URL,
        .method = HTTP_METHOD_POST,
        .timeout_ms = 5000,
        .event_handler = http_response_event_handler,
        .user_data = &response,
    };

    esp_http_client_handle_t client = esp_http_client_init(&config);
    if (client == NULL) {
        free(response.data);
        return QR_SENZA_RETE;
    }

    // Il backend decodifica il corpo come JSON: il payload va tra virgolette con escape
    char *corpo = calloc(strlen(payload) + 3, 1);
    if (corpo == NULL) {
        esp_http_client_cleanup(client);
        return QR_SENZA_RETE;
    }

    size_t n = 0;
    corpo[n++] = '"';
    for (const char *c = payload; *c != '\0'; c++) {
        if (*c == '"' || *c == '\\') {
            corpo[n++] = '\\';
        }
        corpo[n++] = *c;
    }
    corpo[n++] = '"';

    // L'id della camera accompagna il codice, il backend lo usa per riconoscere l'evento
    char header_id[16];
    snprintf(header_id, sizeof(header_id), "%d", stato_dispositivo_id_cam());

    esp_http_client_set_header(client, "Content-Type", "application/json");
    esp_http_client_set_header(client, "id", header_id);
    esp_http_client_set_post_field(client, corpo, n);

    // set_post_field non copia il buffer: corpo si libera solo a richiesta conclusa
    esito_qr_t esito;
    esp_err_t err = esp_http_client_perform(client);

    free(corpo);

    if (err == ESP_OK) {
        int status = esp_http_client_get_status_code(client);
        ESP_LOGI(TAG, "POST status = %d", status);
        ESP_LOGI(TAG, "POST response: '%.*s'", response.len, response.data);
        esito = (status >= 200 && status < 300) ? QR_INVIATO : QR_RIFIUTATO;
    } else {
        ESP_LOGE(TAG, "POST failed: %s", esp_err_to_name(err));
        esito = QR_SENZA_RETE;
    }

    esp_http_client_cleanup(client);
    free(response.data);
    return esito;
}

// Doppio lampeggio del LED per segnalare una richiesta HTTP fallita
static void led_blink_error(void)
{
    // Impedisce al ciclo principale di spegnere il LED a meta sequenza
    s_led_on_until = esp_timer_get_time() + ERROR_BLINK_DURATION_US;
    for (int i = 0; i < 2; i++) {
        gpio_set_level(CAM_FLASH_LED_GPIO, 1);
        vTaskDelay(pdMS_TO_TICKS(ERROR_BLINK_ON_US / 1000));
        gpio_set_level(CAM_FLASH_LED_GPIO, 0);
        vTaskDelay(pdMS_TO_TICKS(ERROR_BLINK_ON_US / 1000));
    }
}

// Verifica se un payload e gia in attesa nella lista FIFO
static bool qr_list_contains(const char *payload)
{
    for (qr_node_t *n = s_qr_head; n != NULL; n = n->next) {
        if (strcmp(n->payload, payload) == 0) {
            return true;
        }
    }
    return false;
}

// Task HTTP: accoda i QR decodificati e invia il piu vecchio appena la rete e disponibile
static void http_task(void *arg)
{
    QueueHandle_t input_queue = (QueueHandle_t) arg;
    qr_payload_t item;

    ESP_LOGI(TAG, "HTTP task ready");
    while (true) {
        // Sposta i QR decodificati dalla coda alla lista FIFO
        if (xQueueReceive(input_queue, &item, pdMS_TO_TICKS(100)) == pdPASS) {
            if (qr_list_contains(item.payload)) {
                // Scarta il codice se un identico e gia in attesa nella lista
                ESP_LOGI(TAG, "Duplicate QR code ignored: '%s'", item.payload);
            } else {
                qr_node_t *node = malloc(sizeof(qr_node_t));
                char *payload = strdup(item.payload);
                if (node != NULL && payload != NULL) {
                    node->payload = payload;
                    node->next = NULL;
                    if (s_qr_tail == NULL) {
                        s_qr_head = node;
                    } else {
                        s_qr_tail->next = node;
                    }
                    s_qr_tail = node;
                    ESP_LOGI(TAG, "QR code queued: '%s'", node->payload);
                } else {
                    free(node);
                    free(payload);
                    ESP_LOGE(TAG, "Out of memory, dropping QR code");
                }
            }
        }

        // Procede solo con i bit Wi-Fi e IP entrambi impostati
        EventBits_t bits = xEventGroupWaitBits(s_wifi_event_group,
                                               WIFI_CONNECTED_BIT | WIFI_GOT_IP_BIT,
                                               pdFALSE, pdTRUE, 0);
        if ((bits & (WIFI_CONNECTED_BIT | WIFI_GOT_IP_BIT)) != (WIFI_CONNECTED_BIT | WIFI_GOT_IP_BIT)) {
            continue;
        }

        // Invia il QR piu vecchio, che lascia la lista anche se rifiutato dal server
        if (s_qr_head != NULL) {
            esito_qr_t esito = http_post_qrcode(s_qr_head->payload);

            if (esito == QR_SENZA_RETE) {
                ESP_LOGW(TAG, "Server unreachable, QR code not sent, retrying...");
                led_blink_error();
                vTaskDelay(pdMS_TO_TICKS(1000));
                continue;
            }

            qr_node_t *sent = s_qr_head;
            s_qr_head = sent->next;
            if (s_qr_head == NULL) {
                s_qr_tail = NULL;
            }
            if (esito == QR_INVIATO) {
                ESP_LOGI(TAG, "Sent and removed QR code: '%s'", sent->payload);
            } else {
                ESP_LOGW(TAG, "Server rejected QR code, discarded: '%s'", sent->payload);
            }
            free(sent->payload);
            free(sent);
        }
    }
}


// Funzioni di conversione da RGB565 a scala di grigi
typedef union {
    uint16_t val;
    struct {
        uint16_t b: 5;
        uint16_t g: 6;
        uint16_t r: 5;
    };
} rgb565_t;

static uint8_t rgb565_to_grayscale(const uint8_t *img)
{
    uint16_t *img_16 = (uint16_t *) img;
    rgb565_t rgb = {.val = __builtin_bswap16(*img_16)};
    uint16_t val = (rgb.r * 8 + rgb.g * 4 + rgb.b * 8) / 3;
    return (uint8_t) MIN(255, val);
}

static void rgb565_to_grayscale_buf(const uint8_t *src, uint8_t *dst, int qr_width, int qr_height)
{
    for (size_t y = 0; y < qr_height; y++) {
        for (size_t x = 0; x < qr_width; x++) {
            dst[y * qr_width + x] = rgb565_to_grayscale(&src[(y * qr_width + x) * 2]);
        }
    }
}
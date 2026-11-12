/*
 * SPDX-FileCopyrightText: 2022-2023 Espressif Systems (Shanghai) CO LTD
 *
 * SPDX-License-Identifier: Apache-2.0
 */

#pragma once


#ifdef __cplusplus
extern "C" {
#endif

/**
 * @brief Classificatore dei QR code, ogni riga di configurazione e "regex stringa"
 */

/** Inizializza il classificatore dal file di configurazione */
void classifier_init(const char *config_path);

/** Restituisce la classificazione del contenuto di un QR code */
const char *classifier_get_pic_from_qrcode_data(const char *text);

#ifdef __cplusplus
}
#endif
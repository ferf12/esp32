idf_component_register(SRCS "qrcode_demo_main.c"
                       "stato_dispositivo.c"
                       INCLUDE_DIRS "."
                       PRIV_REQUIRES esp_timer vfs fatfs sdmmc esp_driver_sdmmc esp_driver_gpio classifier espressif__esp32-camera espressif__quirc esp_wifi esp_netif esp_event nvs_flash esp_http_client esp_system)

target_compile_options(${COMPONENT_LIB} PRIVATE -O3)

idf_component_get_property(quirc_lib_name espressif__quirc COMPONENT_LIB)
target_compile_options(${quirc_lib_name} PRIVATE -O3)
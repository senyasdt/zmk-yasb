#include <zephyr/kernel.h>
#include <zephyr/init.h>
#include <zephyr/sys/byteorder.h>
#include <zephyr/sys/util.h>
#include <zmk/battery.h>
#include <zmk/event_manager.h>
#include <zmk/events/battery_state_changed.h>
#include <zmk/events/layer_state_changed.h>
#include <zmk/keymap.h>
#include <raw_hid/events.h>

#define ZMK_YASB_REPORT_ID 0x7A
#define ZMK_YASB_REPORT_SIZE 32

struct zmk_yasb_layer_status_report {
    uint8_t report_id;
    uint8_t top_layer;
    uint16_t effective_mask;
    uint16_t default_mask;
    uint16_t temp_mask;
    uint8_t battery_percent;
    uint8_t reserved[23];
} __packed;

static uint8_t last_top_layer = 0xff;
static uint32_t last_effective_mask = 0xffffffff;
static uint8_t last_battery_percent = 0xff;
static struct k_work_delayable heartbeat_work;

static uint32_t layer_bit(uint8_t layer) {
    return layer < 32 ? BIT(layer) : 0;
}

static void send_layer_report(bool force) {
    const uint8_t top = zmk_keymap_highest_layer_active();
    const uint32_t default_mask = layer_bit(zmk_keymap_layer_default());
    const uint32_t effective = zmk_keymap_layer_state() | default_mask;
    const uint32_t temp_mask = effective & ~default_mask;
    const uint8_t battery_percent = zmk_battery_state_of_charge();

    if (!force && top == last_top_layer && effective == last_effective_mask &&
        battery_percent == last_battery_percent) {
        return;
    }

    last_top_layer = top;
    last_effective_mask = effective;
    last_battery_percent = battery_percent;

    struct zmk_yasb_layer_status_report report = {
        .report_id = ZMK_YASB_REPORT_ID,
        .top_layer = top,
        .effective_mask = sys_cpu_to_le16((uint16_t)effective),
        .default_mask = sys_cpu_to_le16((uint16_t)default_mask),
        .temp_mask = sys_cpu_to_le16((uint16_t)temp_mask),
        .battery_percent = battery_percent,
    };

    raise_raw_hid_sent_event((struct raw_hid_sent_event){
        .data = (uint8_t *)&report,
        .length = ZMK_YASB_REPORT_SIZE,
    });
}

static int layer_status_listener(const zmk_event_t *eh) {
    ARG_UNUSED(eh);

    send_layer_report(false);
    return ZMK_EV_EVENT_BUBBLE;
}

static void heartbeat_handler(struct k_work *work) {
    ARG_UNUSED(work);

    send_layer_report(true);
    k_work_schedule(&heartbeat_work, K_MSEC(CONFIG_ZMK_YASB_LAYER_STATUS_HEARTBEAT_MS));
}

static int layer_status_init(void) {
    k_work_init_delayable(&heartbeat_work, heartbeat_handler);
    k_work_schedule(&heartbeat_work, K_MSEC(1000));
    return 0;
}

ZMK_LISTENER(zmk_yasb_layer_status, layer_status_listener);
ZMK_SUBSCRIPTION(zmk_yasb_layer_status, zmk_layer_state_changed);
ZMK_SUBSCRIPTION(zmk_yasb_layer_status, zmk_battery_state_changed);
SYS_INIT(layer_status_init, APPLICATION, CONFIG_APPLICATION_INIT_PRIORITY);

if ((NOT CONFIG_ZMK_SPLIT) OR CONFIG_ZMK_SPLIT_ROLE_CENTRAL)
  target_sources_ifdef(CONFIG_ZMK_YASB_LAYER_STATUS app PRIVATE src/layer_status.c)
endif()

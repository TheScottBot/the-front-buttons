# Bricks

Every brick kind and preset, with each field a key can set. For how a key is
written in the file, see [CONFIGURATION.md](CONFIGURATION.md).

## Brick reference

Generated from each kind's and preset's declaration. Do not edit by hand: run
`UPDATE_BRICKS=1 go test ./internal/reference/` after changing a declaration.

### `brightness`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `level` | integer | required | 1 to 100 |  | How bright the deck goes, in percent, when the key is pressed; pressing it again goes back to the usual brightness. The key shows the percentage unless it has a label. |

### `deck_off`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `level` | integer | optional | 1 to 100 |  | How bright the deck is, in percent, while every other key is off; 20 if left out. Pressing this key, or any other, turns the keys back on. |

### `gitkraken_close_tab`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_close_terminal_tab`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_command_palette`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_commit`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_commit_message`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_create_branch`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_diff_tool`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_fetch_all`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_file_history`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_first_commit`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_full_screen`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_keyboard_shortcuts`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_last_commit`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_new_tab`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_new_terminal_tab`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_next_tab`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_next_terminal_tab`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_open_in_file_manager`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_open_in_terminal`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_open_repo`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_previous_tab`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_previous_terminal_tab`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_redo`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_reopen_tab`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_search`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_stage_all`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_stage_and_commit`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_tabs_list`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_toggle_commit_detail`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_toggle_left_panel`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_undo`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_unstage_all`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_zoom_in`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_zoom_out`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `gitkraken_zoom_reset`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_back`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_build`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_comment_line`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_commit`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_debug`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_debug_window`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_delete_line`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_duplicate_line`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_evaluate`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_extract_method`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_extract_variable`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_file_structure`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_find_action`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_find_usages`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_forward`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_git_window`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_go_to_declaration`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_go_to_file`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_go_to_implementation`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_go_to_symbol`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_hide_tool_windows`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_inline`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_move_line_down`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_move_line_up`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_next_error`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_next_window`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_optimise_imports`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_previous_error`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_previous_window`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_problems_window`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_project_window`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_push`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_recent_files`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_refactor_this`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_reformat`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_rename`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_rerun`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_resume`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_rollback`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_run`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_run_choose`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_run_tests`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_run_to_cursor`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_run_window`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_show_intentions`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_step_into`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_step_out`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_step_over`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_stop`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_terminal`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_toggle_breakpoint`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `jetbrains_update_project`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `open_application`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `application` | text | required | 1 to 512 characters |  | The application to launch: a bundle identifier or name on macOS; an executable path or application user model ID on Windows. |
| `use_app_icon` | boolean | optional |  |  | Draw the application's own icon instead of Icon or Icon file, and colour the key to match the icon's edge; a Background you set still wins. Not available on Windows yet, where the key shows the missing icon marker. |
| `app_screen` | boolean | optional |  |  | Also show the application's screen on the deck: a page of its own keys, with the application's icon top left to go back. Saving the key creates the screen if there is none. |
| `show_notifications` | boolean | optional |  |  | Show the badge the application shows on its Dock icon, such as a count of unread messages, top right on the key, within a few seconds of it changing. Not available on Windows yet. |

### `open_editor`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `open_url`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `url` | text | required | 1 to 2048 characters | link_url | The address the key opens: https, or a discord:// link, which opens the Discord app. |

### `page_marker`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `pipeline_status`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `repository` | text | required | 3 to 140 characters |  | The GitHub repository, written owner/name. |
| `workflow` | text | required | 1 to 255 characters | token | The workflow's file name, such as ci.yml, or its numeric ID. |
| `branch` | text | required | 1 to 255 characters | branch | The branch whose runs the key shows. |
| `credential` | text, secret | required | 1 to 59 characters | identifier | The name of the stored GitHub token, a fine grained token with read access to Actions. |
| `interval` | duration | optional | 30s to 1h |  | How often to poll when no run is active; a minute if left out. Every 15 seconds while a run is queued or in progress. GitHub may ask for longer, and is obeyed. |
| `api_base_url` | text | optional | 9 to 2048 characters | https_url | The GitHub API address for GitHub Enterprise; api.github.com if left out. The credential must be stored for this host on the credentials page. |

### `press_key`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `key` | text | required |  | function_key | The key to press: F13 to F24. macOS has only F13 to F20, and asks for Accessibility permission the first time. |

### `press_shortcut`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `key` | text | required |  | shortcut_key | The key to press: a letter, a digit, F1 to F20, or a named key. |
| `control` | boolean | optional |  |  | Hold Control. |
| `alt` | boolean | optional |  |  | Hold Option on macOS, Alt on Windows. |
| `shift` | boolean | optional |  |  | Hold Shift. |
| `command` | boolean | optional |  |  | Hold Command on macOS, the Windows key on Windows. |

### `spotify_album`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `spotify_next`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `spotify_play_pause`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `spotify_previous`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `discord_channel`

A preset that expands into `open_url`. Its action, which the editor shows, is `open_server`.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `server_id` | text | required | 1 to 32 characters | digits | The server ID, from Copy Server ID with Developer Mode on. |
| `channel_id` | text | required | 1 to 32 characters | digits | The channel ID, from Copy Channel ID with Developer Mode on. |

### `dndbeyond_character`

A preset that expands into `open_url`. Its action, which the editor shows, is `open_character`.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `character_id` | text | required | 1 to 32 characters | digits | The number at the end of the character sheet's address. |

### `roll20_campaign`

A preset that expands into `open_url`. Its action, which the editor shows, is `open_campaign`.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `campaign_id` | text | required | 1 to 32 characters | digits | The number at the end of the campaign's address. |

### `teams_channel`

A preset that expands into `open_url`. Its action, which the editor shows, is `open_channel`.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `channel_id` | text | required | 4 to 256 characters |  | The channel ID beginning 19:, from the channel's URL. |
| `channel_name` | text | required | 1 to 256 characters |  | The channel's name. |
| `group_id` | text | required | 1 to 64 characters | identifier | The team's group ID, from the channel's URL. |
| `tenant_id` | text | required | 1 to 64 characters | identifier | The organisation's tenant ID, from the channel's URL. |

### `teams_chat`

A preset that expands into `open_url`. Its action, which the editor shows, is `open_chat`.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `participants` | text | required | 3 to 2048 characters |  | Comma separated user principal names, usually email addresses. |

### `teams_chat_by_id`

A preset that expands into `open_url`. Its action, which the editor shows, is `open_chat`.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `chat_id` | text | required | 4 to 256 characters |  | The chat ID beginning 19:, from a message's Copy link. |

### `teams_private_channel`

A preset that expands into `open_url`. Its action, which the editor shows, is `open_channel`.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `channel_id` | text | required | 4 to 256 characters |  | The channel ID beginning 19:, from the channel's URL. |
| `channel_name` | text | required | 1 to 256 characters |  | The channel's name. |
| `group_id` | text | required | 1 to 64 characters | identifier | The team's group ID, from the channel's URL. |
| `tenant_id` | text | required | 1 to 64 characters | identifier | The organisation's tenant ID, from the channel's URL. |

### `teams_shared_channel`

A preset that expands into `open_url`. Its action, which the editor shows, is `open_channel`.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `channel_id` | text | required | 4 to 256 characters |  | The channel ID beginning 19:, from the channel's URL. |
| `channel_name` | text | required | 1 to 256 characters |  | The channel's name. |
| `group_id` | text | required | 1 to 64 characters | identifier | The team's group ID, from the channel's URL. |
| `tenant_id` | text | required | 1 to 64 characters | identifier | The organisation's tenant ID, from the channel's URL. |


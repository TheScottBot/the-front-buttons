# The editor

The editor is at <http://127.0.0.1:48080/>, listening on this machine only. If
another program already holds port 48080, the daemon uses any free port instead,
logs a warning, and prints the address it chose; `-editor-port` picks another
number, and `-editor-port 0` always lets the system choose. Every key, page and
form in it works from the keyboard.

Every page needs a session, given by a one time login link, so other accounts on
the machine cannot use it. Open the editor with an `open_editor` key on the deck,
or with:

```bash
the-front-buttons -open-editor
```

which asks the running daemon for a link and opens it in your browser. The
session lasts until the browser closes, and survives the daemon restarting. On
macOS the install script puts the binary in
`~/Library/Application Support/the-front-buttons/bin`, so from a terminal:

```bash
"$HOME/Library/Application Support/the-front-buttons/bin/the-front-buttons" -open-editor
```

## Render like deck

The **Render like deck** switch, under **Dark mode**, draws every key and paging
tile on the grid as the deck draws it: its label, icon and colours, a page
marker's page number, and the paging keys' page count and neighbours. Switched
off, the grid names each key and its kind in words. The choice is remembered in
the browser. Keys whose look depends on live state show it as the deck does:
the current album art, a pipeline's status, a Dock badge, a press's error, each
updating as it changes on the deck, even for pages the deck is not showing.
Each key pressed on the deck lights on the grid for two seconds, blinks for one,
then stops. A live key the daemon has
not yet drawn, such as Spotify's album key while nothing plays, shows a
placeholder icon. Switched off, the editor shows none of the deck's live
state.

## Moving keys between pages

Drag a key onto a free tile to move it; hold Command (Ctrl on Windows) as you
drop to copy it instead. Option (Alt) copies too. Drop a key on a page's or app screen's name in the page list
to move it to the same tile there. Hold it over the name for a moment instead
and that page opens in place, the name filling as it counts down, so you can
drop the key on any free tile of it. Let go anywhere else and you stay on that
page with nothing moved. Each key's form does the same from the keyboard.

## Copying keys between pages

To put the same keys on several pages, such as cut, copy, paste and Spotify on
every app screen, choose **Select keys to copy** under the grid. Each key
becomes a check box: tick the keys, tick every page and app screen to copy them
onto, and choose **Copy the ticked keys**. Each key lands on the same tile it is
on, on every page ticked.

Or hold Shift and click a key on the grid to open select mode with that key
ticked. In select mode a click ticks or unticks one key, with Shift held or not,
so you can keep holding Shift as you go. **Done selecting**, or Escape, leaves
select mode without copying.

If any key would land on a key already there, on a tile kept for paging or for
an app screen's icon, or on a page that cannot hold it, such as a JetBrains key
on a regular page, nothing is copied anywhere. Every such tile is named, and
what was ticked stays ticked to change and try again.

**Copy this page's keys** copies a whole page, and a single key's form, or
dragging it with Command (Ctrl on Windows) held, copies one key to any free tile.

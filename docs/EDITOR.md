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
Each key pressed on the deck lights on the grid only while it is held down, and
letting go clears it at once. Held for two seconds, it blinks for one more, then
stops even if still held. A live key the daemon has
not yet drawn, such as Spotify's album key while nothing plays, shows a
placeholder icon. Switched off, the editor shows none of the deck's live
state.

## All pages

**All pages**, in the editor's sections, draws every page's grid small, side
by side, the regular pages first and then the app screens, each named and
linked to its page. Drag a key onto a free tile on any of them to move it
there, hold Command (Ctrl on Windows) as you drop to copy it, drop it on
another key to swap the two, or drop it in the bin to remove it; each updates
the grids in place. Command-Z and Shift-Command-Z undo and redo there too.
Choose a key to open its page with it marked. With Render like deck on, the
grids show the deck's pictures and live state, as the pages screen does.

## Finding a key

**Find a key**, the first drawer under the grid, lists every key on any page
or app screen whose label, kind, the name its kind is known by, or page holds
every word typed, case aside, as its page and tile. The list follows the box as
you type. Choose one, or press Enter for the first, and its page opens with the
key outlined and brought into view. Press / anywhere but a field to open the
drawer with the box ready; Escape clears the box, then closes the drawer.
Without the script, the Find button sends the search.

## Page maintenance and the shortcuts

**Page maintenance**, under the grid, opens in place to the page's own forms,
grouped on a darker ground: **Rename this page**, **Move this page to**,
**Copy this page's keys**, **Delete this page and its keys**, **Add another
screen** on an app screen, and **New page name** to add one. It is closed while
there is a page, and open while there are none.

Press ? for a sheet of every shortcut, or use the button at the foot of the
page; Escape closes it.

## The brick drawer

The **Bricks** tab on the left opens the drawer of bricks to drag onto an empty
tile. Its search box takes focus as it opens and narrows the drawer as you
type, to the bricks whose names hold every word typed; a section's name and a
brick's kind count too, so "spotify" or "goland" finds a whole section and
"session" finds Claude's session keys. An application's own bricks are named by
what they do, such as GoLand: Build or Claude: Back, in that order. Click a
brick's name to open it and see what it is: an application's brick names the
command it was made for and the shortcut it presses on macOS and on Windows;
any other brick, or preset, the fields it asks for. Escape
clears the search, or with the search empty closes the drawer, so two presses
close it from a search, wherever focus is, as after dragging a brick back into
the drawer. The drawer is wide enough for the longest name it can show, and
stays that width however its sections open.

## Moving keys between pages

Drag a key onto a free tile to move it; hold Command (Ctrl on Windows) as you
drop to copy it instead. Drop it on another key, here or on a page a held
drag opened, to swap the two; carried over a key, the two slide into each
other's tiles as they would land, and slide back when it moves on. A key's form offers the same swap from the keyboard, with any key on
any page. Option (Alt) copies too. Drop a key on a page's or app screen's name in the page list
to move it to the same tile there. Hold it over the name for a moment instead
and that page opens in place, the name filling as it counts down, so you can
drop the key on any free tile of it. Let go anywhere else and you stay on that
page with nothing moved. Each key's form does the same from the keyboard.

## Undo and redo

**Undo** and **Redo**, beside the Render like deck switch, undo and redo any
change made in the editor: a key saved, moved, copied, swapped or removed, a
page added, renamed, reordered or deleted. Command-Z undoes and Shift-Command-Z
redoes, Ctrl on Windows, unless you are typing. The last 20 changes are kept,
until the daemon restarts, and a new change after an undo means what was
undone can no longer be redone. If the file has been changed by hand since,
undo and redo refuse rather than throw that change away.

## Moving around the grid

Command-C, Ctrl-C on Windows, copies the key on the tile focused, or in select
mode the ticked keys; Command-X cuts them instead. Command-V pastes them onto
the page open, each on the tile it came from, or with a tile focused, shifted
so their top left lands there. A paste that would land on anything in the way
pastes nothing and names the tiles; pasting a cut moves the keys, once. What
is copied lasts while the tab is open and never reaches the system's
clipboard, and text being typed or selected keeps its usual Command-C and V.

A change made from the grid that stays on the page on show, a drop, a swap, a
paste, Delete or Undo, updates only the tiles it changed, without loading the
page again, so nothing else on the grid flashes. Every other change, from the
key dialog, or going into select mode and out of it, loads the page and keeps
the grid where it was in the window.

With a tile focused, the arrow keys move between tiles, passing over any that
cannot take focus, and Enter opens one. Delete, or Backspace, removes the key
on a tile, and the page offers Undo to put it back. In select mode Space ticks
the key focused.

## Renaming a page

Double-click the name of the page you are on in the page list to rename it in
place: Enter saves, and Escape or clicking away leaves it as it was. A single
click on that name does nothing, as the page is already open. **Rename this
page**, in Page maintenance, does the same from the keyboard.

## Reordering pages

Drag a page's name along the page list and it moves as you go, the other
names sliding aside to show where it will land; let go and the page takes that
place. Anywhere past the last name counts as the last place, and letting go
outside the list puts the names back. Regular pages and app screens are
listed apart, and each moves only within its own list: the regular pages' order
is the order the deck pages through, and an application's screens are paged
between in theirs. **Move this page to**, in Page maintenance, does the same from
the keyboard. The deck stays on the page it was showing, wherever that went.

A page marker's **Page number** box, ticked for a new marker, shows the page's
number on the key, Page 2 and the icon of a 2, so the marker follows its page
as pages move. Untick it to give the marker a label and icon of your own,
which a marker made before the box existed has, and keeps, until it is ticked.

## Copying, moving and removing keys

To put the same keys on several pages, such as cut, copy, paste and Spotify on
every app screen, open **Select keys** under the grid and tick **Edit mode**.
The grid switches in place, without loading the page, and each key becomes a
check box: tick the keys, tick every page and app
screen to put them on, and choose **Copy the ticked keys**. Each key lands on
the same tile it is on, on every page ticked. **Move the ticked keys** does the
same and takes them off the page they came from. **Remove the ticked keys**
needs no page ticked; the page then offers Undo, which puts every one back.
**Colour the ticked keys** gives each the background colour chosen above it,
and **Clear their colour** takes it off again; neither needs a page ticked.

In select mode, drag a key to carry every ticked key with it, ticking it if
it was not. Drop it on a free tile and each key shifts as far as the one
carried; on a page's name, or held there to open that page, each goes to its
own tile there; in the bin, all are removed. Hold Command (Ctrl on Windows) to
copy instead of move. The drag carries a picture of every ticked key, laid out
as on the grid, and over a tile each key's ghost shows where it would land:
outlined where it fits, crossed on a tile in the way, and all crossed if one
would fall off the deck. Only a drop where every key fits is taken.

Or hold Shift and click a key on the grid to open select mode with that key
ticked. In select mode a click ticks or unticks one key, with Shift held or not,
so you can keep holding Shift as you go. Unticking edit mode, or Escape, leaves
select mode without changing anything, and copying, moving, removing or
colouring leaves it once done, all without loading the page.

If any key would land on a key already there, on a tile kept for paging or for
an app screen's icon, or on a page that cannot hold it, such as a JetBrains key
on a regular page, nothing is copied anywhere. Every such tile is named, and
what was ticked stays ticked to change and try again. A move refused this way
moves nothing, so every key stays where it was.

**Copy this page's keys** copies a whole page, and a single key's form, or
dragging it with Command (Ctrl on Windows) held, copies one key to any free tile.

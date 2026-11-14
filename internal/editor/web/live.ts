// The deck as it runs, on a page drawn like the deck: the server's event
// stream names the tile of each key pressed and let go of on the deck, lit
// here while it is held, and given a flash after a hold of half a second; and of
// each key whose picture changed, such as new album art, which is fetched
// again. A reload fetches every key's picture again. The stream carries tiles
// alone; the pictures come through the same guarded previews as every other.
// A page drawn in words never opens it.

const heldClass = "held";
const pressedClass = "pressed";
// holdMilliseconds is how long a key is held down before it counts as a
// hold, as the daemon counts it (holdThreshold in hold.go).
const holdMilliseconds = 500;
// flashMilliseconds is how long a held key shows it once let go of: lit for
// two seconds, then blinking for one, as the flash animation in editor.css
// runs.
const flashMilliseconds = 3000;
// downSince is when each tile held down was pressed; flashTimers ends each
// tile's flash, so a key pressed again starts afresh.
const downSince = new Map<string, number>();
const flashTimers = new Map<HTMLTableCellElement, number>();

function shownTile(tile: string): HTMLTableCellElement | null {
  if (!/^\d+\.\d+\.\d+$/.test(tile)) {
    return null;
  }

  return document.querySelector<HTMLTableCellElement>(`table.grid:not(.grid-left-behind) td[data-tile="${tile}"]`);
}

// refetch asks again for a live picture, past anything the browser kept.
function refetch(picture: HTMLImageElement): void {
  const address = new URL(picture.src, window.location.href);
  if (address.searchParams.get("live") !== "1") {
    return;
  }

  address.searchParams.set("drawn", String(Date.now()));
  picture.src = address.pathname + address.search;
}

// press lights a tile for as long as its key is held down.
function press(tile: string): void {
  downSince.set(tile, Date.now());

  const cell = shownTile(tile);
  if (cell !== null) {
    window.clearTimeout(flashTimers.get(cell));
    cell.classList.remove(pressedClass);
    cell.classList.add(heldClass);
  }
}

// release ends a quick press's light at once, and gives a hold its flash.
function release(tile: string): void {
  const since = downSince.get(tile);
  downSince.delete(tile);

  const cell = shownTile(tile);
  if (cell === null) {
    return;
  }

  cell.classList.remove(heldClass);

  if (since !== undefined && Date.now() - since >= holdMilliseconds) {
    flash(cell);
  }
}

function flash(cell: HTMLTableCellElement): void {
  window.clearTimeout(flashTimers.get(cell));
  cell.classList.remove(pressedClass);
  // Reading the layout restarts the flash for a key pressed again.
  void cell.offsetWidth;
  cell.classList.add(pressedClass);
  flashTimers.set(cell, window.setTimeout(() => {
    cell.classList.remove(pressedClass);
    flashTimers.delete(cell);
  }, flashMilliseconds));
}

export function enableLive(page: Document): void {
  if (page.documentElement.dataset["render"] !== "deck" || page.querySelector("table.grid") === null) {
    return;
  }

  const stream = new EventSource("/events");

  stream.addEventListener("press", (event: MessageEvent<string>) => press(event.data));
  stream.addEventListener("release", (event: MessageEvent<string>) => release(event.data));

  stream.addEventListener("frame", (event: MessageEvent<string>) => {
    const picture = shownTile(event.data)?.querySelector<HTMLImageElement>("img.deck-face");
    if (picture !== null && picture !== undefined) {
      refetch(picture);
    }
  });

  stream.addEventListener("reload", () => {
    for (const picture of page.querySelectorAll<HTMLImageElement>("table.grid img.deck-face")) {
      refetch(picture);
    }
  });

  window.addEventListener("pagehide", () => stream.close());
}

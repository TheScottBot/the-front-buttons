// The deck as it runs, on a page drawn like the deck: the server's event
// stream names the tile of each press on the deck, which flashes here, and of
// each key whose picture changed, such as new album art, which is fetched
// again. A reload fetches every key's picture again. The stream carries tiles
// alone; the pictures come through the same guarded previews as every other.
// A page drawn in words never opens it.

const pressedClass = "pressed";
// flashMilliseconds is how long a pressed tile shows it: lit for two seconds,
// then blinking for one, as the flash animation in editor.css runs.
const flashMilliseconds = 3000;
// flashTimers ends each tile's flash, so a key pressed again starts afresh.
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

  stream.addEventListener("press", (event: MessageEvent<string>) => {
    const cell = shownTile(event.data);
    if (cell !== null) {
      flash(cell);
    }
  });

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

// The deck as it runs, on a page drawn like the deck: the server's event
// stream names the tile of each key pressed and let go of on the deck, lit
// here only while it is held; held for two seconds it blinks for one, then
// stops though still held; and of
// each key whose picture changed, such as new album art, which is fetched
// again. A reload fetches every key's picture again. The stream carries tiles
// alone; the pictures come through the same guarded previews as every other.
// A page drawn in words never opens it.

const heldClass = "held";
const holdingClass = "holding";
// A tile is lit while its key is held down. Held for litMilliseconds, it
// blinks for the rest of shownMilliseconds, as the animation in editor.css
// runs, and then stops though the key is still held, so a release the page
// never hears cannot leave it lit.
const litMilliseconds = 2000;
const shownMilliseconds = 3000;
// holdTimers starts each held tile's blink and ends it.
const holdTimers = new Map<string, number[]>();

function clearTimers(tile: string): void {
  for (const timer of holdTimers.get(tile) ?? []) {
    window.clearTimeout(timer);
  }

  holdTimers.delete(tile);
}

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

// press lights a tile while its key is held down, blinks it once held for
// two seconds, and stops after three.
function press(tile: string): void {
  const cell = shownTile(tile);
  if (cell === null) {
    return;
  }

  clearTimers(tile);
  cell.classList.remove(holdingClass);
  cell.classList.add(heldClass);
  holdTimers.set(tile, [
    window.setTimeout(() => cell.classList.add(holdingClass), litMilliseconds),
    window.setTimeout(() => release(tile), shownMilliseconds),
  ]);
}

// release clears the tile at once, however long it was held.
function release(tile: string): void {
  clearTimers(tile);
  shownTile(tile)?.classList.remove(heldClass, holdingClass);
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

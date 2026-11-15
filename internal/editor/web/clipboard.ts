// Command-C, Command-X and Command-V for keys, Ctrl on Windows. Copying takes
// the ticked keys in select mode, or else the key on the tile focused; cutting
// does the same, for a paste that moves them. Pasting puts them on the page
// open, each on the tile it came from, or, with a tile focused, shifted so the
// top left of the keys lands there. A paste is the same POST select mode's
// Copy and Move buttons send, through the same checks, which refuse it whole,
// naming the tiles in the way, if any key cannot land.
//
// What is copied is kept in this tab's session storage, which may be refused,
// so it lasts as pages are opened, and never touches the system's clipboard.
// Text being typed or selected keeps the usual Command-C and Command-V.

import { announce, send } from "./drag";

const clipboardKey = "front-buttons-keys";
const tileAttribute = "data-tile";

interface Copied {
  readonly page: string;
  readonly tiles: readonly string[];
  readonly cut: boolean;
}

function save(copied: Copied): void {
  try {
    window.sessionStorage.setItem(clipboardKey, JSON.stringify(copied));
  } catch {
    announce("Error: this browser keeps nothing for the page, so keys cannot be copied this way.");
  }
}

function load(): Copied | null {
  try {
    const stored = window.sessionStorage.getItem(clipboardKey);
    const copied = stored === null ? null : (JSON.parse(stored) as Partial<Copied>);
    if (typeof copied?.page !== "string" || !Array.isArray(copied.tiles) || typeof copied.cut !== "boolean") {
      return null;
    }

    return { page: copied.page, tiles: copied.tiles.filter((tile) => typeof tile === "string"), cut: copied.cut };
  } catch {
    return null;
  }
}

function forget(): void {
  try {
    window.sessionStorage.removeItem(clipboardKey);
  } catch {
    // Nothing was kept.
  }
}

function shownGrid(): HTMLTableElement | null {
  return document.querySelector<HTMLTableElement>("table.grid:not(.grid-left-behind)");
}

// focusedTile is the grid tile holding focus, as "page.row.column".
function focusedTile(): string[] | null {
  const cell = document.activeElement?.closest(`table.grid td[${tileAttribute}]`);
  const parts = (cell?.getAttribute(tileAttribute) ?? "").split(".");

  return parts.length === 3 ? parts : null;
}

// toCopy is what Command-C takes: the ticked keys, or the key focused.
function toCopy(cut: boolean): Copied | null {
  const ticked = [...document.querySelectorAll<HTMLInputElement>('.select-key input[name="tile"]:checked')];
  const page = shownGrid()?.querySelector(`td[${tileAttribute}]`)?.getAttribute(tileAttribute)?.split(".")[0];

  if (ticked.length > 0 && page !== undefined) {
    return { page, tiles: ticked.map((box) => box.value), cut };
  }

  const focused = focusedTile();
  const cell = document.activeElement?.closest(`td[${tileAttribute}]`);
  if (focused === null || cell?.getAttribute("data-state") !== "key" || cell.querySelector(".select-key") !== null) {
    return null;
  }

  return { page: focused[0], tiles: [`${focused[1]}.${focused[2]}`], cut };
}

function keysWord(count: number): string {
  return count === 1 ? "1 key" : `${count} keys`;
}

function copy(cut: boolean): boolean {
  const copied = toCopy(cut);
  if (copied === null) {
    return false;
  }

  save(copied);
  announce(`${cut ? "Cut" : "Copied"} ${keysWord(copied.tiles.length)}: open a page and press Command-V, ` +
    "Ctrl-V on Windows, to paste on the same tiles, or focus a tile with the arrow keys to paste from there.");

  return true;
}

async function paste(): Promise<void> {
  const copied = load();
  const page = shownGrid()?.querySelector(`td[${tileAttribute}]`)?.getAttribute(tileAttribute)?.split(".")[0];
  if (copied === null || copied.tiles.length === 0 || page === undefined) {
    announce("Error: no keys are copied; press Command-C, Ctrl-C on Windows, on a key or the ticked keys first.");

    return;
  }

  // With a tile focused, the top left of the copied keys lands on it.
  let rowShift = 0;
  let columnShift = 0;
  const focused = focusedTile();
  if (focused !== null) {
    const places = copied.tiles.map((tile) => tile.split(".").map(Number));
    rowShift = Number(focused[1]) - Math.min(...places.map((place) => place[0]));
    columnShift = Number(focused[2]) - Math.min(...places.map((place) => place[1]));
  }

  if (page === copied.page && rowShift === 0 && columnShift === 0) {
    announce("Error: those keys are already on these tiles; open another page, or focus a tile with the arrow " +
      "keys to paste from there.");

    return;
  }

  // A cut is used once: the keys it named have moved.
  if (copied.cut) {
    forget();
  }

  const accepted = await send(copied.cut ? "/keys/move" : "/keys/copy", {
    page: copied.page, tile: copied.tiles, to: page, showing: page,
    row_offset: String(rowShift), column_offset: String(columnShift),
  });
  if (!accepted && copied.cut) {
    save(copied);
  }
}

export function enableClipboard(): void {
  document.addEventListener("keydown", (event: KeyboardEvent) => {
    const key = event.key.toLowerCase();
    const typing = event.target instanceof HTMLElement &&
      event.target.matches('input:not([type="checkbox"]), textarea, select, [contenteditable]');
    const selecting = (window.getSelection()?.toString() ?? "") !== "";

    if (!(event.metaKey || event.ctrlKey) || event.altKey || event.shiftKey || typing || selecting ||
      document.querySelector("dialog[open]") !== null || !["c", "x", "v"].includes(key)) {
      return;
    }

    if (key === "v") {
      event.preventDefault();
      void paste();

      return;
    }

    if (copy(key === "x")) {
      event.preventDefault();
    }
  });
}

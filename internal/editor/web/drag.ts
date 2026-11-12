// Drag and drop for the editor's grid of keys: drag a key onto a free tile to
// move it, or hold Option (Alt) while dropping to copy it. It only adds to the
// page: every move it makes is the same POST /key/move the move form on each
// key's page sends, through the same guard and the same checks, and that form
// stays the way to do it from the keyboard (engineering datum 0.16).
//
// The server marks each grid cell with data-tile ("page.row.column") and
// data-state ("key", "empty" or "reserved"); only a key can be picked up and
// only an empty tile accepts a drop, so a paging or occupied tile never does.

const tileAttribute = "data-tile";
const stateAttribute = "data-state";
const draggingClass = "dragging";
const dropTargetClass = "drop-target";
const tileType = "application/x-front-buttons-tile";

type TileState = "key" | "empty" | "reserved";

interface Tile {
  readonly page: string;
  readonly row: string;
  readonly column: string;
}

function parseTile(value: string | null): Tile | null {
  const parts = (value ?? "").split(".");
  if (parts.length !== 3 || parts.some((part) => !/^\d+$/.test(part))) {
    return null;
  }

  return { page: parts[0], row: parts[1], column: parts[2] };
}

function cellOf(target: EventTarget | null): HTMLTableCellElement | null {
  return target instanceof Element ? target.closest<HTMLTableCellElement>(`td[${tileAttribute}]`) : null;
}

function stateOf(cell: HTMLTableCellElement): TileState | null {
  const state = cell.getAttribute(stateAttribute);

  return state === "key" || state === "empty" || state === "reserved" ? state : null;
}

// announce writes to the page's polite live region, which every editor page
// has, so a refused drop is read out as well as shown.
function announce(message: string): void {
  const region = document.querySelector<HTMLElement>(".announcement");
  if (region !== null) {
    region.textContent = message;
  }
}

// refusalFrom reads the reason from a refused move's page: the server writes
// it in the errors alert, starting "Error: ".
function refusalFrom(page: string): string {
  const parsed = new DOMParser().parseFromString(page, "text/html");
  const alert = parsed.querySelector(".errors");

  return alert?.textContent?.trim() || "Error: the key could not be moved.";
}

// send posts one change the way the editor's own forms do, with the page's
// token, and follows the redirect a change the server accepted answers with.
async function send(path: string, fields: Record<string, string>): Promise<void> {
  const token = document.querySelector<HTMLInputElement>('input[name="token"]')?.value;
  if (token === undefined) {
    announce("Error: this page has no editor token; reload it and try again.");

    return;
  }

  const body = new URLSearchParams({ token, ...fields });

  try {
    const response = await fetch(path, { method: "POST", body, credentials: "same-origin" });
    if (response.ok && response.redirected) {
      window.location.assign(response.url);

      return;
    }

    announce(refusalFrom(await response.text()));
  } catch {
    announce("Error: the editor could not be reached; is the daemon running?");
  }
}

function relocate(source: Tile, destination: string, copying: boolean): Promise<void> {
  return send("/key/move", {
    page: source.page,
    row: source.row,
    column: source.column,
    to: destination,
    action: copying ? "copy" : "move",
  });
}

// remove is the same POST /key/delete as a key's Remove button; the page it
// returns to offers Undo.
function remove(source: Tile): Promise<void> {
  return send("/key/delete", { page: source.page, row: source.row, column: source.column });
}

// The bin the author asked for: shown only while a key is carried, in the
// window's bottom right corner, away from the tiles, or its top right corner
// when the grid reaches the bottom one, so it never sits over a key.
const binArmedClass = "armed";
const binTopClass = "key-bin-top";

function overlaps(first: DOMRect, second: DOMRect): boolean {
  return first.left < second.right && second.left < first.right && first.top < second.bottom && second.top < first.bottom;
}

function showBin(bin: HTMLElement, grid: HTMLElement): void {
  bin.classList.remove(binTopClass, binArmedClass);
  bin.hidden = false;

  if (overlaps(bin.getBoundingClientRect(), grid.getBoundingClientRect())) {
    bin.classList.add(binTopClass);
  }
}

function hideBin(bin: HTMLElement): void {
  bin.hidden = true;
  bin.classList.remove(binArmedClass);
}

function clearDropTargets(grid: HTMLElement): void {
  for (const cell of grid.querySelectorAll(`.${dropTargetClass}`)) {
    cell.classList.remove(dropTargetClass);
  }
}

export function enableDrag(grid: HTMLTableElement): void {
  let source: Tile | null = null;
  const bin = document.getElementById("key-bin");

  if (bin !== null) {
    bin.addEventListener("dragover", (event: DragEvent) => {
      if (source === null || event.dataTransfer === null) {
        return;
      }

      event.preventDefault();
      event.dataTransfer.dropEffect = "move";
      bin.classList.add(binArmedClass);
    });

    bin.addEventListener("dragleave", () => bin.classList.remove(binArmedClass));

    bin.addEventListener("drop", (event: DragEvent) => {
      if (source === null) {
        return;
      }

      event.preventDefault();
      const carried = source;
      hideBin(bin);
      void remove(carried);
    });
  }

  grid.addEventListener("dragstart", (event: DragEvent) => {
    const cell = cellOf(event.target);
    if (cell === null || stateOf(cell) !== "key" || event.dataTransfer === null) {
      return;
    }

    source = parseTile(cell.getAttribute(tileAttribute));
    event.dataTransfer.effectAllowed = "copyMove";
    event.dataTransfer.setData(tileType, cell.getAttribute(tileAttribute) ?? "");
    cell.classList.add(draggingClass);

    if (bin !== null) {
      showBin(bin, grid);
    }
  });

  grid.addEventListener("dragend", (event: DragEvent) => {
    cellOf(event.target)?.classList.remove(draggingClass);
    clearDropTargets(grid);
    source = null;

    if (bin !== null) {
      hideBin(bin);
    }
  });

  grid.addEventListener("dragover", (event: DragEvent) => {
    const cell = cellOf(event.target);
    if (source === null || cell === null || stateOf(cell) !== "empty" || event.dataTransfer === null) {
      return;
    }

    event.preventDefault();
    event.dataTransfer.dropEffect = event.altKey ? "copy" : "move";
    cell.classList.add(dropTargetClass);
  });

  grid.addEventListener("dragleave", (event: DragEvent) => {
    cellOf(event.target)?.classList.remove(dropTargetClass);
  });

  grid.addEventListener("drop", (event: DragEvent) => {
    const cell = cellOf(event.target);
    const destination = cell?.getAttribute(tileAttribute) ?? null;
    if (source === null || cell === null || stateOf(cell) !== "empty" || parseTile(destination) === null) {
      return;
    }

    event.preventDefault();
    clearDropTargets(grid);
    void relocate(source, destination as string, event.altKey);
  });

  document.getElementById("drag-help")?.removeAttribute("hidden");
}

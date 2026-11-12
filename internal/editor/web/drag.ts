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

async function relocate(source: Tile, destination: string, copying: boolean): Promise<void> {
  const token = document.querySelector<HTMLInputElement>('input[name="token"]')?.value;
  if (token === undefined) {
    announce("Error: this page has no editor token; reload it and try again.");

    return;
  }

  const body = new URLSearchParams({
    token,
    page: source.page,
    row: source.row,
    column: source.column,
    to: destination,
    action: copying ? "copy" : "move",
  });

  try {
    const response = await fetch("/key/move", { method: "POST", body, credentials: "same-origin" });
    if (response.ok && response.redirected) {
      window.location.assign(response.url);

      return;
    }

    announce(refusalFrom(await response.text()));
  } catch {
    announce("Error: the editor could not be reached; is the daemon running?");
  }
}

function clearDropTargets(grid: HTMLElement): void {
  for (const cell of grid.querySelectorAll(`.${dropTargetClass}`)) {
    cell.classList.remove(dropTargetClass);
  }
}

export function enableDrag(grid: HTMLTableElement): void {
  let source: Tile | null = null;

  grid.addEventListener("dragstart", (event: DragEvent) => {
    const cell = cellOf(event.target);
    if (cell === null || stateOf(cell) !== "key" || event.dataTransfer === null) {
      return;
    }

    source = parseTile(cell.getAttribute(tileAttribute));
    event.dataTransfer.effectAllowed = "copyMove";
    event.dataTransfer.setData(tileType, cell.getAttribute(tileAttribute) ?? "");
    cell.classList.add(draggingClass);
  });

  grid.addEventListener("dragend", (event: DragEvent) => {
    cellOf(event.target)?.classList.remove(draggingClass);
    clearDropTargets(grid);
    source = null;
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

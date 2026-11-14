// Drag and drop for the editor's grid of keys: drag a key onto a free tile to
// move it, or hold Command, Option or Ctrl while dropping to copy it. It only adds to the
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

// Page names as drop targets: a key dropped on a page's name in the page
// list moves, or held to copy copies, to the same tile on that page; held over
// the name a moment, that page's grid is fetched and shown in place of this
// one, without leaving the page, so the drag carries on and the key can be
// dropped on any free tile there. The grid left behind is kept, out of sight,
// as it holds the key being dragged, which the browser needs to finish the
// drag. Every drop is still the same POST /key/move.
const pageAttribute = "data-page";
const pageDropClass = "page-drop";
const pageArmingClass = "page-arming";
const leftBehindClass = "grid-left-behind";
// holdMilliseconds is how long a key is held over a page's name to open that
// page; the arming animation in editor.css runs as long.
const holdMilliseconds = 800;

// copying reports a drop that copies rather than moves: Command, Option or
// Ctrl held, or the browser itself choosing a copy. Some browsers, on macOS
// especially, report no keys held during a drag, but still choose a copy for
// the platform's own copy key, so that choice counts too.
function copying(event: DragEvent): boolean {
  return event.metaKey || event.altKey || event.ctrlKey || event.dataTransfer?.dropEffect === "copy";
}

// showCopying shows a copy as the drag's cursor when a copy key is reported,
// and otherwise leaves the browser's own choice, never forcing a move over it.
function showCopying(event: DragEvent): void {
  if (event.dataTransfer !== null && (event.metaKey || event.altKey || event.ctrlKey)) {
    event.dataTransfer.dropEffect = "copy";
  }
}

function pageLinkOf(target: EventTarget | null): HTMLAnchorElement | null {
  const link = target instanceof Element ? target.closest<HTMLAnchorElement>(`a[${pageAttribute}]`) : null;

  return link !== null && /^\d+$/.test(link.getAttribute(pageAttribute) ?? "") ? link : null;
}

// shownGrid is the grid on show: the page's own, or the one a held drag
// opened.
function shownGrid(): HTMLTableElement | null {
  return document.querySelector<HTMLTableElement>(`table.grid:not(.${leftBehindClass})`);
}

// openPageDuringDrag shows another page's grid and page list in place of
// these, keeping the grid left behind out of sight.
async function openPageDuringDrag(page: string): Promise<boolean> {
  try {
    const response = await fetch(`/page?page=${page}`, { credentials: "same-origin" });
    if (!response.ok) {
      return false;
    }

    const parsed = new DOMParser().parseFromString(await response.text(), "text/html");
    const incoming = parsed.querySelector("table.grid");
    const incomingPages = parsed.querySelector('nav[aria-label="Pages"]');
    const current = shownGrid();
    if (incoming === null || current === null) {
      return false;
    }

    current.classList.add(leftBehindClass);
    current.setAttribute("aria-hidden", "true");
    current.after(document.importNode(incoming, true));

    if (incomingPages !== null) {
      document.querySelector('nav[aria-label="Pages"]')?.replaceWith(document.importNode(incomingPages, true));
    }

    window.history.replaceState(null, "", `/page?page=${page}`);
    announce("Opened that page: drop the key on a free tile.");

    return true;
  } catch {
    announce("Error: that page could not be opened; is the daemon running?");

    return false;
  }
}

export function enableDrag(grid: HTMLTableElement): void {
  // Listening on the page's main part, not the grid, keeps the drag working
  // on a grid a held drag brings in.
  const root: HTMLElement = grid.closest("main") ?? document.body;
  let source: Tile | null = null;
  let shownPage = "";
  let opened = false;
  let dropped = false;
  let armed: HTMLAnchorElement | null = null;
  let armTimer: number | undefined;
  const bin = document.getElementById("key-bin");

  const disarm = (): void => {
    window.clearTimeout(armTimer);
    armed?.classList.remove(pageDropClass, pageArmingClass);
    armed = null;
  };

  const arm = (link: HTMLAnchorElement): void => {
    if (armed === link) {
      return;
    }

    disarm();
    armed = link;
    link.classList.add(pageDropClass, pageArmingClass);

    const page = link.getAttribute(pageAttribute) ?? "";
    armTimer = window.setTimeout(() => {
      disarm();

      // Taken as shown at once, so the name, still under the key while the
      // page loads, does not arm again.
      const before = shownPage;
      shownPage = page;
      void openPageDuringDrag(page).then((done) => {
        opened ||= done;
        if (!done) {
          shownPage = before;
        }
      });
    }, holdMilliseconds);
  };

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
      dropped = true;
      const carried = source;
      hideBin(bin);
      void remove(carried);
    });
  }

  root.addEventListener("dragstart", (event: DragEvent) => {
    const cell = cellOf(event.target);
    if (cell === null || stateOf(cell) !== "key" || event.dataTransfer === null) {
      return;
    }

    source = parseTile(cell.getAttribute(tileAttribute));
    shownPage = source?.page ?? "";
    opened = false;
    dropped = false;
    event.dataTransfer.effectAllowed = "copyMove";
    event.dataTransfer.setData(tileType, cell.getAttribute(tileAttribute) ?? "");
    cell.classList.add(draggingClass);

    const shown = shownGrid();
    if (bin !== null && shown !== null) {
      showBin(bin, shown);
    }
  });

  root.addEventListener("dragend", (event: DragEvent) => {
    cellOf(event.target)?.classList.remove(draggingClass);
    clearDropTargets(root);
    disarm();
    source = null;

    if (bin !== null) {
      hideBin(bin);
    }

    // A drag that opened another page and was let go of nowhere stays on
    // that page, drawn afresh so everything on it works.
    if (opened && !dropped) {
      window.location.assign(`/page?page=${shownPage}`);
    }
  });

  root.addEventListener("dragover", (event: DragEvent) => {
    if (source === null || event.dataTransfer === null) {
      return;
    }

    const link = pageLinkOf(event.target);
    if (link !== null && link.getAttribute(pageAttribute) !== shownPage) {
      event.preventDefault();
      showCopying(event);
      arm(link);

      return;
    }

    const cell = cellOf(event.target);
    if (cell === null || stateOf(cell) !== "empty") {
      return;
    }

    event.preventDefault();
    showCopying(event);
    cell.classList.add(dropTargetClass);
  });

  root.addEventListener("dragleave", (event: DragEvent) => {
    cellOf(event.target)?.classList.remove(dropTargetClass);

    if (armed !== null && pageLinkOf(event.target) === armed && !armed.contains(event.relatedTarget as Node | null)) {
      disarm();
    }
  });

  root.addEventListener("drop", (event: DragEvent) => {
    if (source === null) {
      return;
    }

    // A quick drop on a page's name: the same tile on that page.
    const link = pageLinkOf(event.target);
    if (link !== null && link.getAttribute(pageAttribute) !== shownPage) {
      event.preventDefault();
      disarm();
      dropped = true;
      void relocate(source, `${link.getAttribute(pageAttribute) ?? ""}.${source.row}.${source.column}`, copying(event));

      return;
    }

    const cell = cellOf(event.target);
    const destination = cell?.getAttribute(tileAttribute) ?? null;
    if (cell === null || stateOf(cell) !== "empty" || parseTile(destination) === null) {
      return;
    }

    event.preventDefault();
    clearDropTargets(root);
    dropped = true;
    void relocate(source, destination as string, copying(event));
  });

  document.getElementById("drag-help")?.removeAttribute("hidden");
}

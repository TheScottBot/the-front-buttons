// Drag and drop for the editor's grid of keys: drag a key onto a free tile to
// move it, or hold Command, Option or Ctrl while dropping to copy it; drop it
// on another key to swap the two. It only adds to the
// page: every move it makes is the same POST /key/move the move form on each
// key's page sends, through the same guard and the same checks, and that form
// stays the way to do it from the keyboard (engineering datum 0.16).
//
// The server marks each grid cell with data-tile ("page.row.column") and
// data-state ("key", "empty" or "reserved"); only a key can be picked up and
// only an empty tile accepts a drop, so a paging or occupied tile never does.

import { updateInPlace } from "./inplace";

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
export function announce(message: string): void {
  const region = document.querySelector<HTMLElement>(".announcement");
  if (region !== null) {
    region.textContent = message;
  }
}

// refusalFrom reads the reason from a refused change's page: the server
// writes it starting "Error: ", in the key form's errors alert, or in the
// page's list of problems.
function refusalFrom(page: string): string {
  const parsed = new DOMParser().parseFromString(page, "text/html");
  const alert = parsed.querySelector(".errors");
  const problems = [...parsed.querySelectorAll(".problems li")].map((item) => item.textContent?.trim() ?? "");

  return alert?.textContent?.trim() || problems.join(" ") || "Error: the change could not be made.";
}

// send posts one change the way the editor's own forms do, with the page's
// token, and follows the redirect a change the server accepted answers with,
// reporting whether it was accepted.
export async function send(path: string, fields: Record<string, string | readonly string[]>): Promise<boolean> {
  const token = document.querySelector<HTMLInputElement>('input[name="token"]')?.value;
  if (token === undefined) {
    announce("Error: this page has no editor token; reload it and try again.");

    return false;
  }

  const body = new URLSearchParams({ token });
  for (const [name, value] of Object.entries(fields)) {
    for (const each of typeof value === "string" ? [value] : value) {
      body.append(name, each);
    }
  }

  try {
    const response = await fetch(path, { method: "POST", body, credentials: "same-origin" });
    if (response.ok && response.redirected) {
      if (!(await updateInPlace(await response.text(), response.url))) {
        window.location.assign(response.url);
      }

      return true;
    }

    announce(refusalFrom(await response.text()));
  } catch {
    announce("Error: the editor could not be reached; is the daemon running?");
  }

  return false;
}

// swap trades the places of two keys, the same POST /key/move as the Swap
// keys button on a key's page.
function swap(source: Tile, destination: string): Promise<boolean> {
  return send("/key/move", {
    page: source.page, row: source.row, column: source.column, to: destination, action: "swap",
  });
}

// swapClass marks the key a carried key would trade places with, when the
// key carried is on a page no longer on show and so cannot slide.
const swapClass = "swap-target";

// A swap shown as it would happen: carried over another key, the two keys
// slide into each other's tiles, as the page list's names do, and slide back
// when the key moves on. Copies of the two keys do the sliding, over the real
// ones, hidden meanwhile, so the drag itself is never disturbed.
const swapGhostClass = "swap-ghost";
const swapHiddenClass = "swap-hidden";
const swapSlideMilliseconds = 150;

interface SwapPreview {
  readonly target: HTMLTableCellElement;
  readonly source: HTMLTableCellElement;
  readonly ghosts: readonly HTMLElement[];
}

function reducedMotion(): boolean {
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

// slideGhost draws a copy of one tile's key in another tile, sliding there
// from where the key is.
function slideGhost(from: HTMLTableCellElement, to: HTMLTableCellElement): HTMLElement | null {
  const key = from.querySelector("a");
  if (key === null) {
    return null;
  }

  const ghost = key.cloneNode(true) as HTMLElement;
  ghost.removeAttribute("href");
  ghost.removeAttribute("draggable");
  ghost.setAttribute("aria-hidden", "true");
  ghost.tabIndex = -1;
  ghost.classList.add(swapGhostClass);
  to.append(ghost);

  if (!reducedMotion()) {
    const start = from.getBoundingClientRect();
    const end = to.getBoundingClientRect();
    ghost.animate(
      [{ transform: `translate(${start.left - end.left}px, ${start.top - end.top}px)` }, { transform: "none" }],
      { duration: swapSlideMilliseconds, easing: "ease-out" },
    );
  }

  return ghost;
}

function showSwap(source: HTMLTableCellElement, target: HTMLTableCellElement): SwapPreview {
  const ghosts = [slideGhost(target, source), slideGhost(source, target)]
    .filter((ghost): ghost is HTMLElement => ghost !== null);
  source.classList.add(swapHiddenClass);
  target.classList.add(swapHiddenClass);

  return { target, source, ghosts };
}

// hideSwap slides the two keys back to their own tiles.
function hideSwap(preview: SwapPreview): void {
  const back = (ghost: HTMLElement, home: HTMLTableCellElement): Promise<void> => {
    if (reducedMotion()) {
      return Promise.resolve();
    }

    const here = ghost.getBoundingClientRect();
    const there = home.getBoundingClientRect();

    return ghost.animate(
      [{ transform: "none" }, { transform: `translate(${there.left - here.left}px, ${there.top - here.top}px)` }],
      { duration: swapSlideMilliseconds, easing: "ease-in" },
    ).finished.then(() => undefined, () => undefined);
  };

  const homes = [preview.target, preview.source];
  void Promise.all(preview.ghosts.map((ghost, index) => back(ghost, homes[index]))).then(() => {
    for (const ghost of preview.ghosts) {
      ghost.remove();
    }

    for (const cell of homes) {
      if (cell.querySelector(`.${swapGhostClass}`) === null) {
        cell.classList.remove(swapHiddenClass);
      }
    }
  });
}

function relocate(source: Tile, destination: string, copying: boolean): Promise<boolean> {
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
function remove(source: Tile): Promise<boolean> {
  return send("/key/delete", { page: source.page, row: source.row, column: source.column });
}

// Ticked keys carried together: in select mode a key picked up carries every
// ticked key, ticking itself if it was not. Dropped on a free tile, each key
// shifts as far as the one carried; on a page's name each goes to its own
// tile there; in the bin all are removed. Each is the same POST the select
// mode form sends, with the shift added, through the same checks, which
// refuse the whole drop, moving nothing, if any key cannot land.
interface Group {
  readonly source: Tile;
  readonly tiles: readonly string[];
}

function groupOf(cell: HTMLTableCellElement, source: Tile | null): Group | null {
  const box = cell.querySelector<HTMLInputElement>('.select-key input[type="checkbox"]');
  if (box === null || source === null) {
    return null;
  }

  box.checked = true;
  const tiles = [...document.querySelectorAll<HTMLInputElement>('.select-key input[name="tile"]:checked')]
    .map((ticked) => ticked.value);

  return { source, tiles };
}

// carriesTile reports a tile one of the carried keys is on, which another
// may land on as they all move.
function carriesTile(group: Group | null, tile: Tile | null): boolean {
  return group !== null && tile !== null && tile.page === group.source.page &&
    group.tiles.includes(`${tile.row}.${tile.column}`);
}

function relocateGroup(group: Group, destination: Tile, copying: boolean): Promise<boolean> {
  return send(copying ? "/keys/copy" : "/keys/move", {
    page: group.source.page,
    tile: group.tiles,
    to: destination.page,
    row_offset: String(Number(destination.row) - Number(group.source.row)),
    column_offset: String(Number(destination.column) - Number(group.source.column)),
  });
}

function removeGroup(group: Group): Promise<boolean> {
  return send("/keys/remove", { page: group.source.page, tile: group.tiles });
}

// The ghost of carried keys: over a tile, each carried key's picture is
// drawn where it would land, so where they will go, and whether they fit, can
// be seen before letting go. A tile that cannot take one, a key staying put or
// one kept for paging, is crossed; a key that would fall off the deck crosses
// them all. Only a drop where every key fits is taken; the server checks again.
const ghostClass = "ghost";
const ghostFitsClass = "ghost-fits";
const ghostBlockedClass = "ghost-blocked";

// ghostPicture copies a ticked key's picture, without its check box, to draw
// where it would land.
function ghostPicture(cell: HTMLTableCellElement): HTMLElement {
  const ghost = document.createElement("span");
  ghost.className = ghostClass;
  ghost.setAttribute("aria-hidden", "true");

  for (const node of cell.querySelector(".select-key label")?.childNodes ?? []) {
    ghost.append(node.cloneNode(true));
  }

  return ghost;
}

function clearGhosts(root: HTMLElement): void {
  for (const ghost of root.querySelectorAll(`.${ghostClass}`)) {
    ghost.remove();
  }

  for (const cell of root.querySelectorAll(`.${ghostFitsClass}, .${ghostBlockedClass}`)) {
    cell.classList.remove(ghostFitsClass, ghostBlockedClass);
  }
}

// showGhosts draws the carried keys landing as the key carried lands on a
// tile, and reports whether every one fits.
function showGhosts(group: Group, grid: HTMLElement, destination: Tile, pictures: ReadonlyMap<string, HTMLElement>): boolean {
  const rowShift = Number(destination.row) - Number(group.source.row);
  const columnShift = Number(destination.column) - Number(group.source.column);
  const landed: HTMLTableCellElement[] = [];
  let fits = true;
  let offDeck = false;

  for (const tile of group.tiles) {
    const [row, column] = tile.split(".").map(Number);
    const landing = `${destination.page}.${row + rowShift}.${column + columnShift}`;
    const cell = grid.querySelector<HTMLTableCellElement>(`td[${tileAttribute}="${landing}"]`);
    if (cell === null) {
      fits = false;
      offDeck = true;
      continue;
    }

    const free = stateOf(cell) === "empty" || carriesTile(group, parseTile(landing));
    fits &&= free;
    cell.classList.add(free ? ghostFitsClass : ghostBlockedClass);
    landed.push(cell);

    const picture = pictures.get(tile);
    if (picture !== undefined) {
      cell.append(picture.cloneNode(true));
    }
  }

  if (offDeck) {
    for (const cell of landed) {
      cell.classList.replace(ghostFitsClass, ghostBlockedClass);
    }
  }

  return fits;
}

// groupDragImage is the picture the browser drags for carried keys: every
// ticked key, laid out as on the grid, held where the key carried was picked
// up. It is drawn out of sight for the browser to copy, then taken away.
function groupDragImage(group: Group, event: DragEvent): void {
  const cells = group.tiles
    .map((tile) => document.querySelector<HTMLTableCellElement>(`td[${tileAttribute}="${group.source.page}.${tile}"]`))
    .filter((cell): cell is HTMLTableCellElement => cell !== null);
  if (cells.length === 0 || event.dataTransfer === null) {
    return;
  }

  const boxes = cells.map((cell) => cell.getBoundingClientRect());
  const left = Math.min(...boxes.map((box) => box.left));
  const top = Math.min(...boxes.map((box) => box.top));
  const image = document.createElement("div");
  image.className = "grid group-drag-image";
  image.style.width = `${Math.max(...boxes.map((box) => box.right)) - left}px`;
  image.style.height = `${Math.max(...boxes.map((box) => box.bottom)) - top}px`;

  cells.forEach((cell, index) => {
    const picture = cell.querySelector(".select-key")?.cloneNode(true);
    if (!(picture instanceof HTMLElement)) {
      return;
    }

    picture.querySelector("input")?.remove();
    picture.querySelector("label")?.removeAttribute("for");
    picture.style.left = `${boxes[index].left - left}px`;
    picture.style.top = `${boxes[index].top - top}px`;
    picture.style.width = `${boxes[index].width}px`;
    picture.style.height = `${boxes[index].height}px`;
    image.append(picture);
  });

  // Held where the pointer took the key carried, as it is on the grid.
  document.body.append(image);
  event.dataTransfer.setDragImage(image, event.clientX - left, event.clientY - top);
  window.setTimeout(() => image.remove(), 0);
}

function markGroup(group: Group, root: HTMLElement, marked: boolean): void {
  for (const tile of group.tiles) {
    root.querySelector(`td[${tileAttribute}="${group.source.page}.${tile}"]`)?.classList.toggle(draggingClass, marked);
  }
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
  for (const cell of grid.querySelectorAll(`.${dropTargetClass}, .${swapClass}`)) {
    cell.classList.remove(dropTargetClass, swapClass);
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
  let group: Group | null = null;
  let pictures = new Map<string, HTMLElement>();
  let ghostedAt = "";
  let ghostFits = false;
  let swapping: SwapPreview | null = null;

  const stopSwapping = (): void => {
    if (swapping !== null) {
      hideSwap(swapping);
      swapping = null;
    }
  };
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

  // A free tile takes a drop, and with keys carried together, a tile one of
  // them is leaving.
  const accepts = (cell: HTMLTableCellElement): boolean =>
    stateOf(cell) === "empty" || carriesTile(group, parseTile(cell.getAttribute(tileAttribute)));

  // A single key carried over another key, not itself, would swap with it.
  const swapsWith = (cell: HTMLTableCellElement): boolean => {
    const tile = parseTile(cell.getAttribute(tileAttribute));

    return group === null && source !== null && stateOf(cell) === "key" && tile !== null &&
      !(tile.page === source.page && tile.row === source.row && tile.column === source.column);
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
      void (group === null ? remove(carried) : removeGroup(group));
    });
  }

  root.addEventListener("dragstart", (event: DragEvent) => {
    const cell = cellOf(event.target);
    if (cell === null || stateOf(cell) !== "key" || event.dataTransfer === null) {
      return;
    }

    source = parseTile(cell.getAttribute(tileAttribute));
    group = groupOf(cell, source);
    shownPage = source?.page ?? "";
    opened = false;
    dropped = false;
    event.dataTransfer.effectAllowed = "copyMove";
    event.dataTransfer.setData(tileType, cell.getAttribute(tileAttribute) ?? "");
    cell.classList.add(draggingClass);

    // A ticked key's own layer is an empty label over it, so the drag shows
    // every ticked key's picture instead.
    if (group !== null) {
      const carried = group;
      groupDragImage(carried, event);
      markGroup(carried, root, true);
      pictures = new Map(carried.tiles.flatMap((tile) => {
        const ticked = root.querySelector<HTMLTableCellElement>(`td[${tileAttribute}="${carried.source.page}.${tile}"]`);

        return ticked === null ? [] : [[tile, ghostPicture(ticked)] as const];
      }));
    }

    const shown = shownGrid();
    if (bin !== null && shown !== null) {
      showBin(bin, shown);
    }
  });

  root.addEventListener("dragend", (event: DragEvent) => {
    cellOf(event.target)?.classList.remove(draggingClass);
    if (group !== null) {
      markGroup(group, root, false);
    }

    clearDropTargets(root);
    clearGhosts(root);
    ghostedAt = "";
    stopSwapping();
    disarm();
    source = null;
    group = null;

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
      clearGhosts(root);
      ghostedAt = "";
      event.preventDefault();
      showCopying(event);
      arm(link);

      return;
    }

    const cell = cellOf(event.target);

    if (group !== null) {
      const tile = cell?.getAttribute(tileAttribute) ?? "";
      const destination = parseTile(tile);
      const shown = cell?.closest<HTMLElement>("table.grid") ?? null;
      if (tile !== ghostedAt) {
        clearGhosts(root);
        ghostedAt = tile;
        ghostFits = destination !== null && shown !== null && showGhosts(group, shown, destination, pictures);
      }

      if (ghostFits) {
        event.preventDefault();
        showCopying(event);
      }

      return;
    }

    if (cell !== null && swapsWith(cell)) {
      event.preventDefault();

      // The key carried slides only where its own tile is on show.
      const home = source === null ? null :
        shownGrid()?.querySelector<HTMLTableCellElement>(
          `td[${tileAttribute}="${source.page}.${source.row}.${source.column}"]`) ?? null;
      if (home === null) {
        cell.classList.add(swapClass);
      } else if (swapping?.target !== cell) {
        stopSwapping();
        swapping = showSwap(home, cell);
      }

      return;
    }

    stopSwapping();

    if (cell === null || !accepts(cell)) {
      return;
    }

    event.preventDefault();
    showCopying(event);
    cell.classList.add(dropTargetClass);
  });

  root.addEventListener("dragleave", (event: DragEvent) => {
    cellOf(event.target)?.classList.remove(dropTargetClass, swapClass);

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
      const sameTile = `${link.getAttribute(pageAttribute) ?? ""}.${source.row}.${source.column}`;
      const target = parseTile(sameTile);
      void (group !== null && target !== null ?
        relocateGroup(group, target, copying(event)) :
        relocate(source, sameTile, copying(event)));

      return;
    }

    const cell = cellOf(event.target);
    const destination = cell?.getAttribute(tileAttribute) ?? null;
    const target = parseTile(destination);

    // A key dropped on another trades places with it; a copy needs a free
    // tile, so holding the copy key there does nothing.
    if (cell !== null && destination !== null && swapsWith(cell)) {
      event.preventDefault();
      clearDropTargets(root);
      dropped = true;

      // The keys stay where they slid while the swap is saved, and slide
      // back only if it is refused.
      const shown = swapping;
      swapping = null;

      if (copying(event)) {
        announce("Error: a copy needs a free tile; let go of Command to swap the two keys instead.");
        if (shown !== null) {
          hideSwap(shown);
        }
      } else {
        void swap(source, destination).then((accepted) => {
          if (!accepted && shown !== null) {
            hideSwap(shown);
          }
        });
      }

      return;
    }

    if (cell === null || target === null || (group === null ? !accepts(cell) : !ghostFits)) {
      return;
    }

    event.preventDefault();
    clearDropTargets(root);
    clearGhosts(root);
    dropped = true;
    void (group !== null ? relocateGroup(group, target, copying(event)) : relocate(source, destination as string, copying(event)));
  });

  document.getElementById("drag-help")?.removeAttribute("hidden");
}

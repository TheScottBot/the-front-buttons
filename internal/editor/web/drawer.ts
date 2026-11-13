// The brick drawer the author asked for (evaluation, deck testing): a tab in
// the left gutter that opens on a click into drawers of bricks to drag, 1x1
// for every brick and preset, 1x2 and 1x3 for keys placed down a column. A
// brick dropped on an empty tile opens that tile's key editor with the brick
// chosen; a group needs as many empty tiles down the column, numbered as it is
// carried over them, and opens its own form there. Nothing is saved until that
// form is. While anything is carried the grid is marked "placing", and the
// stylesheet crosses each key and paging tile to say it cannot be replaced
// this way. The grid's empty tiles stay the way to add a key from the keyboard.

import { openKeyDialog } from "./dialog";

const brickType = "application/x-front-buttons-brick";
const placingClass = "placing";
const carryingClass = "carrying";
const stackClass = "stack-target";

// Carried is what is being dragged: one brick or preset, or a group of keys
// that many tiles high.
interface Carried {
  readonly type: string | null;
  readonly group: string | null;
  readonly size: number;
}

interface Place {
  readonly page: string;
  readonly row: number;
  readonly column: string;
}

function placeOf(cell: HTMLTableCellElement): Place | null {
  const parts = (cell.dataset.tile ?? "").split(".");
  if (parts.length !== 3) {
    return null;
  }

  return { page: parts[0], row: Number(parts[1]), column: parts[2] };
}

function cellOf(target: EventTarget | null): HTMLTableCellElement | null {
  return target instanceof Element ? target.closest<HTMLTableCellElement>("td[data-tile]") : null;
}

// stackFrom is the tiles a carried brick or group would fill from a cell down
// its column, or null unless every one is there and empty.
function stackFrom(grid: HTMLTableElement, cell: HTMLTableCellElement, size: number): HTMLTableCellElement[] | null {
  const place = placeOf(cell);
  if (place === null) {
    return null;
  }

  const stack: HTMLTableCellElement[] = [];

  for (let offset = 0; offset < size; offset++) {
    const tile = `${place.page}.${place.row + offset}.${place.column}`;
    const below = grid.querySelector<HTMLTableCellElement>(`td[data-tile="${tile}"]`);
    if (below === null || below.dataset.state !== "empty") {
      return null;
    }

    stack.push(below);
  }

  return stack;
}

function setOpen(toggle: HTMLButtonElement, panel: HTMLElement, open: boolean): void {
  toggle.setAttribute("aria-expanded", String(open));
  panel.hidden = !open;
}

function carriedFrom(item: HTMLElement): Carried | null {
  if (item.dataset.type !== undefined) {
    return { type: item.dataset.type, group: null, size: 1 };
  }

  const size = Number(item.dataset.size);
  if (item.dataset.group !== undefined && Number.isInteger(size) && size > 1) {
    return { type: null, group: item.dataset.group, size };
  }

  return null;
}

export function enableDrawer(grid: HTMLTableElement): void {
  const drawer = document.getElementById("brick-drawer");
  const toggle = drawer?.querySelector<HTMLButtonElement>(".drawer-toggle");
  const panel = document.getElementById("brick-list");
  if (drawer === null || toggle === null || toggle === undefined || panel === null) {
    return;
  }

  let carried: Carried | null = null;
  let marked: HTMLTableCellElement[] = [];

  const unmark = (): void => {
    for (const cell of marked) {
      cell.classList.remove(stackClass);
      delete cell.dataset.slot;
    }

    marked = [];
  };

  const mark = (stack: HTMLTableCellElement[]): void => {
    unmark();
    stack.forEach((cell, index) => {
      cell.classList.add(stackClass);
      if (stack.length > 1) {
        cell.dataset.slot = String(index + 1);
      }
    });
    marked = stack;
  };

  toggle.addEventListener("click", () => setOpen(toggle, panel, panel.hidden));

  panel.addEventListener("dragstart", (event: DragEvent) => {
    const item = event.target instanceof Element ? event.target.closest<HTMLElement>("li[draggable]") : null;
    const picked = item === null ? null : carriedFrom(item);
    if (picked === null || event.dataTransfer === null) {
      return;
    }

    carried = picked;
    event.dataTransfer.effectAllowed = "copy";
    event.dataTransfer.setData(brickType, picked.type ?? picked.group ?? "");
    grid.classList.add(placingClass);

    // Once the browser has its picture of the brick, the panel fades and lets
    // the drag through to any tile it covers on a narrow window.
    window.setTimeout(() => drawer.classList.add(carryingClass), 0);
  });

  panel.addEventListener("dragend", () => {
    carried = null;
    unmark();
    grid.classList.remove(placingClass);
    drawer.classList.remove(carryingClass);
  });

  grid.addEventListener("dragover", (event: DragEvent) => {
    const cell = cellOf(event.target);
    if (carried === null || cell === null || event.dataTransfer === null) {
      return;
    }

    const stack = stackFrom(grid, cell, carried.size);
    if (stack === null) {
      unmark();

      return;
    }

    event.preventDefault();
    event.dataTransfer.dropEffect = "copy";
    mark(stack);
  });

  grid.addEventListener("dragleave", (event: DragEvent) => {
    if (carried !== null && !(event.relatedTarget instanceof Node && grid.contains(event.relatedTarget))) {
      unmark();
    }
  });

  grid.addEventListener("drop", (event: DragEvent) => {
    const cell = cellOf(event.target);
    const link = cell?.querySelector<HTMLAnchorElement>('a[href^="/key?"]');
    const place = cell === null ? null : placeOf(cell);
    if (carried === null || cell === null || link === null || link === undefined || place === null ||
      stackFrom(grid, cell, carried.size) === null) {
      return;
    }

    event.preventDefault();
    unmark();
    setOpen(toggle, panel, false);

    if (carried.group !== null) {
      const query = new URLSearchParams({
        page: place.page, row: String(place.row), column: place.column, size: carried.group,
      });
      openKeyDialog(`/group?${query.toString()}`, link);

      return;
    }

    const address = new URL(link.href);
    address.searchParams.set("type", carried.type ?? "");
    openKeyDialog(address.pathname + address.search, link);
  });

  drawer.hidden = false;
}

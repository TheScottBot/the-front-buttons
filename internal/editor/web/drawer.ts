// The brick drawer the author asked for (evaluation, deck testing): a tab in
// the left gutter that opens on a click into drawers of bricks to drag, 1x1
// for every brick and preset, and bigger ones, width by height, for keys placed
// at once. A brick dropped on an empty tile opens that tile's key editor with
// the brick chosen; a group needs as many empty tiles from its top left,
// numbered as it is carried over them, and opens its own form there. Nothing is saved until that
// form is. While anything is carried the grid is marked "placing", and the
// stylesheet crosses each key and paging tile to say it cannot be replaced
// this way. The grid's empty tiles stay the way to add a key from the keyboard.

import { openKeyDialog } from "./dialog";

const brickType = "application/x-front-buttons-brick";
const placingClass = "placing";
const carryingClass = "carrying";
const stackClass = "stack-target";

// Carried is what is being dragged: one brick or preset, or a group of keys
// that many tiles wide and high.
interface Carried {
  readonly type: string | null;
  readonly group: string | null;
  readonly width: number;
  readonly height: number;
}

interface Place {
  readonly page: string;
  readonly row: number;
  readonly column: number;
}

function placeOf(cell: HTMLTableCellElement): Place | null {
  const parts = (cell.dataset.tile ?? "").split(".");
  if (parts.length !== 3) {
    return null;
  }

  return { page: parts[0], row: Number(parts[1]), column: Number(parts[2]) };
}

function cellOf(target: EventTarget | null): HTMLTableCellElement | null {
  return target instanceof Element ? target.closest<HTMLTableCellElement>("td[data-tile]") : null;
}

// stackFrom is the tiles a carried brick or group would fill from a cell as
// its top left, left to right then top to bottom, or null unless every one is
// there and empty.
function stackFrom(grid: HTMLTableElement, cell: HTMLTableCellElement, carried: Carried): HTMLTableCellElement[] | null {
  const place = placeOf(cell);
  if (place === null) {
    return null;
  }

  const stack: HTMLTableCellElement[] = [];

  for (let down = 0; down < carried.height; down++) {
    for (let across = 0; across < carried.width; across++) {
      const tile = `${place.page}.${place.row + down}.${place.column + across}`;
      const covered = grid.querySelector<HTMLTableCellElement>(`td[data-tile="${tile}"]`);
      if (covered === null || covered.dataset.state !== "empty") {
        return null;
      }

      stack.push(covered);
    }
  }

  return stack;
}

// enableSearch narrows the drawer to the bricks whose names hold every word
// typed, a section's name counting for each brick in it, so "spotify" or
// "goland" finds a whole section. Sections with a match open and the rest
// hide; clearing the box, or Escape, puts every section back as it was. It
// returns the box, to take focus when the drawer opens.
function enableSearch(panel: HTMLElement): HTMLInputElement | null {
  const box = panel.querySelector<HTMLInputElement>("input.drawer-search");
  const empty = panel.querySelector<HTMLElement>(".drawer-empty");
  const sections = Array.from(panel.querySelectorAll<HTMLDetailsElement>("details.drawer-section"));
  if (box === null) {
    return null;
  }

  const wasOpen = new Map(sections.map((section) => [section, section.open]));

  const narrow = (): void => {
    const words = box.value.toLowerCase().split(/\s+/).filter((word) => word !== "");
    let matches = 0;

    for (const section of sections) {
      const sectionName = section.querySelector("summary")?.textContent?.toLowerCase() ?? "";
      let shown = 0;

      for (const item of section.querySelectorAll<HTMLElement>("li")) {
        const haystack = `${sectionName} ${item.textContent?.toLowerCase() ?? ""}`;
        const match = words.every((word) => haystack.includes(word));

        item.hidden = !match;
        shown += match ? 1 : 0;
      }

      section.hidden = shown === 0;
      section.open = words.length === 0 ? (wasOpen.get(section) ?? false) : shown > 0;
      matches += shown;
    }

    if (empty !== null) {
      empty.hidden = matches > 0;
    }
  };

  box.addEventListener("input", narrow);

  box.addEventListener("keydown", (event: KeyboardEvent) => {
    if (event.key === "Escape" && box.value !== "") {
      event.preventDefault();
      event.stopPropagation();
      box.value = "";
      narrow();
    }
  });

  return box;
}

function setOpen(toggle: HTMLButtonElement, panel: HTMLElement, open: boolean): void {
  toggle.setAttribute("aria-expanded", String(open));
  panel.hidden = !open;
}

function carriedFrom(item: HTMLElement): Carried | null {
  if (item.dataset.type !== undefined) {
    return { type: item.dataset.type, group: null, width: 1, height: 1 };
  }

  const width = Number(item.dataset.width);
  const height = Number(item.dataset.height);
  if (item.dataset.group !== undefined && Number.isInteger(width) && Number.isInteger(height) &&
    width > 0 && height > 0) {
    return { type: null, group: item.dataset.group, width, height };
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

  const search = enableSearch(panel);

  toggle.addEventListener("click", () => {
    setOpen(toggle, panel, panel.hidden);

    if (!panel.hidden) {
      search?.focus();
    }
  });

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

    const stack = stackFrom(grid, cell, carried);
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
      stackFrom(grid, cell, carried) === null) {
      return;
    }

    event.preventDefault();
    unmark();
    setOpen(toggle, panel, false);

    if (carried.group !== null) {
      const query = new URLSearchParams({
        page: place.page, row: String(place.row), column: String(place.column), size: carried.group,
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

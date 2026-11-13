// The brick drawer the author asked for (evaluation, deck testing): a tab in
// the left gutter that opens on a click and offers every brick and preset to
// drag. Dropped on an empty tile, a brick opens that tile's key editor with
// the brick chosen, as choosing the tile and the type would; nothing is saved
// until that form is. Only an empty tile takes one: while a brick is carried
// the grid is marked "placing", and the stylesheet crosses each key and paging
// tile to say it cannot be replaced this way. The grid's empty tiles stay the
// way to add a key from the keyboard.

import { openKeyDialog } from "./dialog";

const brickType = "application/x-front-buttons-brick";
const placingClass = "placing";
const carryingClass = "carrying";
const dropTargetClass = "drop-target";

function emptyCellOf(target: EventTarget | null): HTMLTableCellElement | null {
  const cell = target instanceof Element ? target.closest<HTMLTableCellElement>("td[data-tile]") : null;

  return cell?.dataset.state === "empty" ? cell : null;
}

function setOpen(toggle: HTMLButtonElement, panel: HTMLElement, open: boolean): void {
  toggle.setAttribute("aria-expanded", String(open));
  panel.hidden = !open;
}

export function enableDrawer(grid: HTMLTableElement): void {
  const drawer = document.getElementById("brick-drawer");
  const toggle = drawer?.querySelector<HTMLButtonElement>(".drawer-toggle");
  const panel = document.getElementById("brick-list");
  if (drawer === null || toggle === null || toggle === undefined || panel === null) {
    return;
  }

  let carried: string | null = null;

  toggle.addEventListener("click", () => setOpen(toggle, panel, panel.hidden));

  panel.addEventListener("dragstart", (event: DragEvent) => {
    const item = event.target instanceof Element ? event.target.closest<HTMLElement>("li[data-type]") : null;
    if (item === null || event.dataTransfer === null) {
      return;
    }

    carried = item.dataset.type ?? null;
    event.dataTransfer.effectAllowed = "copy";
    event.dataTransfer.setData(brickType, carried ?? "");
    grid.classList.add(placingClass);

    // Once the browser has its picture of the brick, the panel fades and lets
    // the drag through to any tile it covers on a narrow window.
    window.setTimeout(() => drawer.classList.add(carryingClass), 0);
  });

  panel.addEventListener("dragend", () => {
    carried = null;
    grid.classList.remove(placingClass);
    drawer.classList.remove(carryingClass);

    for (const cell of grid.querySelectorAll(`.${dropTargetClass}`)) {
      cell.classList.remove(dropTargetClass);
    }
  });

  grid.addEventListener("dragover", (event: DragEvent) => {
    const cell = emptyCellOf(event.target);
    if (carried === null || cell === null || event.dataTransfer === null) {
      return;
    }

    event.preventDefault();
    event.dataTransfer.dropEffect = "copy";
    cell.classList.add(dropTargetClass);
  });

  grid.addEventListener("dragleave", (event: DragEvent) => {
    if (carried !== null) {
      emptyCellOf(event.target)?.classList.remove(dropTargetClass);
    }
  });

  grid.addEventListener("drop", (event: DragEvent) => {
    const cell = emptyCellOf(event.target);
    const link = cell?.querySelector<HTMLAnchorElement>('a[href^="/key?"]');
    if (carried === null || cell === null || link === null || link === undefined) {
      return;
    }

    event.preventDefault();
    cell.classList.remove(dropTargetClass);

    const address = new URL(link.href);
    address.searchParams.set("type", carried);
    setOpen(toggle, panel, false);
    openKeyDialog(address.pathname + address.search, link);
  });

  drawer.hidden = false;
}

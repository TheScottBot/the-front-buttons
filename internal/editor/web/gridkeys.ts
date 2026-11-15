// Arrow keys on the grid: with focus on a tile, the arrows move it to the
// next tile in that direction that can take focus, passing over those that
// cannot, such as a paging tile with nowhere to go. Enter opens a tile, as it
// always has. Delete or Backspace on a key removes it, the same POST
// /key/delete as its Remove button, and the page it returns to offers Undo.
// In select mode each key's check box takes the focus, and Space ticks it.

import { send } from "./drag";

const tileAttribute = "data-tile";
const focusableSelector = 'a[href], input[type="checkbox"]';

const steps: Readonly<Record<string, readonly [number, number]>> = {
  ArrowUp: [-1, 0],
  ArrowDown: [1, 0],
  ArrowLeft: [0, -1],
  ArrowRight: [0, 1],
};

function cellOf(target: EventTarget | null): HTMLTableCellElement | null {
  return target instanceof Element ? target.closest<HTMLTableCellElement>(`td[${tileAttribute}]`) : null;
}

// focusable is what in a tile takes focus, if anything does.
function focusable(cell: HTMLTableCellElement | undefined): HTMLElement | null {
  return cell?.querySelector<HTMLElement>(focusableSelector) ?? null;
}

// nextTile is the first tile from a cell in a direction that can take focus.
function nextTile(grid: HTMLTableElement, cell: HTMLTableCellElement, step: readonly [number, number]): HTMLElement | null {
  const row = cell.parentElement instanceof HTMLTableRowElement ? cell.parentElement.sectionRowIndex : -1;
  const rows = grid.tBodies[0]?.rows ?? grid.rows;

  for (let rowIndex = row + step[0], column = cell.cellIndex + step[1];
    rowIndex >= 0 && rowIndex < rows.length && column >= 0 && column < rows[rowIndex].cells.length;
    rowIndex += step[0], column += step[1]) {
    const target = focusable(rows[rowIndex].cells[column]);
    if (target !== null) {
      return target;
    }
  }

  return null;
}

// removeKey sends a key's removal, for a tile holding a key outside select
// mode.
function removeKey(cell: HTMLTableCellElement): boolean {
  const parts = (cell.getAttribute(tileAttribute) ?? "").split(".");
  if (cell.getAttribute("data-state") !== "key" || cell.querySelector(".select-key") !== null || parts.length !== 3) {
    return false;
  }

  void send("/key/delete", { page: parts[0], row: parts[1], column: parts[2] });

  return true;
}

export function enableGridKeys(grid: HTMLTableElement): void {
  // Listening on the page's main part keeps the keys working on a grid a held
  // drag brings in.
  const root: HTMLElement = grid.closest("main") ?? document.body;

  root.addEventListener("keydown", (event: KeyboardEvent) => {
    const cell = cellOf(event.target);
    const table = cell?.closest<HTMLTableElement>("table.grid");
    if (cell === null || table === null || table === undefined || event.altKey || event.ctrlKey || event.metaKey) {
      return;
    }

    const step = steps[event.key];
    if (step !== undefined) {
      event.preventDefault();
      nextTile(table, cell, step)?.focus();

      return;
    }

    if ((event.key === "Delete" || event.key === "Backspace") && removeKey(cell)) {
      event.preventDefault();
    }
  });

}

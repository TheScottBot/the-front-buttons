// Shift and a click for copying keys. Select mode itself is a plain form the
// server draws, which works without this script; this only adds two
// shortcuts, and shows their hints, which stay hidden without it.
//
// On the grid, Shift and a click on a key opens select mode with that key
// ticked: the server ticks each tile named in the address. In select mode,
// Shift and a click on a key ticks every key in the rectangle from the last
// key clicked to this one.

const tileAttribute = "data-tile";
const keyState = "key";

interface Place {
  readonly page: string;
  readonly row: number;
  readonly column: number;
}

function placeOf(cell: Element | null): Place | null {
  const parts = (cell?.getAttribute(tileAttribute) ?? "").split(".");
  if (parts.length !== 3 || parts.some((part) => !/^\d+$/.test(part))) {
    return null;
  }

  return { page: parts[0], row: Number(parts[1]), column: Number(parts[2]) };
}

function cellOf(target: EventTarget | null): HTMLTableCellElement | null {
  return target instanceof Element ? target.closest<HTMLTableCellElement>(`td[${tileAttribute}]`) : null;
}

function boxOf(cell: Element): HTMLInputElement | null {
  return cell.querySelector<HTMLInputElement>('.select-key input[type="checkbox"]');
}

function showHints(): void {
  for (const hint of document.querySelectorAll<HTMLElement>(".select-help")) {
    hint.hidden = false;
  }
}

// openSelecting goes to select mode with the key clicked ticked.
function openSelecting(place: Place): void {
  const address = new URLSearchParams({ page: place.page, select: "1", tile: `${place.row}.${place.column}` });
  window.location.assign(`/page?${address.toString()}`);
}

// tickRectangle ticks every key between two corners, inclusive.
function tickRectangle(grid: HTMLTableElement, from: Place, to: Place): void {
  const top = Math.min(from.row, to.row);
  const bottom = Math.max(from.row, to.row);
  const left = Math.min(from.column, to.column);
  const right = Math.max(from.column, to.column);

  for (const cell of grid.querySelectorAll(`td[${tileAttribute}]`)) {
    const place = placeOf(cell);
    const box = boxOf(cell);

    if (place !== null && box !== null && place.row >= top && place.row <= bottom &&
      place.column >= left && place.column <= right) {
      box.checked = true;
    }
  }
}

export function enableSelecting(grid: HTMLTableElement): void {
  const selecting = grid.querySelector(".select-key") !== null;
  let anchor: Place | null = null;

  showHints();

  // Capture, so this runs before the key dialog's own click handler.
  grid.addEventListener("click", (event: MouseEvent) => {
    const cell = cellOf(event.target);
    const place = placeOf(cell);
    if (cell === null || place === null) {
      return;
    }

    if (!selecting) {
      if (event.shiftKey && event.button === 0 && cell.getAttribute("data-state") === keyState) {
        event.preventDefault();
        event.stopPropagation();
        openSelecting(place);
      }

      return;
    }

    if (boxOf(cell) === null) {
      return;
    }

    if (event.shiftKey && anchor !== null) {
      event.preventDefault();
      tickRectangle(grid, anchor, place);
    }

    anchor = place;
  }, true);
}

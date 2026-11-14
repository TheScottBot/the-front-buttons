// Shift and a click for copying keys. Select mode itself is a plain form the
// server draws, which works without this script; this only adds a shortcut,
// and shows its hint, which stays hidden without it.
//
// On the grid, Shift and a click on a key opens select mode with that key
// ticked: the server ticks each tile named in the address. In select mode,
// Shift and a click ticks or unticks the one key, as a plain click does, so
// Shift may be held throughout. Escape leaves select mode, as Done selecting
// does, unless it is typing in a field or closing a dialog.

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

// enableShiftTicking makes a click with Shift held in select mode tick or
// untick the one key clicked, as a plain click does. Some browsers do nothing
// for a label clicked with Shift held, which swallowed every click made with
// Shift still down from opening select mode; this does it in every browser.
function enableShiftTicking(grid: HTMLTableElement): void {
  grid.addEventListener("click", (event: MouseEvent) => {
    const box = cellOf(event.target)?.querySelector<HTMLInputElement>('.select-key input[type="checkbox"]');
    if (!event.shiftKey || box === null || box === undefined || event.target === box) {
      return;
    }

    event.preventDefault();
    box.checked = !box.checked;
  });
}

// enableEscapeToLeave makes Escape follow the Done selecting link. A field
// being typed in keeps its own Escape, and an open dialog closes first.
function enableEscapeToLeave(): void {
  const done = document.querySelector<HTMLAnchorElement>("a.select-done");
  if (done === null) {
    return;
  }

  document.addEventListener("keydown", (event: KeyboardEvent) => {
    const typing = event.target instanceof HTMLElement &&
      event.target.matches('input:not([type="checkbox"]), textarea, select, [contenteditable]');

    if (event.key !== "Escape" || event.defaultPrevented || typing || document.querySelector("dialog[open]") !== null) {
      return;
    }

    event.preventDefault();
    window.location.assign(done.href);
  });
}

export function enableSelecting(grid: HTMLTableElement): void {
  showHints();

  if (grid.querySelector(".select-key") !== null) {
    enableShiftTicking(grid);
    enableEscapeToLeave();

    return;
  }

  // Capture, so this runs before the key dialog's own click handler.
  grid.addEventListener("click", (event: MouseEvent) => {
    const cell = cellOf(event.target);
    const place = placeOf(cell);

    if (cell !== null && place !== null && event.shiftKey && event.button === 0 &&
      cell.getAttribute("data-state") === keyState) {
      event.preventDefault();
      event.stopPropagation();
      openSelecting(place);
    }
  }, true);
}

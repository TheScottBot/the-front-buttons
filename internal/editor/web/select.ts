// Select mode without loading the page. Select mode is a plain form the
// server draws, which works without this script: the Select keys drawer's
// edit mode check box and its Apply button load the page in or out of it.
// This switches it in place instead: ticking or unticking edit mode, Shift and
// a click on a key, which opens it with that key ticked, and Escape, which
// leaves it, each fetch the page as the server draws it in the other mode and
// put its grid and drawer on show, without loading anything else.
//
// In select mode, Shift and a click ticks or unticks the one key, as a plain
// click does, so Shift may be held throughout.

import { updateInPlace } from "./inplace";

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

// selecting reports whether the grid on show is in select mode.
function selecting(): boolean {
  return document.querySelector("table.grid:not(.grid-left-behind) .select-key") !== null;
}

// shownPage is the page on show, from its grid.
function shownPage(): string {
  return document.querySelector(`table.grid:not(.grid-left-behind) td[${tileAttribute}]`)
    ?.getAttribute(tileAttribute)?.split(".")[0] ?? "0";
}

// showHints shows what only the script makes true, and hides the Apply
// button the edit mode box needs without it.
function showHints(): void {
  for (const hint of document.querySelectorAll<HTMLElement>(".select-help")) {
    hint.hidden = false;
  }

  for (const apply of document.querySelectorAll<HTMLElement>(".select-apply")) {
    apply.hidden = true;
  }
}

// switchMode puts the page at an address on show in place, or loads it if it
// cannot be.
async function switchMode(address: string): Promise<void> {
  try {
    const response = await fetch(address, { credentials: "same-origin" });
    if (response.ok && await updateInPlace(await response.text(), address, false)) {
      showHints();

      return;
    }
  } catch {
    // Loaded instead, below.
  }

  window.location.assign(address);
}

function modeAddress(on: boolean, tile: string | null = null): string {
  const address = new URLSearchParams({ page: shownPage() });
  if (on) {
    address.set("select", "1");
  }

  if (tile !== null) {
    address.set("tile", tile);
  }

  return `/page?${address.toString()}`;
}

// enableEditMode switches select mode as the edit mode box is ticked.
function enableEditMode(): void {
  document.addEventListener("change", (event: Event) => {
    if (event.target instanceof HTMLInputElement && event.target.id === "select-mode-box") {
      void switchMode(modeAddress(event.target.checked));
    }
  });
}

// enableShiftTicking makes a click with Shift held in select mode tick or
// untick the one key clicked, as a plain click does. Some browsers do nothing
// for a label clicked with Shift held, which swallowed every click made with
// Shift still down from opening select mode; this does it in every browser.
function enableShiftTicking(): void {
  document.addEventListener("click", (event: MouseEvent) => {
    const box = cellOf(event.target)?.querySelector<HTMLInputElement>('.select-key input[type="checkbox"]');
    if (!event.shiftKey || box === null || box === undefined || event.target === box) {
      return;
    }

    event.preventDefault();
    box.checked = !box.checked;
  });
}

// enableEscapeToLeave makes Escape leave select mode. A field being typed in
// keeps its own Escape, and an open dialog closes first.
function enableEscapeToLeave(): void {
  document.addEventListener("keydown", (event: KeyboardEvent) => {
    const typing = event.target instanceof HTMLElement &&
      event.target.matches('input:not([type="checkbox"]), textarea, select, [contenteditable]');

    if (event.key !== "Escape" || event.defaultPrevented || typing || !selecting() ||
      document.querySelector("dialog[open]") !== null) {
      return;
    }

    event.preventDefault();
    void switchMode(modeAddress(false));
  });
}

// enablePlainShiftClicks takes a Shift-click on a link in the grid that
// nothing else here answers, such as a paging tile or an empty tile, as a
// plain click, in this tab, rather than the browser opening the link in a new
// window. It listens last, so Shift and a click on a key still opens select
// mode; Command and Ctrl clicks keep opening a new tab, as asked.
function enablePlainShiftClicks(): void {
  document.addEventListener("click", (event: MouseEvent) => {
    const link = event.target instanceof Element ? event.target.closest<HTMLAnchorElement>("table.grid a[href]") : null;
    if (link === null || event.defaultPrevented || !event.shiftKey || event.button !== 0 || event.metaKey ||
      event.ctrlKey || event.altKey) {
      return;
    }

    event.preventDefault();
    link.click();
  });
}

export function enableSelecting(grid: HTMLTableElement): void {
  showHints();
  enablePlainShiftClicks();
  enableEditMode();
  enableShiftTicking();
  enableEscapeToLeave();

  // A change made from the grid draws the drawer afresh, its hints hidden.
  document.addEventListener("front-buttons:updated", showHints);

  // Shift and a click on a key outside select mode opens it with that key
  // ticked. Capture, so this runs before the key dialog's own click handler.
  grid.closest("main")?.addEventListener("click", (event: MouseEvent) => {
    const cell = cellOf(event.target);
    const place = placeOf(cell);

    if (cell !== null && place !== null && event.shiftKey && event.button === 0 && !selecting() &&
      cell.getAttribute("data-state") === keyState) {
      event.preventDefault();
      event.stopPropagation();
      void switchMode(modeAddress(true, `${place.row}.${place.column}`));
    }
  }, true);
}

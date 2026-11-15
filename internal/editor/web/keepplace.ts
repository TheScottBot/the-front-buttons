// Keeping the grid in place: every change on the pages screen, a drop, a
// save from the key dialog, Undo, entering or leaving select mode, loads the
// page again, which would start it at the top. As the page goes, where the
// grid sits in the window is kept, and the next page of the grid, if it comes
// within a few seconds, scrolls to put its grid in the same place. Keeping the
// grid rather than the scroll position holds it still when what is above or
// below it changes, as select mode's form does. A page too short to scroll
// that far is given room at its bottom, so the grid stays put.
//
// The place is kept in this tab's session storage, which may be refused; the
// page then simply opens at the top, as it would without the script.

const placeKey = "front-buttons-grid-place";
// freshMilliseconds is how soon the next page must come for the place to
// count, so a page opened later starts at the top as usual.
const freshMilliseconds = 10000;

interface Kept {
  readonly top: number;
  readonly at: number;
}

function keep(grid: HTMLElement): void {
  try {
    const kept: Kept = { top: grid.getBoundingClientRect().top, at: Date.now() };
    window.sessionStorage.setItem(placeKey, JSON.stringify(kept));
  } catch {
    // Without storage the next page opens at the top.
  }
}

function takeKept(): Kept | null {
  try {
    const stored = window.sessionStorage.getItem(placeKey);
    window.sessionStorage.removeItem(placeKey);

    const kept = stored === null ? null : (JSON.parse(stored) as Partial<Kept>);
    if (typeof kept?.top !== "number" || typeof kept.at !== "number" || Date.now() - kept.at > freshMilliseconds) {
      return null;
    }

    return { top: kept.top, at: kept.at };
  } catch {
    return null;
  }
}

function restore(grid: HTMLElement, kept: Kept): void {
  const wanted = window.scrollY + grid.getBoundingClientRect().top - kept.top;
  const room = wanted - (document.documentElement.scrollHeight - window.innerHeight);

  if (room > 0) {
    document.documentElement.style.setProperty("--keep-room", `${Math.ceil(room)}px`);
  }

  window.scrollTo(0, Math.max(Math.round(wanted), 0));
}

export function enableKeepPlace(grid: HTMLElement): void {
  const kept = takeKept();
  if (kept !== null) {
    restore(grid, kept);
  }

  // The grid on show may be another page's, brought in by a held drag.
  window.addEventListener("pagehide", () => {
    keep(document.querySelector<HTMLElement>("table.grid:not(.grid-left-behind)") ?? grid);
  });
}

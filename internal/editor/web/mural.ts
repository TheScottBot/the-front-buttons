// Zooming out from the pages screen: - (or the Zoom out button) puts the
// page's grid aside and, in its place, without loading the page, draws the
// pages around it side by side, small, from All pages' own grids: this page
// in the middle, outlined, and as many either side as the window holds, in
// the order the deck pages through, an app screen's among its application's
// screens. The arrows, or the buttons either side, move along a page at a
// time, wrapping round as the deck does. Keys drag between the pages shown,
// as on All pages. A page's name zooms into it, and + into the page in the
// middle of those shown; Escape zooms back into the page zoomed out from. Either way the page comes back scrolled as it was
// before zooming out.

import { updateInPlace } from "./inplace";
import { keepPlace } from "./keepplace";

const currentClass = "mural-current";

interface Mural {
  readonly element: HTMLElement;
  readonly grid: HTMLTableElement;
  readonly pages: readonly HTMLElement[];
  readonly from: string;
  // Where the grid sat in the window, and how far the page was scrolled,
  // when it was zoomed out of, to come back to.
  readonly top: number;
  readonly scroll: number;
  start: number;
}

let mural: Mural | null = null;

function shownPage(grid: HTMLTableElement): string {
  return grid.querySelector("td[data-tile]")?.getAttribute("data-tile")?.split(".")[0] ?? "0";
}

function pageOf(section: HTMLElement): string {
  return section.querySelector("table[data-overview-page]")?.getAttribute("data-overview-page") ?? "";
}

// fit shows as many pages as the strip holds, the page at start first,
// wrapping round.
function fit(shown: Mural): void {
  const strip = shown.element.querySelector<HTMLElement>(".mural-strip");
  if (strip === null) {
    return;
  }

  for (const section of shown.pages) {
    section.hidden = false;
  }

  const width = shown.pages[0]?.getBoundingClientRect().width ?? 1;
  const gap = Number.parseFloat(getComputedStyle(strip).columnGap) || 0;
  const count = Math.max(1, Math.min(shown.pages.length, Math.floor((strip.clientWidth + gap) / (width + gap))));

  for (const section of shown.pages) {
    section.hidden = true;
  }

  for (let offset = 0; offset < count; offset++) {
    const section = shown.pages[(shown.start + offset) % shown.pages.length];
    section.hidden = false;
    strip.append(section);
  }
}

// centre puts the page zoomed out from in the middle of what fits.
function centre(shown: Mural): void {
  const index = Math.max(0, shown.pages.findIndex((section) => pageOf(section) === shown.from));
  shown.start = 0;
  fit(shown);

  const count = shown.pages.filter((section) => !section.hidden).length;
  shown.start = (index - Math.floor((count - 1) / 2) + shown.pages.length * 2) % shown.pages.length;
  fit(shown);
}

// middlePage is the page in the middle of those shown, the left of the two
// when an even number are, as the page zoomed out from first sits.
function middlePage(shown: Mural): string {
  const visible = [...shown.element.querySelectorAll<HTMLElement>(".mural-strip .overview-page:not([hidden])")];

  return pageOf(visible[Math.floor((visible.length - 1) / 2)] ?? visible[0]) || shown.from;
}

function move(step: number): void {
  if (mural === null) {
    return;
  }

  mural.start = (mural.start + step + mural.pages.length) % mural.pages.length;
  fit(mural);
}

function button(label: string, text: string, action: () => void): HTMLButtonElement {
  const made = document.createElement("button");
  made.type = "button";
  made.textContent = text;
  made.setAttribute("aria-label", label);
  made.addEventListener("click", action);

  return made;
}

// zoomIn leaves the mural for a page: the one zoomed out from, its grid put
// back and brought up to date in place, or another, opened.
async function zoomIn(page: string): Promise<void> {
  if (mural === null) {
    return;
  }

  const shown = mural;
  if (page !== shown.from) {
    // The page opened puts its grid where this one sat before zooming out.
    keepPlace(shown.top);
    window.location.assign(`/page?page=${page}`);

    return;
  }

  mural = null;
  shown.element.replaceWith(shown.grid);
  document.body.classList.remove("zoomed-out");
  window.scrollTo(0, shown.scroll);

  try {
    const response = await fetch(window.location.href, { credentials: "same-origin" });
    if (response.ok) {
      await updateInPlace(await response.text(), window.location.href);
    }
  } catch {
    // The grid stays as it was put aside.
  }

  shown.grid.querySelector<HTMLElement>("td a, td input")?.focus({ preventScroll: true });
}

async function zoomOut(grid: HTMLTableElement): Promise<void> {
  if (mural !== null || grid.querySelector(".select-key") !== null || !grid.isConnected) {
    return;
  }

  const response = await fetch("/overview", { credentials: "same-origin" });
  if (!response.ok) {
    return;
  }

  const incoming = new DOMParser().parseFromString(await response.text(), "text/html");
  const from = shownPage(grid);
  const application = incoming.querySelector(`table[data-overview-page="${from}"]`)?.getAttribute("data-application") ??
    "";
  const pages = [...incoming.querySelectorAll<HTMLElement>("section.overview-page")]
    .filter((section) => section.querySelector("table")?.getAttribute("data-application") === application)
    .map((section) => document.importNode(section, true));

  if (pages.length === 0) {
    return;
  }

  const element = document.createElement("div");
  element.className = "mural";
  element.setAttribute("role", "region");
  element.setAttribute("aria-label", "The pages around this one");
  element.tabIndex = -1;

  const strip = document.createElement("div");
  strip.className = "mural-strip";

  const head = document.createElement("div");
  head.className = "mural-head";
  head.append(
    button("Earlier pages", "‹", () => move(-1)),
    button("Back to this page", "Zoom in", () => void zoomIn(from)),
    button("Later pages", "›", () => move(1)),
  );

  for (const section of pages) {
    section.classList.toggle(currentClass, pageOf(section) === from);
    section.querySelector("h2 a")?.addEventListener("click", (event: Event) => {
      event.preventDefault();
      void zoomIn(pageOf(section));
    });
  }

  element.append(head, strip);
  strip.append(...pages);

  const top = grid.getBoundingClientRect().top;
  const scroll = window.scrollY;
  grid.replaceWith(element);
  document.body.classList.add("zoomed-out");

  mural = { element, grid, pages, from, top, scroll, start: 0 };
  centre(mural);

  // Taking focus, the zoomed out pages leave the page scrolled where it was.
  element.focus({ preventScroll: true });
}

export function enableMural(grid: HTMLTableElement): void {
  const zoom = document.querySelector<HTMLButtonElement>(".zoom-out");
  if (zoom === null) {
    return;
  }

  zoom.hidden = false;
  zoom.addEventListener("click", () => void (mural === null ? zoomOut(grid) : zoomIn(mural.from)));

  // The button says what it will do: zoom out, or zoomed out, zoom back in.
  new MutationObserver(() => {
    const zoomed = document.body.classList.contains("zoomed-out");
    zoom.textContent = zoomed ? "Zoom in" : "Zoom out";
    zoom.setAttribute("aria-pressed", String(zoomed));
  }).observe(document.body, { attributes: true, attributeFilter: ["class"] });

  document.addEventListener("keydown", (event: KeyboardEvent) => {
    const typing = event.target instanceof HTMLElement &&
      event.target.matches('input:not([type="checkbox"]), textarea, select, [contenteditable]');
    if (typing || event.metaKey || event.ctrlKey || event.altKey || document.querySelector("dialog[open]") !== null) {
      return;
    }

    const inTile = event.target instanceof Element && event.target.closest("td[data-tile]") !== null;

    if (event.key === "-" && mural === null) {
      event.preventDefault();
      void zoomOut(grid);
    } else if (mural !== null && event.key === "Escape") {
      event.preventDefault();
      void zoomIn(mural.from);
    } else if (mural !== null && (event.key === "+" || event.key === "=")) {
      event.preventDefault();
      void zoomIn(middlePage(mural));
    } else if (mural !== null && !inTile && (event.key === "ArrowLeft" || event.key === "ArrowRight")) {
      event.preventDefault();
      move(event.key === "ArrowLeft" ? -1 : 1);
    }
  });

  window.addEventListener("resize", () => {
    if (mural !== null) {
      fit(mural);
    }
  });
}

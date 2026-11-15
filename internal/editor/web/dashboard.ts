// The dashboard preview ticks as the deck does. Each second, on the second,
// every clock key whose figure may have changed is drawn again: a key says in
// data-period how many seconds pass between changes, and is asked for at that
// second, so all its figures turn together. The new picture is loaded before
// it replaces the old, so a key never flashes blank. Each second the page also
// asks which page the deck shows, and when the deck has turned to another,
// draws the pages again, so the page marked follows the deck.

const previewSelector = ".dashboard-previews";
const keySelector = "table.dashboard-preview img";
const pagesSelector = 'table.dashboard-preview[data-dashboard-view="pages"] img';

// drawAt loads a key's picture as it is at a second, then shows it.
function drawAt(key: HTMLImageElement, second: number): void {
  const address = new URL(key.src, document.baseURI);
  address.searchParams.set("at", String(second));

  const next = new Image();
  next.addEventListener("load", () => {
    key.src = next.src;
  });
  next.src = address.toString();
}

// followDeck draws the pages again when the deck shows another page.
async function followDeck(previews: HTMLElement, second: number): Promise<void> {
  const response = await fetch("/dashboard/showing", { credentials: "same-origin" });
  if (!response.ok) {
    return;
  }

  const showing = await response.text();
  if (showing === previews.dataset.showing) {
    return;
  }

  previews.dataset.showing = showing;

  for (const key of previews.querySelectorAll<HTMLImageElement>(pagesSelector)) {
    drawAt(key, second);
  }
}

// tick draws the keys due this second, then waits for the next.
function tick(previews: HTMLElement): void {
  const second = Math.floor(Date.now() / 1000);

  for (const key of previews.querySelectorAll<HTMLImageElement>(keySelector)) {
    const period = Number(key.dataset.period ?? "0");
    if (period > 0 && second % period === 0) {
      drawAt(key, second);
    }
  }

  followDeck(previews, second).catch(() => undefined);
  window.setTimeout(() => tick(previews), 1000 - (Date.now() % 1000));
}

export function enableDashboardClock(root: Document): void {
  const previews = root.querySelector<HTMLElement>(previewSelector);
  if (previews === null) {
    return;
  }

  window.setTimeout(() => tick(previews), 1000 - (Date.now() % 1000));
}

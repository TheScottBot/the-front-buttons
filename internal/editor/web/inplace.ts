// Updating the grid in place: a change made from the grid, such as a swap, a
// drop or a paste, answers with the page it lands on. Loading that page again
// would fetch every key's picture afresh, as the editor lets nothing be kept,
// and they pop in one by one. Where the answer is the page already on show,
// in the same mode, only the tiles that changed are replaced, each picture
// loaded before it is put in, so nothing on the grid flashes. Anything else,
// another page, select mode opened or closed, loads the page as before.

const tileAttribute = "data-tile";
// decodeMilliseconds is how long a changed picture may take to load before
// it is put in anyway.
const decodeMilliseconds = 2000;

function shownGrid(): HTMLTableElement | null {
  return document.querySelector<HTMLTableElement>("table.grid:not(.grid-left-behind)");
}

function pageOf(grid: Element | null): string | null {
  return grid?.querySelector(`td[${tileAttribute}]`)?.getAttribute(tileAttribute)?.split(".")[0] ?? null;
}

function pageNames(root: ParentNode): string {
  return [...root.querySelectorAll('nav[aria-label="Pages"] a')].map((link) => link.textContent ?? "").join("|");
}

// preload loads a picture into an image of its own, which, unlike one not
// yet in the page, always loads, so the page's own is drawn at once from it.
function preload(address: string): Promise<void> {
  return new Promise<void>((resolve) => {
    const image = new Image();
    const timer = window.setTimeout(resolve, decodeMilliseconds);
    const done = (): void => {
      window.clearTimeout(timer);
      resolve();
    };

    image.addEventListener("load", done);
    image.addEventListener("error", done);
    image.src = address;
  });
}

// fresh asks for a changed tile's pictures past the ones the page already
// holds for the same address, and waits for them to load.
function fresh(cell: HTMLTableCellElement): Promise<void>[] {
  return [...cell.querySelectorAll<HTMLImageElement>("img.deck-face")].map((picture) => {
    const address = new URL(picture.getAttribute("src") ?? "", window.location.href);
    address.searchParams.set("drawn", String(Date.now()));
    const source = address.pathname + address.search;

    return preload(source).then(() => {
      picture.src = source;
    });
  });
}

// sameTile reports a tile drawn as the answer draws it, leaving out what the
// script adds while it works, such as a drag's marks.
function sameTile(now: HTMLTableCellElement, next: HTMLTableCellElement): boolean {
  return now.getAttribute("data-state") === next.getAttribute("data-state") && now.innerHTML === next.innerHTML;
}

// replaceRegion brings one of the page's message regions into line with the
// answer's, filling the live region in place so it is read out.
function replaceRegion(incoming: Document, selector: string): void {
  const now = document.querySelector(selector);
  const next = incoming.querySelector(selector);

  if (now !== null && next !== null) {
    now.replaceChildren(...[...next.childNodes].map((node) => document.importNode(node, true)));
  } else if (now !== null) {
    now.remove();
  } else if (next !== null) {
    document.querySelector(".announcement")?.after(document.importNode(next, true));
  }
}

// updateInPlace puts the answer's grid on show without loading the page,
// reporting false where it cannot and the page must be loaded instead.
export async function updateInPlace(page: string, address: string): Promise<boolean> {
  const incoming = new DOMParser().parseFromString(page, "text/html");
  const nextGrid = incoming.querySelector<HTMLTableElement>("table.grid");
  const grid = shownGrid();

  if (grid === null || nextGrid === null || document.querySelector(".grid-left-behind") !== null ||
    pageOf(grid) !== pageOf(nextGrid) || pageNames(document) !== pageNames(incoming) ||
    (grid.querySelector(".select-key") === null) !== (nextGrid.querySelector(".select-key") === null) ||
    document.querySelector("dialog[open]") !== null) {
    return false;
  }

  const changed: [HTMLTableCellElement, HTMLTableCellElement][] = [];

  for (const next of nextGrid.querySelectorAll<HTMLTableCellElement>(`td[${tileAttribute}]`)) {
    const now = grid.querySelector<HTMLTableCellElement>(`td[${tileAttribute}="${next.getAttribute(tileAttribute)}"]`);
    if (now === null) {
      return false;
    }

    if (!sameTile(now, next)) {
      changed.push([now, document.importNode(next, true)]);
    }
  }

  await Promise.all(changed.flatMap(([, next]) => fresh(next)));

  // Focus on a tile replaced stays on that tile.
  const focused = document.activeElement?.closest(`td[${tileAttribute}]`)?.getAttribute(tileAttribute) ?? null;

  for (const [now, next] of changed) {
    now.replaceWith(next);
  }

  if (focused !== null && document.activeElement === document.body) {
    grid.querySelector<HTMLElement>(`td[${tileAttribute}="${focused}"] a, td[${tileAttribute}="${focused}"] input`)
      ?.focus();
  }

  const caption = grid.querySelector("caption");
  const nextCaption = nextGrid.querySelector("caption");
  if (caption !== null && nextCaption !== null) {
    caption.textContent = nextCaption.textContent;
  }

  replaceRegion(incoming, ".announcement");
  replaceRegion(incoming, ".problems");
  window.history.replaceState(null, "", address);

  return true;
}

// Reordering pages by dragging their names: a page's name carried along the
// page list moves as it goes, the names around it sliding aside into their
// new places, and let go of, the page takes the place it is shown in.
// Anywhere past the last name counts as the last place. It only adds to the
// page: the drop is the same POST /page/reorder the Move this page form sends,
// through the same guard and checks, and that form stays the way to do it
// from the keyboard. Regular pages and app screens are listed apart and each
// moves only within its own list, as the server also insists. A drag let go
// of anywhere else puts the names back as they were.

import { send } from "./drag";

const pageType = "application/x-front-buttons-page";
const pageAttribute = "data-page";
const movingClass = "reorder-moving";
// slideMilliseconds is how long a name takes to slide into its new place.
const slideMilliseconds = 150;
// The list takes a drop this far beyond its own edges, in pixels, so a name
// let go of just below or beside the list still counts.
const slack = 24;

interface Carried {
  readonly item: HTMLLIElement;
  readonly list: HTMLElement;
  // order is the pages' indices as listed when the drag began, so a place
  // in the list names the page that was there.
  readonly order: readonly string[];
}

function pageLinkOf(target: EventTarget | null): HTMLAnchorElement | null {
  const link = target instanceof Element ? target.closest<HTMLAnchorElement>(`nav a[${pageAttribute}]`) : null;

  return link !== null && /^\d+$/.test(link.getAttribute(pageAttribute) ?? "") ? link : null;
}

function itemsOf(list: HTMLElement): HTMLLIElement[] {
  return [...list.children].filter((child): child is HTMLLIElement => child instanceof HTMLLIElement);
}

function pageOf(item: HTMLLIElement): string {
  return item.querySelector(`a[${pageAttribute}]`)?.getAttribute(pageAttribute) ?? "";
}

// showingPage is the page on show, which the editor returns to wherever it
// moves.
function showingPage(): string {
  return document.querySelector(`nav a[${pageAttribute}][aria-current="page"]`)?.getAttribute(pageAttribute) ?? "";
}

function reducedMotion(): boolean {
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

// slide moves the names with a change to the list: each is drawn where it
// was, then slides to where it now is.
function slide(list: HTMLElement, change: () => void): void {
  const items = itemsOf(list);
  const before = new Map(items.map((item) => [item, item.getBoundingClientRect()]));

  change();

  if (reducedMotion()) {
    return;
  }

  for (const item of items) {
    const was = before.get(item);
    const now = item.getBoundingClientRect();
    if (was === undefined || (was.left === now.left && was.top === now.top)) {
      continue;
    }

    item.animate(
      [{ transform: `translate(${was.left - now.left}px, ${was.top - now.top}px)` }, { transform: "none" }],
      { duration: slideMilliseconds, easing: "ease-out" },
    );
  }
}

// within reports a point inside the list or just beyond it.
function within(list: HTMLElement, x: number, y: number): boolean {
  const box = list.getBoundingClientRect();

  return x >= box.left - slack && x <= box.right + slack && y >= box.top - slack && y <= box.bottom + slack;
}

// Where an item sits in the list's own layout, leaving out any slide under
// way, so a name mid slide does not move the carried one back and forth.
interface Place {
  readonly left: number;
  readonly top: number;
  readonly right: number;
  readonly bottom: number;
}

function placeOf(item: HTMLLIElement): Place {
  return {
    left: item.offsetLeft,
    top: item.offsetTop,
    right: item.offsetLeft + item.offsetWidth,
    bottom: item.offsetTop + item.offsetHeight,
  };
}

// newSpot is where the carried name goes for a point in the list, as the
// item it goes before (null for the end), or undefined to leave it be. Over
// another name, it goes past that name once the point crosses its middle;
// past the last name, it goes last.
function newSpot(carried: Carried, x: number, y: number): HTMLLIElement | null | undefined {
  const box = carried.list.getBoundingClientRect();
  const pointX = x - box.left - carried.list.clientLeft;
  const pointY = y - box.top - carried.list.clientTop;
  const items = itemsOf(carried.list);
  const others = items.filter((item) => item !== carried.item);
  const last = others.at(-1);

  if (last !== undefined) {
    const end = placeOf(last);
    if (pointY > end.bottom || (pointY >= end.top && pointX > end.right)) {
      return null;
    }
  }

  const over = others.find((item) => {
    const place = placeOf(item);

    return pointX >= place.left && pointX <= place.right && pointY >= place.top && pointY <= place.bottom;
  });
  if (over === undefined) {
    return undefined;
  }

  const place = placeOf(over);
  const pastMiddle = pointX > (place.left + place.right) / 2;
  const carriedFirst = items.indexOf(carried.item) < items.indexOf(over);

  if (carriedFirst && pastMiddle) {
    return over.nextElementSibling instanceof HTMLLIElement ? over.nextElementSibling : null;
  }

  return !carriedFirst && !pastMiddle ? over : undefined;
}

function moveTo(carried: Carried, spot: HTMLLIElement | null): void {
  const current = carried.item.nextElementSibling;
  if (spot === carried.item || spot === current || (spot === null && current === null)) {
    return;
  }

  slide(carried.list, () => carried.list.insertBefore(carried.item, spot));
}

// putBack lists the names as they were when the drag began.
function putBack(carried: Carried): void {
  const byPage = new Map(itemsOf(carried.list).map((item) => [pageOf(item), item]));

  slide(carried.list, () => {
    for (const page of carried.order) {
      const item = byPage.get(page);
      if (item !== undefined) {
        carried.list.append(item);
      }
    }
  });
}

export function enablePageReorder(root: Document): void {
  let carried: Carried | null = null;
  let dropped = false;

  root.addEventListener("dragstart", (event: DragEvent) => {
    const link = pageLinkOf(event.target);
    const item = link?.closest("li");
    const list = item?.parentElement;
    if (link === null || !(item instanceof HTMLLIElement) || !(list instanceof HTMLElement) ||
      event.dataTransfer === null) {
      return;
    }

    carried = { item, list, order: itemsOf(list).map(pageOf) };
    dropped = false;
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData(pageType, pageOf(item));

    // Marked once the browser has taken its picture of the name to drag.
    window.setTimeout(() => item.classList.add(movingClass), 0);
  });

  root.addEventListener("dragover", (event: DragEvent) => {
    if (carried === null || event.dataTransfer === null || !within(carried.list, event.clientX, event.clientY)) {
      return;
    }

    event.preventDefault();
    event.dataTransfer.dropEffect = "move";

    const spot = newSpot(carried, event.clientX, event.clientY);
    if (spot !== undefined) {
      moveTo(carried, spot);
    }
  });

  root.addEventListener("drop", (event: DragEvent) => {
    if (carried === null || !within(carried.list, event.clientX, event.clientY)) {
      return;
    }

    event.preventDefault();
    dropped = true;

    const from = pageOf(carried.item);
    const to = carried.order[itemsOf(carried.list).indexOf(carried.item)] ?? from;
    if (to !== from) {
      void send("/page/reorder", { page: from, to, showing: showingPage() });
    }
  });

  root.addEventListener("dragend", () => {
    if (carried === null) {
      return;
    }

    carried.item.classList.remove(movingClass);
    if (!dropped) {
      putBack(carried);
    }

    carried = null;
  });
}

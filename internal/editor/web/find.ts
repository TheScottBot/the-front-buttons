// The In use drawer: finding a key in use on any page. Without the script it
// is a plain search form on the page, sent with its button. The script makes
// it a tab on the window's left edge under Bricks, opening into a panel whose
// results follow the box as it is typed in, fetched as the server lists them,
// without loading the page. Enter opens the first, and the page it opens
// brings the key found into view with the drawer still open; Escape clears
// the box, or empty, closes the drawer; and / anywhere but a field opens it
// with the box ready.

const readyClass = "drawer-ready";
const waitMilliseconds = 200;

function shownPage(): string {
  return document.querySelector("table.grid:not(.grid-left-behind) td[data-tile]")
    ?.getAttribute("data-tile")?.split(".")[0] ?? "0";
}

async function showResults(search: string): Promise<void> {
  const results = document.getElementById("find-results");
  if (results === null) {
    return;
  }

  if (search.trim() === "") {
    results.replaceChildren();

    return;
  }

  try {
    const address = `/page?${new URLSearchParams({ page: shownPage(), find: search }).toString()}`;
    const response = await fetch(address, { credentials: "same-origin" });
    const incoming = new DOMParser().parseFromString(await response.text(), "text/html").getElementById("find-results");

    if (response.ok && incoming !== null) {
      results.replaceChildren(...[...incoming.childNodes].map((node) => document.importNode(node, true)));
    }
  } catch {
    // The results stay as they were; the form still sends.
  }
}

export function enableFind(): void {
  const drawer = document.getElementById("find-drawer");
  const toggle = drawer?.querySelector<HTMLButtonElement>(".find-toggle") ?? null;
  const panel = document.getElementById("find-panel");
  const box = document.getElementById("find-box");
  if (drawer === null || toggle === null || panel === null || !(box instanceof HTMLInputElement)) {
    return;
  }

  const setOpen = (open: boolean): void => {
    toggle.setAttribute("aria-expanded", String(open));
    panel.hidden = !open;
  };

  drawer.classList.add(readyClass);
  toggle.hidden = false;
  setOpen(toggle.getAttribute("aria-expanded") === "true");

  for (const button of document.querySelectorAll<HTMLElement>(".find-submit")) {
    button.hidden = true;
  }

  toggle.addEventListener("click", () => {
    setOpen(panel.hidden);

    if (!panel.hidden) {
      box.focus();
    }
  });

  // A page opened from a result brings the key found into view, after the
  // grid has been put back where it was.
  const found = document.querySelector<HTMLElement>("table.grid td.found");
  if (found !== null) {
    window.setTimeout(() => found.scrollIntoView({ block: "center" }), 0);
  }

  let timer: number | undefined;

  box.addEventListener("input", () => {
    window.clearTimeout(timer);
    timer = window.setTimeout(() => void showResults(box.value), waitMilliseconds);
  });

  box.addEventListener("keydown", (event: KeyboardEvent) => {
    if (event.key !== "Enter") {
      return;
    }

    event.preventDefault();
    document.querySelector<HTMLAnchorElement>("#find-results a")?.click();
  });

  // Escape, while the drawer is open and nothing is being typed elsewhere,
  // clears the box first, then closes the drawer.
  document.addEventListener("keydown", (event: KeyboardEvent) => {
    const target = event.target instanceof HTMLElement ? event.target : null;
    const typingElsewhere = target !== null && !panel.contains(target) &&
      target.matches('input:not([type="checkbox"]), textarea, select, [contenteditable]');

    if (event.key !== "Escape" || event.defaultPrevented || panel.hidden || typingElsewhere ||
      document.querySelector("dialog[open]") !== null) {
      return;
    }

    event.preventDefault();
    event.stopPropagation();

    if (box.value !== "") {
      box.value = "";
      void showResults("");
    } else {
      setOpen(false);
      toggle.focus();
    }
  }, true);

  document.addEventListener("keydown", (event: KeyboardEvent) => {
    const typing = event.target instanceof HTMLElement &&
      event.target.matches('input:not([type="checkbox"]), textarea, select, [contenteditable]');

    if (event.key !== "/" || typing || event.metaKey || event.ctrlKey || event.altKey ||
      document.querySelector("dialog[open]") !== null) {
      return;
    }

    event.preventDefault();
    setOpen(true);
    box.focus();
    box.select();
  });
}

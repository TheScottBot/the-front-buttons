// Find a key as one types: the drawer's search form works without the
// script, sent with its button. With it, the results follow the box as it is
// typed in, fetched as the server lists them, without loading the page; Enter
// opens the first, and the page it opens brings the key found into view;
// Escape clears the box, or empty, closes the drawer; and / anywhere but a
// field opens the drawer with the box ready.

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
  const box = document.getElementById("find-box");
  if (!(drawer instanceof HTMLDetailsElement) || !(box instanceof HTMLInputElement)) {
    return;
  }

  for (const button of document.querySelectorAll<HTMLElement>(".find-submit")) {
    button.hidden = true;
  }

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
    if (event.key === "Enter") {
      const first = document.querySelector<HTMLAnchorElement>("#find-results a");
      event.preventDefault();
      first?.click();
    } else if (event.key === "Escape") {
      event.preventDefault();
      event.stopPropagation();

      if (box.value !== "") {
        box.value = "";
        void showResults("");
      } else {
        drawer.open = false;
        drawer.querySelector("summary")?.focus();
      }
    }
  });

  document.addEventListener("keydown", (event: KeyboardEvent) => {
    const typing = event.target instanceof HTMLElement &&
      event.target.matches('input:not([type="checkbox"]), textarea, select, [contenteditable]');

    if (event.key !== "/" || typing || event.metaKey || event.ctrlKey || event.altKey ||
      document.querySelector("dialog[open]") !== null) {
      return;
    }

    event.preventDefault();
    drawer.open = true;
    box.focus();
    box.select();
  });
}

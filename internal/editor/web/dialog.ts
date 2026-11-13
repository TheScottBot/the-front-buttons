// Opens a key's editor over the grid in a modal dialog instead of on a page of
// its own, as the author asked (evaluation, deck testing). It only adds to the
// page: the grid's links still go to each key's own page, which is what this
// fetches and lifts the #key-editor part out of, and every form inside still
// posts where it always did, through the same guard and checks.
//
// The browser's <dialog> does the hard parts: shown modal, it keeps focus
// inside, closes on Escape and leaves the page behind it inert. When it closes,
// focus goes back to the tile that opened it.

import { showGroupMoves } from "./group";
import { applyReplacements } from "./replaces";
import { showSliders } from "./slider";

interface Elements {
  readonly dialog: HTMLDialogElement;
  readonly title: HTMLElement;
  readonly body: HTMLElement;
}

// htmx swaps the key editor when its type changes; content this script puts
// in the dialog has to be handed to htmx so its attributes work there too.
interface Htmx {
  process(element: Element): void;
}

const keyLinkSelector = 'a[href^="/key?"]';
const editorID = "key-editor";
const refusedMessage = "Error: the editor refused that request; close this and reload the page.";
const unreachableMessage = "Error: the editor could not be reached; is the daemon running?";

function elements(): Elements | null {
  const dialog = document.getElementById("key-dialog");
  const title = document.getElementById("key-dialog-title");
  const body = document.getElementById("key-dialog-body");

  if (!(dialog instanceof HTMLDialogElement) || title === null || body === null) {
    return null;
  }

  return { dialog, title, body };
}

function htmx(): Htmx | null {
  const candidate = (window as unknown as { htmx?: Htmx }).htmx;

  return candidate !== undefined && typeof candidate.process === "function" ? candidate : null;
}

// formParameters turns a form's fields, and the button that sent it, into the
// request's parameters. Only text values exist in the editor's forms.
function formParameters(form: HTMLFormElement, submitter: HTMLElement | null): URLSearchParams {
  const data = new FormData(form, submitter);
  const parameters = new URLSearchParams();

  for (const [name, value] of data) {
    if (typeof value === "string") {
      parameters.append(name, value);
    }
  }

  return parameters;
}

// focusFirst lands focus where the page asked (a field in error has
// autofocus), or on the first control.
function focusFirst(body: HTMLElement): void {
  const target =
    body.querySelector<HTMLElement>("[autofocus]") ??
    body.querySelector<HTMLElement>('select, input:not([type="hidden"]), button, a[href]');

  target?.focus();
}

function showMessage(parts: Elements, message: string): void {
  const paragraph = document.createElement("p");
  paragraph.className = "errors";
  paragraph.setAttribute("role", "alert");
  paragraph.textContent = message;
  parts.body.replaceChildren(paragraph);
}

// load fetches a key's page, or a form's answer, and shows its editor in the
// dialog. A change the server accepted answers with a redirect to the grid,
// which is followed, so the grid shows the change and the saved message.
async function load(parts: Elements, url: string, init?: RequestInit): Promise<void> {
  let response: Response;

  try {
    response = await fetch(url, { credentials: "same-origin", ...init });
  } catch {
    showMessage(parts, unreachableMessage);

    return;
  }

  if (response.redirected) {
    window.location.assign(response.url);

    return;
  }

  const page = new DOMParser().parseFromString(await response.text(), "text/html");
  const editor = page.getElementById(editorID);

  if (editor === null) {
    if (init === undefined) {
      window.location.assign(url);
    } else {
      showMessage(parts, refusedMessage);
    }

    return;
  }

  parts.title.textContent = page.querySelector("h1")?.textContent ?? "Key";
  parts.body.replaceChildren(document.importNode(editor, true));
  htmx()?.process(parts.body);
  applyReplacements(document);
  showSliders(document);
  showGroupMoves(document);

  if (!parts.dialog.open) {
    parts.dialog.showModal();
  }

  focusFirst(parts.body);
}

// openFrom is set once the dialog is enabled: it opens the key editor at an
// address, with focus going back to the element given when it closes.
let openFrom: ((address: string, from: HTMLElement) => void) | null = null;

// openKeyDialog opens the key editor at an address, as a tile's click does;
// the brick drawer uses it once a brick is dropped on an empty tile. Without
// the dialog it goes to the key's own page.
export function openKeyDialog(address: string, from: HTMLElement): void {
  if (openFrom === null) {
    window.location.assign(address);

    return;
  }

  openFrom(address, from);
}

function isPlainClick(event: MouseEvent): boolean {
  return event.button === 0 && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey;
}

export function enableKeyDialog(grid: HTMLTableElement): void {
  const parts = elements();
  if (parts === null) {
    return;
  }

  let opener: HTMLElement | null = null;

  // htmx takes the page title from the key editor it swaps in; the grid's own
  // title comes back when the dialog closes.
  const pageTitle = document.title;

  // A plain click, or Enter on a focused tile, opens the dialog; a click with
  // a modifier still opens the key's page in a new tab or window as usual.
  grid.addEventListener("click", (event: MouseEvent) => {
    const link = event.target instanceof Element ? event.target.closest<HTMLAnchorElement>(keyLinkSelector) : null;
    if (link === null || event.defaultPrevented || !isPlainClick(event)) {
      return;
    }

    event.preventDefault();
    opener = link;
    void load(parts, link.href);
  });

  openFrom = (address: string, from: HTMLElement): void => {
    opener = from;
    void load(parts, address);
  };

  // Every form in the dialog is sent from here, so its answer lands in the
  // dialog rather than replacing the page.
  parts.body.addEventListener("submit", (event: SubmitEvent) => {
    const form = event.target;
    if (!(form instanceof HTMLFormElement)) {
      return;
    }

    event.preventDefault();

    const parameters = formParameters(form, event.submitter);
    if (form.method.toLowerCase() === "get") {
      void load(parts, `${form.action}?${parameters.toString()}`);
    } else {
      void load(parts, form.action, { method: "POST", body: parameters });
    }
  });

  parts.dialog.querySelector(".dialog-close")?.addEventListener("click", () => parts.dialog.close());

  // A click on the backdrop, outside the dialog's box, closes it too.
  parts.dialog.addEventListener("click", (event: MouseEvent) => {
    if (event.target === parts.dialog) {
      parts.dialog.close();
    }
  });

  parts.dialog.addEventListener("close", () => {
    parts.body.replaceChildren();
    document.title = pageTitle;
    opener?.focus();
    opener = null;
  });
}

// Undo and Redo from the page: the Undo and Redo buttons, Undo after a
// removal, and select mode's buttons are plain forms that work without the
// script; with it they are sent as a change from the grid is, so the grid
// updates in place without flashing. Command-Z undoes and Shift-Command-Z redoes, Ctrl on Windows,
// unless text is being typed, where they keep their usual meaning.

import { send } from "./drag";

// sendForm posts an in-place form's fields, which carry the token already,
// to the address of the button that sent it, with that button's own field,
// as the browser would.
function sendForm(form: HTMLFormElement, submitter: HTMLElement | null): void {
  const fields: Record<string, string[]> = {};
  for (const [name, value] of new FormData(form, submitter)) {
    if (typeof value === "string" && name !== "token") {
      (fields[name] ??= []).push(value);
    }
  }

  const action = submitter?.getAttribute("formaction") ?? form.getAttribute("action") ?? "/undo";
  void send(action, fields);
}

// showingPage is the page on show, which an undo returns to.
function showingPage(): string {
  return document.querySelector<HTMLInputElement>('.history input[name="showing"]')?.value ?? "0";
}

export function enableHistory(): void {
  document.addEventListener("submit", (event: SubmitEvent) => {
    if (event.target instanceof HTMLFormElement && event.target.matches("form.in-place")) {
      event.preventDefault();
      sendForm(event.target, event.submitter);
    }
  });

  document.addEventListener("keydown", (event: KeyboardEvent) => {
    const typing = event.target instanceof HTMLElement &&
      event.target.matches('input:not([type="checkbox"]), textarea, select, [contenteditable]');

    if (!(event.metaKey || event.ctrlKey) || event.altKey || event.key.toLowerCase() !== "z" || typing ||
      document.querySelector("dialog[open]") !== null) {
      return;
    }

    event.preventDefault();
    void send(event.shiftKey ? "/redo" : "/undo", { showing: showingPage() });
  });
}

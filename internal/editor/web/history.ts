// Undo and Redo from the page: the Undo and Redo buttons, and Undo after a
// removal, are plain forms that work without the script; with it they are
// sent as a change from the grid is, so the grid updates in place without
// flashing. Command-Z undoes and Shift-Command-Z redoes, Ctrl on Windows,
// unless text is being typed, where they keep their usual meaning.

import { send } from "./drag";

// sendForm posts an in-place form's fields, which carry the token already.
function sendForm(form: HTMLFormElement): void {
  const fields: Record<string, string> = {};
  for (const [name, value] of new FormData(form)) {
    if (typeof value === "string" && name !== "token") {
      fields[name] = value;
    }
  }

  void send(form.getAttribute("action") ?? "/undo", fields);
}

// showingPage is the page on show, which an undo returns to.
function showingPage(): string {
  return document.querySelector<HTMLInputElement>('.history input[name="showing"]')?.value ?? "0";
}

export function enableHistory(): void {
  document.addEventListener("submit", (event: SubmitEvent) => {
    if (event.target instanceof HTMLFormElement && event.target.matches("form.in-place")) {
      event.preventDefault();
      sendForm(event.target);
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

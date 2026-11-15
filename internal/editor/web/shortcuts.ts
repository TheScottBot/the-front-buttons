// The shortcuts sheet: every way of working the pages screen, in one table in
// a dialog the page carries, opened with ? or the button in the line pointing
// to it, which, like the sheet, only means anything with the script. The
// browser closes it on Escape.

export function enableShortcuts(): void {
  const sheet = document.getElementById("shortcuts-dialog");
  if (!(sheet instanceof HTMLDialogElement)) {
    return;
  }

  const open = (): void => {
    if (!sheet.open) {
      sheet.showModal();
    }
  };

  document.querySelector(".shortcuts-open")?.addEventListener("click", open);
  sheet.querySelector(".dialog-close")?.addEventListener("click", () => sheet.close());

  document.addEventListener("keydown", (event: KeyboardEvent) => {
    const typing = event.target instanceof HTMLElement &&
      event.target.matches('input:not([type="checkbox"]), textarea, select, [contenteditable]');

    if (event.key !== "?" || typing || event.metaKey || event.ctrlKey || event.altKey ||
      document.querySelector("dialog[open]") !== null) {
      return;
    }

    event.preventDefault();
    open();
  });

  for (const hint of document.querySelectorAll<HTMLElement>(".shortcuts-help")) {
    hint.hidden = false;
  }
}

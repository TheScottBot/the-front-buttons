// The page drawer: the page's own forms, adding a screen, renaming, moving,
// copying and deleting the page, and adding a new one, sit in a panel behind a
// Page tab on the window's right edge, mirroring the brick drawer on the left.
// Without the script they stay on the page, open; the script only gathers them
// away. Escape closes it, unless text is being typed outside it.

const readyClass = "drawer-ready";

export function enablePageDrawer(): void {
  const drawer = document.getElementById("page-drawer");
  const toggle = drawer?.querySelector<HTMLButtonElement>(".page-drawer-toggle") ?? null;
  const tools = document.getElementById("page-tools");
  if (drawer === null || toggle === null || tools === null) {
    return;
  }

  const setOpen = (open: boolean): void => {
    toggle.setAttribute("aria-expanded", String(open));
    tools.hidden = !open;
  };

  drawer.classList.add(readyClass);
  toggle.hidden = false;
  setOpen(false);

  toggle.addEventListener("click", () => {
    setOpen(tools.hidden);

    if (!tools.hidden) {
      tools.querySelector<HTMLElement>('input:not([type="hidden"]), select, button')?.focus();
    }
  });

  document.addEventListener("keydown", (event: KeyboardEvent) => {
    const target = event.target instanceof HTMLElement ? event.target : null;
    const typingElsewhere = target !== null && !tools.contains(target) &&
      target.matches('input:not([type="checkbox"]), textarea, select, [contenteditable]');

    if (event.key !== "Escape" || event.defaultPrevented || tools.hidden || typingElsewhere ||
      document.querySelector("dialog[open]") !== null) {
      return;
    }

    event.preventDefault();
    event.stopPropagation();
    setOpen(false);
    toggle.focus();
  }, true);
}

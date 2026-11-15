// The side drawers, Bricks and In use, open one at a time: opening one
// closes any other, whose tab names the panel it controls, so two panels
// never sit over each other.

export function closeOtherDrawers(opening: HTMLButtonElement): void {
  for (const toggle of document.querySelectorAll<HTMLButtonElement>(".drawer-toggle[aria-expanded=\"true\"]")) {
    if (toggle === opening) {
      continue;
    }

    toggle.setAttribute("aria-expanded", "false");

    const panel = document.getElementById(toggle.getAttribute("aria-controls") ?? "");
    if (panel !== null) {
      panel.hidden = true;
    }
  }
}

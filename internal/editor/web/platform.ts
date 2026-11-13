// The shortcut key's modifiers are named as the Mac or Windows names them,
// and a switch flips between the two, as the author asked (evaluation, deck
// testing): each machine reads its own configuration, so the author may set
// up the Dell's shortcuts from the Mac. The names are all it changes; what is
// saved is the same. Without the script the names are this machine's, so the
// switch stays hidden until shown. Listening on the document covers a form
// the key dialog adds later.

type Platform = "mac" | "windows";

function choose(root: Document, platform: Platform): void {
  for (const label of root.querySelectorAll<HTMLLabelElement>("label[data-mac][data-windows]")) {
    label.textContent = (platform === "mac" ? label.dataset.mac : label.dataset.windows) ?? label.textContent;
  }

  for (const button of root.querySelectorAll<HTMLButtonElement>("button.platform-choice")) {
    button.setAttribute("aria-pressed", String(button.dataset.platform === platform));
  }
}

// showPlatformSwitch shows the switch, for a page as it loads and for a form
// the key dialog puts in later.
export function showPlatformSwitch(root: Document): void {
  for (const platformSwitch of root.querySelectorAll<HTMLElement>(".modifier-platform")) {
    platformSwitch.hidden = false;
  }
}

export function enablePlatformSwitch(root: Document): void {
  root.addEventListener("click", (event: MouseEvent) => {
    const button = event.target instanceof Element ? event.target.closest<HTMLButtonElement>("button.platform-choice") : null;
    const platform = button?.dataset.platform;
    if (platform === "mac" || platform === "windows") {
      choose(root, platform);
    }
  });

  showPlatformSwitch(root);
}

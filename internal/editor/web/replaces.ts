// A check box that stands in for other fields, as "Use app icon" does for Icon
// and Icon file, sets them aside while it is ticked, as the author asked
// (evaluation, deck testing). The fields become read only and are marked, not
// disabled: a disabled field is not sent, which would wipe the value saved in
// it for when the box is unticked. The daemon ignores them while it is ticked.

const replacedClass = "replaced";

function setAside(checkBox: HTMLInputElement, root: Document): void {
  for (const id of (checkBox.dataset.replaces ?? "").split(" ")) {
    const field = root.getElementById(id);
    if (!(field instanceof HTMLInputElement)) {
      continue;
    }

    field.readOnly = checkBox.checked;
    field.setAttribute("aria-disabled", checkBox.checked ? "true" : "false");
    field.closest(".field")?.classList.toggle(replacedClass, checkBox.checked);
  }
}

// applyReplacements brings every such check box's fields into line, for a page
// as it loads and for content the key dialog or htmx puts in later.
export function applyReplacements(root: Document): void {
  for (const checkBox of root.querySelectorAll<HTMLInputElement>("input[type=checkbox][data-replaces]")) {
    setAside(checkBox, root);
  }
}

export function enableReplacements(root: Document): void {
  root.addEventListener("change", (event: Event) => {
    const checkBox = event.target;
    if (checkBox instanceof HTMLInputElement && checkBox.dataset.replaces !== undefined) {
      setAside(checkBox, root);
    }
  });

  root.addEventListener("htmx:afterSwap", () => applyReplacements(root));
  applyReplacements(root);
}

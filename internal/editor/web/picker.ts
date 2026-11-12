// The app picker: choosing an installed application fills the field beside it
// with the identifier the launcher takes, so the author picks by name instead
// of typing a bundle identifier or AppID (evaluation, deck testing). The field
// stays a plain text field that can be typed into; the picker has no name, so
// it is never sent. Listening on the document covers pickers the key dialog
// adds later.

export function enablePickers(root: Document): void {
  root.addEventListener("change", (event: Event) => {
    const picker = event.target;
    if (!(picker instanceof HTMLSelectElement) || !picker.classList.contains("app-picker")) {
      return;
    }

    const field = root.getElementById(picker.dataset.fills ?? "");
    if (!(field instanceof HTMLInputElement) || picker.value === "") {
      return;
    }

    field.value = picker.value;
    field.dispatchEvent(new Event("input", { bubbles: true }));
  });
}

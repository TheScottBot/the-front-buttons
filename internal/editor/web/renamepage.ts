// Renaming a page from the page list: double-click the name of the page on
// show and it becomes a text box; Enter saves the new name, through the same
// POST /page/rename as the Rename this page form, and Escape, or leaving the
// box, puts the name back as it was. A single click on that name, which would
// only open the page already open, does nothing, so it does not reload the
// page between the two clicks. The form under the grid stays the way to do it
// from the keyboard.

import { send } from "./drag";

const pageAttribute = "data-page";
const inputClass = "rename-input";
// The longest name the configuration takes, as the rename form allows.
const maximumName = 24;

function currentLink(target: EventTarget | null): HTMLAnchorElement | null {
  const link = target instanceof Element ?
    target.closest<HTMLAnchorElement>(`nav a[${pageAttribute}][aria-current="page"]`) :
    null;

  return link !== null && /^\d+$/.test(link.getAttribute(pageAttribute) ?? "") ? link : null;
}

function startRenaming(link: HTMLAnchorElement): void {
  const box = document.createElement("input");
  box.type = "text";
  box.className = inputClass;
  box.value = link.textContent ?? "";
  box.maxLength = maximumName;
  box.setAttribute("aria-label", `Rename page ${link.textContent ?? ""}`);

  let finished = false;
  const finish = (save: boolean): void => {
    if (finished) {
      return;
    }

    finished = true;
    const name = box.value.trim();
    box.remove();
    link.hidden = false;

    if (save && name !== "" && name !== link.textContent) {
      link.textContent = name;
      void send("/page/rename", { page: link.getAttribute(pageAttribute) ?? "", name });
    } else {
      link.focus();
    }
  };

  box.addEventListener("keydown", (event: KeyboardEvent) => {
    if (event.key === "Enter" || event.key === "Escape") {
      event.preventDefault();
      event.stopPropagation();
      finish(event.key === "Enter");
    }
  });
  box.addEventListener("blur", () => finish(false));

  link.hidden = true;
  link.after(box);
  box.focus();
  box.select();
}

export function enableRenamePage(root: Document): void {
  root.addEventListener("click", (event: MouseEvent) => {
    if (currentLink(event.target) !== null && event.button === 0 && !event.metaKey && !event.ctrlKey) {
      event.preventDefault();
    }
  });

  root.addEventListener("dblclick", (event: MouseEvent) => {
    const link = currentLink(event.target);
    if (link !== null) {
      event.preventDefault();
      startRenaming(link);
    }
  });

  for (const hint of root.querySelectorAll<HTMLElement>(".rename-help")) {
    hint.hidden = false;
  }
}

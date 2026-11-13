// The group form's parts can be put in any order, as the author asked
// (evaluation, deck testing): each part's Up and Down buttons move it, and its
// number follows. The form sends the parts' hidden order fields in the order
// they sit, which is the order the keys go down the column. Without the script
// the parts keep their first order, so the buttons stay hidden until shown.
// Listening on the document covers a form the key dialog adds later.

const partSelector = "li.group-part";

function renumber(list: Element): void {
  list.querySelectorAll(partSelector).forEach((part, index) => {
    const slot = part.querySelector(".slot");
    if (slot !== null) {
      slot.textContent = String(index + 1);
    }
  });
}

// showGroupMoves shows each part's buttons, for a page as it loads and for a
// form the key dialog puts in later.
export function showGroupMoves(root: Document): void {
  for (const moves of root.querySelectorAll<HTMLElement>(".group-moves")) {
    moves.hidden = false;
  }
}

export function enableGroupOrder(root: Document): void {
  root.addEventListener("click", (event: MouseEvent) => {
    const button = event.target instanceof Element ? event.target.closest<HTMLButtonElement>(".move-up, .move-down") : null;
    const part = button?.closest(partSelector);
    const list = part?.parentElement;
    if (button === null || button === undefined || part === null || part === undefined || list === null ||
      list === undefined) {
      return;
    }

    if (button.classList.contains("move-up") && part.previousElementSibling !== null) {
      list.insertBefore(part, part.previousElementSibling);
    } else if (button.classList.contains("move-down") && part.nextElementSibling !== null) {
      list.insertBefore(part.nextElementSibling, part);
    }

    renumber(list);
    button.focus();
  });

  showGroupMoves(root);
}

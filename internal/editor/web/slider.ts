// A bounded number, such as a brightness key's level, can be set by digit or
// by slider, as the author asked (evaluation, deck testing). The number box is
// what is sent; the slider beside it has no name, and moving either moves the
// other. Listening on the document covers sliders the key dialog adds later.

const sliderSelector = "input.slider-input";

// showSliders shows each slider and sets it from its number, for a page as it
// loads and for content the key dialog puts in later.
export function showSliders(root: Document): void {
  for (const slider of root.querySelectorAll<HTMLInputElement>(sliderSelector)) {
    const number = root.getElementById(slider.dataset.mirrors ?? "");
    if (number instanceof HTMLInputElement && number.value !== "") {
      slider.value = number.value;
    }

    const holder = slider.closest<HTMLElement>(".slider");
    if (holder !== null) {
      holder.hidden = false;
    }
  }
}

export function enableSliders(root: Document): void {
  root.addEventListener("input", (event: Event) => {
    const moved = event.target;
    if (!(moved instanceof HTMLInputElement)) {
      return;
    }

    if (moved.matches(sliderSelector)) {
      const number = root.getElementById(moved.dataset.mirrors ?? "");
      if (number instanceof HTMLInputElement) {
        number.value = moved.value;
      }

      return;
    }

    const slider = root.querySelector<HTMLInputElement>(`${sliderSelector}[data-mirrors="${CSS.escape(moved.id)}"]`);
    if (slider !== null && moved.value !== "") {
      slider.value = moved.value;
    }
  });

  showSliders(root);
}

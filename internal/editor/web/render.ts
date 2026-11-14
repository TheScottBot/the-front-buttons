// The Render like deck switch: the grid's keys drawn as the deck draws them,
// or as words. The server draws the grid either way, reading the choice from
// a cookie, so the switch keeps the choice there and draws the page again.

const oneYearSeconds = 60 * 60 * 24 * 365;

export function enableRenderToggle(page: Document): void {
  const toggle = page.querySelector<HTMLButtonElement>("button.render-switch");
  if (toggle === null) {
    return;
  }

  toggle.addEventListener("click", () => {
    const on = toggle.getAttribute("aria-checked") !== "true";

    toggle.setAttribute("aria-checked", String(on));
    // The name matches deckRenderCookie in deckrender.go, which reads it.
    page.cookie = `deck_render=${on ? "on" : "off"}; path=/; max-age=${oneYearSeconds}; samesite=strict`;
    window.location.reload();
  });

  toggle.hidden = false;
}

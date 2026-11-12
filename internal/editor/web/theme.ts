// The theme switch: light or dark, whatever the system's setting. The choice
// is kept in a cookie the server reads, so each page is drawn in it from the
// start; until a choice is made the switch follows the system.

const oneYearSeconds = 60 * 60 * 24 * 365;
const systemDark = "(prefers-color-scheme: dark)";

export function enableThemeToggle(page: Document): void {
  const toggle = page.querySelector<HTMLButtonElement>("button.theme-switch");
  if (toggle === null) {
    return;
  }

  const root = page.documentElement;
  const system = window.matchMedia(systemDark);
  const isDark = (): boolean =>
    root.dataset["theme"] === "dark" || (root.dataset["theme"] === undefined && system.matches);
  const show = (): void => {
    toggle.setAttribute("aria-checked", String(isDark()));
  };

  toggle.addEventListener("click", () => {
    const chosen = isDark() ? "light" : "dark";

    root.dataset["theme"] = chosen;
    // The name matches themeCookie in pages.go, which reads it.
    page.cookie = `theme=${chosen}; path=/; max-age=${oneYearSeconds}; samesite=strict`;
    show();
  });

  system.addEventListener("change", show);
  show();
  toggle.hidden = false;
}

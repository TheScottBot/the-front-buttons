(() => {
  // internal/editor/web/replaces.ts
  var replacedClass = "replaced";
  function setAside(checkBox, root) {
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
  function applyReplacements(root) {
    for (const checkBox of root.querySelectorAll("input[type=checkbox][data-replaces]")) {
      setAside(checkBox, root);
    }
  }
  function enableReplacements(root) {
    root.addEventListener("change", (event) => {
      const checkBox = event.target;
      if (checkBox instanceof HTMLInputElement && checkBox.dataset.replaces !== void 0) {
        setAside(checkBox, root);
      }
    });
    root.addEventListener("htmx:afterSwap", () => applyReplacements(root));
    applyReplacements(root);
  }

  // internal/editor/web/dialog.ts
  var keyLinkSelector = 'a[href^="/key?"]';
  var editorID = "key-editor";
  var refusedMessage = "Error: the editor refused that request; close this and reload the page.";
  var unreachableMessage = "Error: the editor could not be reached; is the daemon running?";
  function elements() {
    const dialog = document.getElementById("key-dialog");
    const title = document.getElementById("key-dialog-title");
    const body = document.getElementById("key-dialog-body");
    if (!(dialog instanceof HTMLDialogElement) || title === null || body === null) {
      return null;
    }
    return { dialog, title, body };
  }
  function htmx() {
    const candidate = window.htmx;
    return candidate !== void 0 && typeof candidate.process === "function" ? candidate : null;
  }
  function formParameters(form, submitter) {
    const data = new FormData(form, submitter);
    const parameters = new URLSearchParams();
    for (const [name, value] of data) {
      if (typeof value === "string") {
        parameters.append(name, value);
      }
    }
    return parameters;
  }
  function focusFirst(body) {
    const target = body.querySelector("[autofocus]") ?? body.querySelector('select, input:not([type="hidden"]), button, a[href]');
    target?.focus();
  }
  function showMessage(parts, message) {
    const paragraph = document.createElement("p");
    paragraph.className = "errors";
    paragraph.setAttribute("role", "alert");
    paragraph.textContent = message;
    parts.body.replaceChildren(paragraph);
  }
  async function load(parts, url, init) {
    let response;
    try {
      response = await fetch(url, { credentials: "same-origin", ...init });
    } catch {
      showMessage(parts, unreachableMessage);
      return;
    }
    if (response.redirected) {
      window.location.assign(response.url);
      return;
    }
    const page = new DOMParser().parseFromString(await response.text(), "text/html");
    const editor = page.getElementById(editorID);
    if (editor === null) {
      if (init === void 0) {
        window.location.assign(url);
      } else {
        showMessage(parts, refusedMessage);
      }
      return;
    }
    parts.title.textContent = page.querySelector("h1")?.textContent ?? "Key";
    parts.body.replaceChildren(document.importNode(editor, true));
    htmx()?.process(parts.body);
    applyReplacements(document);
    if (!parts.dialog.open) {
      parts.dialog.showModal();
    }
    focusFirst(parts.body);
  }
  function isPlainClick(event) {
    return event.button === 0 && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey;
  }
  function enableKeyDialog(grid2) {
    const parts = elements();
    if (parts === null) {
      return;
    }
    let opener = null;
    const pageTitle = document.title;
    grid2.addEventListener("click", (event) => {
      const link = event.target instanceof Element ? event.target.closest(keyLinkSelector) : null;
      if (link === null || event.defaultPrevented || !isPlainClick(event)) {
        return;
      }
      event.preventDefault();
      opener = link;
      void load(parts, link.href);
    });
    parts.body.addEventListener("submit", (event) => {
      const form = event.target;
      if (!(form instanceof HTMLFormElement)) {
        return;
      }
      event.preventDefault();
      const parameters = formParameters(form, event.submitter);
      if (form.method.toLowerCase() === "get") {
        void load(parts, `${form.action}?${parameters.toString()}`);
      } else {
        void load(parts, form.action, { method: "POST", body: parameters });
      }
    });
    parts.dialog.querySelector(".dialog-close")?.addEventListener("click", () => parts.dialog.close());
    parts.dialog.addEventListener("click", (event) => {
      if (event.target === parts.dialog) {
        parts.dialog.close();
      }
    });
    parts.dialog.addEventListener("close", () => {
      parts.body.replaceChildren();
      document.title = pageTitle;
      opener?.focus();
      opener = null;
    });
  }

  // internal/editor/web/drag.ts
  var tileAttribute = "data-tile";
  var stateAttribute = "data-state";
  var draggingClass = "dragging";
  var dropTargetClass = "drop-target";
  var tileType = "application/x-front-buttons-tile";
  function parseTile(value) {
    const parts = (value ?? "").split(".");
    if (parts.length !== 3 || parts.some((part) => !/^\d+$/.test(part))) {
      return null;
    }
    return { page: parts[0], row: parts[1], column: parts[2] };
  }
  function cellOf(target) {
    return target instanceof Element ? target.closest(`td[${tileAttribute}]`) : null;
  }
  function stateOf(cell) {
    const state = cell.getAttribute(stateAttribute);
    return state === "key" || state === "empty" || state === "reserved" ? state : null;
  }
  function announce(message) {
    const region = document.querySelector(".announcement");
    if (region !== null) {
      region.textContent = message;
    }
  }
  function refusalFrom(page) {
    const parsed = new DOMParser().parseFromString(page, "text/html");
    const alert = parsed.querySelector(".errors");
    return alert?.textContent?.trim() || "Error: the key could not be moved.";
  }
  async function send(path, fields) {
    const token = document.querySelector('input[name="token"]')?.value;
    if (token === void 0) {
      announce("Error: this page has no editor token; reload it and try again.");
      return;
    }
    const body = new URLSearchParams({ token, ...fields });
    try {
      const response = await fetch(path, { method: "POST", body, credentials: "same-origin" });
      if (response.ok && response.redirected) {
        window.location.assign(response.url);
        return;
      }
      announce(refusalFrom(await response.text()));
    } catch {
      announce("Error: the editor could not be reached; is the daemon running?");
    }
  }
  function relocate(source, destination, copying) {
    return send("/key/move", {
      page: source.page,
      row: source.row,
      column: source.column,
      to: destination,
      action: copying ? "copy" : "move"
    });
  }
  function remove(source) {
    return send("/key/delete", { page: source.page, row: source.row, column: source.column });
  }
  var binArmedClass = "armed";
  var binTopClass = "key-bin-top";
  function overlaps(first, second) {
    return first.left < second.right && second.left < first.right && first.top < second.bottom && second.top < first.bottom;
  }
  function showBin(bin, grid2) {
    bin.classList.remove(binTopClass, binArmedClass);
    bin.hidden = false;
    if (overlaps(bin.getBoundingClientRect(), grid2.getBoundingClientRect())) {
      bin.classList.add(binTopClass);
    }
  }
  function hideBin(bin) {
    bin.hidden = true;
    bin.classList.remove(binArmedClass);
  }
  function clearDropTargets(grid2) {
    for (const cell of grid2.querySelectorAll(`.${dropTargetClass}`)) {
      cell.classList.remove(dropTargetClass);
    }
  }
  function enableDrag(grid2) {
    let source = null;
    const bin = document.getElementById("key-bin");
    if (bin !== null) {
      bin.addEventListener("dragover", (event) => {
        if (source === null || event.dataTransfer === null) {
          return;
        }
        event.preventDefault();
        event.dataTransfer.dropEffect = "move";
        bin.classList.add(binArmedClass);
      });
      bin.addEventListener("dragleave", () => bin.classList.remove(binArmedClass));
      bin.addEventListener("drop", (event) => {
        if (source === null) {
          return;
        }
        event.preventDefault();
        const carried = source;
        hideBin(bin);
        void remove(carried);
      });
    }
    grid2.addEventListener("dragstart", (event) => {
      const cell = cellOf(event.target);
      if (cell === null || stateOf(cell) !== "key" || event.dataTransfer === null) {
        return;
      }
      source = parseTile(cell.getAttribute(tileAttribute));
      event.dataTransfer.effectAllowed = "copyMove";
      event.dataTransfer.setData(tileType, cell.getAttribute(tileAttribute) ?? "");
      cell.classList.add(draggingClass);
      if (bin !== null) {
        showBin(bin, grid2);
      }
    });
    grid2.addEventListener("dragend", (event) => {
      cellOf(event.target)?.classList.remove(draggingClass);
      clearDropTargets(grid2);
      source = null;
      if (bin !== null) {
        hideBin(bin);
      }
    });
    grid2.addEventListener("dragover", (event) => {
      const cell = cellOf(event.target);
      if (source === null || cell === null || stateOf(cell) !== "empty" || event.dataTransfer === null) {
        return;
      }
      event.preventDefault();
      event.dataTransfer.dropEffect = event.altKey ? "copy" : "move";
      cell.classList.add(dropTargetClass);
    });
    grid2.addEventListener("dragleave", (event) => {
      cellOf(event.target)?.classList.remove(dropTargetClass);
    });
    grid2.addEventListener("drop", (event) => {
      const cell = cellOf(event.target);
      const destination = cell?.getAttribute(tileAttribute) ?? null;
      if (source === null || cell === null || stateOf(cell) !== "empty" || parseTile(destination) === null) {
        return;
      }
      event.preventDefault();
      clearDropTargets(grid2);
      void relocate(source, destination, event.altKey);
    });
    document.getElementById("drag-help")?.removeAttribute("hidden");
  }

  // internal/editor/web/picker.ts
  function enablePickers(root) {
    root.addEventListener("change", (event) => {
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

  // internal/editor/web/theme.ts
  var oneYearSeconds = 60 * 60 * 24 * 365;
  var systemDark = "(prefers-color-scheme: dark)";
  function enableThemeToggle(page) {
    const toggle = page.querySelector("button.theme-switch");
    if (toggle === null) {
      return;
    }
    const root = page.documentElement;
    const system = window.matchMedia(systemDark);
    const isDark = () => root.dataset["theme"] === "dark" || root.dataset["theme"] === void 0 && system.matches;
    const show = () => {
      toggle.setAttribute("aria-checked", String(isDark()));
    };
    toggle.addEventListener("click", () => {
      const chosen = isDark() ? "light" : "dark";
      root.dataset["theme"] = chosen;
      page.cookie = `theme=${chosen}; path=/; max-age=${oneYearSeconds}; samesite=strict`;
      show();
    });
    system.addEventListener("change", show);
    show();
    toggle.hidden = false;
  }

  // internal/editor/web/editor.ts
  enablePickers(document);
  enableReplacements(document);
  enableThemeToggle(document);
  var grid = document.querySelector("table.grid");
  if (grid !== null) {
    enableDrag(grid);
    enableKeyDialog(grid);
  }
})();

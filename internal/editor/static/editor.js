(() => {
  // internal/editor/web/group.ts
  var partSelector = "li.group-part";
  function renumber(list) {
    list.querySelectorAll(partSelector).forEach((part, index) => {
      const slot = part.querySelector(".slot");
      if (slot !== null) {
        slot.textContent = String(index + 1);
      }
    });
  }
  function showGroupMoves(root) {
    for (const moves of root.querySelectorAll(".group-moves")) {
      moves.hidden = false;
    }
  }
  function enableGroupOrder(root) {
    root.addEventListener("click", (event) => {
      const button = event.target instanceof Element ? event.target.closest(".move-up, .move-down") : null;
      const part = button?.closest(partSelector);
      const list = part?.parentElement;
      if (button === null || button === void 0 || part === null || part === void 0 || list === null || list === void 0) {
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

  // internal/editor/web/platform.ts
  function choose(root, platform) {
    for (const label of root.querySelectorAll("label[data-mac][data-windows]")) {
      label.textContent = (platform === "mac" ? label.dataset.mac : label.dataset.windows) ?? label.textContent;
    }
    for (const button of root.querySelectorAll("button.platform-choice")) {
      button.setAttribute("aria-pressed", String(button.dataset.platform === platform));
    }
  }
  function showPlatformSwitch(root) {
    for (const platformSwitch of root.querySelectorAll(".modifier-platform")) {
      platformSwitch.hidden = false;
    }
  }
  function enablePlatformSwitch(root) {
    root.addEventListener("click", (event) => {
      const button = event.target instanceof Element ? event.target.closest("button.platform-choice") : null;
      const platform = button?.dataset.platform;
      if (platform === "mac" || platform === "windows") {
        choose(root, platform);
      }
    });
    showPlatformSwitch(root);
  }

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

  // internal/editor/web/slider.ts
  var sliderSelector = "input.slider-input";
  function showSliders(root) {
    for (const slider of root.querySelectorAll(sliderSelector)) {
      const number = root.getElementById(slider.dataset.mirrors ?? "");
      if (number instanceof HTMLInputElement && number.value !== "") {
        slider.value = number.value;
      }
      const holder = slider.closest(".slider");
      if (holder !== null) {
        holder.hidden = false;
      }
    }
  }
  function enableSliders(root) {
    root.addEventListener("input", (event) => {
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
      const slider = root.querySelector(`${sliderSelector}[data-mirrors="${CSS.escape(moved.id)}"]`);
      if (slider !== null && moved.value !== "") {
        slider.value = moved.value;
      }
    });
    showSliders(root);
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
    showSliders(document);
    showGroupMoves(document);
    showPlatformSwitch(document);
    if (!parts.dialog.open) {
      parts.dialog.showModal();
    }
    focusFirst(parts.body);
  }
  var openFrom = null;
  function openKeyDialog(address, from) {
    if (openFrom === null) {
      window.location.assign(address);
      return;
    }
    openFrom(address, from);
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
    openFrom = (address, from) => {
      opener = from;
      void load(parts, address);
    };
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
  function relocate(source, destination, copying2) {
    return send("/key/move", {
      page: source.page,
      row: source.row,
      column: source.column,
      to: destination,
      action: copying2 ? "copy" : "move"
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
  var pageAttribute = "data-page";
  var pageDropClass = "page-drop";
  var pageArmingClass = "page-arming";
  var leftBehindClass = "grid-left-behind";
  var holdMilliseconds = 800;
  function copying(event) {
    return event.metaKey || event.altKey || event.ctrlKey || event.dataTransfer?.dropEffect === "copy";
  }
  function showCopying(event) {
    if (event.dataTransfer !== null && (event.metaKey || event.altKey || event.ctrlKey)) {
      event.dataTransfer.dropEffect = "copy";
    }
  }
  function pageLinkOf(target) {
    const link = target instanceof Element ? target.closest(`a[${pageAttribute}]`) : null;
    return link !== null && /^\d+$/.test(link.getAttribute(pageAttribute) ?? "") ? link : null;
  }
  function shownGrid() {
    return document.querySelector(`table.grid:not(.${leftBehindClass})`);
  }
  async function openPageDuringDrag(page) {
    try {
      const response = await fetch(`/page?page=${page}`, { credentials: "same-origin" });
      if (!response.ok) {
        return false;
      }
      const parsed = new DOMParser().parseFromString(await response.text(), "text/html");
      const incoming = parsed.querySelector("table.grid");
      const incomingPages = parsed.querySelector('nav[aria-label="Pages"]');
      const current = shownGrid();
      if (incoming === null || current === null) {
        return false;
      }
      current.classList.add(leftBehindClass);
      current.setAttribute("aria-hidden", "true");
      current.after(document.importNode(incoming, true));
      if (incomingPages !== null) {
        document.querySelector('nav[aria-label="Pages"]')?.replaceWith(document.importNode(incomingPages, true));
      }
      window.history.replaceState(null, "", `/page?page=${page}`);
      announce("Opened that page: drop the key on a free tile.");
      return true;
    } catch {
      announce("Error: that page could not be opened; is the daemon running?");
      return false;
    }
  }
  function enableDrag(grid2) {
    const root = grid2.closest("main") ?? document.body;
    let source = null;
    let shownPage = "";
    let opened = false;
    let dropped = false;
    let armed = null;
    let armTimer;
    const bin = document.getElementById("key-bin");
    const disarm = () => {
      window.clearTimeout(armTimer);
      armed?.classList.remove(pageDropClass, pageArmingClass);
      armed = null;
    };
    const arm = (link) => {
      if (armed === link) {
        return;
      }
      disarm();
      armed = link;
      link.classList.add(pageDropClass, pageArmingClass);
      const page = link.getAttribute(pageAttribute) ?? "";
      armTimer = window.setTimeout(() => {
        disarm();
        const before = shownPage;
        shownPage = page;
        void openPageDuringDrag(page).then((done) => {
          opened || (opened = done);
          if (!done) {
            shownPage = before;
          }
        });
      }, holdMilliseconds);
    };
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
        dropped = true;
        const carried = source;
        hideBin(bin);
        void remove(carried);
      });
    }
    root.addEventListener("dragstart", (event) => {
      const cell = cellOf(event.target);
      if (cell === null || stateOf(cell) !== "key" || event.dataTransfer === null) {
        return;
      }
      source = parseTile(cell.getAttribute(tileAttribute));
      shownPage = source?.page ?? "";
      opened = false;
      dropped = false;
      event.dataTransfer.effectAllowed = "copyMove";
      event.dataTransfer.setData(tileType, cell.getAttribute(tileAttribute) ?? "");
      cell.classList.add(draggingClass);
      const shown = shownGrid();
      if (bin !== null && shown !== null) {
        showBin(bin, shown);
      }
    });
    root.addEventListener("dragend", (event) => {
      cellOf(event.target)?.classList.remove(draggingClass);
      clearDropTargets(root);
      disarm();
      source = null;
      if (bin !== null) {
        hideBin(bin);
      }
      if (opened && !dropped) {
        window.location.assign(`/page?page=${shownPage}`);
      }
    });
    root.addEventListener("dragover", (event) => {
      if (source === null || event.dataTransfer === null) {
        return;
      }
      const link = pageLinkOf(event.target);
      if (link !== null && link.getAttribute(pageAttribute) !== shownPage) {
        event.preventDefault();
        showCopying(event);
        arm(link);
        return;
      }
      const cell = cellOf(event.target);
      if (cell === null || stateOf(cell) !== "empty") {
        return;
      }
      event.preventDefault();
      showCopying(event);
      cell.classList.add(dropTargetClass);
    });
    root.addEventListener("dragleave", (event) => {
      cellOf(event.target)?.classList.remove(dropTargetClass);
      if (armed !== null && pageLinkOf(event.target) === armed && !armed.contains(event.relatedTarget)) {
        disarm();
      }
    });
    root.addEventListener("drop", (event) => {
      if (source === null) {
        return;
      }
      const link = pageLinkOf(event.target);
      if (link !== null && link.getAttribute(pageAttribute) !== shownPage) {
        event.preventDefault();
        disarm();
        dropped = true;
        void relocate(source, `${link.getAttribute(pageAttribute) ?? ""}.${source.row}.${source.column}`, copying(event));
        return;
      }
      const cell = cellOf(event.target);
      const destination = cell?.getAttribute(tileAttribute) ?? null;
      if (cell === null || stateOf(cell) !== "empty" || parseTile(destination) === null) {
        return;
      }
      event.preventDefault();
      clearDropTargets(root);
      dropped = true;
      void relocate(source, destination, copying(event));
    });
    document.getElementById("drag-help")?.removeAttribute("hidden");
  }

  // internal/editor/web/drawer.ts
  var brickType = "application/x-front-buttons-brick";
  var placingClass = "placing";
  var carryingClass = "carrying";
  var stackClass = "stack-target";
  function placeOf(cell) {
    const parts = (cell.dataset.tile ?? "").split(".");
    if (parts.length !== 3) {
      return null;
    }
    return { page: parts[0], row: Number(parts[1]), column: Number(parts[2]) };
  }
  function cellOf2(target) {
    return target instanceof Element ? target.closest("td[data-tile]") : null;
  }
  function stackFrom(grid2, cell, carried) {
    const place = placeOf(cell);
    if (place === null) {
      return null;
    }
    const stack = [];
    for (let down = 0; down < carried.height; down++) {
      for (let across = 0; across < carried.width; across++) {
        const tile = `${place.page}.${place.row + down}.${place.column + across}`;
        const covered = grid2.querySelector(`td[data-tile="${tile}"]`);
        if (covered === null || covered.dataset.state !== "empty") {
          return null;
        }
        stack.push(covered);
      }
    }
    return stack;
  }
  function setOpen(toggle, panel, open) {
    toggle.setAttribute("aria-expanded", String(open));
    panel.hidden = !open;
  }
  function carriedFrom(item) {
    if (item.dataset.type !== void 0) {
      return { type: item.dataset.type, group: null, width: 1, height: 1 };
    }
    const width = Number(item.dataset.width);
    const height = Number(item.dataset.height);
    if (item.dataset.group !== void 0 && Number.isInteger(width) && Number.isInteger(height) && width > 0 && height > 0) {
      return { type: null, group: item.dataset.group, width, height };
    }
    return null;
  }
  function enableDrawer(grid2) {
    const drawer = document.getElementById("brick-drawer");
    const toggle = drawer?.querySelector(".drawer-toggle");
    const panel = document.getElementById("brick-list");
    if (drawer === null || toggle === null || toggle === void 0 || panel === null) {
      return;
    }
    let carried = null;
    let marked = [];
    const unmark = () => {
      for (const cell of marked) {
        cell.classList.remove(stackClass);
        delete cell.dataset.slot;
      }
      marked = [];
    };
    const mark = (stack) => {
      unmark();
      stack.forEach((cell, index) => {
        cell.classList.add(stackClass);
        if (stack.length > 1) {
          cell.dataset.slot = String(index + 1);
        }
      });
      marked = stack;
    };
    toggle.addEventListener("click", () => setOpen(toggle, panel, panel.hidden));
    panel.addEventListener("dragstart", (event) => {
      const item = event.target instanceof Element ? event.target.closest("li[draggable]") : null;
      const picked = item === null ? null : carriedFrom(item);
      if (picked === null || event.dataTransfer === null) {
        return;
      }
      carried = picked;
      event.dataTransfer.effectAllowed = "copy";
      event.dataTransfer.setData(brickType, picked.type ?? picked.group ?? "");
      grid2.classList.add(placingClass);
      window.setTimeout(() => drawer.classList.add(carryingClass), 0);
    });
    panel.addEventListener("dragend", () => {
      carried = null;
      unmark();
      grid2.classList.remove(placingClass);
      drawer.classList.remove(carryingClass);
    });
    grid2.addEventListener("dragover", (event) => {
      const cell = cellOf2(event.target);
      if (carried === null || cell === null || event.dataTransfer === null) {
        return;
      }
      const stack = stackFrom(grid2, cell, carried);
      if (stack === null) {
        unmark();
        return;
      }
      event.preventDefault();
      event.dataTransfer.dropEffect = "copy";
      mark(stack);
    });
    grid2.addEventListener("dragleave", (event) => {
      if (carried !== null && !(event.relatedTarget instanceof Node && grid2.contains(event.relatedTarget))) {
        unmark();
      }
    });
    grid2.addEventListener("drop", (event) => {
      const cell = cellOf2(event.target);
      const link = cell?.querySelector('a[href^="/key?"]');
      const place = cell === null ? null : placeOf(cell);
      if (carried === null || cell === null || link === null || link === void 0 || place === null || stackFrom(grid2, cell, carried) === null) {
        return;
      }
      event.preventDefault();
      unmark();
      setOpen(toggle, panel, false);
      if (carried.group !== null) {
        const query = new URLSearchParams({
          page: place.page,
          row: String(place.row),
          column: String(place.column),
          size: carried.group
        });
        openKeyDialog(`/group?${query.toString()}`, link);
        return;
      }
      const address = new URL(link.href);
      address.searchParams.set("type", carried.type ?? "");
      openKeyDialog(address.pathname + address.search, link);
    });
    drawer.hidden = false;
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

  // internal/editor/web/render.ts
  var oneYearSeconds = 60 * 60 * 24 * 365;
  function enableRenderToggle(page) {
    const toggle = page.querySelector("button.render-switch");
    if (toggle === null) {
      return;
    }
    toggle.addEventListener("click", () => {
      const on = toggle.getAttribute("aria-checked") !== "true";
      toggle.setAttribute("aria-checked", String(on));
      page.cookie = `deck_render=${on ? "on" : "off"}; path=/; max-age=${oneYearSeconds}; samesite=strict`;
      window.location.reload();
    });
    toggle.hidden = false;
  }

  // internal/editor/web/select.ts
  var tileAttribute2 = "data-tile";
  var keyState = "key";
  function placeOf2(cell) {
    const parts = (cell?.getAttribute(tileAttribute2) ?? "").split(".");
    if (parts.length !== 3 || parts.some((part) => !/^\d+$/.test(part))) {
      return null;
    }
    return { page: parts[0], row: Number(parts[1]), column: Number(parts[2]) };
  }
  function cellOf3(target) {
    return target instanceof Element ? target.closest(`td[${tileAttribute2}]`) : null;
  }
  function showHints() {
    for (const hint of document.querySelectorAll(".select-help")) {
      hint.hidden = false;
    }
  }
  function openSelecting(place) {
    const address = new URLSearchParams({ page: place.page, select: "1", tile: `${place.row}.${place.column}` });
    window.location.assign(`/page?${address.toString()}`);
  }
  function enableShiftTicking(grid2) {
    grid2.addEventListener("click", (event) => {
      const box = cellOf3(event.target)?.querySelector('.select-key input[type="checkbox"]');
      if (!event.shiftKey || box === null || box === void 0 || event.target === box) {
        return;
      }
      event.preventDefault();
      box.checked = !box.checked;
    });
  }
  function enableEscapeToLeave() {
    const done = document.querySelector("a.select-done");
    if (done === null) {
      return;
    }
    document.addEventListener("keydown", (event) => {
      const typing = event.target instanceof HTMLElement && event.target.matches('input:not([type="checkbox"]), textarea, select, [contenteditable]');
      if (event.key !== "Escape" || event.defaultPrevented || typing || document.querySelector("dialog[open]") !== null) {
        return;
      }
      event.preventDefault();
      window.location.assign(done.href);
    });
  }
  function enableSelecting(grid2) {
    showHints();
    if (grid2.querySelector(".select-key") !== null) {
      enableShiftTicking(grid2);
      enableEscapeToLeave();
      return;
    }
    grid2.addEventListener("click", (event) => {
      const cell = cellOf3(event.target);
      const place = placeOf2(cell);
      if (cell !== null && place !== null && event.shiftKey && event.button === 0 && cell.getAttribute("data-state") === keyState) {
        event.preventDefault();
        event.stopPropagation();
        openSelecting(place);
      }
    }, true);
  }

  // internal/editor/web/theme.ts
  var oneYearSeconds2 = 60 * 60 * 24 * 365;
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
      page.cookie = `theme=${chosen}; path=/; max-age=${oneYearSeconds2}; samesite=strict`;
      show();
    });
    system.addEventListener("change", show);
    show();
    toggle.hidden = false;
  }

  // internal/editor/web/editor.ts
  enablePickers(document);
  enableReplacements(document);
  enableSliders(document);
  enableGroupOrder(document);
  enablePlatformSwitch(document);
  enableThemeToggle(document);
  enableRenderToggle(document);
  var grid = document.querySelector("table.grid");
  if (grid !== null) {
    enableSelecting(grid);
    enableDrag(grid);
    enableKeyDialog(grid);
    enableDrawer(grid);
  }
})();

(() => {
  // internal/editor/web/inplace.ts
  var tileAttribute = "data-tile";
  var decodeMilliseconds = 2e3;
  function shownGrid() {
    return document.querySelector("table.grid:not(.grid-left-behind)");
  }
  function pageOf(grid2) {
    return grid2?.querySelector(`td[${tileAttribute}]`)?.getAttribute(tileAttribute)?.split(".")[0] ?? null;
  }
  function pageNames(root) {
    return [...root.querySelectorAll('nav[aria-label="Pages"] a')].map((link) => link.textContent ?? "").join("|");
  }
  function preload(address) {
    return new Promise((resolve) => {
      const image = new Image();
      const timer = window.setTimeout(resolve, decodeMilliseconds);
      const done = () => {
        window.clearTimeout(timer);
        resolve();
      };
      image.addEventListener("load", done);
      image.addEventListener("error", done);
      image.src = address;
    });
  }
  function fresh(cell) {
    return [...cell.querySelectorAll("img.deck-face")].map((picture) => {
      const address = new URL(picture.getAttribute("src") ?? "", window.location.href);
      address.searchParams.set("drawn", String(Date.now()));
      const source = address.pathname + address.search;
      return preload(source).then(() => {
        picture.src = source;
      });
    });
  }
  function sameTile(now, next) {
    return now.getAttribute("data-state") === next.getAttribute("data-state") && now.innerHTML === next.innerHTML;
  }
  function replaceRegion(incoming, selector) {
    const now = document.querySelector(selector);
    const next = incoming.querySelector(selector);
    if (now !== null && next !== null) {
      now.replaceChildren(...[...next.childNodes].map((node) => document.importNode(node, true)));
    } else if (now !== null) {
      now.remove();
    } else if (next !== null) {
      document.querySelector(".announcement")?.after(document.importNode(next, true));
    }
  }
  async function updateInPlace(page, address, refresh = true) {
    const incoming = new DOMParser().parseFromString(page, "text/html");
    const nextGrid = incoming.querySelector("table.grid");
    const grid2 = shownGrid();
    if (grid2 === null || nextGrid === null || document.querySelector(".grid-left-behind") !== null || pageOf(grid2) !== pageOf(nextGrid) || pageNames(document) !== pageNames(incoming) || document.querySelector("dialog[open]") !== null) {
      return false;
    }
    const changed = [];
    for (const next of nextGrid.querySelectorAll(`td[${tileAttribute}]`)) {
      const now = grid2.querySelector(`td[${tileAttribute}="${next.getAttribute(tileAttribute)}"]`);
      if (now === null) {
        return false;
      }
      if (!sameTile(now, next)) {
        changed.push([now, document.importNode(next, true)]);
      }
    }
    if (refresh) {
      await Promise.all(changed.flatMap(([, next]) => fresh(next)));
    }
    const focused = document.activeElement?.closest(`td[${tileAttribute}]`)?.getAttribute(tileAttribute) ?? null;
    for (const [now, next] of changed) {
      now.replaceWith(next);
    }
    if (focused !== null && document.activeElement === document.body) {
      grid2.querySelector(`td[${tileAttribute}="${focused}"] a, td[${tileAttribute}="${focused}"] input`)?.focus();
    }
    const caption = grid2.querySelector("caption");
    const nextCaption = nextGrid.querySelector("caption");
    if (caption !== null && nextCaption !== null) {
      caption.textContent = nextCaption.textContent;
    }
    replaceRegion(incoming, ".announcement");
    replaceRegion(incoming, ".history");
    replaceRegion(incoming, "#select-drawer > .maintenance-body");
    const drawer = document.querySelector("#select-drawer");
    if (drawer !== null && incoming.querySelector("#select-drawer")?.hasAttribute("open") === true) {
      drawer.open = true;
    }
    replaceRegion(incoming, ".problems");
    window.history.replaceState(null, "", address);
    document.dispatchEvent(new CustomEvent("front-buttons:updated"));
    return true;
  }

  // internal/editor/web/drag.ts
  var tileAttribute2 = "data-tile";
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
    return target instanceof Element ? target.closest(`td[${tileAttribute2}]`) : null;
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
    const problems = [...parsed.querySelectorAll(".problems li")].map((item) => item.textContent?.trim() ?? "");
    return alert?.textContent?.trim() || problems.join(" ") || "Error: the change could not be made.";
  }
  async function send(path, fields) {
    const token = document.querySelector('input[name="token"]')?.value;
    if (token === void 0) {
      announce("Error: this page has no editor token; reload it and try again.");
      return false;
    }
    const body = new URLSearchParams({ token });
    for (const [name, value] of Object.entries(fields)) {
      for (const each of typeof value === "string" ? [value] : value) {
        body.append(name, each);
      }
    }
    try {
      const response = await fetch(path, { method: "POST", body, credentials: "same-origin" });
      if (response.ok && response.redirected) {
        if (!await updateInPlace(await response.text(), response.url)) {
          window.location.assign(response.url);
        }
        return true;
      }
      announce(refusalFrom(await response.text()));
    } catch {
      announce("Error: the editor could not be reached; is the daemon running?");
    }
    return false;
  }
  function swap(source, destination) {
    return send("/key/move", {
      page: source.page,
      row: source.row,
      column: source.column,
      to: destination,
      action: "swap"
    });
  }
  var swapClass = "swap-target";
  var swapGhostClass = "swap-ghost";
  var swapHiddenClass = "swap-hidden";
  var swapSlideMilliseconds = 150;
  function reducedMotion() {
    return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  }
  function slideGhost(from, to) {
    const key = from.querySelector("a");
    if (key === null) {
      return null;
    }
    const ghost = key.cloneNode(true);
    ghost.removeAttribute("href");
    ghost.removeAttribute("draggable");
    ghost.setAttribute("aria-hidden", "true");
    ghost.tabIndex = -1;
    ghost.classList.add(swapGhostClass);
    to.append(ghost);
    if (!reducedMotion()) {
      const start = from.getBoundingClientRect();
      const end = to.getBoundingClientRect();
      ghost.animate(
        [{ transform: `translate(${start.left - end.left}px, ${start.top - end.top}px)` }, { transform: "none" }],
        { duration: swapSlideMilliseconds, easing: "ease-out" }
      );
    }
    return ghost;
  }
  function showSwap(source, target) {
    const ghosts = [slideGhost(target, source), slideGhost(source, target)].filter((ghost) => ghost !== null);
    source.classList.add(swapHiddenClass);
    target.classList.add(swapHiddenClass);
    return { target, source, ghosts };
  }
  function hideSwap(preview) {
    const back = (ghost, home) => {
      if (reducedMotion()) {
        return Promise.resolve();
      }
      const here = ghost.getBoundingClientRect();
      const there = home.getBoundingClientRect();
      return ghost.animate(
        [{ transform: "none" }, { transform: `translate(${there.left - here.left}px, ${there.top - here.top}px)` }],
        { duration: swapSlideMilliseconds, easing: "ease-in" }
      ).finished.then(() => void 0, () => void 0);
    };
    const homes = [preview.target, preview.source];
    void Promise.all(preview.ghosts.map((ghost, index) => back(ghost, homes[index]))).then(() => {
      for (const ghost of preview.ghosts) {
        ghost.remove();
      }
      for (const cell of homes) {
        if (cell.querySelector(`.${swapGhostClass}`) === null) {
          cell.classList.remove(swapHiddenClass);
        }
      }
    });
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
  function groupOf(cell, source) {
    const box = cell.querySelector('.select-key input[type="checkbox"]');
    if (box === null || source === null) {
      return null;
    }
    box.checked = true;
    const tiles = [...document.querySelectorAll('.select-key input[name="tile"]:checked')].map((ticked) => ticked.value);
    return { source, tiles };
  }
  function carriesTile(group, tile) {
    return group !== null && tile !== null && tile.page === group.source.page && group.tiles.includes(`${tile.row}.${tile.column}`);
  }
  function relocateGroup(group, destination, copying2) {
    return send(copying2 ? "/keys/copy" : "/keys/move", {
      page: group.source.page,
      tile: group.tiles,
      to: destination.page,
      row_offset: String(Number(destination.row) - Number(group.source.row)),
      column_offset: String(Number(destination.column) - Number(group.source.column))
    });
  }
  function removeGroup(group) {
    return send("/keys/remove", { page: group.source.page, tile: group.tiles });
  }
  var ghostClass = "ghost";
  var ghostFitsClass = "ghost-fits";
  var ghostBlockedClass = "ghost-blocked";
  function ghostPicture(cell) {
    const ghost = document.createElement("span");
    ghost.className = ghostClass;
    ghost.setAttribute("aria-hidden", "true");
    for (const node of cell.querySelector(".select-key label")?.childNodes ?? []) {
      ghost.append(node.cloneNode(true));
    }
    return ghost;
  }
  function clearGhosts(root) {
    for (const ghost of root.querySelectorAll(`.${ghostClass}`)) {
      ghost.remove();
    }
    for (const cell of root.querySelectorAll(`.${ghostFitsClass}, .${ghostBlockedClass}`)) {
      cell.classList.remove(ghostFitsClass, ghostBlockedClass);
    }
  }
  function showGhosts(group, grid2, destination, pictures) {
    const rowShift = Number(destination.row) - Number(group.source.row);
    const columnShift = Number(destination.column) - Number(group.source.column);
    const landed = [];
    let fits = true;
    let offDeck = false;
    for (const tile of group.tiles) {
      const [row, column] = tile.split(".").map(Number);
      const landing = `${destination.page}.${row + rowShift}.${column + columnShift}`;
      const cell = grid2.querySelector(`td[${tileAttribute2}="${landing}"]`);
      if (cell === null) {
        fits = false;
        offDeck = true;
        continue;
      }
      const free = stateOf(cell) === "empty" || carriesTile(group, parseTile(landing));
      fits && (fits = free);
      cell.classList.add(free ? ghostFitsClass : ghostBlockedClass);
      landed.push(cell);
      const picture = pictures.get(tile);
      if (picture !== void 0) {
        cell.append(picture.cloneNode(true));
      }
    }
    if (offDeck) {
      for (const cell of landed) {
        cell.classList.replace(ghostFitsClass, ghostBlockedClass);
      }
    }
    return fits;
  }
  function groupDragImage(group, event) {
    const cells = group.tiles.map((tile) => document.querySelector(`td[${tileAttribute2}="${group.source.page}.${tile}"]`)).filter((cell) => cell !== null);
    if (cells.length === 0 || event.dataTransfer === null) {
      return;
    }
    const boxes = cells.map((cell) => cell.getBoundingClientRect());
    const left = Math.min(...boxes.map((box) => box.left));
    const top = Math.min(...boxes.map((box) => box.top));
    const image = document.createElement("div");
    image.className = "grid group-drag-image";
    image.style.width = `${Math.max(...boxes.map((box) => box.right)) - left}px`;
    image.style.height = `${Math.max(...boxes.map((box) => box.bottom)) - top}px`;
    cells.forEach((cell, index) => {
      const picture = cell.querySelector(".select-key")?.cloneNode(true);
      if (!(picture instanceof HTMLElement)) {
        return;
      }
      picture.querySelector("input")?.remove();
      picture.querySelector("label")?.removeAttribute("for");
      picture.style.left = `${boxes[index].left - left}px`;
      picture.style.top = `${boxes[index].top - top}px`;
      picture.style.width = `${boxes[index].width}px`;
      picture.style.height = `${boxes[index].height}px`;
      image.append(picture);
    });
    document.body.append(image);
    event.dataTransfer.setDragImage(image, event.clientX - left, event.clientY - top);
    window.setTimeout(() => image.remove(), 0);
  }
  function markGroup(group, root, marked) {
    for (const tile of group.tiles) {
      root.querySelector(`td[${tileAttribute2}="${group.source.page}.${tile}"]`)?.classList.toggle(draggingClass, marked);
    }
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
    for (const cell of grid2.querySelectorAll(`.${dropTargetClass}, .${swapClass}`)) {
      cell.classList.remove(dropTargetClass, swapClass);
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
  function shownGrid2() {
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
      const current = shownGrid2();
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
    let group = null;
    let pictures = /* @__PURE__ */ new Map();
    let ghostedAt = "";
    let ghostFits = false;
    let swapping = null;
    const stopSwapping = () => {
      if (swapping !== null) {
        hideSwap(swapping);
        swapping = null;
      }
    };
    let shownPage3 = "";
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
        const before = shownPage3;
        shownPage3 = page;
        void openPageDuringDrag(page).then((done) => {
          opened || (opened = done);
          if (!done) {
            shownPage3 = before;
          }
        });
      }, holdMilliseconds);
    };
    const accepts = (cell) => stateOf(cell) === "empty" || carriesTile(group, parseTile(cell.getAttribute(tileAttribute2)));
    const swapsWith = (cell) => {
      const tile = parseTile(cell.getAttribute(tileAttribute2));
      return group === null && source !== null && stateOf(cell) === "key" && tile !== null && !(tile.page === source.page && tile.row === source.row && tile.column === source.column);
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
        void (group === null ? remove(carried) : removeGroup(group));
      });
    }
    root.addEventListener("dragstart", (event) => {
      const cell = cellOf(event.target);
      if (cell === null || stateOf(cell) !== "key" || event.dataTransfer === null) {
        return;
      }
      source = parseTile(cell.getAttribute(tileAttribute2));
      group = groupOf(cell, source);
      shownPage3 = source?.page ?? "";
      opened = false;
      dropped = false;
      event.dataTransfer.effectAllowed = "copyMove";
      event.dataTransfer.setData(tileType, cell.getAttribute(tileAttribute2) ?? "");
      cell.classList.add(draggingClass);
      if (group !== null) {
        const carried = group;
        groupDragImage(carried, event);
        markGroup(carried, root, true);
        pictures = new Map(carried.tiles.flatMap((tile) => {
          const ticked = root.querySelector(`td[${tileAttribute2}="${carried.source.page}.${tile}"]`);
          return ticked === null ? [] : [[tile, ghostPicture(ticked)]];
        }));
      }
      const shown = shownGrid2();
      if (bin !== null && shown !== null) {
        showBin(bin, shown);
      }
    });
    root.addEventListener("dragend", (event) => {
      cellOf(event.target)?.classList.remove(draggingClass);
      if (group !== null) {
        markGroup(group, root, false);
      }
      clearDropTargets(root);
      clearGhosts(root);
      ghostedAt = "";
      stopSwapping();
      disarm();
      source = null;
      group = null;
      if (bin !== null) {
        hideBin(bin);
      }
      if (opened && !dropped) {
        window.location.assign(`/page?page=${shownPage3}`);
      }
    });
    root.addEventListener("dragover", (event) => {
      if (source === null || event.dataTransfer === null) {
        return;
      }
      const link = pageLinkOf(event.target);
      if (link !== null && link.getAttribute(pageAttribute) !== shownPage3) {
        clearGhosts(root);
        ghostedAt = "";
        event.preventDefault();
        showCopying(event);
        arm(link);
        return;
      }
      const cell = cellOf(event.target);
      if (group !== null) {
        const tile = cell?.getAttribute(tileAttribute2) ?? "";
        const destination = parseTile(tile);
        const shown = cell?.closest("table.grid") ?? null;
        if (tile !== ghostedAt) {
          clearGhosts(root);
          ghostedAt = tile;
          ghostFits = destination !== null && shown !== null && showGhosts(group, shown, destination, pictures);
        }
        if (ghostFits) {
          event.preventDefault();
          showCopying(event);
        }
        return;
      }
      if (cell !== null && swapsWith(cell)) {
        event.preventDefault();
        const home = source === null ? null : shownGrid2()?.querySelector(
          `td[${tileAttribute2}="${source.page}.${source.row}.${source.column}"]`
        ) ?? null;
        if (home === null) {
          cell.classList.add(swapClass);
        } else if (swapping?.target !== cell) {
          stopSwapping();
          swapping = showSwap(home, cell);
        }
        return;
      }
      stopSwapping();
      if (cell === null || !accepts(cell)) {
        return;
      }
      event.preventDefault();
      showCopying(event);
      cell.classList.add(dropTargetClass);
    });
    root.addEventListener("dragleave", (event) => {
      cellOf(event.target)?.classList.remove(dropTargetClass, swapClass);
      if (armed !== null && pageLinkOf(event.target) === armed && !armed.contains(event.relatedTarget)) {
        disarm();
      }
    });
    root.addEventListener("drop", (event) => {
      if (source === null) {
        return;
      }
      const link = pageLinkOf(event.target);
      if (link !== null && link.getAttribute(pageAttribute) !== shownPage3) {
        event.preventDefault();
        disarm();
        dropped = true;
        const sameTile2 = `${link.getAttribute(pageAttribute) ?? ""}.${source.row}.${source.column}`;
        const target2 = parseTile(sameTile2);
        void (group !== null && target2 !== null ? relocateGroup(group, target2, copying(event)) : relocate(source, sameTile2, copying(event)));
        return;
      }
      const cell = cellOf(event.target);
      const destination = cell?.getAttribute(tileAttribute2) ?? null;
      const target = parseTile(destination);
      if (cell !== null && destination !== null && swapsWith(cell)) {
        event.preventDefault();
        clearDropTargets(root);
        dropped = true;
        const shown = swapping;
        swapping = null;
        if (copying(event)) {
          announce("Error: a copy needs a free tile; let go of Command to swap the two keys instead.");
          if (shown !== null) {
            hideSwap(shown);
          }
        } else {
          void swap(source, destination).then((accepted) => {
            if (!accepted && shown !== null) {
              hideSwap(shown);
            }
          });
        }
        return;
      }
      if (cell === null || target === null || (group === null ? !accepts(cell) : !ghostFits)) {
        return;
      }
      event.preventDefault();
      clearDropTargets(root);
      clearGhosts(root);
      dropped = true;
      void (group !== null ? relocateGroup(group, target, copying(event)) : relocate(source, destination, copying(event)));
    });
  }

  // internal/editor/web/clipboard.ts
  var clipboardKey = "front-buttons-keys";
  var tileAttribute3 = "data-tile";
  function save(copied) {
    try {
      window.sessionStorage.setItem(clipboardKey, JSON.stringify(copied));
    } catch {
      announce("Error: this browser keeps nothing for the page, so keys cannot be copied this way.");
    }
  }
  function load() {
    try {
      const stored = window.sessionStorage.getItem(clipboardKey);
      const copied = stored === null ? null : JSON.parse(stored);
      if (typeof copied?.page !== "string" || !Array.isArray(copied.tiles) || typeof copied.cut !== "boolean") {
        return null;
      }
      return { page: copied.page, tiles: copied.tiles.filter((tile) => typeof tile === "string"), cut: copied.cut };
    } catch {
      return null;
    }
  }
  function forget() {
    try {
      window.sessionStorage.removeItem(clipboardKey);
    } catch {
    }
  }
  function shownGrid3() {
    return document.querySelector("table.grid:not(.grid-left-behind)");
  }
  function focusedTile() {
    const cell = document.activeElement?.closest(`table.grid td[${tileAttribute3}]`);
    const parts = (cell?.getAttribute(tileAttribute3) ?? "").split(".");
    return parts.length === 3 ? parts : null;
  }
  function toCopy(cut) {
    const ticked = [...document.querySelectorAll('.select-key input[name="tile"]:checked')];
    const page = shownGrid3()?.querySelector(`td[${tileAttribute3}]`)?.getAttribute(tileAttribute3)?.split(".")[0];
    if (ticked.length > 0 && page !== void 0) {
      return { page, tiles: ticked.map((box) => box.value), cut };
    }
    const focused = focusedTile();
    const cell = document.activeElement?.closest(`td[${tileAttribute3}]`);
    if (focused === null || cell?.getAttribute("data-state") !== "key" || cell.querySelector(".select-key") !== null) {
      return null;
    }
    return { page: focused[0], tiles: [`${focused[1]}.${focused[2]}`], cut };
  }
  function keysWord(count) {
    return count === 1 ? "1 key" : `${count} keys`;
  }
  function copy(cut) {
    const copied = toCopy(cut);
    if (copied === null) {
      return false;
    }
    save(copied);
    announce(`${cut ? "Cut" : "Copied"} ${keysWord(copied.tiles.length)}: open a page and press Command-V, Ctrl-V on Windows, to paste on the same tiles, or focus a tile with the arrow keys to paste from there.`);
    return true;
  }
  async function paste() {
    const copied = load();
    const page = shownGrid3()?.querySelector(`td[${tileAttribute3}]`)?.getAttribute(tileAttribute3)?.split(".")[0];
    if (copied === null || copied.tiles.length === 0 || page === void 0) {
      announce("Error: no keys are copied; press Command-C, Ctrl-C on Windows, on a key or the ticked keys first.");
      return;
    }
    let rowShift = 0;
    let columnShift = 0;
    const focused = focusedTile();
    if (focused !== null) {
      const places = copied.tiles.map((tile) => tile.split(".").map(Number));
      rowShift = Number(focused[1]) - Math.min(...places.map((place) => place[0]));
      columnShift = Number(focused[2]) - Math.min(...places.map((place) => place[1]));
    }
    if (page === copied.page && rowShift === 0 && columnShift === 0) {
      announce("Error: those keys are already on these tiles; open another page, or focus a tile with the arrow keys to paste from there.");
      return;
    }
    if (copied.cut) {
      forget();
    }
    const accepted = await send(copied.cut ? "/keys/move" : "/keys/copy", {
      page: copied.page,
      tile: copied.tiles,
      to: page,
      showing: page,
      row_offset: String(rowShift),
      column_offset: String(columnShift)
    });
    if (!accepted && copied.cut) {
      save(copied);
    }
  }
  function enableClipboard() {
    document.addEventListener("keydown", (event) => {
      const key = event.key.toLowerCase();
      const typing = event.target instanceof HTMLElement && event.target.matches('input:not([type="checkbox"]), textarea, select, [contenteditable]');
      const selecting2 = (window.getSelection()?.toString() ?? "") !== "";
      if (!(event.metaKey || event.ctrlKey) || event.altKey || event.shiftKey || typing || selecting2 || document.querySelector("dialog[open]") !== null || !["c", "x", "v"].includes(key)) {
        return;
      }
      if (key === "v") {
        event.preventDefault();
        void paste();
        return;
      }
      if (copy(key === "x")) {
        event.preventDefault();
      }
    });
  }

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
  async function load2(parts, url, init) {
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
      void load2(parts, link.href);
    });
    openFrom = (address, from) => {
      opener = from;
      void load2(parts, address);
    };
    parts.body.addEventListener("submit", (event) => {
      const form = event.target;
      if (!(form instanceof HTMLFormElement)) {
        return;
      }
      event.preventDefault();
      const parameters = formParameters(form, event.submitter);
      if (form.method.toLowerCase() === "get") {
        void load2(parts, `${form.action}?${parameters.toString()}`);
      } else {
        void load2(parts, form.action, { method: "POST", body: parameters });
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
  function enableSearch(panel) {
    const box = panel.querySelector("input.drawer-search");
    const empty = panel.querySelector(".drawer-empty");
    const sections = Array.from(panel.querySelectorAll("details.drawer-section"));
    if (box === null) {
      return null;
    }
    const wasOpen = new Map(sections.map((section) => [section, section.open]));
    const narrow = () => {
      const words = box.value.toLowerCase().split(/\s+/).filter((word) => word !== "");
      let matches = 0;
      for (const section of sections) {
        const sectionName = section.querySelector("summary")?.textContent?.toLowerCase() ?? "";
        let shown = 0;
        for (const item of section.querySelectorAll("li[draggable]")) {
          const kindName = (item.dataset.type ?? item.dataset.group ?? "").replace(/_/g, " ");
          const haystack = `${sectionName} ${item.textContent?.toLowerCase() ?? ""} ${kindName.toLowerCase()}`;
          const match = words.every((word) => haystack.includes(word));
          item.hidden = !match;
          shown += match ? 1 : 0;
        }
        section.hidden = shown === 0;
        section.open = words.length === 0 ? wasOpen.get(section) ?? false : shown > 0;
        matches += shown;
      }
      if (empty !== null) {
        empty.hidden = matches > 0;
      }
    };
    box.addEventListener("input", narrow);
    return {
      box,
      clear: () => {
        if (box.value === "") {
          return false;
        }
        box.value = "";
        narrow();
        return true;
      }
    };
  }
  function enableEscapeToClose(toggle, panel, clear) {
    document.addEventListener("keydown", (event) => {
      const target = event.target instanceof HTMLElement ? event.target : null;
      const typingElsewhere = target !== null && !panel.contains(target) && target.matches('input:not([type="checkbox"]), textarea, select, [contenteditable]');
      if (event.key !== "Escape" || event.defaultPrevented || panel.hidden || typingElsewhere || document.querySelector("dialog[open]") !== null) {
        return;
      }
      event.preventDefault();
      event.stopPropagation();
      if (!clear()) {
        setOpen(toggle, panel, false);
        toggle.focus();
      }
    }, true);
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
    const chars = Number(panel.dataset.chars);
    if (Number.isInteger(chars) && chars > 0) {
      panel.style.setProperty("--drawer-chars", String(chars));
    }
    const search = enableSearch(panel);
    enableEscapeToClose(toggle, panel, search?.clear ?? (() => false));
    toggle.addEventListener("click", () => {
      setOpen(toggle, panel, panel.hidden);
      if (!panel.hidden) {
        search?.box.focus();
      }
    });
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

  // internal/editor/web/keepplace.ts
  var placeKey = "front-buttons-grid-place";
  var freshMilliseconds = 1e4;
  function keep(grid2) {
    try {
      const kept = { top: grid2.getBoundingClientRect().top, at: Date.now() };
      window.sessionStorage.setItem(placeKey, JSON.stringify(kept));
    } catch {
    }
  }
  function takeKept() {
    try {
      const stored = window.sessionStorage.getItem(placeKey);
      window.sessionStorage.removeItem(placeKey);
      const kept = stored === null ? null : JSON.parse(stored);
      if (typeof kept?.top !== "number" || typeof kept.at !== "number" || Date.now() - kept.at > freshMilliseconds) {
        return null;
      }
      return { top: kept.top, at: kept.at };
    } catch {
      return null;
    }
  }
  function restore(grid2, kept) {
    const wanted = window.scrollY + grid2.getBoundingClientRect().top - kept.top;
    const room = wanted - (document.documentElement.scrollHeight - window.innerHeight);
    if (room > 0) {
      document.documentElement.style.setProperty("--keep-room", `${Math.ceil(room)}px`);
    }
    window.scrollTo(0, Math.max(Math.round(wanted), 0));
  }
  function enableKeepPlace(grid2) {
    const kept = takeKept();
    if (kept !== null) {
      restore(grid2, kept);
    }
    window.addEventListener("pagehide", () => {
      keep(document.querySelector("table.grid:not(.grid-left-behind)") ?? grid2);
    });
  }

  // internal/editor/web/live.ts
  var heldClass = "held";
  var holdingClass = "holding";
  var litMilliseconds = 2e3;
  var shownMilliseconds = 3e3;
  var holdTimers = /* @__PURE__ */ new Map();
  function clearTimers(tile) {
    for (const timer of holdTimers.get(tile) ?? []) {
      window.clearTimeout(timer);
    }
    holdTimers.delete(tile);
  }
  function shownTile(tile) {
    if (!/^\d+\.\d+\.\d+$/.test(tile)) {
      return null;
    }
    return document.querySelector(`table.grid:not(.grid-left-behind) td[data-tile="${tile}"]`);
  }
  function refetch(picture) {
    const address = new URL(picture.src, window.location.href);
    if (address.searchParams.get("live") !== "1") {
      return;
    }
    address.searchParams.set("drawn", String(Date.now()));
    picture.src = address.pathname + address.search;
  }
  function press(tile) {
    const cell = shownTile(tile);
    if (cell === null) {
      return;
    }
    clearTimers(tile);
    cell.classList.remove(holdingClass);
    cell.classList.add(heldClass);
    holdTimers.set(tile, [
      window.setTimeout(() => cell.classList.add(holdingClass), litMilliseconds),
      window.setTimeout(() => release(tile), shownMilliseconds)
    ]);
  }
  function release(tile) {
    clearTimers(tile);
    shownTile(tile)?.classList.remove(heldClass, holdingClass);
  }
  function enableLive(page) {
    if (page.documentElement.dataset["render"] !== "deck" || page.querySelector("table.grid") === null) {
      return;
    }
    const stream = new EventSource("/events");
    stream.addEventListener("press", (event) => press(event.data));
    stream.addEventListener("release", (event) => release(event.data));
    stream.addEventListener("frame", (event) => {
      const picture = shownTile(event.data)?.querySelector("img.deck-face");
      if (picture !== null && picture !== void 0) {
        refetch(picture);
      }
    });
    stream.addEventListener("reload", () => {
      for (const picture of page.querySelectorAll("table.grid img.deck-face")) {
        refetch(picture);
      }
    });
    window.addEventListener("pagehide", () => stream.close());
  }

  // internal/editor/web/find.ts
  var waitMilliseconds = 200;
  function shownPage() {
    return document.querySelector("table.grid:not(.grid-left-behind) td[data-tile]")?.getAttribute("data-tile")?.split(".")[0] ?? "0";
  }
  async function showResults(search) {
    const results = document.getElementById("find-results");
    if (results === null) {
      return;
    }
    if (search.trim() === "") {
      results.replaceChildren();
      return;
    }
    try {
      const address = `/page?${new URLSearchParams({ page: shownPage(), find: search }).toString()}`;
      const response = await fetch(address, { credentials: "same-origin" });
      const incoming = new DOMParser().parseFromString(await response.text(), "text/html").getElementById("find-results");
      if (response.ok && incoming !== null) {
        results.replaceChildren(...[...incoming.childNodes].map((node) => document.importNode(node, true)));
      }
    } catch {
    }
  }
  function enableFind() {
    const drawer = document.getElementById("find-drawer");
    const box = document.getElementById("find-box");
    if (!(drawer instanceof HTMLDetailsElement) || !(box instanceof HTMLInputElement)) {
      return;
    }
    for (const button of document.querySelectorAll(".find-submit")) {
      button.hidden = true;
    }
    const found = document.querySelector("table.grid td.found");
    if (found !== null) {
      window.setTimeout(() => found.scrollIntoView({ block: "center" }), 0);
    }
    let timer;
    box.addEventListener("input", () => {
      window.clearTimeout(timer);
      timer = window.setTimeout(() => void showResults(box.value), waitMilliseconds);
    });
    box.addEventListener("keydown", (event) => {
      if (event.key === "Enter") {
        const first = document.querySelector("#find-results a");
        event.preventDefault();
        first?.click();
      } else if (event.key === "Escape") {
        event.preventDefault();
        event.stopPropagation();
        if (box.value !== "") {
          box.value = "";
          void showResults("");
        } else {
          drawer.open = false;
          drawer.querySelector("summary")?.focus();
        }
      }
    });
    document.addEventListener("keydown", (event) => {
      const typing = event.target instanceof HTMLElement && event.target.matches('input:not([type="checkbox"]), textarea, select, [contenteditable]');
      if (event.key !== "/" || typing || event.metaKey || event.ctrlKey || event.altKey || document.querySelector("dialog[open]") !== null) {
        return;
      }
      event.preventDefault();
      drawer.open = true;
      box.focus();
      box.select();
    });
  }

  // internal/editor/web/gridkeys.ts
  var tileAttribute4 = "data-tile";
  var focusableSelector = 'a[href], input[type="checkbox"]';
  var steps = {
    ArrowUp: [-1, 0],
    ArrowDown: [1, 0],
    ArrowLeft: [0, -1],
    ArrowRight: [0, 1]
  };
  function cellOf3(target) {
    return target instanceof Element ? target.closest(`td[${tileAttribute4}]`) : null;
  }
  function focusable(cell) {
    return cell?.querySelector(focusableSelector) ?? null;
  }
  function nextTile(grid2, cell, step) {
    const row = cell.parentElement instanceof HTMLTableRowElement ? cell.parentElement.sectionRowIndex : -1;
    const rows = grid2.tBodies[0]?.rows ?? grid2.rows;
    for (let rowIndex = row + step[0], column = cell.cellIndex + step[1]; rowIndex >= 0 && rowIndex < rows.length && column >= 0 && column < rows[rowIndex].cells.length; rowIndex += step[0], column += step[1]) {
      const target = focusable(rows[rowIndex].cells[column]);
      if (target !== null) {
        return target;
      }
    }
    return null;
  }
  function removeKey(cell) {
    const parts = (cell.getAttribute(tileAttribute4) ?? "").split(".");
    if (cell.getAttribute("data-state") !== "key" || cell.querySelector(".select-key") !== null || parts.length !== 3) {
      return false;
    }
    void send("/key/delete", { page: parts[0], row: parts[1], column: parts[2] });
    return true;
  }
  function enableGridKeys(grid2) {
    const root = grid2.closest("main") ?? document.body;
    root.addEventListener("keydown", (event) => {
      const cell = cellOf3(event.target);
      const table = cell?.closest("table.grid");
      if (cell === null || table === null || table === void 0 || event.altKey || event.ctrlKey || event.metaKey) {
        return;
      }
      const step = steps[event.key];
      if (step !== void 0) {
        event.preventDefault();
        nextTile(table, cell, step)?.focus();
        return;
      }
      if ((event.key === "Delete" || event.key === "Backspace") && removeKey(cell)) {
        event.preventDefault();
      }
    });
  }

  // internal/editor/web/history.ts
  function sendForm(form, submitter) {
    const fields = {};
    for (const [name, value] of new FormData(form, submitter)) {
      if (typeof value === "string" && name !== "token") {
        (fields[name] ?? (fields[name] = [])).push(value);
      }
    }
    const action = submitter?.getAttribute("formaction") ?? form.getAttribute("action") ?? "/undo";
    void send(action, fields);
  }
  function showingPage() {
    return document.querySelector('.history input[name="showing"]')?.value ?? "0";
  }
  function enableHistory() {
    document.addEventListener("submit", (event) => {
      if (event.target instanceof HTMLFormElement && event.target.matches("form.in-place")) {
        event.preventDefault();
        sendForm(event.target, event.submitter);
      }
    });
    document.addEventListener("keydown", (event) => {
      const typing = event.target instanceof HTMLElement && event.target.matches('input:not([type="checkbox"]), textarea, select, [contenteditable]');
      if (!(event.metaKey || event.ctrlKey) || event.altKey || event.key.toLowerCase() !== "z" || typing || document.querySelector("dialog[open]") !== null) {
        return;
      }
      event.preventDefault();
      void send(event.shiftKey ? "/redo" : "/undo", { showing: showingPage() });
    });
  }

  // internal/editor/web/pagedrag.ts
  var pageType = "application/x-front-buttons-page";
  var pageAttribute2 = "data-page";
  var movingClass = "reorder-moving";
  var slideMilliseconds = 150;
  var slack = 24;
  function pageLinkOf2(target) {
    const link = target instanceof Element ? target.closest(`nav a[${pageAttribute2}]`) : null;
    return link !== null && /^\d+$/.test(link.getAttribute(pageAttribute2) ?? "") ? link : null;
  }
  function itemsOf(list) {
    return [...list.children].filter((child) => child instanceof HTMLLIElement);
  }
  function pageOf2(item) {
    return item.querySelector(`a[${pageAttribute2}]`)?.getAttribute(pageAttribute2) ?? "";
  }
  function showingPage2() {
    return document.querySelector(`nav a[${pageAttribute2}][aria-current="page"]`)?.getAttribute(pageAttribute2) ?? "";
  }
  function reducedMotion2() {
    return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  }
  function slide(list, change) {
    const items = itemsOf(list);
    const before = new Map(items.map((item) => [item, item.getBoundingClientRect()]));
    change();
    if (reducedMotion2()) {
      return;
    }
    for (const item of items) {
      const was = before.get(item);
      const now = item.getBoundingClientRect();
      if (was === void 0 || was.left === now.left && was.top === now.top) {
        continue;
      }
      item.animate(
        [{ transform: `translate(${was.left - now.left}px, ${was.top - now.top}px)` }, { transform: "none" }],
        { duration: slideMilliseconds, easing: "ease-out" }
      );
    }
  }
  function within(list, x, y) {
    const box = list.getBoundingClientRect();
    return x >= box.left - slack && x <= box.right + slack && y >= box.top - slack && y <= box.bottom + slack;
  }
  function placeOf2(item) {
    return {
      left: item.offsetLeft,
      top: item.offsetTop,
      right: item.offsetLeft + item.offsetWidth,
      bottom: item.offsetTop + item.offsetHeight
    };
  }
  function newSpot(carried, x, y) {
    const box = carried.list.getBoundingClientRect();
    const pointX = x - box.left - carried.list.clientLeft;
    const pointY = y - box.top - carried.list.clientTop;
    const items = itemsOf(carried.list);
    const others = items.filter((item) => item !== carried.item);
    const last = others.at(-1);
    if (last !== void 0) {
      const end = placeOf2(last);
      if (pointY > end.bottom || pointY >= end.top && pointX > end.right) {
        return null;
      }
    }
    const over = others.find((item) => {
      const place2 = placeOf2(item);
      return pointX >= place2.left && pointX <= place2.right && pointY >= place2.top && pointY <= place2.bottom;
    });
    if (over === void 0) {
      return void 0;
    }
    const place = placeOf2(over);
    const pastMiddle = pointX > (place.left + place.right) / 2;
    const carriedFirst = items.indexOf(carried.item) < items.indexOf(over);
    if (carriedFirst && pastMiddle) {
      return over.nextElementSibling instanceof HTMLLIElement ? over.nextElementSibling : null;
    }
    return !carriedFirst && !pastMiddle ? over : void 0;
  }
  function moveTo(carried, spot) {
    const current = carried.item.nextElementSibling;
    if (spot === carried.item || spot === current || spot === null && current === null) {
      return;
    }
    slide(carried.list, () => carried.list.insertBefore(carried.item, spot));
  }
  function putBack(carried) {
    const byPage = new Map(itemsOf(carried.list).map((item) => [pageOf2(item), item]));
    slide(carried.list, () => {
      for (const page of carried.order) {
        const item = byPage.get(page);
        if (item !== void 0) {
          carried.list.append(item);
        }
      }
    });
  }
  function enablePageReorder(root) {
    let carried = null;
    let dropped = false;
    root.addEventListener("dragstart", (event) => {
      const link = pageLinkOf2(event.target);
      const item = link?.closest("li");
      const list = item?.parentElement;
      if (link === null || !(item instanceof HTMLLIElement) || !(list instanceof HTMLElement) || event.dataTransfer === null) {
        return;
      }
      carried = { item, list, order: itemsOf(list).map(pageOf2) };
      dropped = false;
      event.dataTransfer.effectAllowed = "move";
      event.dataTransfer.setData(pageType, pageOf2(item));
      window.setTimeout(() => item.classList.add(movingClass), 0);
    });
    root.addEventListener("dragover", (event) => {
      if (carried === null || event.dataTransfer === null || !within(carried.list, event.clientX, event.clientY)) {
        return;
      }
      event.preventDefault();
      event.dataTransfer.dropEffect = "move";
      const spot = newSpot(carried, event.clientX, event.clientY);
      if (spot !== void 0) {
        moveTo(carried, spot);
      }
    });
    root.addEventListener("drop", (event) => {
      if (carried === null || !within(carried.list, event.clientX, event.clientY)) {
        return;
      }
      event.preventDefault();
      dropped = true;
      const from = pageOf2(carried.item);
      const to = carried.order[itemsOf(carried.list).indexOf(carried.item)] ?? from;
      if (to !== from) {
        void send("/page/reorder", { page: from, to, showing: showingPage2() });
      }
    });
    root.addEventListener("dragend", () => {
      if (carried === null) {
        return;
      }
      carried.item.classList.remove(movingClass);
      if (!dropped) {
        putBack(carried);
      }
      carried = null;
    });
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

  // internal/editor/web/renamepage.ts
  var pageAttribute3 = "data-page";
  var inputClass = "rename-input";
  var maximumName = 24;
  function currentLink(target) {
    const link = target instanceof Element ? target.closest(`nav a[${pageAttribute3}][aria-current="page"]`) : null;
    return link !== null && /^\d+$/.test(link.getAttribute(pageAttribute3) ?? "") ? link : null;
  }
  function startRenaming(link) {
    const box = document.createElement("input");
    box.type = "text";
    box.className = inputClass;
    box.value = link.textContent ?? "";
    box.maxLength = maximumName;
    box.setAttribute("aria-label", `Rename page ${link.textContent ?? ""}`);
    let finished = false;
    const finish = (save2) => {
      if (finished) {
        return;
      }
      finished = true;
      const name = box.value.trim();
      box.remove();
      link.hidden = false;
      if (save2 && name !== "" && name !== link.textContent) {
        link.textContent = name;
        void send("/page/rename", { page: link.getAttribute(pageAttribute3) ?? "", name });
      } else {
        link.focus();
      }
    };
    box.addEventListener("keydown", (event) => {
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
  function enableRenamePage(root) {
    root.addEventListener("click", (event) => {
      if (currentLink(event.target) !== null && event.button === 0 && !event.metaKey && !event.ctrlKey) {
        event.preventDefault();
      }
    });
    root.addEventListener("dblclick", (event) => {
      const link = currentLink(event.target);
      if (link !== null) {
        event.preventDefault();
        startRenaming(link);
      }
    });
    for (const hint of root.querySelectorAll(".rename-help")) {
      hint.hidden = false;
    }
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
  var tileAttribute5 = "data-tile";
  var keyState = "key";
  function placeOf3(cell) {
    const parts = (cell?.getAttribute(tileAttribute5) ?? "").split(".");
    if (parts.length !== 3 || parts.some((part) => !/^\d+$/.test(part))) {
      return null;
    }
    return { page: parts[0], row: Number(parts[1]), column: Number(parts[2]) };
  }
  function cellOf4(target) {
    return target instanceof Element ? target.closest(`td[${tileAttribute5}]`) : null;
  }
  function selecting() {
    return document.querySelector("table.grid:not(.grid-left-behind) .select-key") !== null;
  }
  function shownPage2() {
    return document.querySelector(`table.grid:not(.grid-left-behind) td[${tileAttribute5}]`)?.getAttribute(tileAttribute5)?.split(".")[0] ?? "0";
  }
  function showHints() {
    for (const hint of document.querySelectorAll(".select-help")) {
      hint.hidden = false;
    }
    for (const apply of document.querySelectorAll(".select-apply")) {
      apply.hidden = true;
    }
  }
  async function switchMode(address) {
    try {
      const response = await fetch(address, { credentials: "same-origin" });
      if (response.ok && await updateInPlace(await response.text(), address, false)) {
        showHints();
        return;
      }
    } catch {
    }
    window.location.assign(address);
  }
  function modeAddress(on, tile = null) {
    const address = new URLSearchParams({ page: shownPage2() });
    if (on) {
      address.set("select", "1");
    }
    if (tile !== null) {
      address.set("tile", tile);
    }
    return `/page?${address.toString()}`;
  }
  function enableEditMode() {
    document.addEventListener("change", (event) => {
      if (event.target instanceof HTMLInputElement && event.target.id === "select-mode-box") {
        void switchMode(modeAddress(event.target.checked));
      }
    });
  }
  function enableShiftTicking() {
    document.addEventListener("click", (event) => {
      const box = cellOf4(event.target)?.querySelector('.select-key input[type="checkbox"]');
      if (!event.shiftKey || box === null || box === void 0 || event.target === box) {
        return;
      }
      event.preventDefault();
      box.checked = !box.checked;
    });
  }
  function enableEscapeToLeave() {
    document.addEventListener("keydown", (event) => {
      const typing = event.target instanceof HTMLElement && event.target.matches('input:not([type="checkbox"]), textarea, select, [contenteditable]');
      if (event.key !== "Escape" || event.defaultPrevented || typing || !selecting() || document.querySelector("dialog[open]") !== null) {
        return;
      }
      event.preventDefault();
      void switchMode(modeAddress(false));
    });
  }
  function enablePlainShiftClicks() {
    document.addEventListener("click", (event) => {
      const link = event.target instanceof Element ? event.target.closest("table.grid a[href]") : null;
      if (link === null || event.defaultPrevented || !event.shiftKey || event.button !== 0 || event.metaKey || event.ctrlKey || event.altKey) {
        return;
      }
      event.preventDefault();
      link.click();
    });
  }
  function enableSelecting(grid2) {
    showHints();
    enablePlainShiftClicks();
    enableEditMode();
    enableShiftTicking();
    enableEscapeToLeave();
    document.addEventListener("front-buttons:updated", showHints);
    grid2.closest("main")?.addEventListener("click", (event) => {
      const cell = cellOf4(event.target);
      const place = placeOf3(cell);
      if (cell !== null && place !== null && event.shiftKey && event.button === 0 && !selecting() && cell.getAttribute("data-state") === keyState) {
        event.preventDefault();
        event.stopPropagation();
        void switchMode(modeAddress(true, `${place.row}.${place.column}`));
      }
    }, true);
  }

  // internal/editor/web/shortcuts.ts
  function enableShortcuts() {
    const sheet = document.getElementById("shortcuts-dialog");
    if (!(sheet instanceof HTMLDialogElement)) {
      return;
    }
    const open = () => {
      if (!sheet.open) {
        sheet.showModal();
      }
    };
    document.querySelector(".shortcuts-open")?.addEventListener("click", open);
    sheet.querySelector(".dialog-close")?.addEventListener("click", () => sheet.close());
    document.addEventListener("keydown", (event) => {
      const typing = event.target instanceof HTMLElement && event.target.matches('input:not([type="checkbox"]), textarea, select, [contenteditable]');
      if (event.key !== "?" || typing || event.metaKey || event.ctrlKey || event.altKey || document.querySelector("dialog[open]") !== null) {
        return;
      }
      event.preventDefault();
      open();
    });
    for (const hint of document.querySelectorAll(".shortcuts-help")) {
      hint.hidden = false;
    }
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
  enableLive(document);
  enablePageReorder(document);
  enableRenamePage(document);
  var grid = document.querySelector("table.grid");
  if (grid !== null) {
    enableSelecting(grid);
    enableDrag(grid);
    enableGridKeys(grid);
    enableClipboard();
    enableHistory();
    enableShortcuts();
    enableFind();
    enableKeyDialog(grid);
    enableDrawer(grid);
    enableKeepPlace(grid);
  }
})();

(() => {
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
  async function relocate(source, destination, copying) {
    const token = document.querySelector('input[name="token"]')?.value;
    if (token === void 0) {
      announce("Error: this page has no editor token; reload it and try again.");
      return;
    }
    const body = new URLSearchParams({
      token,
      page: source.page,
      row: source.row,
      column: source.column,
      to: destination,
      action: copying ? "copy" : "move"
    });
    try {
      const response = await fetch("/key/move", { method: "POST", body, credentials: "same-origin" });
      if (response.ok && response.redirected) {
        window.location.assign(response.url);
        return;
      }
      announce(refusalFrom(await response.text()));
    } catch {
      announce("Error: the editor could not be reached; is the daemon running?");
    }
  }
  function clearDropTargets(grid2) {
    for (const cell of grid2.querySelectorAll(`.${dropTargetClass}`)) {
      cell.classList.remove(dropTargetClass);
    }
  }
  function enable(grid2) {
    let source = null;
    grid2.addEventListener("dragstart", (event) => {
      const cell = cellOf(event.target);
      if (cell === null || stateOf(cell) !== "key" || event.dataTransfer === null) {
        return;
      }
      source = parseTile(cell.getAttribute(tileAttribute));
      event.dataTransfer.effectAllowed = "copyMove";
      event.dataTransfer.setData(tileType, cell.getAttribute(tileAttribute) ?? "");
      cell.classList.add(draggingClass);
    });
    grid2.addEventListener("dragend", (event) => {
      cellOf(event.target)?.classList.remove(draggingClass);
      clearDropTargets(grid2);
      source = null;
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
  var grid = document.querySelector("table.grid");
  if (grid !== null) {
    enable(grid);
  }
})();

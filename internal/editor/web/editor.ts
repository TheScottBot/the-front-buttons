// The editor's script: everything here only adds to pages that work without
// it. esbuild bundles this file and the modules it imports into
// static/editor.js (scripts/build-editor-script.sh, run by go generate).

import { enableKeyDialog } from "./dialog";
import { enableDrag } from "./drag";
import { enableDrawer } from "./drawer";
import { enableGroupOrder } from "./group";
import { enablePickers } from "./picker";
import { enableReplacements } from "./replaces";
import { enableSliders } from "./slider";
import { enableThemeToggle } from "./theme";

enablePickers(document);
enableReplacements(document);
enableSliders(document);
enableGroupOrder(document);
enableThemeToggle(document);

const grid = document.querySelector<HTMLTableElement>("table.grid");

if (grid !== null) {
  enableDrag(grid);
  enableKeyDialog(grid);
  enableDrawer(grid);
}

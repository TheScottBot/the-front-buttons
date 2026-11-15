// The editor's script: everything here only adds to pages that work without
// it. esbuild bundles this file and the modules it imports into
// static/editor.js (scripts/build-editor-script.sh, run by go generate).

import { enableKeyDialog } from "./dialog";
import { enableDrag } from "./drag";
import { enableDrawer } from "./drawer";
import { enableKeepPlace } from "./keepplace";
import { enableLive } from "./live";
import { enableGridKeys } from "./gridkeys";
import { enableGroupOrder } from "./group";
import { enablePlatformSwitch } from "./platform";
import { enablePageReorder } from "./pagedrag";
import { enablePickers } from "./picker";
import { enableReplacements } from "./replaces";
import { enableRenderToggle } from "./render";
import { enableSelecting } from "./select";
import { enableSliders } from "./slider";
import { enableThemeToggle } from "./theme";

enablePickers(document);
enableReplacements(document);
enableSliders(document);
enableGroupOrder(document);
enablePlatformSwitch(document);
enableThemeToggle(document);
enableRenderToggle(document);
enableLive(document);
enablePageReorder(document);

const grid = document.querySelector<HTMLTableElement>("table.grid");

if (grid !== null) {
  enableSelecting(grid);
  enableDrag(grid);
  enableGridKeys(grid);
  enableKeyDialog(grid);
  enableDrawer(grid);

  // Last, once every hint the script shows above the grid is showing, so the
  // grid is where it will stay.
  enableKeepPlace(grid);
}

import type { LayoutItem, Widget } from "@/lib/types";

export const GRID_COLS = 12;

const DEFAULT_W = 6;
const DEFAULT_H = 4;

// mergeLayout ensures every widget has a layout entry: existing ones are
// kept as-is (preserving any manual drag/resize), and any widget missing
// one (e.g. just added) gets placed in a simple 2-column flow so it's
// visible somewhere sensible before anyone's dragged it.
export function mergeLayout(widgets: Widget[], existing: LayoutItem[]): LayoutItem[] {
  const byId = new Map(existing.map((l) => [l.i, l]));
  const result: LayoutItem[] = [];
  let index = 0;

  for (const w of widgets) {
    const found = byId.get(w.id);
    if (found) {
      result.push(found);
    } else {
      const col = index % 2;
      const row = Math.floor(index / 2);
      result.push({ i: w.id, x: col * DEFAULT_W, y: row * DEFAULT_H, w: DEFAULT_W, h: DEFAULT_H });
    }
    index++;
  }

  return result;
}

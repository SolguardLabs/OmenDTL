import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(fileURLToPath(new URL("..", import.meta.url)));
const src = join(root, "src");
let lines = 0;

function visit(dir) {
  for (const entry of readdirSync(dir)) {
    const path = join(dir, entry);
    const stat = statSync(path);
    if (stat.isDirectory()) {
      visit(path);
      continue;
    }
    if (!entry.endsWith(".go")) continue;
    const text = readFileSync(path, "utf8");
    lines += text.split(/\r?\n/).filter((line) => line.trim().length > 0).length;
  }
}

visit(src);
console.log(lines);


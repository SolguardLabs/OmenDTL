import { spawnSync } from "node:child_process";
import { readdirSync, readFileSync } from "node:fs";
import { join } from "node:path";

const gofmt = process.platform === "win32" ? "gofmt.exe" : "gofmt";
for (const name of readdirSync("src").filter((entry) => entry.endsWith(".go"))) {
  const path = join("src", name);
  const source = readFileSync(path, "utf8").replace(/\r\n/g, "\n");
  const result = spawnSync(gofmt, [], { input: source, encoding: "utf8", shell: false });
  if (result.error || result.status !== 0 || result.stdout !== source) {
    console.error(`${path}: gofmt required or unavailable`, result.error?.message ?? result.stderr);
    process.exitCode = 1;
  }
}

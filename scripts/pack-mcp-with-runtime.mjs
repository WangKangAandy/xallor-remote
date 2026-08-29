/**
 * 把已构建的 Runtime 打进 MCP 包再 npm pack，控制端 / 被控端只装一个 tgz。
 * 用法（仓库根）：node scripts/pack-mcp-with-runtime.mjs
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const mcp = path.join(root, "packages", "mcp");
const release = path.join(root, "dist", "release");

const copies = [
  {
    from: path.join(release, "windows-amd64", "xallor-remote.exe"),
    to: path.join(mcp, "vendor", "win32-x64", "xallor-remote.exe"),
  },
  {
    from: path.join(release, "linux-amd64", "xallor-remote"),
    to: path.join(mcp, "vendor", "linux-x64", "xallor-remote"),
  },
];

for (const { from, to } of copies) {
  if (!fs.existsSync(from)) {
    console.error("缺少二进制，先构建:", from);
    process.exit(1);
  }
  fs.mkdirSync(path.dirname(to), { recursive: true });
  fs.copyFileSync(from, to);
  if (process.platform !== "win32" || to.endsWith("xallor-remote")) {
    try {
      fs.chmodSync(to, 0o755);
    } catch {
      /* windows exe */
    }
  }
  console.log("vendor", path.relative(root, to), fs.statSync(to).size);
}

const build = spawnSync("npm", ["run", "build"], { cwd: mcp, stdio: "inherit", shell: true });
if (build.status !== 0) process.exit(build.status ?? 1);
const test = spawnSync("npm", ["test"], { cwd: mcp, stdio: "inherit", shell: true });
if (test.status !== 0) process.exit(test.status ?? 1);

fs.mkdirSync(release, { recursive: true });
const pack = spawnSync("npm", ["pack", "--pack-destination", release], {
  cwd: mcp,
  stdio: "inherit",
  shell: true,
});
if (pack.status !== 0) process.exit(pack.status ?? 1);
console.log("packed into", release);

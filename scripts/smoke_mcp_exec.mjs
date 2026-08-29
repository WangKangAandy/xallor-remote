import { spawn } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const mcp = path.join(root, "packages", "mcp", "dist", "index.js");
const bin = path.join(process.env.APPDATA, "XallorRemote", "bin", "xallor-remote.exe");

const st = await new Promise((resolve, reject) => {
  const child = spawn(process.execPath, [path.join(root, "scripts", "ipc_rpc.mjs"), "status"], {
    cwd: root,
    stdio: ["ignore", "pipe", "pipe"],
  });
  let out = "";
  child.stdout.on("data", (d) => (out += d));
  child.on("exit", (code) => (code === 0 ? resolve(JSON.parse(out)) : reject(new Error(out))));
});
const g = await new Promise((resolve, reject) => {
  const child = spawn(process.execPath, [path.join(root, "scripts", "ipc_rpc.mjs"), "grant.show"], {
    cwd: root,
    stdio: ["ignore", "pipe", "pipe"],
  });
  let out = "";
  child.stdout.on("data", (d) => (out += d));
  child.on("exit", (code) => (code === 0 ? resolve(JSON.parse(out)) : reject(new Error(out || "grant"))));
});

const init = {
  jsonrpc: "2.0",
  id: 1,
  method: "initialize",
  params: {
    protocolVersion: "2024-11-05",
    capabilities: {},
    clientInfo: { name: "smoke", version: "0" },
  },
};
const toolsCall = {
  jsonrpc: "2.0",
  id: 2,
  method: "tools/call",
  params: {
    name: "remote.exec",
    arguments: { command: "whoami", device_id: st.device_id },
  },
};

const child = spawn(process.execPath, [mcp], {
  env: {
    ...process.env,
    XALLOR_REMOTE_BIN: bin,
    XALLOR_REMOTE_DEVICE_ID: st.device_id,
    XALLOR_REMOTE_DEVICE_GRANT: g.grant,
    XALLOR_REMOTE_RELAY_URL: "wss://api.xallor.com/remote",
  },
  stdio: ["pipe", "pipe", "pipe"],
});

let buf = "";
const lines = [];
child.stdout.setEncoding("utf8");
child.stderr.setEncoding("utf8");
child.stderr.on("data", (d) => process.stderr.write(d));

function send(obj) {
  child.stdin.write(JSON.stringify(obj) + "\n");
}

const result = await new Promise((resolve, reject) => {
  const t = setTimeout(() => reject(new Error("mcp timeout")), 30000);
  child.stdout.on("data", (chunk) => {
    buf += chunk;
    for (;;) {
      const i = buf.indexOf("\n");
      if (i < 0) break;
      const line = buf.slice(0, i);
      buf = buf.slice(i + 1);
      if (!line.trim()) continue;
      let msg;
      try {
        msg = JSON.parse(line);
      } catch {
        continue;
      }
      lines.push(msg);
      if (msg.id === 1) {
        send({ jsonrpc: "2.0", method: "notifications/initialized" });
        send(toolsCall);
      }
      if (msg.id === 2) {
        clearTimeout(t);
        child.kill();
        resolve(msg);
      }
    }
  });
  child.on("error", reject);
  send(init);
});

const text = result?.result?.content?.[0]?.text || JSON.stringify(result);
console.log(text.includes("andy") || text.includes("\\") ? "mcp_exec_ok" : "mcp_exec_check");
console.log(text.slice(0, 300));

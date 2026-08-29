import type { IncomingMessage, ServerResponse } from "node:http";
import net from "node:net";
import { spawn } from "node:child_process";
import { randomUUID } from "node:crypto";

export function ipcPath(): string {
  if (process.platform === "win32") {
    return "\\\\.\\pipe\\XallorRemote";
  }
  if (process.env.XDG_RUNTIME_DIR) {
    return `${process.env.XDG_RUNTIME_DIR}/xallor-remote.sock`;
  }
  return `${process.env.HOME || ""}/.config/xallor-remote/ipc.sock`;
}

function findBin(): string {
  return process.env.XALLOR_REMOTE_BIN || "xallor-remote";
}

export function ensureRuntime(): Promise<void> {
  return new Promise((resolve, reject) => {
    const child = spawn(findBin(), ["ensure"], {
      windowsHide: true,
      stdio: ["ignore", "pipe", "pipe"],
    });
    const t = setTimeout(() => {
      child.kill();
      reject(new Error("无法拉起 Runtime。"));
    }, 12000);
    child.on("error", (e) => {
      clearTimeout(t);
      reject(e);
    });
    child.on("exit", (code) => {
      clearTimeout(t);
      if (code === 0) resolve();
      else reject(new Error("无法拉起 Runtime。"));
    });
  });
}

function connect(timeoutMs: number): Promise<net.Socket> {
  return new Promise((resolve, reject) => {
    const s = net.createConnection(ipcPath());
    const t = setTimeout(() => {
      s.destroy();
      reject(new Error("Runtime 未运行。"));
    }, timeoutMs);
    s.once("connect", () => {
      clearTimeout(t);
      resolve(s);
    });
    s.once("error", (e) => {
      clearTimeout(t);
      reject(e);
    });
  });
}

export async function rpc(method: string, params: Record<string, unknown> = {}): Promise<unknown> {
  let sock: net.Socket;
  try {
    sock = await connect(400);
  } catch {
    await ensureRuntime();
    sock = await connect(8000);
  }
  const id = randomUUID();
  const line = JSON.stringify({ id, method, params }) + "\n";
  return new Promise((resolve, reject) => {
    let buf = "";
    sock.setEncoding("utf8");
    sock.on("data", (chunk: string) => {
      buf += chunk;
      for (;;) {
        const i = buf.indexOf("\n");
        if (i < 0) break;
        const raw = buf.slice(0, i);
        buf = buf.slice(i + 1);
        if (!raw.trim()) continue;
        const f = JSON.parse(raw) as { id?: string; ok?: boolean; result?: unknown; message?: string; event?: string };
        if (f.id !== id || f.event) continue;
        sock.destroy();
        if (f.ok === false) reject(new Error(f.message || "失败。"));
        else resolve(f.result ?? {});
        return;
      }
    });
    sock.on("error", reject);
    sock.write(line);
  });
}

export async function execStream(
  command: string,
  deviceId: string,
  write: (s: string) => void,
): Promise<void> {
  let sock: net.Socket;
  try {
    sock = await connect(400);
  } catch {
    await ensureRuntime();
    sock = await connect(8000);
  }
  const id = randomUUID();
  const params: Record<string, unknown> = { command };
  if (deviceId) params.device_id = deviceId;
  sock.setEncoding("utf8");
  let buf = "";
  await new Promise<void>((resolve, reject) => {
    sock.on("data", (chunk: string) => {
      buf += chunk;
      for (;;) {
        const i = buf.indexOf("\n");
        if (i < 0) break;
        const raw = buf.slice(0, i);
        buf = buf.slice(i + 1);
        const f = JSON.parse(raw) as { id?: string; ok?: boolean; event?: string; data?: string; message?: string };
        if (f.id !== id) continue;
        if (f.event === "stdout" || f.event === "stderr") {
          if (f.data) write(f.data);
          continue;
        }
        sock.destroy();
        if (f.ok === false) reject(new Error(f.message || "失败。"));
        else resolve();
        return;
      }
    });
    sock.on("error", reject);
    sock.write(JSON.stringify({ id, method: "exec", params }) + "\n");
  });
}

function readBody(req: IncomingMessage): Promise<string> {
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];
    req.on("data", (c) => chunks.push(c as Buffer));
    req.on("end", () => resolve(Buffer.concat(chunks).toString("utf8")));
    req.on("error", reject);
  });
}

export async function handleDevIpc(req: IncomingMessage, res: ServerResponse): Promise<boolean> {
  if (!req.url?.startsWith("/__ipc")) return false;
  res.setHeader("Content-Type", "application/json; charset=utf-8");
  try {
    const body = JSON.parse((await readBody(req)) || "{}") as {
      method?: string;
      params?: Record<string, unknown>;
      command?: string;
      device_id?: string;
    };
    if (req.url === "/__ipc/exec") {
      res.setHeader("Content-Type", "text/plain; charset=utf-8");
      await execStream(String(body.command || ""), String(body.device_id || ""), (s) => res.write(s));
      res.end();
      return true;
    }
    const result = await rpc(String(body.method), body.params || {});
    res.end(JSON.stringify({ ok: true, result }));
  } catch (e) {
    res.statusCode = 500;
    res.end(JSON.stringify({ ok: false, message: e instanceof Error ? e.message : "失败。" }));
  }
  return true;
}

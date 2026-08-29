import net from "node:net";
import { randomUUID } from "node:crypto";

const method = process.argv[2];
const params = JSON.parse(process.argv[3] || "{}");
const path = "\\\\.\\pipe\\XallorRemote";

const sock = net.createConnection(path);
const id = randomUUID();
let buf = "";
await new Promise((resolve, reject) => {
  const t = setTimeout(() => reject(new Error("Runtime 未运行。")), 4000);
  sock.once("connect", () => {
    clearTimeout(t);
    resolve();
  });
  sock.once("error", reject);
});
sock.setEncoding("utf8");
const result = await new Promise((resolve, reject) => {
  sock.on("data", (chunk) => {
    buf += chunk;
    for (;;) {
      const i = buf.indexOf("\n");
      if (i < 0) break;
      const raw = buf.slice(0, i);
      buf = buf.slice(i + 1);
      if (!raw.trim()) continue;
      const f = JSON.parse(raw);
      if (f.id !== id || f.event) continue;
      sock.destroy();
      if (f.ok === false) reject(new Error(f.message || "失败。"));
      else resolve(f.result ?? {});
      return;
    }
  });
  sock.on("error", reject);
  sock.write(JSON.stringify({ id, method, params }) + "\n");
});
if (method === "status") {
  const { device_id, online, relay, inbound, has_grant, version } = result;
  console.log(JSON.stringify({ device_id, online, relay, inbound, has_grant, version }));
} else {
  console.log(JSON.stringify(result));
}

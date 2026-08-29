import net from "node:net";
import { randomUUID } from "node:crypto";

const params = { relay_url: "wss://api.xallor.com/remote" };
const sock = net.createConnection("\\\\.\\pipe\\XallorRemote");
const id = randomUUID();
await new Promise((resolve, reject) => {
  sock.once("connect", resolve);
  sock.once("error", reject);
});
sock.setEncoding("utf8");
let buf = "";
await new Promise((resolve, reject) => {
  sock.on("data", (chunk) => {
    buf += chunk;
    const i = buf.indexOf("\n");
    if (i < 0) return;
    const f = JSON.parse(buf.slice(0, i));
    sock.destroy();
    if (f.ok === false) reject(new Error(f.message || "失败。"));
    else resolve(f.result ?? {});
  });
  sock.write(JSON.stringify({ id, method: "config.set", params }) + "\n");
});
console.log("relay set");

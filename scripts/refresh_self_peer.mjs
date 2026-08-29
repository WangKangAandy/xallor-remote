import net from "node:net";
import { randomUUID } from "node:crypto";

function call(method, params = {}) {
  const sock = net.createConnection("\\\\.\\pipe\\XallorRemote");
  const id = randomUUID();
  return new Promise((resolve, reject) => {
    let buf = "";
    sock.once("error", reject);
    sock.setEncoding("utf8");
    sock.on("data", (chunk) => {
      buf += chunk;
      for (;;) {
        const i = buf.indexOf("\n");
        if (i < 0) break;
        const f = JSON.parse(buf.slice(0, i));
        buf = buf.slice(i + 1);
        if (f.id !== id || f.event) continue;
        sock.destroy();
        if (f.ok === false) reject(new Error(f.message || "失败。"));
        else resolve(f.result ?? {});
        return;
      }
    });
    sock.once("connect", () => {
      sock.write(JSON.stringify({ id, method, params }) + "\n");
    });
  });
}

const st = await call("status");
const g = await call("grant.show");
if (!st.device_id || !g.grant) {
  console.log("no_id_or_grant");
  process.exit(1);
}
await call("peer.add", { device_id: st.device_id, grant: g.grant });
console.log("peer_refreshed", st.device_id);

const url = process.argv[2] || "wss://api.xallor.com/remote";
const ws = new WebSocket(url);
const t = setTimeout(() => {
  console.log("timeout");
  process.exit(2);
}, 8000);
ws.addEventListener("open", () => {
  console.log("open");
  ws.send(JSON.stringify({ type: "hello_device", device_id: "dev_probe", secret: "nope" }));
});
ws.addEventListener("message", (ev) => {
  console.log("msg", ev.data);
  clearTimeout(t);
  ws.close();
  process.exit(0);
});
ws.addEventListener("error", (ev) => {
  console.log("error", ev.message || ev);
});
ws.addEventListener("close", (ev) => {
  console.log("close", ev.code, ev.reason);
  clearTimeout(t);
  process.exit(ev.code === 1000 ? 0 : 1);
});

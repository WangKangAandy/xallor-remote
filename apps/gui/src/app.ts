import * as api from "./bridge";

type Page = "home" | "peers" | "session" | "approval" | "audit" | "settings";

const pages: { id: Page; label: string }[] = [
  { id: "home", label: "本机" },
  { id: "peers", label: "对方" },
  { id: "session", label: "会话" },
  { id: "approval", label: "审批" },
  { id: "audit", label: "审计" },
  { id: "settings", label: "设置" },
];

export function mount(root: HTMLElement): void {
  root.innerHTML = `
    <aside class="nav">
      <h1>XallorRemote</h1>
      ${pages.map((p) => `<button type="button" data-page="${p.id}">${p.label}</button>`).join("")}
    </aside>
    <main id="main"></main>
  `;
  let page: Page = "home";
  const main = root.querySelector("#main") as HTMLElement;
  const go = (next: Page) => {
    page = next;
    root.querySelectorAll(".nav button").forEach((b) => {
      b.classList.toggle("on", (b as HTMLElement).dataset.page === page);
    });
    void render(main, page);
  };
  root.querySelector(".nav")?.addEventListener("click", (e) => {
    const t = e.target as HTMLElement;
    if (t.dataset.page) go(t.dataset.page as Page);
  });
  go("home");
}

async function render(main: HTMLElement, page: Page): Promise<void> {
  main.innerHTML = "<p class='hint'>加载中…</p>";
  try {
    if (page === "home") return void (await home(main));
    if (page === "peers") return void (await peers(main));
    if (page === "session") return void session(main);
    if (page === "settings") return void (await settings(main));
    if (page === "approval") {
      main.innerHTML = "<h2>审批</h2><p class='hint'>当前没有等待确认的操作。</p>";
      return;
    }
    main.innerHTML = "<h2>审计</h2><p class='hint'>这里不展示命令全文。</p>";
  } catch (e) {
    main.innerHTML = `<p class="err">${esc(e instanceof Error ? e.message : "失败。")}</p>`;
  }
}

async function home(main: HTMLElement): Promise<void> {
  const st = await api.status();
  const grant = await api.grantShow().catch(() => "");
  const inbound = st.inbound && st.has_grant ? "开" : st.has_grant ? "关" : "关（还没有授权码）";
  main.innerHTML = `
    <h2>本机</h2>
    <dl>
      <dt>Device ID</dt><dd>${esc(st.device_id || "")}</dd>
      <dt>Workspace</dt><dd>${esc(st.workspace || "")}</dd>
      <dt>Relay</dt><dd>${esc(st.relay || "")}</dd>
      <dt>入站</dt><dd>${inbound}</dd>
      <dt>在线</dt><dd>${st.online ? "是" : "否"}</dd>
    </dl>
    <div class="row">
      <button type="button" id="issue">签发授权码</button>
      <button type="button" id="rotate">换新授权码</button>
      <button type="button" id="on">开入站</button>
      <button type="button" id="off">关入站</button>
    </div>
    <pre id="grant">${grant ? "授权码: " + esc(grant) + "\n把这一行给对方。" : "还没有授权码。"}</pre>
  `;
  const showGrant = (g: string) => {
    (main.querySelector("#grant") as HTMLElement).textContent = "授权码: " + g + "\n把这一行给对方。";
  };
  bind(main, "#issue", async () => {
    showGrant(await api.grantIssue());
  });
  bind(main, "#rotate", async () => {
    if (!confirm("换新授权码后，旧码立刻失效。继续？")) return;
    showGrant(await api.grantRotate());
  });
  bind(main, "#on", async () => {
    await api.inboundSet(true);
    await home(main);
  });
  bind(main, "#off", async () => {
    await api.inboundSet(false);
    await home(main);
  });
}

async function peers(main: HTMLElement): Promise<void> {
  const ids = await api.peerList();
  main.innerHTML = `
    <h2>对方</h2>
    <ul id="list">${ids.length ? ids.map((id) => `<li>${esc(id)}</li>`).join("") : "<li class='hint'>还没有对方设备。</li>"}</ul>
    <form id="add" class="stack">
      <input name="id" placeholder="设备 ID" autocomplete="off" />
      <input name="grant" placeholder="授权码" autocomplete="off" />
      <button type="submit">添加</button>
    </form>
  `;
  (main.querySelector("#add") as HTMLFormElement).onsubmit = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target as HTMLFormElement);
    const id = String(fd.get("id") || "").trim();
    const grant = String(fd.get("grant") || "").trim();
    if (!id || !grant) {
      alert("请填写设备 ID 和授权码。");
      return;
    }
    await api.peerAdd(id, grant);
    await peers(main);
  };
}

function session(main: HTMLElement): void {
  main.innerHTML = `
    <h2>会话</h2>
    <form id="run" class="stack">
      <input name="device" placeholder="设备 ID（仅一台可空）" autocomplete="off" />
      <input name="cmd" placeholder="命令" autocomplete="off" />
      <button type="submit">执行</button>
    </form>
    <pre id="out"></pre>
  `;
  (main.querySelector("#run") as HTMLFormElement).onsubmit = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target as HTMLFormElement);
    const out = main.querySelector("#out") as HTMLElement;
    const cmd = String(fd.get("cmd") || "").trim();
    if (!cmd) {
      alert("请输入命令。");
      return;
    }
    out.textContent = "";
    try {
      await api.execCommand(cmd, String(fd.get("device") || "").trim(), (s) => {
        out.textContent += s;
        out.scrollTop = out.scrollHeight;
      });
    } catch (err) {
      out.textContent += (err instanceof Error ? err.message : "失败。") + "\n";
    }
  };
}

async function settings(main: HTMLElement): Promise<void> {
  const st = await api.status();
  main.innerHTML = `
    <h2>设置</h2>
    <p>Relay 和 workspace 跟本机 Runtime 走。要改地址，请在本机用命令行。</p>
    <dl>
      <dt>Relay</dt><dd>${esc(st.relay || "")}</dd>
      <dt>Workspace</dt><dd>${esc(st.workspace || "")}</dd>
      <dt>版本</dt><dd>${esc(st.version || "")}</dd>
    </dl>
  `;
}

function bind(root: HTMLElement, sel: string, fn: () => Promise<void>): void {
  root.querySelector(sel)?.addEventListener("click", () => {
    void fn().catch((e: unknown) => {
      alert(e instanceof Error ? e.message : "失败。");
    });
  });
}

function esc(s: string): string {
  return s.replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c] || c));
}

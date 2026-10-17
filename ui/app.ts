// @anchors
//   code: APIMP
//   ref: APPUI

// The Polvo window: the nodes, the folder tree, the editor and the terminals. The network, the
// browser storage, the editor and the terminal come in as dependencies (main.ts wires the real
// ones), so the window's behaviour is tested on its own.

/** The messages the user reads: APPUI-M01 (no secret), APPUI-M02 (an error), APPUI-M03 (saved),
 * APPUI-M04 (the shell ended). */
export const messages = {
  noToken: "Abra o Polvo pelo endereço que o comando polvo mostrou (falta o segredo da sessão).",
  error: (m: string) => `Erro: ${m}`,
  saved: (path: string) => `Gravado: ${path}`,
  ended: (code: number) => `[terminou: código ${code}]`,
};

/** The session secret from the address fragment, `#t=…` (APPUI-B01). */
export function tokenFromHash(hash: string): string {
  const m = /(?:^#|&)t=([^&]*)/.exec(hash);
  return m ? decodeURIComponent(m[1]) : "";
}

export interface Editor {
  setText(text: string): void;
  getText(): string;
  onChange(cb: () => void): void;
}

export interface Terminal {
  rows: number;
  cols: number;
  write(data: string | Uint8Array): void;
  onData(cb: (data: string) => void): void;
  onResize(cb: (rows: number, cols: number) => void): void;
  dispose(): void;
}

export interface SocketLike {
  binaryType: string;
  send(data: string): void;
  close(): void;
  onmessage: ((ev: { data: unknown }) => void) | null;
  onclose: (() => void) | null;
}

export interface Deps {
  fetch: typeof fetch;
  openSocket(url: string): SocketLike;
  makeEditor(el: HTMLElement): Editor;
  makeTerminal(el: HTMLElement): Terminal;
  storage: { getItem(k: string): string | null; setItem(k: string, v: string): void };
  location: { hash: string; host: string; protocol: string; pathname: string };
  replaceUrl(url: string): void;
}

interface Entry {
  name: string;
  dir: boolean;
  size: number;
}

const nodesKey = "polvo.nodes";

function el<K extends keyof HTMLElementTagNameMap>(tag: K, cls: string, text = ""): HTMLElementTagNameMap[K] {
  const e = document.createElement(tag);
  e.className = cls;
  e.textContent = text;
  return e;
}

function toBase64(text: string): string {
  let bin = "";
  for (const b of new TextEncoder().encode(text)) bin += String.fromCharCode(b);
  return btoa(bin);
}

function fromBase64(data: string): string {
  const bin = atob(data);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return new TextDecoder().decode(bytes);
}

function join(dir: string, name: string): string {
  return dir.endsWith("/") ? dir + name : `${dir}/${name}`;
}

function parent(dir: string): string {
  const i = dir.replace(/\/+$/, "").lastIndexOf("/");
  return i <= 0 ? "/" : dir.slice(0, i);
}

/** The control calls through the UI server, with the session secret. */
export class Api {
  constructor(private fetchFn: typeof fetch, private token: string) {}

  async call(node: string, method: string, params: unknown): Promise<any> {
    const resp = await this.fetchFn("/api/call", {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-Polvo-Token": this.token },
      body: JSON.stringify({ node, method, params }),
    });
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    const answer = await resp.json();
    if (answer.error) throw new Error(answer.error.message);
    return answer.result;
  }
}

export class App {
  private token = "";
  private api!: Api;
  private nodes: string[] = [];
  private node = "";
  private file = "";
  private editor!: Editor;
  private ui!: {
    nodes: HTMLUListElement;
    add: HTMLInputElement;
    path: HTMLInputElement;
    tree: HTMLUListElement;
    title: HTMLElement;
    mark: HTMLElement;
    terminals: HTMLElement;
    status: HTMLElement;
  };

  constructor(private root: HTMLElement, private deps: Deps) {}

  /** Builds the window; false when the address has no secret (APPUI-E02, APPUI-S01). */
  start(): boolean {
    this.token = tokenFromHash(this.deps.location.hash);
    if (!this.token) {
      this.root.textContent = messages.noToken;
      return false;
    }
    // APPUI-B01: the secret leaves the address bar.
    this.deps.replaceUrl(this.deps.location.pathname);
    this.api = new Api(this.deps.fetch, this.token);
    this.build();
    try {
      this.nodes = JSON.parse(this.deps.storage.getItem(nodesKey) ?? "[]");
    } catch {
      this.nodes = [];
    }
    this.renderNodes();
    // APPUI-S02: ready, with the node list, the add field and the status bar.
    return true;
  }

  private build() {
    const side = el("aside", "side");
    const nodes = el("ul", "nodes");
    const add = el("input", "add-node");
    add.placeholder = "máquina:porta";
    add.addEventListener("keydown", (e) => {
      if (e.key === "Enter") {
        this.addNode(add.value.trim());
        add.value = "";
      }
    });
    const path = el("input", "path");
    path.value = "/";
    path.addEventListener("keydown", (e) => {
      if (e.key === "Enter") this.openFolder(path.value || "/");
    });
    const tree = el("ul", "tree");
    side.append(nodes, add, path, tree);

    const main = el("main", "main");
    const head = el("header", "head");
    const title = el("span", "title");
    const mark = el("span", "mark");
    const save = el("button", "save", "Gravar");
    save.addEventListener("click", () => this.save());
    head.append(title, mark, save);
    const editorBox = el("div", "editor");
    main.append(head, editorBox);

    const bottom = el("section", "bottom");
    const newTerm = el("button", "new-terminal", "+ terminal");
    newTerm.addEventListener("click", () => this.newTerminal());
    const terminals = el("div", "terminals");
    bottom.append(newTerm, terminals);

    const status = el("footer", "status");
    this.root.append(side, main, bottom, status);
    this.root.addEventListener("keydown", (e) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "s") {
        e.preventDefault();
        this.save();
      }
    });
    this.ui = { nodes, add, path, tree, title, mark, terminals, status };
    this.editor = this.deps.makeEditor(editorBox);
    // APPUI-B05: a change marks the file.
    this.editor.onChange(() => {
      if (this.file) this.ui.mark.textContent = "●";
    });
  }

  status(): string {
    return this.ui.status.textContent ?? "";
  }

  private fail(e: unknown) {
    // APPUI-E01
    // APPUI-M02
    this.ui.status.textContent = messages.error(e instanceof Error ? e.message : String(e));
  }

  private renderNodes() {
    this.ui.nodes.replaceChildren(
      ...this.nodes.map((n) => {
        const li = el("li", n === this.node ? "node selected" : "node", n);
        li.addEventListener("click", () => this.selectNode(n));
        return li;
      }),
    );
  }

  /** Adds a node and remembers it (APPUI-B02). */
  addNode(addr: string) {
    if (!addr) return;
    if (!this.nodes.includes(addr)) {
      this.nodes.push(addr);
      this.deps.storage.setItem(nodesKey, JSON.stringify(this.nodes));
    }
    this.selectNode(addr);
  }

  selectNode(addr: string) {
    this.node = addr;
    this.renderNodes();
    this.openFolder(this.ui.path.value || "/");
  }

  /** Lists a folder: up, folders, files, each by name (APPUI-B03). */
  async openFolder(path: string) {
    try {
      const r = await this.api.call(this.node, "fs.list", { path });
      const entries: Entry[] = r.entries ?? [];
      const sorted = [...entries.filter((e) => e.dir), ...entries.filter((e) => !e.dir)];
      sorted.sort((a, b) => (a.dir === b.dir ? a.name.localeCompare(b.name) : a.dir ? -1 : 1));
      this.ui.path.value = path;
      const up = el("li", "dir", "..");
      up.addEventListener("click", () => this.openFolder(parent(path)));
      this.ui.tree.replaceChildren(
        up,
        ...sorted.map((e) => {
          const li = el("li", e.dir ? "dir" : "file", e.dir ? `${e.name}/` : e.name);
          li.addEventListener("click", () =>
            e.dir ? this.openFolder(join(path, e.name)) : this.openFile(join(path, e.name)),
          );
          return li;
        }),
      );
    } catch (e) {
      this.fail(e);
    }
  }

  /** Opens a file in the editor (APPUI-B04). */
  async openFile(path: string) {
    try {
      const r = await this.api.call(this.node, "fs.read", { path });
      this.file = path;
      this.editor.setText(fromBase64(r.data));
      this.ui.title.textContent = path;
      this.ui.mark.textContent = "";
    } catch (e) {
      this.fail(e);
    }
  }

  /** Writes the editor's text to the node (APPUI-B05). */
  async save() {
    if (!this.file) return;
    try {
      await this.api.call(this.node, "fs.write", { path: this.file, data: toBase64(this.editor.getText()) });
      this.ui.mark.textContent = "";
      // APPUI-M03
      this.ui.status.textContent = messages.saved(this.file);
    } catch (e) {
      this.fail(e);
    }
  }

  /** Opens a terminal on the node over the WebSocket (APPUI-B06). */
  newTerminal() {
    if (!this.node) return;
    const box = el("div", "terminal");
    this.ui.terminals.append(box);
    const t = this.deps.makeTerminal(box);
    const proto = this.deps.location.protocol === "https:" ? "wss:" : "ws:";
    const q = new URLSearchParams({ t: this.token, node: this.node, rows: String(t.rows), cols: String(t.cols) });
    const s = this.deps.openSocket(`${proto}//${this.deps.location.host}/ws/term?${q}`);
    s.binaryType = "arraybuffer";
    t.onData((d) => s.send(JSON.stringify({ input: d })));
    t.onResize((r, c) => s.send(JSON.stringify({ resize: [r, c] })));
    s.onmessage = (ev) => {
      if (typeof ev.data === "string") {
        const m = JSON.parse(ev.data);
        if (m.error) t.write(`\r\n${messages.error(m.error)}\r\n`);
        // APPUI-M04
        else if ("exit" in m) t.write(`\r\n${messages.ended(m.exit)}\r\n`);
      } else {
        t.write(new Uint8Array(ev.data as ArrayBuffer));
      }
    };
  }
}

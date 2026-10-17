// @anchors
//   code: APTST
//   ref: APPUI

import { describe, expect, it } from "vitest";
import { App, Deps, Editor, SocketLike, Terminal, messages, tokenFromHash } from "./app";

type Call = { node: string; method: string; params: any };

function b64(s: string): string {
  let bin = "";
  for (const b of new TextEncoder().encode(s)) bin += String.fromCharCode(b);
  return btoa(bin);
}

class FakeEditor implements Editor {
  text = "";
  private cb: () => void = () => {};
  setText(t: string) { this.text = t; }
  getText() { return this.text; }
  onChange(cb: () => void) { this.cb = cb; }
  type(t: string) { this.text = t; this.cb(); }
}

class FakeTerminal implements Terminal {
  rows = 24;
  cols = 80;
  out = "";
  private data: (d: string) => void = () => {};
  private resize: (r: number, c: number) => void = () => {};
  write(d: string | Uint8Array) { this.out += typeof d === "string" ? d : new TextDecoder().decode(d); }
  onData(cb: (d: string) => void) { this.data = cb; }
  onResize(cb: (r: number, c: number) => void) { this.resize = cb; }
  dispose() {}
  typeIn(d: string) { this.data(d); }
  resizeTo(r: number, c: number) { this.rows = r; this.cols = c; this.resize(r, c); }
}

class FakeSocket implements SocketLike {
  binaryType = "blob";
  sent: string[] = [];
  onmessage: ((ev: { data: unknown }) => void) | null = null;
  onclose: (() => void) | null = null;
  constructor(public url: string) {}
  send(s: string) { this.sent.push(s); }
  close() { this.onclose?.(); }
  receive(data: unknown) { this.onmessage?.({ data }); }
}

function setup(hash = "#t=abc", answer: (c: Call) => any = () => ({ result: {} }), saved: string[] = []) {
  document.body.innerHTML = "";
  const root = document.createElement("div");
  document.body.appendChild(root);
  const calls: Call[] = [];
  const tokens: (string | null)[] = [];
  const store = new Map<string, string>();
  if (saved.length) store.set("polvo.nodes", JSON.stringify(saved));
  const editor = new FakeEditor();
  const terminals: FakeTerminal[] = [];
  const sockets: FakeSocket[] = [];
  const urls: string[] = [];
  const deps: Deps = {
    fetch: (async (_url: string, init: any) => {
      tokens.push(init.headers["X-Polvo-Token"]);
      const c = JSON.parse(init.body) as Call;
      calls.push(c);
      return { ok: true, status: 200, json: async () => answer(c) } as any;
    }) as any,
    openSocket: (url: string) => { const s = new FakeSocket(url); sockets.push(s); return s; },
    makeEditor: () => editor,
    makeTerminal: () => { const t = new FakeTerminal(); terminals.push(t); return t; },
    storage: { getItem: (k) => store.get(k) ?? null, setItem: (k, v) => { store.set(k, v); } },
    location: { hash, host: "127.0.0.1:8080", protocol: "http:", pathname: "/" },
    replaceUrl: (u) => { urls.push(u); },
  };
  const app = new App(root, deps);
  const started = app.start();
  return { app, root, calls, tokens, store, editor, terminals, sockets, urls, started };
}

const flush = () => new Promise((r) => setTimeout(r, 0));

describe("App", () => {
  it("APPUI-B01: O segredo lido do fragmento sai da barra de endereço", async () => {
    expect(tokenFromHash("#t=abc")).toBe("abc");
    const w = setup("#t=abc", () => ({ result: { entries: [] } }));
    w.app.addNode("10.0.0.1:2222");
    await flush();
    expect(w.tokens[0]).toBe("abc");
    expect(w.urls).toEqual(["/"]);
  });

  it("APPUI-B02: O nó acrescentado fica lembrado e escolhê-lo lista a raiz", async () => {
    const w = setup("#t=abc", () => ({ result: { entries: [] } }));
    w.app.addNode("10.0.0.1:2222");
    await flush();
    expect(JSON.parse(w.store.get("polvo.nodes")!)).toEqual(["10.0.0.1:2222"]);
    expect(w.calls[0]).toEqual({ node: "10.0.0.1:2222", method: "fs.list", params: { path: "/" } });
    const again = setup("#t=abc", () => ({ result: { entries: [] } }), ["10.0.0.9:2222"]);
    expect([...again.root.querySelectorAll(".node")].map((n) => n.textContent)).toEqual(["10.0.0.9:2222"]);
  });

  it("APPUI-B03: A árvore mostra a pasta de cima, as pastas e os ficheiros por ordem", async () => {
    const w = setup("#t=abc", () => ({ result: { entries: [
      { name: "z.txt", dir: false, size: 3 }, { name: "b", dir: true, size: 0 }, { name: "a", dir: true, size: 0 },
    ] } }));
    w.app.addNode("n:1");
    await w.app.openFolder("/home");
    const names = [...w.root.querySelectorAll(".tree li")].map((li) => li.textContent);
    expect(names).toEqual(["..", "a/", "b/", "z.txt"]);
  });

  it("APPUI-B03: Clicar numa pasta abre-a", async () => {
    const w = setup("#t=abc", () => ({ result: { entries: [{ name: "a", dir: true, size: 0 }] } }));
    w.app.addNode("n:1");
    await w.app.openFolder("/home");
    const a = [...w.root.querySelectorAll<HTMLElement>(".tree li")].find((li) => li.textContent === "a/")!;
    a.click();
    await flush();
    expect(w.calls.at(-1)).toEqual({ node: "n:1", method: "fs.list", params: { path: "/home/a" } });
  });

  it("APPUI-B04: Abrir um ficheiro põe o texto no editor", async () => {
    const w = setup("#t=abc", (c) => c.method === "fs.read" ? { result: { data: b64("ola polvo"), size: 9 } } : { result: { entries: [] } });
    w.app.addNode("n:1");
    await w.app.openFile("/home/z.txt");
    expect(w.editor.text).toBe("ola polvo");
    expect(w.root.querySelector(".title")!.textContent).toBe("/home/z.txt");
  });

  it("APPUI-B05: Alterar põe a marca e gravar escreve, tira a marca e diz gravado", async () => {
    const w = setup("#t=abc", (c) => c.method === "fs.read" ? { result: { data: b64("velho"), size: 5 } } : { result: { entries: [] } });
    w.app.addNode("n:1");
    await w.app.openFile("/home/z.txt");
    w.editor.type("novo texto");
    expect(w.root.querySelector(".mark")!.textContent).toBe("●");
    await w.app.save();
    const write = w.calls.find((c) => c.method === "fs.write")!;
    expect(write.params).toEqual({ path: "/home/z.txt", data: b64("novo texto") });
    expect(w.root.querySelector(".mark")!.textContent).toBe("");
    expect(w.app.status()).toBe("Gravado: /home/z.txt");
  });

  it("APPUI-B06: O terminal envia a entrada e o tamanho, mostra a saída e o fim", () => {
    const w = setup("#t=abc", () => ({ result: { entries: [] } }));
    w.app.addNode("10.0.0.1:2222");
    w.app.newTerminal();
    const s = w.sockets[0];
    const t = w.terminals[0];
    expect(s.url).toBe("ws://127.0.0.1:8080/ws/term?t=abc&node=10.0.0.1%3A2222&rows=24&cols=80");
    expect(s.binaryType).toBe("arraybuffer");
    t.typeIn("ls");
    t.resizeTo(30, 100);
    expect(s.sent.map((m) => JSON.parse(m))).toEqual([{ input: "ls" }, { resize: [30, 100] }]);
    s.receive(new TextEncoder().encode("ola").buffer);
    s.receive(JSON.stringify({ exit: 0, signal: "" }));
    expect(t.out).toContain("ola");
    expect(t.out).toContain("[terminou: código 0]");
  });

  it("APPUI-E01: Uma chamada que falha mostra o erro na barra", async () => {
    const w = setup("#t=abc", (c) => c.method === "fs.read"
      ? { error: { code: -32000, message: "open /x: no such file or directory" } } : { result: { entries: [] } });
    w.app.addNode("n:1");
    await w.app.openFile("/x");
    expect(w.app.status()).toBe("Erro: open /x: no such file or directory");
  });

  it("APPUI-E02: Sem o segredo só aparece a mensagem", () => {
    const w = setup("");
    expect(w.started).toBe(false);
    expect(w.root.textContent).toBe(messages.noToken);
    expect(w.root.querySelector(".nodes")).toBeNull();
  });

  it("APPUI-S01: O estado sem segredo esconde a janela", () => {
    const w = setup("");
    for (const sel of [".nodes", ".tree", ".editor", ".terminals"]) expect(w.root.querySelector(sel)).toBeNull();
  });

  it("APPUI-S02: O estado pronto mostra a lista dos nós e a barra de estado", () => {
    const w = setup("#t=abc");
    expect(w.started).toBe(true);
    for (const sel of [".nodes", ".add-node", ".status"]) expect(w.root.querySelector(sel)).not.toBeNull();
  });

  it("APPUI-M01: A mensagem sem segredo", () => {
    expect(messages.noToken).toBe("Abra o Polvo pelo endereço que o comando polvo mostrou (falta o segredo da sessão).");
  });

  it("APPUI-M02: A mensagem de erro", () => {
    expect(messages.error("sem rede")).toBe("Erro: sem rede");
  });

  it("APPUI-M03: A mensagem de gravado", () => {
    expect(messages.saved("/a.txt")).toBe("Gravado: /a.txt");
  });

  it("APPUI-M04: A mensagem de fim do terminal", () => {
    expect(messages.ended(3)).toBe("[terminou: código 3]");
  });
});

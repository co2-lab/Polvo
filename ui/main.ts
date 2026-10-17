// @anchors
//   code: APMAN
//   ref: APPUI

// The wiring of the window to the real pieces: CodeMirror, xterm.js, fetch and WebSocket.
import { basicSetup, EditorView } from "codemirror";
import { Terminal as XTerm } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { App, Editor, SocketLike, Terminal } from "./app";

function makeEditor(parent: HTMLElement): Editor {
  let changed: () => void = () => {};
  let quiet = false;
  const view = new EditorView({
    parent,
    extensions: [basicSetup, EditorView.updateListener.of((u) => { if (u.docChanged && !quiet) changed(); })],
  });
  return {
    setText(text) {
      quiet = true;
      view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text } });
      quiet = false;
    },
    getText: () => view.state.doc.toString(),
    onChange(cb) { changed = cb; },
  };
}

function makeTerminal(parent: HTMLElement): Terminal {
  const xt = new XTerm({ convertEol: false, cursorBlink: true });
  const fit = new FitAddon();
  xt.loadAddon(fit);
  xt.open(parent);
  fit.fit();
  new ResizeObserver(() => fit.fit()).observe(parent);
  return {
    get rows() { return xt.rows; },
    get cols() { return xt.cols; },
    write: (d) => xt.write(d),
    onData: (cb) => { xt.onData(cb); },
    onResize: (cb) => { xt.onResize(({ rows, cols }) => cb(rows, cols)); },
    dispose: () => xt.dispose(),
  };
}

const app = new App(document.getElementById("app")!, {
  fetch: window.fetch.bind(window),
  openSocket: (url) => new WebSocket(url) as unknown as SocketLike,
  makeEditor,
  makeTerminal,
  storage: window.localStorage,
  location: window.location,
  replaceUrl: (url) => history.replaceState(null, "", url),
});
app.start();

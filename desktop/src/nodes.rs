// @anchors
//   code: MQIMP
//   ref: MQNAS

//! The tab of the machines and their sessions (MQNAS): what it asks the installation runs away from the
//! window's thread, for a machine may answer late.

use gpui_kit::base::TestSupportExt as _;
use gpui_kit::component::ActiveTheme as _;
use gpui_kit::component::dock::{BasePanel, Panel, PanelEvent};
use gpui_kit::component::input::{Input, InputState};
use gpui_kit::{
    App, AppContext as _, ClickEvent, ClipboardItem, Context, Entity, EventEmitter, FocusHandle, Focusable, FontWeight, Hsla,
    IntoElement, ParentElement as _, Render, Role, SharedString, Styled as _, Task, Window, div, prelude::*, px,
};

use crate::panel::SessionTab; // @dep: TPITR
use crate::remote::{SessionInfo, Station, Status}; // @dep: RMIMP

/// What the tab tells the window.
#[derive(Clone)]
pub enum NodesEvent {
    /// Open this session in a terminal tab (MQNAS-B03, MQNAS-B04).
    Open(SessionTab),
}

/// A machine as the tab shows it: its sessions, or why it did not answer (MQNAS-E01).
#[derive(Clone)]
struct MachineView {
    name: String,
    local: bool,
    sessions: Result<Vec<SessionInfo>, String>,
}

/// Everything the tab read from the installation.
#[derive(Clone, Default)]
struct Snapshot {
    status: Option<Status>,
    machines: Vec<MachineView>,
}

/// Reads the installation: this machine's node, the machines and the sessions of each (MQNAS-B01).
fn read_all(st: &Station) -> Snapshot {
    let machines = st
        .machines()
        .unwrap_or_default()
        .into_iter()
        .map(|m| {
            let sessions = st.sessions(&m.name).map_err(|e| {
                // MQNAS-E01: the machine says why, the others go on
                eprintln!("polvo: {} did not answer: {e}", m.name);
                e.0
            });
            MachineView { sessions, name: m.name, local: m.local }
        })
        .collect();
    Snapshot { status: st.status().ok(), machines }
}

/// The tab of the machines.
pub struct NodesPanel {
    focus: FocusHandle,
    station: Station,
    addr: Entity<InputState>,
    snap: Snapshot,
    /// Why joining a machine failed (MQNAS-E02).
    join_message: Option<String>,
    _working: Option<Task<()>>,
}

impl NodesPanel {
    /// The tab of the installation's machines, read at once.
    pub fn new(station: Station, window: &mut Window, cx: &mut Context<Self>) -> Self {
        let addr = cx.new(|cx| InputState::new(window, cx).placeholder("máquina:porta"));
        let mut this = NodesPanel { focus: cx.focus_handle(), station, addr, snap: Snapshot::default(), join_message: None, _working: None };
        this.refresh(cx);
        this
    }

    /// Runs `work` on the installation away from the window's thread, then `done` with its result, and
    /// reads everything again (MQNAS-B02).
    fn run<T: Send + 'static>(
        &mut self,
        cx: &mut Context<Self>,
        work: impl FnOnce(&Station) -> T + Send + 'static,
        done: impl FnOnce(&mut Self, T, &mut Context<Self>) + 'static,
    ) {
        let st = self.station.clone();
        let task = cx.background_executor().spawn(async move {
            let out = work(&st);
            (out, read_all(&st))
        });
        self._working = Some(cx.spawn(async move |this, cx| {
            let (out, snap) = task.await;
            let _ = this.update(cx, |this, cx| {
                this.snap = snap;
                done(this, out, cx);
                cx.notify();
            });
        }));
    }

    /// MQNAS-B02
    fn refresh(&mut self, cx: &mut Context<Self>) {
        self.run(cx, |_| (), |_, (), _| {});
    }

    /// MQNAS-B03: a session that exists opens in a tab that is not its owner.
    fn open(&mut self, machine: String, s: &SessionInfo, cx: &mut Context<Self>) {
        let tab = SessionTab { station: self.station.clone(), machine, id: s.id.clone(), owned: false, writable: s.writable };
        cx.emit(NodesEvent::Open(tab));
    }

    /// MQNAS-B04: a new session opens in a tab that owns it.
    fn new_session(&mut self, machine: String, cx: &mut Context<Self>) {
        let m = machine.clone();
        self.run(cx, move |st| st.new_session(&m, 24, 80), move |this, made, cx| match made {
            Ok(id) => {
                let tab = SessionTab { station: this.station.clone(), machine, id, owned: true, writable: true };
                cx.emit(NodesEvent::Open(tab));
            }
            Err(e) => this.tell(e.0),
        });
    }

    /// MQNAS-B05
    fn lock(&mut self, id: String, locked: bool, cx: &mut Context<Self>) {
        self.run(cx, move |st| st.lock(crate::remote::LOCAL, &id, locked), |this, done, _| {
            if let Err(e) = done {
                this.tell(e.0);
            }
        });
    }

    /// MQNAS-B06; MQNAS-E02: the reason, and the machines as they were.
    fn join(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        let addr = self.addr.read(cx).value().trim().to_string();
        if addr.is_empty() {
            return;
        }
        self.addr.update(cx, |s, cx| s.set_value("", window, cx));
        self.run(cx, move |st| st.add(&addr), |this, joined, _| match joined {
            Ok(()) => this.join_message = None,
            // MQNAS-E02: the machines stay as they were
            Err(e) => this.tell(e.0),
        });
    }

    /// MQNAS-B06
    fn remove(&mut self, name: String, cx: &mut Context<Self>) {
        self.run(cx, move |st| st.remove(&name), |this, done, _| {
            if let Err(e) = done {
                this.tell(e.0);
            }
        });
    }

    /// Shows the user why something they did failed.
    fn tell(&mut self, message: String) {
        self.join_message = Some(message);
    }

    fn button(
        id: String,
        label: &'static str,
        hover: Hsla,
        on_click: impl Fn(&ClickEvent, &mut Window, &mut App) + 'static,
    ) -> impl IntoElement {
        div()
            .id(SharedString::from(id))
            .role(Role::Button)
            .aria_label(label)
            .on_click(on_click)
            .test_support()
            .px(px(8.))
            .py(px(2.))
            .rounded(px(4.))
            .cursor_pointer()
            .hover(move |s| s.bg(hover))
            .child(label)
    }

    fn machine(&self, m: &MachineView, cx: &mut Context<Self>) -> impl IntoElement + use<> {
        let theme = cx.theme();
        let (muted, danger, hover, border) = (theme.muted_foreground, theme.danger, theme.list_hover, theme.border);
        let title = if m.local { "Esta máquina".to_string() } else { m.name.clone() };
        let name = m.name.clone();
        let mut head = div()
            .id(SharedString::from(format!("machine-{}", m.name)))
            .role(Role::Heading)
            .aria_label(m.name.clone())
            .test_support()
            .flex()
            .items_center()
            .gap(px(8.))
            .pt(px(12.))
            .child(div().flex_1().font_weight(FontWeight::BOLD).child(title))
            .child(Self::button(format!("new-{}", m.name), "Sessão nova", hover, cx.listener({
                let name = name.clone();
                move |this, _, _, cx| this.new_session(name.clone(), cx)
            })));
        if !m.local {
            head = head.child(Self::button(format!("remove-{}", m.name), "Esquecer", hover, cx.listener({
                let name = name.clone();
                move |this, _, _, cx| this.remove(name.clone(), cx)
            })));
        }
        let mut body = div().border_b_1().border_color(border).pb(px(8.)).child(head);
        if m.local
            && let Some(status) = &self.snap.status
        {
            let listen = match (&status.listen_error, status.listen.is_empty()) {
                (Some(e), _) => format!("Não escuta: {e}"),
                (None, true) => "Não escuta outras máquinas".to_string(),
                (None, false) => format!("Escuta em {}", status.listen),
            };
            let key = status.key.clone();
            body = body.child(div().text_color(muted).child(listen)).child(
                div()
                    .flex()
                    .items_center()
                    .gap(px(8.))
                    .child(
                        div()
                            .id("machine-key")
                            .role(Role::Label)
                            .aria_label(key.clone())
                            .test_support()
                            .flex_1()
                            .overflow_hidden()
                            .whitespace_nowrap()
                            .text_color(muted)
                            .font_family("monospace")
                            .child(key.clone()),
                    )
                    .child(Self::button("copy-key".into(), "Copiar a chave", hover, move |_, _, cx| {
                        cx.write_to_clipboard(ClipboardItem::new_string(key.clone()));
                    })),
            );
        }
        match &m.sessions {
            // MQNAS-E01
            Err(why) => {
                body = body.child(
                    div()
                        .id(SharedString::from(format!("machine-error-{}", m.name)))
                        .role(Role::Status)
                        .aria_label(why.clone())
                        .test_support()
                        .text_color(danger)
                        .child(why.clone()),
                )
            }
            Ok(list) if list.is_empty() => body = body.child(div().text_color(muted).child("Sem sessões.")),
            Ok(list) => {
                for s in list {
                    let mut marks = vec![format!("{}×{}", s.cols, s.rows), format!("{} a ver", s.viewers)];
                    if s.locked {
                        marks.push("travada".into());
                    }
                    if !s.writable {
                        marks.push("só ver".into());
                    }
                    let (machine, info) = (m.name.clone(), s.clone());
                    let mut row = div()
                        .id(SharedString::from(format!("session-{}-{}", m.name, s.id)))
                        .role(Role::ListItem)
                        .aria_label(s.id.clone())
                        .test_support()
                        .flex()
                        .items_center()
                        .gap(px(8.))
                        .px(px(8.))
                        .py(px(2.))
                        .rounded(px(4.))
                        .cursor_pointer()
                        .hover(move |st| st.bg(hover))
                        .on_click(cx.listener(move |this, _, _, cx| this.open(machine.clone(), &info, cx)))
                        .child(div().font_family("monospace").child(s.id.clone()))
                        .child(div().flex_1().overflow_hidden().whitespace_nowrap().child(s.argv.join(" ")))
                        .child(div().text_color(muted).child(marks.join(" · ")));
                    if m.local {
                        let (id, locked) = (s.id.clone(), s.locked);
                        let label = if locked { "Destravar" } else { "Travar" };
                        row = row.child(Self::button(format!("lock-{}", s.id), label, hover, cx.listener(
                            move |this, _, _, cx| {
                                cx.stop_propagation();
                                this.lock(id.clone(), !locked, cx);
                            },
                        )));
                    }
                    body = body.child(row);
                }
            }
        }
        body
    }
}

impl EventEmitter<NodesEvent> for NodesPanel {}
impl EventEmitter<PanelEvent> for NodesPanel {}

impl Focusable for NodesPanel {
    fn focus_handle(&self, _: &App) -> FocusHandle {
        self.focus.clone()
    }
}

impl BasePanel for NodesPanel {
    fn panel_name(&self) -> &'static str {
        "machines"
    }
}

impl Panel for NodesPanel {
    fn title(&mut self, _: &mut Window, _: &mut Context<Self>) -> impl IntoElement {
        "Máquinas"
    }
}

impl Render for NodesPanel {
    fn render(&mut self, _: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let machines: Vec<_> = self.snap.machines.clone().iter().map(|m| self.machine(m, cx).into_any_element()).collect();
        let theme = cx.theme();
        let (bg, fg, danger, hover) = (theme.background, theme.foreground, theme.danger, theme.list_hover);
        let message = self.join_message.clone().map(|m| {
            div().id("join-message").role(Role::Status).aria_label(m.clone()).test_support().text_color(danger).child(m)
        });
        let addr = self.addr.clone();
        div()
            .id("machines")
            .test_support()
            .size_full()
            .flex()
            .flex_col()
            .bg(bg)
            .text_color(fg)
            .track_focus(&self.focus)
            .child(
                div()
                    .flex()
                    .items_center()
                    .gap(px(8.))
                    .p(px(8.))
                    .child(
                        div()
                            .id("join-addr")
                            .test_support()
                            .flex_1()
                            .on_click(move |_, window, cx| addr.update(cx, |s, cx| s.focus(window, cx)))
                            .child(Input::new(&self.addr)),
                    )
                    .child(Self::button("join".into(), "Juntar", hover, cx.listener(|this, _, window, cx| this.join(window, cx))))
                    .child(Self::button("nodes-refresh".into(), "Atualizar", hover, cx.listener(|this, _, _, cx| this.refresh(cx)))),
            )
            .children(message)
            .child(div().id("machines-list").flex_1().min_h_0().overflow_y_scroll().px(px(12.)).children(machines))
    }
}

#[cfg(test)]
#[path = "nodes_test.rs"]
mod tests;

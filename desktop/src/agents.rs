// @anchors
//   code: AGIMP
//   ref: AGNTS

//! The progress of the agents of every machine, and the questions to them (AGNTS).

use std::collections::{BTreeMap, HashMap};

use gpui_kit::base::TestSupportExt as _;
use gpui_kit::component::ActiveTheme as _;
use gpui_kit::component::dock::{BasePanel, Panel, PanelEvent};
use gpui_kit::component::input::{Input, InputState};
use gpui_kit::{
    App, AppContext as _, ClickEvent, Context, Entity, EventEmitter, FocusHandle, Focusable, FontWeight, Hsla,
    IntoElement, ParentElement as _, Render, Role, SharedString, Styled as _, Subscription, Task, Window, div,
    prelude::*, px,
};

use crate::remote::{AgentJob, LOCAL, MachineProgress, Station}; // @dep: RMIMP

/// The last states of the log each agent shows (AGNTS-B02).
pub const LAST_ENTRIES: usize = 8;

/// How long the feed waits for the next change, under the node's 30 s (AGNTS-B05).
pub const WAIT_MS: u64 = 20_000;

/// The questions one asks whoever is working (AGNTS-B03).
pub const COMMON_QUESTIONS: [&str; 3] = ["Em que ponto estás?", "O que falta fazer?", "Estás bloqueado em alguma coisa?"];

/// Where an agent that runs in a session is (AGNTS-B06).
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum AgentMark {
    Working,
    /// A question of Polvo waits for its answer.
    Waiting,
    /// A step is blocked.
    Blocked,
    Finished,
}

impl AgentMark {
    /// The mark on the session's tab.
    pub fn glyph(self) -> &'static str {
        match self {
            AgentMark::Working => "◐",
            AgentMark::Waiting => "?",
            AgentMark::Blocked => "⊘",
            AgentMark::Finished => "✔",
        }
    }

    fn of(job: &AgentJob) -> AgentMark {
        if job.finished {
            AgentMark::Finished
        } else if job.questions.iter().any(|q| q.state == "pending") {
            AgentMark::Waiting
        } else if job.steps.iter().any(|s| s.state == "blocked") {
            AgentMark::Blocked
        } else {
            AgentMark::Working
        }
    }
}

/// What the feed tells whoever hears it.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum FeedEvent {
    /// The progress was read again (AGNTS-B05).
    Changed,
}

/// The progress of the agents of every machine, kept up to date (AGNTS-B05).
pub struct AgentsFeed {
    station: Station,
    machines: Vec<MachineProgress>,
    versions: BTreeMap<String, u64>,
    live: bool,
    _reading: Option<Task<()>>,
}

impl AgentsFeed {
    /// The feed of the installation's machines; a live one waits for each next change, one that is not
    /// reads when asked to (the tests).
    pub fn new(station: Station, live: bool, cx: &mut Context<Self>) -> Self {
        let mut this = AgentsFeed { station, machines: Vec::new(), versions: BTreeMap::new(), live, _reading: None };
        this.read(0, cx);
        this
    }

    /// Reads the progress again, at once.
    pub fn refresh(&mut self, cx: &mut Context<Self>) {
        self.versions.clear();
        self.read(0, cx);
    }

    /// The installation the feed reads.
    pub fn station(&self) -> &Station {
        &self.station
    }

    /// Every machine's progress, as last read.
    pub fn machines(&self) -> &[MachineProgress] {
        &self.machines
    }

    /// The mark of each agent that runs in a session, by the machine and the session's id (AGNTS-B06).
    pub fn marks(&self) -> HashMap<(String, String), AgentMark> {
        let mut out = HashMap::new();
        for m in &self.machines {
            for job in &m.agents {
                if let Some(id) = job.agent.strip_prefix("session-") {
                    out.insert((m.name.clone(), id.to_string()), AgentMark::of(job));
                }
            }
        }
        out
    }

    /// Waits for a change since the versions read, away from the window's thread, then reads on while live.
    fn read(&mut self, wait_ms: u64, cx: &mut Context<Self>) {
        let (st, since) = (self.station.clone(), self.versions.clone());
        let task = cx.background_executor().spawn(async move { st.progress_all(&since, wait_ms) });
        self._reading = Some(cx.spawn(async move |this, cx| {
            let read = task.await;
            let _ = this.update(cx, |this, cx| {
                if let Ok(machines) = read {
                    this.versions = machines.iter().filter(|m| m.error.is_none()).map(|m| (m.name.clone(), m.version)).collect();
                    this.machines = machines;
                    cx.emit(FeedEvent::Changed);
                    cx.notify();
                }
                if this.live {
                    this.read(WAIT_MS, cx);
                }
            });
        }));
    }
}

impl EventEmitter<FeedEvent> for AgentsFeed {}

/// The name the user reads for an agent.
fn shown_name(agent: &str) -> String {
    match agent.strip_prefix("session-") {
        Some(id) => format!("sessão {id}"),
        None => agent.to_string(),
    }
}

/// The text of a state of the log (AGNTS-B02).
fn entry_text(kind: &str, step: &str, state: &str, text: &str) -> String {
    match kind {
        "step" if text.is_empty() => format!("passo {step}: {state}"),
        "step" => format!("passo {step}: {state} — {text}"),
        "start" => format!("começou: {text}"),
        "finish" => format!("terminou: {text}"),
        _ => text.to_string(),
    }
}

/// The tab of the agents.
pub struct AgentsPanel {
    focus: FocusHandle,
    feed: Entity<AgentsFeed>,
    /// The box to write a question, by the machine and the agent (AGNTS-B03).
    inputs: HashMap<(String, String), Entity<InputState>>,
    /// Why asking an agent failed (AGNTS-E02).
    messages: HashMap<(String, String), String>,
    _heard: Subscription,
    _asking: Option<Task<()>>,
}

impl AgentsPanel {
    /// The tab of the feed's agents.
    pub fn new(feed: Entity<AgentsFeed>, _: &mut Window, cx: &mut Context<Self>) -> Self {
        let heard = cx.observe(&feed, |_, _, cx| cx.notify());
        AgentsPanel { focus: cx.focus_handle(), feed, inputs: HashMap::new(), messages: HashMap::new(), _heard: heard, _asking: None }
    }

    /// AGNTS-B03; AGNTS-E02: the reason, by the agent.
    fn ask(&mut self, machine: String, agent: String, question: String, cx: &mut Context<Self>) {
        if question.trim().is_empty() {
            return;
        }
        let st = self.feed.read(cx).station().clone();
        let (m, a) = (machine.clone(), agent.clone());
        let task = cx.background_executor().spawn(async move { st.ask(&m, &a, &question) });
        self._asking = Some(cx.spawn(async move |this, cx| {
            let asked = task.await;
            let _ = this.update(cx, |this, cx| {
                match asked {
                    Ok(_) => {
                        this.messages.remove(&(machine, agent));
                        this.feed.update(cx, |f, cx| f.refresh(cx));
                    }
                    // AGNTS-E02
                    Err(e) => this.tell((machine, agent), e.0),
                }
                cx.notify();
            });
        }));
    }

    /// Shows the user why something they did failed.
    fn tell(&mut self, key: (String, String), message: String) {
        self.messages.insert(key, message);
    }

    fn input(&mut self, key: &(String, String), window: &mut Window, cx: &mut Context<Self>) -> Entity<InputState> {
        self.inputs
            .entry(key.clone())
            .or_insert_with(|| cx.new(|cx| InputState::new(window, cx).placeholder("Outra pergunta…")))
            .clone()
    }

    fn button(id: String, label: String, hover: Hsla, on_click: impl Fn(&ClickEvent, &mut Window, &mut App) + 'static) -> impl IntoElement {
        div()
            .id(SharedString::from(id))
            .role(Role::Button)
            .aria_label(label.clone())
            .on_click(on_click)
            .test_support()
            .px(px(8.))
            .py(px(2.))
            .rounded(px(4.))
            .cursor_pointer()
            .hover(move |s| s.bg(hover))
            .child(label)
    }

    fn agent(&mut self, machine: &str, job: &AgentJob, window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement + use<> {
        let theme = cx.theme();
        let (muted, danger, hover, border) = (theme.muted_foreground, theme.danger, theme.list_hover, theme.border);
        // the elements of an agent of another machine say the machine too
        let k = if machine == LOCAL { job.agent.clone() } else { format!("{machine}/{}", job.agent) };
        let key = (machine.to_string(), job.agent.clone());
        let done = job.steps.iter().filter(|s| s.state == "done").count();
        let summary = format!("{} · {done}/{}", job.goal, job.steps.len());
        let state = if job.finished { "terminado" } else { "a trabalhar" };
        // AGNTS-B01
        let mut body = div().border_b_1().border_color(border).py(px(8.)).child(
            div()
                .id(SharedString::from(format!("agent-{machine}-{}", job.agent)))
                .role(Role::Heading)
                .aria_label(shown_name(&job.agent))
                .aria_value(summary.clone())
                .test_support()
                .flex()
                .gap(px(8.))
                .child(div().font_weight(FontWeight::BOLD).child(shown_name(&job.agent)))
                .child(div().flex_1().child(summary))
                .child(div().text_color(muted).child(state)),
        );
        for s in &job.steps {
            let glyph = match s.state.as_str() {
                "done" => "●",
                "doing" => "◐",
                "blocked" => "⊘",
                _ => "○",
            };
            body = body.child(
                div()
                    .id(SharedString::from(format!("step-{k}-{}", s.id)))
                    .role(Role::ListItem)
                    .aria_label(s.title.clone())
                    .aria_value(s.state.clone())
                    .test_support()
                    .pl(px(12.))
                    .child(format!("{glyph} {}. {}", s.id, s.title)),
            );
        }
        // AGNTS-B02
        for (i, e) in job.log.iter().rev().take(LAST_ENTRIES).enumerate() {
            let line = entry_text(&e.kind, &e.step, &e.state, &e.text);
            let at = e.at.get(11..19).unwrap_or_default().to_string();
            body = body.child(
                div()
                    .id(SharedString::from(format!("entry-{k}-{i}")))
                    .role(Role::ListItem)
                    .aria_label(line.clone())
                    .test_support()
                    .flex()
                    .gap(px(8.))
                    .pl(px(12.))
                    .text_color(muted)
                    .child(at)
                    .child(line),
            );
        }
        // AGNTS-B04
        for q in &job.questions {
            let answer = match q.state.as_str() {
                "answered" => q.answer.clone(),
                "failed" => format!("falhou: {}", q.answer),
                _ => "à espera…".to_string(),
            };
            let line = format!("{} → {answer}", q.text);
            body = body.child(
                div()
                    .id(SharedString::from(format!("question-{k}-{}", q.id)))
                    .role(Role::ListItem)
                    .aria_label(line.clone())
                    .test_support()
                    .pl(px(12.))
                    .when(q.state == "failed", |d| d.text_color(danger))
                    .child(line),
            );
        }
        // AGNTS-B03
        let mut asks = div().flex().flex_wrap().gap(px(4.)).pl(px(12.)).pt(px(4.));
        for (i, q) in COMMON_QUESTIONS.iter().enumerate() {
            let (m, a, q) = (machine.to_string(), job.agent.clone(), q.to_string());
            let this = cx.entity().downgrade();
            asks = asks.child(Self::button(format!("ask-{k}-{i}"), q.clone(), hover, move |_, _, cx| {
                let (m, a, q) = (m.clone(), a.clone(), q.clone());
                let _ = this.update(cx, |this, cx| this.ask(m, a, q, cx));
            }));
        }
        let input = self.input(&key, window, cx);
        let (m, a, field) = (machine.to_string(), job.agent.clone(), input.clone());
        let this = cx.entity().downgrade();
        let send = Self::button(format!("ask-send-{k}"), "Perguntar".to_string(), hover, move |_, window, cx| {
            let text = field.read(cx).value().to_string();
            field.update(cx, |s, cx| s.set_value("", window, cx));
            let (m, a) = (m.clone(), a.clone());
            let _ = this.update(cx, |this, cx| this.ask(m, a, text, cx));
        });
        let focus_input = input.clone();
        body = body.child(asks).child(
            div()
                .flex()
                .gap(px(4.))
                .pl(px(12.))
                .pt(px(4.))
                .child(
                    div()
                        .id(SharedString::from(format!("ask-input-{k}")))
                        .test_support()
                        .flex_1()
                        .on_click(move |_, window, cx| focus_input.update(cx, |s, cx| s.focus(window, cx)))
                        .child(Input::new(&input)),
                )
                .child(send),
        );
        if let Some(m) = self.messages.get(&key).cloned() {
            body = body.child(
                div()
                    .id(SharedString::from(format!("ask-message-{k}")))
                    .role(Role::Status)
                    .aria_label(m.clone())
                    .test_support()
                    .pl(px(12.))
                    .text_color(danger)
                    .child(m),
            );
        }
        body
    }
}

impl EventEmitter<PanelEvent> for AgentsPanel {}

impl Focusable for AgentsPanel {
    fn focus_handle(&self, _: &App) -> FocusHandle {
        self.focus.clone()
    }
}

impl BasePanel for AgentsPanel {
    fn panel_name(&self) -> &'static str {
        "agents"
    }
}

impl Panel for AgentsPanel {
    fn title(&mut self, _: &mut Window, _: &mut Context<Self>) -> impl IntoElement {
        "Agentes"
    }
}

impl Render for AgentsPanel {
    fn render(&mut self, window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let machines = self.feed.read(cx).machines().to_vec();
        let mut list = div().id("agents-list").flex_1().min_h_0().overflow_y_scroll().px(px(12.));
        let theme = cx.theme();
        let (bg, fg, muted, danger) = (theme.background, theme.foreground, theme.muted_foreground, theme.danger);
        let any = machines.iter().any(|m| !m.agents.is_empty() || m.error.is_some());
        if !any {
            list = list.child(div().pt(px(12.)).text_color(muted).child("Nenhum agente reportou ainda."));
        }
        for m in &machines {
            if m.agents.is_empty() && m.error.is_none() {
                continue;
            }
            let title = if m.name == LOCAL { "Esta máquina".to_string() } else { m.name.clone() };
            list = list.child(div().pt(px(12.)).font_weight(FontWeight::BOLD).child(title));
            // AGNTS-E01
            if let Some(why) = &m.error {
                list = list.child(
                    div()
                        .id(SharedString::from(format!("machine-error-{}", m.name)))
                        .role(Role::Status)
                        .aria_label(why.clone())
                        .test_support()
                        .text_color(danger)
                        .child(why.clone()),
                );
            }
            for job in &m.agents {
                let el = self.agent(&m.name, job, window, cx).into_any_element();
                list = list.child(el);
            }
        }
        div().id("agents").test_support().size_full().flex().flex_col().bg(bg).text_color(fg).track_focus(&self.focus).child(list)
    }
}

#[cfg(test)]
#[path = "agents_test.rs"]
mod tests;

// @anchors
//   code: SCIMS
//   ref: SCRNE

//! The terminal without a window (SCRNE): the shell in a pty and the screen in libghostty-vt.

use std::cell::RefCell;
use std::io::{Read, Write};
use std::path::Path;
use std::rc::Rc;
use std::time::Duration;

use futures::channel::mpsc;
use libghostty_vt::key::Encoder;
use libghostty_vt::paste;
use libghostty_vt::render::{CellIterator, RowIterator};
use libghostty_vt::selection::gesture::{DragEvent, Geometry, Gesture, PressEvent, ReleaseEvent};
use libghostty_vt::fmt::Format;
use libghostty_vt::selection::FormatOptions;
use libghostty_vt::style::{RgbColor, Underline};
use libghostty_vt::terminal::{Mode, Point, PointCoordinate, ScrollViewport};
use libghostty_vt::{RenderState, Terminal, TerminalOptions};
use portable_pty::{ChildKiller, CommandBuilder, MasterPty, PtySize, native_pty_system};

use crate::keys::{self, Input}; // @dep: KYIMK

/// A colour, red, green and blue.
pub type Rgb = (u8, u8, u8);

/// The lines of history kept (SCRNE-B05).
pub const SCROLLBACK: usize = 10_000;

/// The longest pause between the clicks of a double or triple click (SCRNE-B06).
pub const REPEAT_CLICK: Duration = Duration::from_millis(500);

/// The size of the terminal: its rows and columns, and the pixels of one cell.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Size {
    pub rows: u16,
    pub cols: u16,
    pub cell_w: u32,
    pub cell_h: u32,
}

/// What comes from the shell: its output, or its end.
#[derive(Debug)]
pub enum Event {
    Output(Vec<u8>),
    Exit,
}

/// The channel of the shell's events.
pub type Events = futures::channel::mpsc::UnboundedReceiver<Event>;

/// A piece of a line with the same colours and style.
#[derive(Clone, Debug, PartialEq)]
pub struct Run {
    pub text: String,
    pub fg: Rgb,
    pub bg: Rgb,
    pub bold: bool,
    pub italic: bool,
    pub underline: bool,
}

/// The drawing of the screen: its lines in runs, the default colours and the palette.
#[derive(Clone, Debug)]
#[cfg_attr(not(test), expect(dead_code, reason = "the default foreground and the palette are for the theme of later panels"))]
pub struct Frame {
    pub lines: Vec<Vec<Run>>,
    pub fg: Rgb,
    pub bg: Rgb,
    pub palette: Vec<Rgb>,
}

/// The terminal.
pub struct Screen {
    term: Terminal<'static, 'static>,
    render: RenderState<'static>,
    rows_it: RowIterator<'static>,
    cells_it: CellIterator<'static>,
    encoder: Encoder<'static>,
    gesture: Gesture<'static>,
    writer: Rc<RefCell<Box<dyn Write>>>,
    pty: Option<Box<dyn MasterPty + Send>>,
    killer: Option<Box<dyn ChildKiller + Send + Sync>>,
    size: Size,
}

impl Drop for Screen {
    /// SCRNE-B11: no shell outlives its terminal.
    fn drop(&mut self) {
        if let Some(mut killer) = self.killer.take() {
            let _ = killer.kill();
        }
    }
}

impl Screen {
    /// A terminal with no shell, that writes to `writer` what it sends to the shell.
    pub fn new(size: Size, writer: Box<dyn Write>) -> Result<Screen, String> {
        let size = size.at_least_one();
        let writer = Rc::new(RefCell::new(writer));
        let fail = |what: &str, e: libghostty_vt::Error| format!("libghostty-vt: {what}: {e}");
        let mut term = Terminal::new(TerminalOptions { cols: size.cols, rows: size.rows, max_scrollback: SCROLLBACK })
            .map_err(|e| fail("terminal", e))?;
        // SCRNE-B03: what the terminal answers goes to the shell.
        let w = writer.clone();
        term.on_pty_write(move |_t, data| {
            let _ = w.borrow_mut().write_all(data);
        })
        .map_err(|e| fail("pty write", e))?;
        let _ = term.resize(size.cols, size.rows, size.cell_w, size.cell_h);
        Ok(Screen {
            term,
            render: RenderState::new().map_err(|e| fail("render state", e))?,
            rows_it: RowIterator::new().map_err(|e| fail("rows", e))?,
            cells_it: CellIterator::new().map_err(|e| fail("cells", e))?,
            encoder: Encoder::new().map_err(|e| fail("encoder", e))?,
            gesture: Gesture::new().map_err(|e| fail("gesture", e))?,
            writer,
            pty: None,
            killer: None,
            size,
        })
    }

    /// Runs `argv` in a pty of `size`, in `cwd` (SCRNE-B01). The events come by the channel: the
    /// output, and the end after the last of it (SCRNE-B09).
    pub fn spawn(argv: &[String], cwd: Option<&Path>, size: Size) -> std::io::Result<(Screen, Events)> {
        let size = size.at_least_one();
        let fail = |e: &dyn std::fmt::Display| {
            std::io::Error::other(format!("{}: {e}", argv.first().map_or("", String::as_str)))
        };
        let Some(program) = argv.first() else {
            return Err(fail(&"no program"));
        };
        let pair = native_pty_system().openpty(size.pty()).map_err(|e| fail(&e))?;
        let mut cmd = CommandBuilder::new(program);
        cmd.args(&argv[1..]);
        cmd.env("TERM", "xterm-256color");
        cmd.env("COLORTERM", "truecolor");
        if let Some(dir) = cwd {
            cmd.cwd(dir);
        }
        // SCRNE-E01
        let mut child = pair.slave.spawn_command(cmd).map_err(|e| fail(&e))?;
        let killer = child.clone_killer();
        drop(pair.slave);
        let mut reader = pair.master.try_clone_reader().map_err(|e| fail(&e))?;
        let writer = pair.master.take_writer().map_err(|e| fail(&e))?;
        let (tx, rx) = mpsc::unbounded();
        std::thread::spawn(move || {
            let mut buf = vec![0u8; 64 * 1024];
            loop {
                match reader.read(&mut buf) {
                    Ok(0) | Err(_) => break,
                    Ok(n) => {
                        if tx.unbounded_send(Event::Output(buf[..n].to_vec())).is_err() {
                            return;
                        }
                    }
                }
            }
            let _ = child.wait();
            let _ = tx.unbounded_send(Event::Exit);
        });
        let mut screen = Screen::new(size, writer).map_err(|e| fail(&e))?;
        screen.pty = Some(pair.master);
        screen.killer = Some(killer);
        Ok((screen, rx))
    }

    /// Passes the shell's output to the terminal (SCRNE-B02).
    pub fn feed(&mut self, data: &[u8]) {
        self.term.vt_write(data);
    }

    fn send(&mut self, data: &[u8]) {
        if data.is_empty() {
            return;
        }
        // SCRNE-B05: writing brings the view back to the end.
        self.term.scroll_viewport(ScrollViewport::Bottom);
        let _ = self.writer.borrow_mut().write_all(data);
    }

    /// Sends a key to the shell, as libghostty-vt encodes it.
    pub fn key(&mut self, input: &Input) {
        let bytes = keys::encode(input, &mut self.encoder, &self.term);
        self.send(&bytes);
    }

    /// Sends pasted text to the shell (SCRNE-B07).
    pub fn paste(&mut self, text: &str) {
        let bracketed = self.term.mode(Mode::BRACKETED_PASTE).unwrap_or(false);
        let mut data = text.as_bytes().to_vec();
        let mut out = vec![0u8; data.len() + 16];
        let n = match paste::encode(&mut data, bracketed, &mut out) {
            Ok(n) => n,
            Err(libghostty_vt::Error::OutOfSpace { required }) => {
                out.resize(required, 0);
                paste::encode(&mut data, bracketed, &mut out).unwrap_or(0)
            }
            Err(_) => 0,
        };
        self.send(&out[..n]);
    }

    /// Changes the size of the terminal and of the pty (SCRNE-B04).
    pub fn resize(&mut self, size: Size) {
        let size = size.at_least_one();
        if size == self.size {
            return;
        }
        self.size = size;
        let _ = self.term.resize(size.cols, size.rows, size.cell_w, size.cell_h);
        if let Some(pty) = &self.pty {
            let _ = pty.resize(size.pty());
        }
    }

    /// The title the shell gave the terminal, empty while it gave none (SCRNE-B10).
    pub fn title(&self) -> String {
        self.term.title().map(str::to_string).unwrap_or_default()
    }

    /// The size the terminal has.
    pub fn size(&self) -> Size {
        self.size
    }

    /// Moves the view through the history, up when negative (SCRNE-B05).
    pub fn scroll(&mut self, lines: isize) {
        self.term.scroll_viewport(ScrollViewport::Delta(lines));
    }

    fn cell_at(&self, x: f64, y: f64) -> PointCoordinate {
        let col = (x / f64::from(self.size.cell_w)).floor().clamp(0., f64::from(self.size.cols - 1));
        let row = (y / f64::from(self.size.cell_h)).floor().clamp(0., f64::from(self.size.rows - 1));
        PointCoordinate { x: col as u16, y: row as u32 }
    }

    fn geometry(&self) -> Geometry {
        Geometry {
            columns: u32::from(self.size.cols),
            cell_width: self.size.cell_w,
            padding_left: 0,
            screen_height: u32::from(self.size.rows) * self.size.cell_h,
        }
    }

    /// The mouse button goes down at (`x`, `y`), pixels from the grid's corner, at the time `at`:
    /// one, two or three clicks in a row (SCRNE-B06).
    pub fn press(&mut self, x: f64, y: f64, at: Duration) {
        let point = self.cell_at(x, y);
        let Ok(grid) = self.term.grid_ref(Point::Viewport(point)) else { return };
        let Ok(mut ev) = PressEvent::new() else { return };
        let repeat = f64::from(self.size.cell_w);
        let _ = ev
            .set_position(x, y)
            .and_then(|e| e.set_time(at))
            .and_then(|e| e.set_repeat_interval(REPEAT_CLICK))
            .and_then(|e| e.set_repeat_distance(repeat));
        if let Ok(sel) = ev.apply(&mut self.gesture, &self.term, grid) {
            let _ = self.term.set_selection(sel.as_ref());
        }
    }

    /// The mouse moves with the button down (SCRNE-B06).
    pub fn drag(&mut self, x: f64, y: f64) {
        let point = self.cell_at(x, y);
        let geometry = self.geometry();
        let Ok(grid) = self.term.grid_ref(Point::Viewport(point)) else { return };
        let Ok(mut ev) = DragEvent::new() else { return };
        let _ = ev.set_position(x, y);
        if let Ok(Some(sel)) = ev.apply(&mut self.gesture, &self.term, grid, geometry) {
            let _ = self.term.set_selection(Some(&sel));
        }
    }

    /// The mouse button goes up (SCRNE-B06).
    pub fn release(&mut self, x: f64, y: f64) {
        let point = self.cell_at(x, y);
        let grid = self.term.grid_ref(Point::Viewport(point)).ok();
        if let Ok(mut ev) = ReleaseEvent::new() {
            let _ = ev.apply(&mut self.gesture, &self.term, grid);
        }
    }

    /// The selected text, without the blanks at the end of the lines (SCRNE-B06).
    pub fn selection_text(&self) -> Option<String> {
        let opts = FormatOptions::new().with_emit_format(Format::Plain).with_trim(true).with_unwrap(true);
        let bytes = self.term.format_selection_alloc(None, opts).ok()??;
        Some(String::from_utf8_lossy(&bytes).into_owned())
    }

    /// The drawing of the screen (SCRNE-B08).
    pub fn frame(&mut self) -> Frame {
        let empty = Frame { lines: Vec::new(), fg: (255, 255, 255), bg: (0, 0, 0), palette: Vec::new() };
        let Ok(snap) = self.render.update(&self.term) else { return empty };
        let Ok(colors) = snap.colors() else { return empty };
        let (fg0, bg0) = (rgb(colors.foreground), rgb(colors.background));
        let cursor = match snap.cursor_visible() {
            Ok(true) => snap.cursor_viewport().ok().flatten().map(|c| (c.x, c.y)),
            _ => None,
        };
        let mut lines = Vec::new();
        let Ok(mut rows) = self.rows_it.update(&snap) else { return empty };
        let mut y = 0u16;
        while let Some(row) = rows.next() {
            let mut runs: Vec<Run> = Vec::new();
            if let Ok(mut cells) = self.cells_it.update(row) {
                let mut x = 0u16;
                while let Some(cell) = cells.next() {
                    let style = cell.style().unwrap_or_default();
                    let mut fg = cell.fg_color().ok().flatten().map_or(fg0, rgb);
                    let mut bg = cell.bg_color().ok().flatten().map_or(bg0, rgb);
                    let selected = cell.is_selected().unwrap_or(false);
                    if style.inverse ^ selected ^ (cursor == Some((x, y))) {
                        std::mem::swap(&mut fg, &mut bg);
                    }
                    let mut text = String::new();
                    let _ = cell.graphemes_utf8(&mut text);
                    if text.is_empty() {
                        text.push(' ');
                    }
                    let run = Run {
                        text,
                        fg,
                        bg,
                        bold: style.bold,
                        italic: style.italic,
                        underline: style.underline != Underline::None,
                    };
                    match runs.last_mut() {
                        Some(r) if (r.fg, r.bg, r.bold, r.italic, r.underline) == (run.fg, run.bg, run.bold, run.italic, run.underline) => {
                            r.text.push_str(&run.text)
                        }
                        _ => runs.push(run),
                    }
                    x += 1;
                }
            }
            let _ = row.set_dirty(false);
            lines.push(runs);
            y += 1;
        }
        Frame { lines, fg: fg0, bg: bg0, palette: colors.palette.iter().copied().map(rgb).collect() }
    }

    #[cfg_attr(not(test), expect(dead_code, reason = "the text of the screen, for the tests and for search later"))]
    /// The text of the screen in view, a line per row, without the blanks at the end of each line
    /// and the empty lines at the end.
    pub fn text(&mut self) -> String {
        let frame = self.frame();
        let mut lines: Vec<String> = frame
            .lines
            .iter()
            .map(|runs| runs.iter().map(|r| r.text.as_str()).collect::<String>().trim_end().to_string())
            .collect();
        while lines.last().is_some_and(String::is_empty) {
            lines.pop();
        }
        lines.join("\n")
    }
}

impl Size {
    fn at_least_one(self) -> Size {
        Size { rows: self.rows.max(1), cols: self.cols.max(1), cell_w: self.cell_w.max(1), cell_h: self.cell_h.max(1) }
    }

    fn pty(self) -> PtySize {
        let px = |n: u16, cell: u32| u16::try_from(u32::from(n) * cell).unwrap_or(u16::MAX);
        PtySize { rows: self.rows, cols: self.cols, pixel_width: px(self.cols, self.cell_w), pixel_height: px(self.rows, self.cell_h) }
    }
}

fn rgb(c: RgbColor) -> Rgb {
    (c.r, c.g, c.b)
}

#[cfg(test)]
#[path = "screen_test.rs"]
pub(crate) mod tests;

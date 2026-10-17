# Agents

This project is kept coherent by [Anchors](https://github.com/co2-lab/anchors).

Read `CONTRIBUTING.md` before any change: it says how work flows here — the order spec → feature → test → code, the layers, the commands, the gates that bar a commit, and how to watch long commands with `anchors monitor`.

- `anchors guide` — the rules for writing each piece; `anchors work <stage> --for <target>`
  — the prompt of one task; `anchors next` — the next task.
- Launch every long command in the background through `anchors monitor run -- <command>`,
  and keep `anchors monitor` running while it works.
- Never commit with `--no-verify`: a gate that fails is fixed, or waived in the commit
  message with its reason (`[skip-<rule>@<CODE>: why]`).

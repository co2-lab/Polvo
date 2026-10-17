# Header guide — project

> The annotation block at the top of EVERY file in this project. Seeded by
> `anchors init`; it is the built-in ruler (`anchors guide header`) instantiated for
> the language here. Mandatory: a file without this header is invisible to what
> Anchors does best.

## The block, in this language's dialect

In CODE/test/feature (references the spec unit):

```
// @anchors
//   ref: LGNN             # references the owning unit (the spec); NOT ownership
//   updated_at: 2026-08-08 # day of the last change (the gate checks vs. git)
//   layer: screen         # Structure layer (normally inferred from the path)
//   @feature: auth
```

In the SPEC (the OWNER of the identity):

```
// @anchors
//   code: LGNN            # the spec OWNS the code
//   updated_at: 2026-08-08
```

## The annotations

- `code:` — OWNERSHIP of the identity (the SPEC is the owner). `ref:` — REFERENCE
  (code/feature/test point to the spec unit; it may be multiple: `ref: A, B`). Every
  file needs one of the two. Generate the code with `anchors code <name>`.
- `updated_at:` — the day of the last change. Whoever changes it updates it; the gate
  `updated-at-atual` checks against git (year-month-day only) and `anchors check --fix`
  fixes it. Do NOT make up the date — let it match the commit.
- `layer:` — the layer; normally inferred from the path, declare it only to override.
- `@feature: <name>` — a free GROUPING label for the vertical module.  Nothing reads it: no gate, no edge.
  The vertical axis that IS confronted is product doctrine — `product/<name>.doctrine.md`,
  which the spec points at with `@realizes` (see `anchors guide product`).
  The tag groups; the doctrine decides. The mechanism grows toward FILTER and VIEW,
  never toward a gate that demands the label.
- `@noPropagation`, `@anchors-shared-code` — honest opt-outs (always with the why alongside).

## Rules

- Always at the TOP of the file: only blank lines, comments and a shebang before it.
  A block below the code is read as text, not as the header. When the language truly
  demands something first, declare why inside the block: `@fixed-header: <why>`.
- `code` is the mandatory minimum (gate `header-valid`).
- `updated_at` matches the day of the last commit (gate `updated-at-atual`; `--fix` repairs it).
- Opt-out always with a why alongside.

## Compliance points

Each item is verifiable in a single file, in isolation. It is what a judgment
gate confronts — and what keeps judgment from turning into heuristics.

- **CK1** — the block is at the TOP of the file, before any code.
- **CK2** — there is a `code:` line with a single identity code.
- **CK3** — the comment dialect is that of the file's language.
- **CK4** — `updated_at`, when present, is the day of the last commit that touched the file.
- **CK5** — every opt-out (`@no-…`, `@allow-…`) carries the why on the same line.

_(Complete, universal ruler: `anchors guide header`.)_

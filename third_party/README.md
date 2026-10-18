# third_party

Código de terceiros que o Polvo usa com mudanças nossas, até o upstream as ter.

- `charmbracelet-x-vt/` — o emulador de terminal do nó (`github.com/charmbracelet/x/vt`).
- `gpui-base/` e `gpui-component/` — os crates 0.7.3 do `gpui-kit` (`github.com/longbridge/gpui-kit`), com
  o `Panel::request_close`: a doca pergunta ao painel antes de o fechar, para o editor com alterações por
  gravar perguntar ao utilizador (`EDTRS-B06`). O patch está em `gpui-request-close.patch`; o
  `desktop/Cargo.toml` usa estas cópias pelo `[patch.crates-io]`. Quando o upstream tiver o gancho, voltam
  a ser os do crates.io.

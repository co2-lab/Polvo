<!-- @anchors
  code: KYSEA
  updated_at: 2026-10-17
-->
# Keys — a tecla que a janela recebe, traduzida para o terminal

> **Code**: `KYSEA`

## Overview
A janela recebe cada tecla com o nome dela (`"a"`, `"enter"`, `"up"`, `"-"`), o carácter que ela
escreveria no teclado do utilizador e os modificadores (Shift, Ctrl, Alt). O terminal precisa da tecla
física da libghostty-vt, dos modificadores e do texto sem modificar. É a libghostty-vt que, com o
estado do terminal, faz a sequência de bytes que vai para a shell: assim o modo das setas, o teclado do
kitty e as outras variações ficam certos sem os reimplementar.

O `Input` é a tecla traduzida, e o `translate` faz a tradução a partir do nome, do carácter e dos
`Modifiers` que a janela recebe. O `encode` dá os bytes do `Input` para o estado atual do terminal.

## Rules

### KYSEA-B01 — As letras e os algarismos são as teclas deles
O nome de uma letra (`"a"` a `"z"`) é a tecla dessa letra, e o de um algarismo (`"0"` a `"9"`) é a tecla
desse algarismo.

### KYSEA-B02 — As teclas com nome são as teclas delas
`enter`, `backspace`, `tab`, `escape`, `space`, as setas (`up`, `down`, `left`, `right`), `home`, `end`,
`pageup`, `pagedown`, `delete`, `insert` e as teclas de função `f1` a `f12` são as teclas da
libghostty-vt com o mesmo significado.

### KYSEA-B03 — A pontuação é a tecla física dela
`-`, `=`, `[`, `]`, `\`, `;`, `'`, `,`, `.`, `/` e `` ` `` são as teclas físicas que os escrevem num
teclado americano.

### KYSEA-B04 — Os modificadores passam
Shift, Ctrl e Alt passam como os modificadores da libghostty-vt.

### KYSEA-B05 — O texto é o carácter sem modificar
O texto é o carácter que a tecla escreve, quando ele não é um carácter de controlo. Sem ele (Ctrl+C), o
texto é o nome da tecla quando é um só carácter (`"c"`). Uma tecla com nome (`enter`, as setas) não tem
texto.

### KYSEA-B06 — Os bytes vêm da libghostty-vt
O `encode` dá a sequência que a libghostty-vt faz para o `Input` no estado atual do terminal: Enter dá
`\r`, Ctrl+C dá `0x03`, a seta para cima dá `ESC [ A`, e com o modo das setas da aplicação ligado dá
`ESC O A`.

### KYSEA-B07 — Uma tecla desconhecida passa o carácter
O nome de uma tecla que nenhuma regra conhece (`"ç"`, `"volumeup"`) é a tecla desconhecida, com o
texto quando o há: assim o carácter ainda chega à shell.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `key` | o nome de uma tecla, como a janela o dá | um nome que nenhuma regra conhece | este módulo: a tecla desconhecida (`KYSEA-B07`) |
| `key_char` | o carácter que a tecla escreve, ou nenhum | um carácter de controlo | este módulo: não é texto (`KYSEA-B05`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `KYSEA-E01` | a libghostty-vt não consegue criar o evento da tecla ou codificá-lo (sem memória) @no-scenario: a falta de memória da libghostty-vt não se provoca num teste | nenhum byte vai para a shell, e o aviso com o erro na saída de erro |

## Rule uses

| Rule | Uses |
| --- | --- |
| `KYSEA-B01` | `key` |
| `KYSEA-B02` | `key` |
| `KYSEA-B03` | `key` |
| `KYSEA-B04` | `key` |
| `KYSEA-B05` | `key`, `key_char` |
| `KYSEA-B06` | `key`, `key_char` |
| `KYSEA-B07` | `key`, `key_char` |
| `KYSEA-E01` | `key` @no-scenario: a falta de memória da libghostty-vt não se provoca num teste |

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none

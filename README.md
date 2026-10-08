# photon

the bits [rush](https://github.com/0xdeafcafe/rush) and [loafer](https://github.com/0xdeafcafe/loafer) both need, so they're written once.

| package | what |
| --- | --- |
| `canvas` | rows of styled text: cut, fit, splice, wrap, and ANSI out with styles only where they change |
| `cellw` | how many cells a string takes in a terminal |
| `fuzzy` | rush's command bar matcher |
| `hl` | rush's syntax highlighter, without the drawing: source in, classed runs out |
| `jsonx` | encoding/json/v2 with the one set of options |
| `keychain` | the macos keychain through `security` |
| `termimg` | which picture protocol the terminal speaks, and pictures in it |
| `theme` | colours written for a dark ground, moved onto whatever the terminal has |
| `uithread` | knows when the ui goroutine is busy, so disk, socket and process work can say so when it runs there |

until it's on github, both repos point at it with `replace github.com/0xdeafcafe/photon => ../photon`, so it has to sit beside them.

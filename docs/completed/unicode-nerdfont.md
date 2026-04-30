# Unicode 11 addon + Symbols Nerd Font

## Summary

Add proper support for emoji, fullwidth Unicode characters, and Nerd Font glyphs
(Powerline symbols, icons) in the terminal.

## Problem

1. **Emoji / fullwidth width**: xterm.js v5 default Unicode 6 mode treats many
   emoji and CJK fullwidth characters incorrectly, causing cursor misalignment.
2. **Nerd Font symbols**: Powerline status bars and icon-heavy TUI applications
   (e.g. Starship prompt, lualine.nvim) rendered tofu (□) because no Nerd Font
   was available in the font stack.

## Fix

### 1. Unicode 11 addon (`js/src/xterm.ts`)

```typescript
import { Unicode11Addon } from "@xterm/addon-unicode11";

const unicode11Addon = new Unicode11Addon();
this.term.loadAddon(unicode11Addon);
this.term.unicode.activeVersion = '11';
```

Xterm.js 5.x requires `allowProposedApi: true` in the terminal options to enable
the `unicode` property:

```typescript
this.term = new XTermTerminal({
    cursorBlink: true,
    allowProposedApi: true,
    ...
});
```

Unicode 11 uses a wider combining-character range (`z` table) and an expanded
fullwidth range that correctly grades emoji (👍, 🚀), many CJK characters,
and box-drawing symbols as `wcwidth=2` while keeping narrow symbols
(e.g. ✗ U+2717) at width 1.

### 2. Symbols Nerd Font (`resources/`)

- Bundled `SymbolsNerdFont-Regular.ttf` (2.4 MB) in `resources/fonts/`.
- Added `@font-face` in `resources/xterm_customize.css` with `unicode-range`
  restricted to the Private Use Area (PUA):

```css
@font-face {
    font-family: "Symbols Nerd Font";
    src: url("../fonts/SymbolsNerdFont-Regular.ttf") format("truetype");
    font-weight: normal;
    font-style: normal;
    unicode-range: U+E000-U+F8FF, U+F0000-U+FFFFD, U+100000-U+10FFFD;
}
```

- Extended `.xterm` font stack to include `"Symbols Nerd Font"` as the last
  fallback — normal text renders with the primary monospace fonts, and PUA
  codepoints (Nerd Font glyphs) are served by the Nerd Font.

### 3. Static file serving (`server/server.go`, `Makefile`)

- Added `fonts/` route in `server.go` to serve font files from embedded `static`.
- Added Makefile rule to copy `resources/fonts/` → `server/static/fonts/` before
  the Go build (`go:embed static`).

## Font stack (final)

```
"DejaVu Sans Mono", "Everson Mono", FreeMono, Menlo, Terminal,
monospace, "Apple Symbols", "Symbols Nerd Font"
```

| Layer | Purpose |
|---|---|
| DejaVu Sans Mono | Primary monospace (excellent Unicode coverage) |
| Everson Mono | Broad script coverage fallback |
| FreeMono / Menlo / Terminal | System fallbacks |
| Apple Symbols | macOS symbol glyphs |
| Symbols Nerd Font (PUA only) | Powerline, devicons, etc. |

## Files changed

| File | Change |
|---|---|
| `js/package.json` | Added `@xterm/addon-unicode11@^0.9.0` |
| `js/package-lock.json` | Lockfile update |
| `js/src/xterm.ts` | Unicode11Addon import + activation, `allowProposedApi: true`, extended fontFamily |
| `js/dist/gotty-bundle.js` | Rebuilt bundle |
| `js/dist/gotty-bundle.js.LICENSE.txt` | New — webpack license output |
| `resources/xterm_customize.css` | `@font-face` for Nerd Font, updated `.xterm` and `.xterm-overlay` font-family |
| `resources/fonts/SymbolsNerdFont-Regular.ttf` | New — bundled font asset |
| `server/server.go` | Added `fonts/` static route |
| `Makefile` | Font copy rule + `asset` dependency |

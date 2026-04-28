# iPadOS Safari Ctrl+C fix

## Problem

When connecting to GoTTY from iPad Safari using a hardware keyboard (Magic Keyboard or
similar), pressing Ctrl+C produces an Enter (`\r`, 0x0d) instead of SIGINT (`\x03`).

Ctrl+D and other Ctrl+letter combinations (such as Ctrl+B for tmux prefix) work
correctly.

## Root cause

The bug is in **iPadOS Safari's keyboard event handling**, not in GoTTY.

When the user presses Ctrl+C on an iPad hardware keyboard:

1. iOS intercepts Ctrl+C at the system level as the **Copy** shortcut.
2. Safari dispatches a `keydown` event, but **`e.keyCode` is incorrectly set to `13`**
   (Enter's keyCode) instead of the expected `67` (keyCode for `C`).
3. `e.key` is correctly reported as `"c"`, and `e.ctrlKey` as `true`.
4. xterm.js v2's `evaluateKeyEscapeSequence()` method dispatches on `e.keyCode`
   (`switch (e.keyCode)`), so `case 13` routes the event to the Enter handler,
   emitting `\r` (0x0d) on the `data` channel.
5. The server receives `\r` and writes it to the PTY, which behaves like pressing
   Enter — not SIGINT.

The key evidence from the debug log (`./gotty --debug -w`):

```
ws input_debug payload=[kb-debug] keydown {
  "key": "c",         ← correct
  "code": "KeyC",     ← correct
  "keyCode": 13,      ← BUG: should be 67
  "ctrlKey": true     ← correct
}

ws input hex=0d       ← Enter (\r) sent instead of \x03
```

## Fix

**File:** `js/src/xterm.ts` — method `setupIPadOSKeyboardFix()`

A capture-phase `keydown` listener is attached to xterm's hidden textarea.
It fires **before** xterm.js own bubbling listener and detects the bogus
`keyCode=13 + ctrlKey + key="c"` combination:

```typescript
textarea.addEventListener(
    "keydown",
    (e: KeyboardEvent) => {
        if (e.ctrlKey && (e.key === "c" || e.key === "C") && e.keyCode === 13) {
            e.preventDefault();
            e.stopImmediatePropagation();
            if (this.inputCallback) {
                this.inputCallback("\x03");  // send correct Ctrl+C
            }
        }
    },
    true // capture phase
);
```

The guard condition is very specific to avoid false positives:

| Scenario | `ctrlKey` | `key` | `keyCode` | Triggers fix? |
|---|---|---|---|---|
| iPadOS Ctrl+C | `true` | `"c"` | `13` | **Yes** |
| Desktop Ctrl+C | `true` | `"c"` | `67` | No — `keyCode !== 13` |
| Normal Enter | `false` | `"Enter"` | `13` | No — `ctrlKey === false` |
| Normal `c` key | `false` | `"c"` | `67` | No — `ctrlKey === false` |

## Debug diagnostics

When GoTTY is started with `--debug` (or `-d`), the frontend registers
additional capture-phase listeners that dump every `keydown`, `keypress`,
`beforeinput`, and `copy` event to both the browser console and the server
log over a dedicated WebSocket message type (`InputDebug = '4'`).

```bash
./gotty --debug -w bash
```

The server log will show:

```
ws input_debug remote=1.2.3.4:56789 payload=[kb-debug] keydown { ... }
ws input_debug remote=1.2.3.4:56789 payload=[kb-debug] beforeinput { ... }
ws input_debug remote=1.2.3.4:56789 payload=[kb-debug] xterm.data "..."
ws input          remote=1.2.3.4:56789 bytes=N hex=... permit_write=true
```

## Files changed

| File | Change |
|---|---|
| `js/src/xterm.ts` | `setupIPadOSKeyboardFix()` — intercepts bogus keyCode=13 for Ctrl+C; `attachKeyboardDebugListener()` — key event dumper for `--debug` mode |
| `js/src/webtty.ts` | Added `msgDebug = '4'` message type, `onDebug()` callback, `onDebug` to `Terminal` interface |
| `js/src/hterm.ts` | Added `onDebug()` stub (interface requirement) |
| `js/src/main.ts` | Passes `gotty_debug` flag to `Xterm` constructor |
| `resources/index.html` | Added `<script src="./debug.js">` |
| `server/server.go` | Registered `/debug.js` route |
| `server/handlers.go` | `handleDebugFlag()` — serves `var gotty_debug = true/false;` |
| `webtty/message_types.go` | Added `InputDebug = '4'` |
| `webtty/webtty.go` | Handles `InputDebug` messages; logs input hex in debug mode |

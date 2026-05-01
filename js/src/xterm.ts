import { Terminal as XTermTerminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { Unicode11Addon } from "@xterm/addon-unicode11";

export class Xterm {
    elem: HTMLElement;
    term: XTermTerminal;
    fitAddon: FitAddon;
    resizeListener: () => void;
    resizeDebounceTimer: number | null;
    debouncedResizeHandler: () => void;

    message: HTMLElement;
    messageTimeout: number;
    messageTimer: number;
    private inputCallback: ((input: string) => void) | null;
    private debugCallback: ((msg: string) => void) | null;
    private debugEnabled: boolean;

    constructor(elem: HTMLElement, debugEnabled?: boolean) {
        this.elem = elem;
        this.debugEnabled = debugEnabled === true;
        this.inputCallback = null;
        this.debugCallback = null;

        this.fitAddon = new FitAddon();
        const webLinksAddon = new WebLinksAddon();

        this.term = new XTermTerminal({
            cursorBlink: true,
            allowProposedApi: true,
            fontFamily:
                '"DejaVu Sans Mono", "Everson Mono", FreeMono, Menlo, Terminal, monospace, "Apple Symbols", "Symbols Nerd Font"',
        });

        // Activate the Unicode 11 character width addon so that emoji
        // and other fullwidth characters get correct 2-cell treatment
        // without affecting narrow symbols (e.g. ✗ U+2717 stays width 1).
        const unicode11Addon = new Unicode11Addon();
        this.term.loadAddon(unicode11Addon);
        this.term.unicode.activeVersion = "11";

        this.term.loadAddon(this.fitAddon);
        this.term.loadAddon(webLinksAddon);

        this.message = elem.ownerDocument.createElement("div");
        this.message.className = "xterm-overlay";
        this.messageTimeout = 2000;
        this.messageTimer = 0;

        this.resizeListener = () => {
            try {
                this.fitAddon.fit();
            } catch {
                // fit() may fail if the terminal element has zero dimensions
            }
            this.term.scrollToBottom();
            this.showMessage(
                `${this.term.cols}x${this.term.rows}`,
                this.messageTimeout,
            );
        };

        this.resizeDebounceTimer = null;
        this.debouncedResizeHandler = () => {
            if (this.resizeDebounceTimer !== null) {
                clearTimeout(this.resizeDebounceTimer);
            }
            this.resizeDebounceTimer = window.setTimeout(() => {
                this.resizeDebounceTimer = null;
                this.resizeListener();
            }, 150);
        };

        this.setupIPadOSKeyboardFix();
        if (this.debugEnabled) {
            this.attachKeyboardDebugListener();
        }

        this.term.open(elem);

        // Fit after the terminal is rendered
        this.resizeListener();
        window.addEventListener("resize", this.debouncedResizeHandler);
    }

    info(): { columns: number; rows: number } {
        return { columns: this.term.cols, rows: this.term.rows };
    }

    output(data: string): void {
        // The server sends base64-encoded UTF-8 text.
        // atob() decodes base64 to a binary string (latin-1 codepoints).
        // Convert to Uint8Array and pass directly to xterm, which handles
        // UTF-8 decoding internally with a stateful parser.  This correctly
        // handles multi-byte sequences (box-drawing, CJK) that may be split
        // across WebSocket message boundaries.
        const bytes = new Uint8Array(data.length);
        for (let i = 0; i < data.length; i++) {
            bytes[i] = data.charCodeAt(i);
        }
        this.term.write(bytes);
    }

    showMessage(message: string, timeout: number): void {
        this.message.textContent = message;
        this.elem.appendChild(this.message);

        if (this.messageTimer) {
            clearTimeout(this.messageTimer);
        }
        if (timeout > 0) {
            this.messageTimer = window.setTimeout(() => {
                if (this.message.parentNode === this.elem) {
                    this.elem.removeChild(this.message);
                }
            }, timeout);
        }
    }

    removeMessage(): void {
        if (this.message.parentNode === this.elem) {
            this.elem.removeChild(this.message);
        }
    }

    setWindowTitle(title: string): void {
        document.title = title;
    }

    setPreferences(_value: object): void {
        // hterm preferences are no-ops with xterm v5
    }

    onInput(callback: (input: string) => void): void {
        this.inputCallback = callback;
        this.term.onData((data: string) => {
            if (this.debugEnabled) {
                this.debugLog("xterm.data", data);
            }
            callback(data);
        });
    }

    // Provide a callback for sending debug messages over the WebSocket.
    // Called by WebTTY with its connection.send wrapped with msgDebug prefix.
    onDebug(callback: (msg: string) => void): void {
        this.debugCallback = callback;
    }

    onResize(callback: (columns: number, rows: number) => void): void {
        this.term.onResize(({ cols, rows }) => {
            callback(cols, rows);
        });
    }

    deactivate(): void {
        this.term.blur();
    }

    // ============================================================
    //  iPadOS keyCode=13 workaround
    // ============================================================

    // iPadOS Safari reports keyCode=13 (Enter) for Ctrl+C (key="c").
    // Intercept via attachCustomKeyEventHandler before xterm maps it
    // to \r and send \x03 instead.
    private setupIPadOSKeyboardFix(): void {
        this.term.attachCustomKeyEventHandler((e: KeyboardEvent): boolean => {
            // iPadOS bug: ctrlKey + key="c" but keyCode=13 instead of 67
            if (e.ctrlKey && (e.key === "c" || e.key === "C") && e.keyCode === 13) {
                if (this.inputCallback) {
                    this.inputCallback("\x03");
                }
                return false; // prevent xterm from processing
            }
            return true;
        });
    }

    // ============================================================
    //  Keyboard debug / diagnostics
    // ============================================================

    private debugLog(tag: string, payload: any): void {
        const msg = "[kb-debug] " + tag + " " + JSON.stringify(payload);
        console.log(msg);
        if (this.debugCallback) {
            this.debugCallback(msg);
        }
    }

    // Attach capture-phase listeners to xterm's textarea that log every
    // keyboard event in full detail.  This helps diagnose platform-specific
    // keyboard bugs such as Ctrl+C on iPadOS Safari.
    private attachKeyboardDebugListener(): void {
        const textarea = this.term.textarea;
        if (!textarea) {
            console.warn("[kb-debug] no textarea found");
            return;
        }

        textarea.addEventListener(
            "keydown",
            (e: KeyboardEvent) => {
                const info = {
                    type: e.type,
                    key: e.key,
                    code: e.code,
                    keyCode: e.keyCode,
                    which: e.which,
                    charCode: (e as any).charCode,
                    ctrlKey: e.ctrlKey,
                    altKey: e.altKey,
                    shiftKey: e.shiftKey,
                    metaKey: e.metaKey,
                    repeat: e.repeat,
                    isComposing: (e as any).isComposing,
                    location: e.location,
                };
                this.debugLog("keydown", info);
            },
            true, // capture phase
        );

        textarea.addEventListener(
            "keypress",
            (e: KeyboardEvent) => {
                const info = {
                    type: e.type,
                    key: e.key,
                    code: e.code,
                    keyCode: e.keyCode,
                    which: e.which,
                    charCode: (e as any).charCode,
                    ctrlKey: e.ctrlKey,
                    altKey: e.altKey,
                    shiftKey: e.shiftKey,
                    metaKey: e.metaKey,
                };
                this.debugLog("keypress", info);
            },
            true,
        );

        // Also capture beforeinput (modern input event used on iOS)
        textarea.addEventListener(
            "beforeinput" as any,
            (e: any) => {
                const info = {
                    type: e.type,
                    inputType: e.inputType,
                    data: e.data,
                    dataTransfer: e.dataTransfer ? "[present]" : null,
                    isComposing: e.isComposing,
                };
                this.debugLog("beforeinput", info);
            },
            true,
        );

        // Also catch copy events (iOS fires this for Ctrl+C)
        textarea.addEventListener(
            "copy",
            (e: ClipboardEvent) => {
                const sel = document.getSelection();
                const info = {
                    type: e.type,
                    selection: sel ? sel.toString() : "",
                    clipboardData: e.clipboardData ? "[present]" : null,
                };
                this.debugLog("copy", info);
            },
            true,
        );
    }

    reset(): void {
        this.removeMessage();
        this.term.reset();
    }

    close(): void {
        if (this.resizeDebounceTimer !== null) {
            clearTimeout(this.resizeDebounceTimer);
            this.resizeDebounceTimer = null;
        }
        window.removeEventListener("resize", this.debouncedResizeHandler);
        this.term.dispose();
    }
}

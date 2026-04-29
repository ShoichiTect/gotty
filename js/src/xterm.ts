import * as bare from "xterm";
import { lib } from "libapps"


bare.loadAddon("fit");

export class Xterm {
    elem: HTMLElement;
    term: bare;
    resizeListener: () => void;
    resizeDebounceTimer: number | null;
    debouncedResizeHandler: () => void;
    decoder: lib.UTF8Decoder;

    message: HTMLElement;
    messageTimeout: number;
    messageTimer: number;
    private inputCallback: (input: string) => void;
    private debugCallback: (msg: string) => void;
    private debugEnabled: boolean;


    constructor(elem: HTMLElement, debugEnabled?: boolean) {
        this.elem = elem;
        this.term = new bare();
        this.debugEnabled = debugEnabled === true;

        this.message = elem.ownerDocument.createElement("div");
        this.message.className = "xterm-overlay";
        this.messageTimeout = 2000;

        this.resizeListener = () => {
            this.term.fit();
            this.term.scrollToBottom();
            this.showMessage(String(this.term.cols) + "x" + String(this.term.rows), this.messageTimeout);
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

        this.term.on("open", () => {
            this.resizeListener();
            window.addEventListener("resize", this.debouncedResizeHandler);

            // iPadOS Safari keyboard fix:
            // On iPadOS with a hardware keyboard, pressing Ctrl+C produces a
            // keydown event with key="c", ctrlKey=true — but Safari reports
            // keyCode=13 (Enter) instead of keyCode=67 (C).  xterm.js v2's
            // evaluateKeyEscapeSequence dispatches on keyCode, so it routes
            // the event to the Enter (\r) handler.  This capture-phase
            // listener detects the bogus keyCode=13 + ctrlKey combination
            // and sends \x03 (the correct Ctrl+C byte) instead.
            //
            // See also: attachKeyboardDebugListener() activated by --debug.
            this.setupIPadOSKeyboardFix();

            // When debug mode is enabled, attach keyboard event listeners that
            // dump every keydown/keypress/beforeinput/copy event to the browser
            // console AND relay them to the server over the WebSocket debug
            // channel (message type 4).  Use this to diagnose platform-specific
            // keyboard bugs such as Ctrl+C on iPadOS Safari.
            if (this.debugEnabled) {
                this.attachKeyboardDebugListener();
            }
        });

        this.term.open(elem, true);

        this.decoder = new lib.UTF8Decoder()
    };

    info(): { columns: number, rows: number } {
        return { columns: this.term.cols, rows: this.term.rows };
    };

    output(data: string) {
        this.term.write(this.decoder.decode(data));
    };

    showMessage(message: string, timeout: number) {
        this.message.textContent = message;
        this.elem.appendChild(this.message);

        if (this.messageTimer) {
            clearTimeout(this.messageTimer);
        }
        if (timeout > 0) {
            this.messageTimer = setTimeout(() => {
                this.elem.removeChild(this.message);
            }, timeout);
        }
    };

    removeMessage(): void {
        if (this.message.parentNode == this.elem) {
            this.elem.removeChild(this.message);
        }
    }

    setWindowTitle(title: string) {
        document.title = title;
    };

    setPreferences(value: object) {
    };

    onInput(callback: (input: string) => void) {
        this.inputCallback = callback;
        this.term.on("data", (data) => {
            if (this.debugEnabled) {
                this.debugLog("xterm.data", data);
            }
            callback(data);
        });
    };

    // Provide a callback for sending debug messages over the WebSocket.
    // Called by WebTTY with its connection.send wrapped with msgDebug prefix.
    onDebug(callback: (msg: string) => void) {
        this.debugCallback = callback;
    };

    onResize(callback: (colmuns: number, rows: number) => void) {
        this.term.on("resize", (data) => {
            callback(data.cols, data.rows);
        });
    };

    deactivate(): void {
        this.term.off("data");
        this.term.off("resize");
        this.term.blur();
    }

    // ============================================================
    //  iPadOS keyCode=13 workaround
    // ============================================================

    // iPadOS Safari reports keyCode=13 (Enter) for Ctrl+C (key="c").
    // Intercept before xterm.js maps it to \r and send \x03 instead.
    private setupIPadOSKeyboardFix(): void {
        const textarea = (this.term as any).textarea as HTMLTextAreaElement;
        if (!textarea) return;

        textarea.addEventListener(
            "keydown",
            (e: KeyboardEvent) => {
                // iPadOS bug: ctrlKey + key="c" but keyCode=13 instead of 67
                if (e.ctrlKey && (e.key === "c" || e.key === "C") && e.keyCode === 13) {
                    e.preventDefault();
                    e.stopImmediatePropagation();
                    if (this.inputCallback) {
                        this.inputCallback("\x03");
                    }
                }
            },
            true // capture phase — fires before xterm.js own handler
        );
    }

    // ============================================================
    //  Keyboard debug / diagnostics
    // ============================================================

    // Log a message to the browser console and relay it to the server
    // over the WebSocket debug channel (message type 4).
    // Only called when debug mode is enabled.
    private debugLog(tag: string, payload: any): void {
        const msg = "[kb-debug] " + tag + " " + JSON.stringify(payload);
        console.log(msg);
        if (this.debugCallback) {
            this.debugCallback(msg);
        }
    }

    // Attach a capture-phase listener to xterm's hidden textarea that
    // logs every keyboard event in full detail.  This helps diagnose
    // platform-specific keyboard bugs such as Ctrl+C on iPadOS Safari.
    private attachKeyboardDebugListener(): void {
        const textarea = (this.term as any).textarea as HTMLTextAreaElement;
        if (!textarea) {
            console.warn("[kb-debug] no textarea found");
            return;
        }

        textarea.addEventListener(
            "keydown",
            (e: KeyboardEvent) => {
                const info = {
                    type:   e.type,
                    key:    e.key,
                    code:   e.code,
                    keyCode: e.keyCode,
                    which:  e.which,
                    charCode: (e as any).charCode,
                    ctrlKey:  e.ctrlKey,
                    altKey:   e.altKey,
                    shiftKey: e.shiftKey,
                    metaKey:  e.metaKey,
                    repeat:   e.repeat,
                    isComposing: (e as any).isComposing,
                    location: e.location,
                };
                this.debugLog("keydown", info);
            },
            true // capture phase
        );

        textarea.addEventListener(
            "keypress",
            (e: KeyboardEvent) => {
                const info = {
                    type:     e.type,
                    key:      e.key,
                    code:     e.code,
                    keyCode:  e.keyCode,
                    which:    e.which,
                    charCode: (e as any).charCode,
                    ctrlKey:  e.ctrlKey,
                    altKey:   e.altKey,
                    shiftKey: e.shiftKey,
                    metaKey:  e.metaKey,
                };
                this.debugLog("keypress", info);
            },
            true
        );

        // Also capture beforeinput (modern input event used on iOS)
        textarea.addEventListener(
            "beforeinput" as any,
            (e: any) => {
                const info = {
                    type:       e.type,
                    inputType:  e.inputType,
                    data:       e.data,
                    dataTransfer: e.dataTransfer ? "[present]" : null,
                    isComposing: e.isComposing,
                };
                this.debugLog("beforeinput", info);
            },
            true
        );

        // Also catch copy events (iOS fires this for Ctrl+C)
        textarea.addEventListener(
            "copy",
            (e: ClipboardEvent) => {
                const sel = document.getSelection();
                const info = {
                    type:         e.type,
                    selection:    sel ? sel.toString() : "",
                    clipboardData: e.clipboardData ? "[present]" : null,
                };
                this.debugLog("copy", info);
            },
            true
        );

        // Log what xterm.js emits as "data" (already done in onInput)
    }

    reset(): void {
        this.removeMessage();
        this.term.clear();
    }

    close(): void {
        if (this.resizeDebounceTimer !== null) {
            clearTimeout(this.resizeDebounceTimer);
            this.resizeDebounceTimer = null;
        }
        window.removeEventListener("resize", this.debouncedResizeHandler);
        this.term.destroy();
    }
}

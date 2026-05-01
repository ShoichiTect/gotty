import { Terminal as XTermTerminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
export declare class Xterm {
    elem: HTMLElement;
    term: XTermTerminal;
    fitAddon: FitAddon;
    resizeListener: () => void;
    resizeDebounceTimer: number | null;
    debouncedResizeHandler: () => void;
    message: HTMLElement;
    messageTimeout: number;
    messageTimer: number;
    private inputCallback;
    private debugCallback;
    private debugEnabled;
    constructor(elem: HTMLElement, debugEnabled?: boolean);
    info(): {
        columns: number;
        rows: number;
    };
    output(data: string): void;
    showMessage(message: string, timeout: number): void;
    removeMessage(): void;
    setWindowTitle(title: string): void;
    setPreferences(_value: object): void;
    onInput(callback: (input: string) => void): void;
    onDebug(callback: (msg: string) => void): void;
    onResize(callback: (columns: number, rows: number) => void): void;
    deactivate(): void;
    private setupIPadOSKeyboardFix;
    private debugLog;
    private attachKeyboardDebugListener;
    reset(): void;
    close(): void;
}

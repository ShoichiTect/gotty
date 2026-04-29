import * as bare from "xterm";
import { lib } from "libapps";
export declare class Xterm {
    elem: HTMLElement;
    term: bare;
    resizeListener: () => void;
    resizeDebounceTimer: number | null;
    debouncedResizeHandler: () => void;
    decoder: lib.UTF8Decoder;
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
    setPreferences(value: object): void;
    onInput(callback: (input: string) => void): void;
    onDebug(callback: (msg: string) => void): void;
    onResize(callback: (colmuns: number, rows: number) => void): void;
    deactivate(): void;
    private setupIPadOSKeyboardFix();
    private debugLog(tag, payload);
    private attachKeyboardDebugListener();
    reset(): void;
    close(): void;
}

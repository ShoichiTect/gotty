import { Xterm } from "./xterm";
import { Terminal, WebTTY, protocols } from "./webtty";
import { ConnectionFactory } from "./websocket";

// Globals injected by Go backend HTML template (see server/asset.go)
declare var gotty_auth_token: string;
declare var _gotty_term: string;
declare var gotty_debug: boolean;
declare var gotty_ping_interval: number;

const elem = document.getElementById("terminal");

if (elem !== null) {
    const term: Terminal = new Xterm(elem, gotty_debug);
    const httpsEnabled = window.location.protocol === "https:";
    const url =
        (httpsEnabled ? "wss://" : "ws://") +
        window.location.host +
        window.location.pathname +
        "ws";
    const args = window.location.search;
    const factory = new ConnectionFactory(url, protocols);
    const wt = new WebTTY(term, factory, args, gotty_auth_token, gotty_ping_interval);
    const closer = wt.open();

    window.addEventListener("unload", () => {
        closer();
        term.close();
    });
}

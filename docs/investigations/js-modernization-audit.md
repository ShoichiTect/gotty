# JS フロントエンド モダナイゼーション診断書

**対象**: `js/` ディレクトリ（GoTTY フロントエンド）  
**診断日**: 2026-05-01  
**診断者**: pi  

---

## 概要

`js/` 配下の TypeScript ソースおよびビルド設定に対し、ECMAScript 2024+ 基準で監査を実施した。
`var` の使用は限定的だが、CommonJS 残留、古い `target`、`==` 演算子、typo など複数の問題が見つかった。

---

## 発見事項

### 🔴 Critical — バグ・誤動作に直結

#### 1. `==` による緩い比較（`websocket.ts:37-38`）

```typescript
// ❌ 現在
if (this.bare.readyState == WebSocket.CONNECTING ||
    this.bare.readyState == WebSocket.OPEN) {
```

`===` にすべき。`==` は型強制が入り、意図しない truthy/falsy 評価を招く。

#### 2. パラメータ名の typo: `colmuns`（`webtty.ts` 複数箇所）

```typescript
// ❌ 現在
onResize(callback: (colmuns: number, rows: number) => void): void;
// ...
const resizeHandler = (colmuns: number, rows: number) => { ... };
// ...
onResize(callback: (colmuns: number, rows: number) => void): void {
```

すべて `columns` に修正すべき。interface 定義と実装で統一されていないとバグの温床になる。

---

### 🟡 Warning — 非推奨パターン・モダン化対象

#### 3. `webpack.config.js` が CommonJS（`require` / `module.exports`）

webpack 5 は ESM 設定ファイルを完全サポートしている。

```javascript
// ❌ 現在
const path = require('path');
module.exports = { ... };
```

```javascript
// ✅ 推奨
import path from 'node:path';
export default { ... };
```

#### 4. `tsconfig.json` のターゲットが古い

| 項目 | 現在 | 推奨 | 理由 |
|------|------|------|------|
| `target` | `es2018` | `es2022` 以上 | 2026年時点で全モダンブラウザが ES2022 をサポート |
| `module` | `commonjs` | `es2022` | webpack 5 が ESM ツリーシェイキングに対応 |
| `lib` | 未指定（暗黙） | `["es2024", "dom", "dom.iterable"]` | 明示指定で型の安全性向上 |

#### 5. `package.json` に `"type": "module"` がない

ESM 移行時に必須。webpack 設定を ESM 化する際に邪魔になる場合は `webpack.config.mjs` にリネームでもよい。

#### 6. 再代入不要な変数に `let` を使用（`webtty.ts`）

```typescript
// ❌ 現在
let connection = this.connectionFactory.create();
let pingTimer: ReturnType<typeof setInterval>;
let reconnectTimeout: ReturnType<typeof setTimeout>;
```

`connection` は reconnect 時に再代入されるので `let` で正しいが、  
`pingTimer` / `reconnectTimeout` は `setup()` クロージャ内でのみ代入 → `let` のままで問題ない。  
ただし他のローカル変数（`termInfo` 等）は `const` にできる。

#### 7. `declare var` のグローバル変数宣言（`main.ts:6-9`）

```typescript
declare var gotty_auth_token: string;
declare var gotty_term: string;
declare var gotty_debug: boolean;
declare var gotty_ping_interval: number;
```

これらは Go バックエンドが HTML テンプレートに埋め込むグローバル変数。  
`declare var` 自体は TypeScript の正しいイディオムであり問題ない。  
ただし `snake_case` は JS の命名慣習に反する（バックエンド側の命名のため修正不可）。

---

### 🟢 Info — スタイル・改善提案

#### 8. `String()` コンストラクタ（`xterm.ts`）

```typescript
// ❌ 現在
this.showMessage(String(this.term.cols) + "x" + String(this.term.rows), ...);
```

```typescript
// ✅ 推奨
this.showMessage(`${this.term.cols}x${this.term.rows}`, ...);
```

#### 9. 余分なメソッド末尾セミコロン（`websocket.ts`, `webtty.ts`）

```typescript
// ❌ 現在
constructor(...) { ... };
method() { ... };
```

TypeScript ではクラスメソッド定義末尾の `;` は不要（エラーにはならないがノイズ）。

#### 10. `as any` キャストの多用（`xterm.ts`）

`charCode` や `which` など非推奨プロパティにアクセスするための `as any`。
これは互換性のために残す。ただし `beforeinput` イベントは `InputEvent` 型で受けられる可能性がある。

#### 11. デバッグ用コードのプロダクション混入

`attachKeyboardDebugListener()` は `gotty_debug` フラグが立ったときのみ動作するが、  
コード自体は常にバンドルされる。webpack の tree-shaking / `DefinePlugin` で除去できるとより良い。

#### 12. `@TODO` コメントの放置（`main.ts:5`）

```typescript
// @TODO remove these
```

`declare var` の削除はできない（バックエンド依存）ため、TODO コメントを具体的な説明に更新すべき。

---

## 依存パッケージの状態

| パッケージ | 現在 | 最新 (2026-04) | 備考 |
|-----------|------|-----------------|------|
| `typescript` | ^5.4.5 | 5.8.x | アップデート推奨 |
| `webpack` | ^5.91.0 | 5.98.x | アップデート推奨 |
| `webpack-cli` | ^5.1.4 | 6.0.x | メジャーアップデートあり |
| `ts-loader` | ^9.5.1 | 9.5.x | ほぼ最新 |
| `@xterm/xterm` | ^5.5.0 | 5.5.x | 最新 |
| `@xterm/addon-fit` | ^0.10.0 | 0.10.x | 最新 |
| `@xterm/addon-web-links` | ^0.11.0 | 0.11.x | 最新 |
| `@xterm/addon-unicode11` | ^0.9.0 | 0.9.x | 最新 |

---

## 修正優先度マトリクス

| # | 項目 | 影響 | 工数 | 優先度 |
|---|------|------|------|--------|
| 1 | `==` → `===` | バグ防止 | 1分 | 🔴 即時 |
| 2 | typo `colmuns` → `columns` | 可読性・型安全性 | 5分 | 🔴 即時 |
| 3 | webpack.config CJS→ESM | モダン化 | 10分 | 🟡 今回 |
| 4 | tsconfig target/lib 更新 | コード生成最適化 | 5分 | 🟡 今回 |
| 5 | `package.json` type: module | ESM 移行の前提 | 1分 | 🟡 今回 |
| 6 | `let` → `const` 最適化 | 可読性 | 5分 | 🟡 今回 |
| 7 | `String()` → テンプレートリテラル | 可読性 | 1分 | 🟢 ついで |
| 8 | メソッド末尾 `;` 削除 | スタイル統一 | 3分 | 🟢 ついで |
| 9 | TODO コメント修正 | ドキュメント | 1分 | 🟢 ついで |
| 10 | husky + prettier + eslint | CI/CD 品質 | 30分 | 🟡 今回 |
| 11 | TS 5.4→5.8 アップデート | 型チェック強化 | 5分 | 🟢 次回 |
| 12 | `as any` 削減 | 型安全性 | 要調査 | 🟢 次回 |

---

## 修正後の理想形

```typescript
// main.ts
import { Xterm } from "./xterm";
import { Terminal, WebTTY, protocols } from "./webtty";
import { ConnectionFactory } from "./websocket";

// Global variables injected by Go backend HTML template
declare var gotty_auth_token: string;
declare var gotty_term: string;
declare var gotty_debug: boolean;
declare var gotty_ping_interval: number;

const elem = document.getElementById("terminal");
if (elem !== null) {
    const term: Terminal = new Xterm(elem, gotty_debug);
    const httpsEnabled = window.location.protocol === "https:";
    const url = `${httpsEnabled ? 'wss://' : 'ws://'}${window.location.host}${window.location.pathname}ws`;
    const args = window.location.search;
    const factory = new ConnectionFactory(url, protocols);
    const wt = new WebTTY(term, factory, args, gotty_auth_token, gotty_ping_interval);
    const closer = wt.open();

    window.addEventListener("unload", () => {
        closer();
        term.close();
    });
}
```

```typescript
// websocket.ts（抜粋）
isOpen(): boolean {
    return this.bare.readyState === WebSocket.CONNECTING
        || this.bare.readyState === WebSocket.OPEN;
}
```

```typescript
// webtty.ts（抜粋）
const resizeHandler = (columns: number, rows: number) => {
    connection.send(
        msgResizeTerminal + JSON.stringify({ columns, rows })
    );
};
```

---

## 関連ドキュメント

- [outdated-patterns.md](../plans/outdated-patterns.md) — Go バックエンド側の旧式パターン一覧
- [maintenance-plan.md](../plans/maintenance-plan.md) — 全体ロードマップ

# GoTTY 文字サイズ（font size）調査報告

## 結論

GoTTY のコマンドライン引数（`--width` や `--height` のような flags）で **font size を直接変更するオプションは存在しない**。

ただし、以下の方法で文字サイズの変更は可能：

1. **hterm を使う場合**：設定ファイル（`~/.gotty`）の `preferences.font_size` で変更可能
2. **xterm を使う場合**：現状、組み込みの設定方法はない（後述の改造 or カスタム HTML が必要）
3. **カスタム HTML を使う場合**：`--index` で独自 `index.html` を指定し、CSS で変更可能

---

## 調査詳細

### 1. CLI オプション一覧の確認

`./gotty --help` で確認した結果、font size 関連の flag は存在しない：

```
--width value    Static width of the screen
--height value   Static height of the screen
--term value     Terminal name to use on the browser (xterm or hterm)
```

これらは **PTY（擬似端末）の行列数** を制御するものであり、ブラウザ上での **文字の見た目の大きさ** ではない。

### 2. 既存の設定機構

GoTTY には `~/.gotty` 設定ファイルで指定できる `preferences` セクションがある。

```hcl
preferences {
    font_size = 15
}
```

この `preferences` は `server/options.go` の `HtermPrefernces` 構造体に対応している：

```go
// server/options.go
type HtermPrefernces struct {
    FontSize  int  `hcl:"font_size" json:"font-size,omitempty"`
    // ...
}
```

しかしこの設定は **hterm 専用** である。

### 3. フロントエンドでの適用フロー

#### hterm の場合

1. `server/handlers.go` の `processWSConn` で `webtty.WithMasterPreferences(server.options.Preferences)` が呼ばれる
2. `webtty/webtty.go` の `sendInitializeMessage()` で `SetPreferences` メッセージが WebSocket 経由で送信される
3. `js/src/webtty.ts` の `connection.onReceive` で `msgSetPreferences` を受信
4. `js/src/hterm.ts` の `setPreferences()` で hterm の pref に反映

```typescript
// js/src/hterm.ts
setPreferences(value: object) {
    Object.keys(value).forEach((key) => {
        this.term.getPrefs().set(key, value[key]);
    });
};
```

#### xterm の場合

1. 同じく `SetPreferences` メッセージが送信される
2. `js/src/xterm.ts` の `setPreferences()` を受信するが、**実装が空**

```typescript
// js/src/xterm.ts
setPreferences(value: object) {
    // 何もしない
};
```

また、xterm.js のインスタンス生成時にオプションを渡していない：

```typescript
// js/src/xterm.ts
constructor(elem: HTMLElement) {
    this.term = new bare();  // ← fontSize 等のオプションなし
}
```

### 4. xterm.js のバージョン

`js/package.json` より：

```json
"xterm": "^2.7.0"
```

xterm 2.x では以下の API で font size を設定可能：
- コンストラクタ: `new Terminal({ fontSize: 15 })`
- 動的変更: `term.setOption('fontSize', 15)`

---

## 現状で文字サイズを変える方法

### 方法A: hterm を使う（簡単）

```bash
./gotty --term hterm -w zsh
```

`~/.gotty` に以下を記述：

```hcl
preferences {
    font_size = 20
}
```

### 方法B: カスタム index.html を使う

```bash
./gotty --index ./my-index.html -w zsh
```

`my-index.html` で CSS を上書き。xterm の文字サイズは `.xterm` クラスや canvas 要素に影響するため、CSS `transform: scale()` やフォント指定で調整可能（完全ではないが応急処置として使える）。

### 方法C: ソースコードを改造する

以下の変更で `--font-size` フラグ（または `preferences.font_size` を xterm にも適用）を追加できる。

---

## ソースコード改造案

### 案1: xterm でも `preferences.font_size` を反映させる（最小変更）

**変更箇所: `js/src/xterm.ts`**

```typescript
setPreferences(value: any) {
    if (value["font-size"] !== undefined) {
        this.term.setOption("fontSize", value["font-size"]);
    }
    // 他のオプションも必要に応じて
};
```

ただし xterm 2.7.0 で `setOption` が動作するかは要検証。動作しない場合はコンストラクタで受け取る必要がある。

### 案2: `--font-size` CLI flag を新設する

**変更箇所①: `server/options.go`**

```go
type Options struct {
    // ... 既存フィールド ...
    FontSize int `hcl:"font_size" flagName:"font-size" flagDescribe:"Font size of the terminal" default:"0"`
}
```

**変更箇所②: `server/handlers.go`**

`handleConfig` で `gotty_font_size` 変数を追加：

```go
func (server *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/javascript")
    w.Write([]byte("var gotty_term = '" + server.options.Term + "';\n"))
    w.Write([]byte("var gotty_font_size = '" + fmt.Sprint(server.options.FontSize) + "';\n"))
}
```

**変更箇所③: `js/src/main.ts`**

```typescript
declare var gotty_font_size: string;

// ...
if (gotty_term == "hterm") {
    term = new Hterm(elem);
} else {
    const fontSize = parseInt(gotty_font_size || "0", 10);
    term = new Xterm(elem, fontSize || undefined);
}
```

**変更箇所④: `js/src/xterm.ts`**

```typescript
export class Xterm {
    // ...
    constructor(elem: HTMLElement, fontSize?: number) {
        this.elem = elem;
        const opts: any = {};
        if (fontSize) opts.fontSize = fontSize;
        this.term = new bare(opts);
        // ...
    }
}
```

### 案3: `--index` で完全に独自フロントエンドを使う（コード変更なし）

`resources/index.html` をコピーして改造：

```html
<style>
.xterm-viewport,
.xterm-rows {
    font-size: 20px !important;
}
</style>
```

ただし xterm.js のレンダリングは canvas ベースなので、単純な CSS では表示が崩れる可能性がある。

---

## ファイル構成まとめ

| ファイル | 役割 |
|---------|------|
| `server/options.go` | CLI flag / 設定ファイルの構造体定義。`HtermPrefernces` に `FontSize` あり |
| `server/handlers.go` | HTTP/WebSocket ハンドラ。`handleConfig` で `gotty_term` を送信。`processWSConn` で preferences を WebTTY に渡す |
| `webtty/webtty.go` | WebTTY プロトコル実装。`sendInitializeMessage` で `SetPreferences` メッセージを送信 |
| `js/src/main.ts` | フロントエンドエントリポイント。`gotty_term` を見て Hterm / Xterm を選択 |
| `js/src/xterm.ts` | xterm.js ラッパー。`setPreferences` が空実装。コンストラクタで `new bare()` を呼ぶ |
| `js/src/hterm.ts` | hterm ラッパー。`setPreferences` で hterm の pref を更新 |
| `js/src/webtty.ts` | WebSocket 通信ロジック。`msgSetPreferences` を受信して `term.setPreferences` を呼ぶ |
| `resources/index.html` | 標準の HTML テンプレート。`--index` で差し替え可能 |
| `.gotty` | 設定ファイルサンプル。`preferences.font_size` の記述例あり |

---

## 最終回答

> 「このコマンドの args で文字サイズって調整できる？」

**できない。** 現状の GoTTY には `--font-size` のような flag はない。

文字サイズを変えるには：
- **hterm 利用時**: `~/.gotty` の `preferences.font_size` を使う
- **xterm 利用時**: ソースコードの改造（上記の改造案参照）または `--index` でカスタム HTML を使う
- **即効性を重視**: `./gotty --term hterm` で hterm に切り替え、`font_size` 設定を使うのが最も簡単

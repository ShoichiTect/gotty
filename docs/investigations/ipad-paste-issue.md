# iPad Safari paste issue

## Problem

iPad Safari から GoTTY 経由でリモートシェルに接続した際、iPad 上でコピーした内容を
ターミナルにペーストできないケースがある。

- コピー元のアプリ・コピー方法・ペースト方法・コンテンツ種別の組み合わせにより、
  成功するケースと失敗するケースが混在する。
- 単一の原因ではなく、**複数の独立したバグ**が重なっている可能性が高い。
- iPad Safari / Chrome ともに影響する可能性がある。

## Expected behavior

すべての組み合わせで、クリップボードのテキスト内容がターミナルにそのまま入力されること。
改行を含む複数行のペーストも、bracketed paste モードにより安全に処理されること。

---

## Test matrix

以下の全組み合わせ (`N × M × P × Q`) でテストし、結果を記録する。

### 凡例

| Mark | Meaning |
|------|---------|
| ✅ | Success — text pasted correctly |
| ⚠️ | Partial — text pasted but garbled/truncated |
| ❌ | Failure — nothing pasted |
| 🔥 | Crash / unexpected behavior |
| — | Not yet tested |

### Axis A: コピー元アプリ

| # | App | Category | Notes |
|---|-----|----------|-------|
| A1 | Safari | Browser | Web ページのテキスト選択 → コピー |
| A2 | Notes (メモ) | Native | プレーンテキスト / 箇条書き |
| A3 | Mail | Native | メール本文の一部 |
| A4 | Files | Native | ファイル名のコピー |
| A5 | Terminal (iOS app) | 3rd party | ターミナルアプリからのコピー |
| A6 | Obsidian | 3rd party | Markdown エディタ |
| A7 | Slack / Discord | 3rd party | メッセージのコピー |
| A8 | iA Writer / Ulysses | 3rd party | ライティングアプリ |
| A9 | Code editor (e.g. Textastic) | 3rd party | コード / シンタックス付き |
| A10 | Shortcuts | Native | ショートカットの出力 |
| A11 | Photos | Native | 写真からのテキスト認識 (Live Text) |
| A12 | Books | Native | Apple Books からの引用 |

### Axis B: コンテンツ種別

| # | Type | Example | Clipboard MIME |
|---|------|---------|----------------|
| B1 | 短いプレーンテキスト (英数字) | `hello world` | `text/plain` |
| B2 | 短い日本語テキスト | `こんにちは` | `text/plain` |
| B3 | 単一行 (80文字超) | 長いURLやパス | `text/plain` |
| B4 | 複数行 (2〜10行) | スクリプト片、設定 | `text/plain` |
| B5 | 複数行 (10行以上) | ログ出力、コードブロック | `text/plain` |
| B6 | タブ・空白を含むインデント | YAML / Python | `text/plain` |
| B7 | 特殊文字 (`$`, `\`, `"`, `'`, etc.) | シェルコマンド | `text/plain` |
| B8 | 制御文字 (`\x00`〜`\x1f`) | バイナリ混入 | `text/plain` (broken) |
| B9 | URL | `https://example.com` | `text/plain` + `text/uri-list` |
| B10 | メールアドレス | `user@example.com` | `text/plain` |
| B11 | 絵文字 | `🚀✨` | `text/plain` |
| B12 | 全角記号・波ダッシュ等 | `〜‖■` | `text/plain` |
| B13 | リッチテキスト (装飾付き) | Safari からの選択範囲 | `text/html` + `text/plain` |
| B14 | コードブロック (シンタックス付き) | VS Code Web 等 | `text/html` + `text/plain` |
| B15 | 空文字列 | (コピー失敗時) | (none or empty) |

### Axis C: コピー方法

| # | Method | Notes |
|---|--------|-------|
| C1 | Magic Keyboard Cmd+C | 標準キーボードショートカット |
| C2 | 長押し → 「コピー」 | タッチ選択後のコンテキストメニュー |
| C3 | ダブルタップ → 「コピー」 | Safari 等での単語選択 |
| C4 | トリプルタップ → 「コピー」 | 段落選択 |
| C5 | 選択ハンドル (グラバー) → コピー | iOS 標準のテキスト選択UI |
| C6 | ツールバーの「コピー」ボタン | アプリ独自のコピーボタン |
| C7 | Share Sheet → 「コピー」 | 共有メニュー経由 |
| C8 | Universal Clipboard (Handoff) | Mac でコピー → iPad でペースト |
| C9 | Shortcuts アプリ経由 | 自動化ワークフロー |

### Axis D: ペースト方法

| # | Method | Notes |
|---|--------|-------|
| D1 | Magic Keyboard Cmd+V | 標準キーボードショートカット |
| D2 | 長押し → 「ペースト」 | ターミナル上のコンテキストメニュー |
| D3 | ソフトウェアキーボード上のペーストボタン | キーボード上部の提案バー |
| D4 | フローティングキーボードのペースト | フローティングモード時 |
| D5 | 3本指ピンチアウト → ペースト | iOS ジェスチャー |
| D6 | AssistiveTouch 経由 | アクセシビリティ |

### Axis E: 環境

| # | Env | Notes |
|---|-----|-------|
| E1 | iPad Safari + Magic Keyboard | 最も標準的な構成 |
| E2 | iPad Safari + ソフトウェアキーボードのみ | タッチ専用 |
| E3 | iPad Chrome + Magic Keyboard | |
| E4 | iPad Chrome + ソフトウェアキーボードのみ | |
| E5 | iPad Safari + Stage Manager (外部ディスプレイ) | |
| E6 | iPad Safari + Tailscale | |
| E7 | iPad Safari + Cloudflare Tunnel | |

---

## Known failure patterns (hypotheses)

### P1: リッチテキスト混入による paste 失敗

Safari 等のブラウザからコピーすると、クリップボードに `text/html` と `text/plain` が
両方入る。xterm.js v2 が `text/html` を優先して読み取り、空または不正なデータを
ペーストしようとして失敗する可能性。

**影響する組み合わせ:** A1 (Safari), A3 (Mail), B13 (リッチテキスト)

### P2: ソフトウェアキーボードのペーストが paste イベントを発火しない

iPadOS のソフトウェアキーボード上部に表示されるペースト提案（またはフローティング
バー）は、`paste` イベントではなく `insertText` 等の入力イベントを生成する可能性。
これが xterm.js の textarea で正しく処理されない。

**影響する組み合わせ:** D3, D4 (ソフトウェアKB / フローティングKB)

### P3: 非表示 textarea への paste が iOS にブロックされる

xterm.js の textarea は `opacity: 0` で画面上に存在するが、iOS Safari が
「非表示の入力フィールド」へのクリップボードアクセスをセキュリティ上の理由で
拒否する可能性。

**影響する組み合わせ:** すべての D1〜D6

### P4: Universal Clipboard のタイミング問題

Mac でコピーしてから iPad でペーストするまでの間に Handoff 同期が完了せず、
空のクリップボードを読み取ってしまう。

**影響する組み合わせ:** C8 (Universal Clipboard)

### P5: 特殊文字・制御文字のエスケープ不足

日本語、絵文字、制御文字が xterm.js または WebSocket 経由で正しく転送されず、
文字化けやペースト中断が発生する。

**影響する組み合わせ:** B2, B7, B8, B11, B12

### P6: 長大テキストのペースト制限

大量の行や文字数を含むテキストをペーストすると、WebSocket フレームの制限、
xterm.js のバッファ制限、または PTY の入力バッファ制限に引っかかる。

**影響する組み合わせ:** B5 (10行以上)

---

## Suspected root causes (技術詳細)

### 1. Clipboard API の制限 (iOS Safari)

iOS Safari はセキュリティ上の理由から、Clipboard API (`navigator.clipboard.readText()`) を
制限している:

- `navigator.clipboard.readText()` は **ユーザージェスチャー（user activation）** が
  必要で、かつ HTTPS または localhost であること。
- 非同期 Clipboard API は `paste` イベント内でのみ許可される。
- xterm.js のデフォルトのペースト処理が、iOS Safari の制限に適合していない可能性がある。

### 2. paste イベントが正しく発火していない

xterm.js は内部の `textarea` で `paste` イベントを listen し、クリップボードデータを
読み取る。iPad Safari では:

- `paste` イベントが textarea に正しく届かない可能性。
- `clipboardData.getData('text/plain')` が空を返す可能性。
- ソフトウェアキーボードのペースト（コンテキストメニュー / フローティングバー）が
  `paste` イベントを生成しない可能性。

### 3. フォーカスと入力モードの問題

iPad Safari の xterm.js textarea:

- textarea が正しくフォーカスされているか。
- `readonly` 属性や `contenteditable` の影響。
- iOS の入力制御（`inputmode`, `autocorrect` 等）が paste イベントの伝播を妨げる可能性。

### 4. xterm.js のバージョン依存

現在の GoTTY fork が使用している xterm.js のバージョン（v2 系）が、
iOS Safari の paste 処理を正しくサポートしていない可能性がある。

### 5. MIME type の優先順位

`clipboardData` には複数の MIME type が格納されており (`text/html`, `text/plain`,
`text/uri-list` 等)、xterm.js が `text/plain` 以外を選択して失敗する可能性。

### 6. WebSocket フレーム制限

ペーストデータが WebSocket のメッセージサイズ制限（デフォルトで 32KB 程度の
実装が多い）を超えた場合、メッセージが分割または破棄される。

---

## Diagnostics

### ブラウザ側デバッグ

Safari Web Inspector (macOS からリモート接続) で以下を確認:

```javascript
// xterm.js の textarea 要素を取得
const textarea = document.querySelector('.xterm-helper-textarea');
if (textarea) {
    console.log('textarea found:', textarea);
    console.log('readOnly:', textarea.readOnly);
    console.log('contentEditable:', textarea.contentEditable);
    console.log('opacity:', getComputedStyle(textarea).opacity);
    console.log('position:', getComputedStyle(textarea).position);
}

// paste イベントのリスナーを追加して確認
document.addEventListener('paste', (e) => {
    console.log('paste event on document');
    console.log('  types:', e.clipboardData?.types);
    for (const type of (e.clipboardData?.types || [])) {
        console.log(`  ${type}:`, e.clipboardData?.getData(type));
    }
});

textarea?.addEventListener('paste', (e) => {
    console.log('paste event on textarea');
    console.log('  types:', e.clipboardData?.types);
    for (const type of (e.clipboardData?.types || [])) {
        console.log(`  ${type}:`, e.clipboardData?.getData(type));
    }
});

// 非表示 textarea の paste がブロックされるか確認
const visibleTextarea = document.createElement('textarea');
visibleTextarea.style.cssText = 'position:fixed;top:10px;left:10px;z-index:99999;width:300px;height:50px;';
document.body.appendChild(visibleTextarea);
visibleTextarea.addEventListener('paste', (e) => {
    console.log('paste on visible textarea:', e.clipboardData?.getData('text/plain'));
});
```

### テスト手順（プロトコル）

1. iPad Safari で GoTTY に接続 (`./gotty --debug -w bash`)
2. **コピー操作:** 指定のアプリから指定の方法でコピー
3. **ペースト操作:** 指定の方法で GoTTY ターミナルにペースト
4. 結果を記録 (✅/⚠️/❌/🔥)
5. Safari Web Inspector で以下を確認:
   - paste イベントの発火の有無
   - `clipboardData.types` 一覧
   - 各 MIME type のデータ内容
   - WebSocket メッセージの内容
   - サーバーログの `ws input hex=...` の内容

### ttyd での動作確認

ttyd でも同様のペースト問題が発生するか確認:

```bash
ttyd -p 7681 -W bash
```

ttyd が `@xterm/xterm` v5 を使用している場合、ペーストの挙動が異なる可能性がある。

---

## Possible fix directions

### Fix A: 非同期 Clipboard API の利用

xterm.js の paste 処理を拡張し、`navigator.clipboard.readText()` を使用する:

```typescript
// xterm.js の paste イベントハンドラ内
textarea.addEventListener('paste', async (e) => {
    e.preventDefault();
    try {
        const text = await navigator.clipboard.readText();
        if (text) {
            term.paste(text);
        }
    } catch (err) {
        // フォールバック: 同期 clipboardData
        const text = e.clipboardData?.getData('text/plain');
        if (text) {
            term.paste(text);
        }
    }
});
```

### Fix B: カスタムペーストハンドラの追加

GoTTY のフロントエンド (`js/src/xterm.ts`) に、capture フェーズでの
paste イベントハンドラを追加（`ipad-ctrl-c-fix.md` の Ctrl+C fix と同様のアプローチ）:

```typescript
textarea.addEventListener(
    'paste',
    (e: ClipboardEvent) => {
        e.preventDefault();
        e.stopImmediatePropagation();
        // 必ず text/plain を優先
        const text = e.clipboardData?.getData('text/plain');
        if (text && this.inputCallback) {
            this.inputCallback(text);
        }
    },
    true // capture phase
);
```

### Fix C: 手動ペーストボタンの追加

UI にペーストボタンを追加し、`navigator.clipboard.readText()` で
クリップボードを読み取って手動で入力する。

### Fix D: xterm.js のバージョンアップグレード

xterm.js を最新版（v5 系）にアップグレードすることで、
iOS Safari の paste 互換性が改善される可能性がある。

現在の GoTTY fork は xterm.js v2 系を使用。ttyd は v5 系を使用。

### Fix E: MIME type の明示的選択

`clipboardData.getData()` で必ず `text/plain` を指定し、
`text/html` 等のリッチテキストが誤って選択されるのを防ぐ。

### Fix F: document-level paste listener

textarea ではなく `document` 全体で paste イベントを capture し、
iOS Safari の非表示 textarea 制限を回避する。

```typescript
document.addEventListener(
    'paste',
    (e: ClipboardEvent) => {
        // xterm がアクティブなときのみ処理
        if (!document.querySelector('.terminal.xterm-focus')) return;
        e.preventDefault();
        const text = e.clipboardData?.getData('text/plain');
        if (text && this.inputCallback) {
            this.inputCallback(text);
        }
    },
    true // capture phase
);
```

---

## Investigation status

- [ ] **Phase 0:** Diagnostics infrastructure — paste イベントダンプを `--debug` に追加
- [ ] **Phase 1:** 最小テストマトリクス (A1×B1×C1×D1×E1) — Safari 基本ケース
- [ ] **Phase 2:** コピー方法バリエーション (C1〜C9)
- [ ] **Phase 3:** ペースト方法バリエーション (D1〜D6)
- [ ] **Phase 4:** コンテンツ種別バリエーション (B1〜B15)
- [ ] **Phase 5:** コピー元アプリバリエーション (A1〜A12)
- [ ] **Phase 6:** 環境バリエーション (E2〜E7)
- [ ] **Phase 7:** ttyd (xterm.js v5) での比較テスト
- [ ] **Phase 8:** Fix の実装と回帰テスト

---

## References

- [ipad-ctrl-c-fix.md](../completed/ipad-ctrl-c-fix.md) — 同様の iPad Safari キーボード問題の修正（capture-phase pattern）
- [ipad-ttyd-investigation.md](./ipad-ttyd-investigation.md) — ttyd vs GoTTY の iPad 互換性の調査
- [MDN: Clipboard API](https://developer.mozilla.org/en-US/docs/Web/API/Clipboard_API)
- [MDN: ClipboardEvent](https://developer.mozilla.org/en-US/docs/Web/API/ClipboardEvent)

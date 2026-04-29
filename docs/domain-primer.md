# GoTTY ドメイン知識入門

このプロジェクトを読むのに必要な用語を、コードのパスと一緒に説明する。
「この言葉がどのファイルのどこに出てくるか」がわかれば、コードを読むときに迷わないはず。

---

## 1. TTY / ターミナルってそもそも何？

**TTY = Teletypewriter（テレタイプライター）** の略。昔の物理的な「キーボード＋印刷機」の端末が語源。

今の意味: **「キーボードから入力を受け取って、文字を画面に表示する」という仕組み**。

あなたがターミナルアプリ（iTerm2, Terminal.app, Ghostty など）を開くと、裏では「疑似端末 = PTY」というものが動いている。

```
あなたのキーボード入力
      ↓
  ターミナルアプリ (iTerm2)
      ↓
  カーネルの PTY デバイス
      ↓
  シェル (bash/zsh) やコマンド
      ↓
  コマンドの出力
      ↓
  カーネルの PTY デバイス
      ↓
  ターミナルアプリ
      ↓
  あなたの画面
```

この「TTY の入力をどこから受け取るか／出力をどこに表示するか」を Web にすり替えたのが GoTTY。

---

## 2. PTY = 疑似端末 (Pseudo Terminal)

「擬似的な端末」。bash や vim のような対話型プログラムは **「自分はターミナルに接続されているか？」** をチェックする。ターミナルじゃないと判断すると、たとえば `ls` は色付き出力をやめる。そこで **「ターミナルっぽいふりをする仕組み」** が PTY。

```
  PTY マスター (プログラム側) ─── PTY スレーブ (シェル/コマンド側)
```

### コード上の出現箇所

| ファイル | 内容 |
|----------|------|
| `backend/localcommand/local_command.go:33` | `pty.StartWithAttrs(cmd, nil, cmd.SysProcAttr)` — コマンドをPTY付きで起動 |
| `backend/localcommand/local_command.go:40` | `lcmd.pty` — ここにPTYのファイルデスクリプタを持ってる |
| `backend/localcommand/local_command.go:52` | `lcmd.pty.Read()` — PTYからコマンドの出力を読む |
| `backend/localcommand/local_command.go:56` | `lcmd.pty.Write()` — PTYにキー入力を書き込む |
| `backend/localcommand/local_command.go:86-97` | `ResizeTerminal()` — `syscall.SYS_IOCTL` + `TIOCSWINSZ` でPTYのサイズを変える |

つまり `local_command.go` が「PTYを直接触っている唯一のファイル」。

---

## 3. Master（マスター）と Slave（スレーブ）

TTY の世界では「操作する側」と「操作される側」をこう呼ぶ。古い用語だが慣習。

| 用語 | GoTTY での意味 | 実体 |
|------|---------------|------|
| **Master** | ターミナルを触ってる側 | WebSocket でつながったブラウザ |
| **Slave** | ターミナルの中で動いてる側 | PTY を通して起動されたコマンド (bash) |

### コード上の出現箇所

| ファイル | 内容 |
|----------|------|
| `webtty/master.go:7` | `type Master io.ReadWriter` — たった1行。読み書きできるものなら何でも Master |
| `webtty/slave.go:8-15` | `type Slave interface` — `ReadWriter` + `WindowTitleVariables()` + `ResizeTerminal()` |
| `server/slave.go:11-14` | `type Slave interface` — webtty.Slave + `Close()` を足した拡張版 |
| `server/handlers.go:86` | `slave, err = server.factory.New(params)` — クライアント接続ごとに Slave を生成 |
| `server/handlers.go:119` | `tty, err := webtty.New(&wsWrapper{conn}, slave, opts...)` — Master(WS) と Slave を結合 |
| `server/ws_wrapper.go:10-25` | `wsWrapper` — `*websocket.Conn` に `io.ReadWriter` のアダプタをかぶせて Master にする |
| `webtty/webtty.go:56-58` | `WebTTY` 構造体が `masterConn` と `slave` の両方を持つ |

ポイント: **Master と Slave は直接通信しない。`webtty.go` の `WebTTY.Run()` が両者の橋渡しをする。**

---

## 4. WebTTY プロトコル

Master と Slave の間の通信ルール。最初の1バイトで「何のデータか」を区別する。

### コード上の出現箇所

| ファイル | 内容 |
|----------|------|
| `webtty/message_types.go:3-20` | 全メッセージタイプの定数定義 (`Input = '1'`, `Ping = '2'`, `Output = '1'` ...) |
| `webtty/webtty.go:13-15` | `Protocols = []string{"webtty"}` — WebSocket のサブプロトコル名 |
| `webtty/webtty.go:79-123` | `handleSlaveReadEvent()` — Slave→Master 方向の処理（出力をBase64エンコードして送信） |
| `webtty/webtty.go:133-191` | `handleMasterReadEvent()` — Master→Slave 方向の処理（入力/リサイズ/Ping を振り分け） |
| `webtty/webtty.go:62-77` | `sendInitializeMessage()` — 接続時にウィンドウタイトルや設定を送る |
| `js/src/webtty.ts:3-14` | TypeScript 側の同名の定数定義。Go側と一致させる必要がある |

### メッセージ早見表

Master → Slave:
| バイト | 定数名 | 意味 |
|--------|--------|------|
| `'1'` | Input | キーボード入力文字 |
| `'2'` | Ping | 生存確認 |
| `'3'` | ResizeTerminal | 画面リサイズ (JSON: `{Columns, Rows}`) |
| `'4'` | InputDebug | デバッグ用ペイロード |

Slave → Master:
| バイト | 定数名 | 意味 |
|--------|--------|------|
| `'1'` | Output | ターミナル出力 (Base64) |
| `'2'` | Pong | Ping の返事 |
| `'3'` | SetWindowTitle | ブラウザのタブタイトル変更 |
| `'4'` | SetPreferences | hterm の設定変更 |
| `'5'` | SetReconnect | 再接続の秒数設定 |

---

## 5. Factory（ファクトリー）

「クライアントが接続してくるたびに、新しいコマンドを起動する」ための仕組み。

```go
type Factory interface {
    Name() string
    New(params map[string][]string) (Slave, error)
}
```

### コード上の出現箇所

| ファイル | 内容 |
|----------|------|
| `server/slave.go:16-19` | `Factory` インタフェースの定義 |
| `backend/localcommand/factory.go:22-30` | `NewFactory()` — bash 等のコマンドを覚えておく |
| `backend/localcommand/factory.go:32-42` | `factory.New()` — 実際にコマンドを起動して Slave を返す |
| `main.go:56-60` | `localcommand.NewFactory(args[0], args[1:], backendOptions)` — 起動時の引数から Factory 作成 |
| `server/handlers.go:86` | `server.factory.New(params)` — WebSocket 接続時に呼ばれる |

### 流れ

1. `main.go` で `Factory` を作る（このとき実行するコマンドと引数を覚える）
2. ブラウザが WebSocket で接続 → `server/handlers.go` の `processWSConn()`
3. その中で `factory.New(params)` → 新しいコマンドプロセスが起動 → Slave が返る
4. `webtty.New(master, slave)` → 通信開始

デフォルトは `localcommand.Factory` だけど、インタフェースさえ満たせれば Docker や SSH 用の Factory も作れる設計。

---

## 6. ioctl / Winsize（端末サイズ変更）

PTY のサイズ（行数・列数）を変更するシステムコール。

### コード上の出現箇所

| ファイル | 内容 |
|----------|------|
| `backend/localcommand/local_command.go:86-97` | `ResizeTerminal()` — `syscall.SYS_IOCTL` + `TIOCSWINSZ` |
| `webtty/webtty.go:160-176` | ブラウザから `ResizeTerminal` メッセージを受け取って `slave.ResizeTerminal()` を呼ぶ |
| `js/src/webtty.ts:51-54` | ブラウザ側のリサイズイベント → WebSocket で `ResizeTerminal` 送信 |
| `server/options.go:27-28` | `Width` / `Height` オプション（0 なら動的リサイズ、>0 なら固定サイズ） |

---

## 7. HCL（設定ファイル）

HashiCorp Configuration Language。Terraform の `.tf` と同じ文法。

### コード上の出現箇所

| ファイル | 内容 |
|----------|------|
| `utils/flags.go:79-95` | `ApplyConfigFile()` — `hcl.Decode()` で設定ファイルを構造体にマッピング |
| `main.go:39-43` | 起動時に `~/.gotty` を読んで設定適用 |
| `.gotty` | 設定ファイルのサンプル（リポジトリルート） |
| `server/options.go` 全体 | 各フィールドに `hcl:"..."` タグで設定キーを指定 |
| `backend/localcommand/options.go` | 同様に hcl タグ |

---

## 8. 各ファイルが何をしてるか一覧

プロジェクトを上から読むときの順序も兼ねて:

| ファイル | 役割 | 読む優先度 |
|----------|------|-----------|
| `main.go` | エントリポイント。CLIフラグをパースし、Server を起動する | ⭐⭐⭐ |
| `help.go` | `--help` のテンプレート | ⭐ |
| `version.go` | `gotty --version` の値 | ⭐ |
| `server/server.go` | HTTPサーバの起動、TLS設定、ルーティング | ⭐⭐⭐ |
| `server/handlers.go` | WebSocket ハンドラ、クライアント接続処理、Slave生成 | ⭐⭐⭐ |
| `server/options.go` | 全設定項目の定義（構造体タグの塊） | ⭐⭐ |
| `server/middleware.go` | Basic認証、ログ、ヘッダーのミドルウェア | ⭐⭐ |
| `server/slave.go` | `Slave` インタフェース + `Factory` インタフェースの定義 | ⭐⭐⭐ |
| `server/handler_atomic.go` | 接続数カウンター + タイムアウト管理 | ⭐⭐ |
| `server/ws_wrapper.go` | WebSocket を `io.ReadWriter` に変換 | ⭐⭐ |
| `server/init_message.go` | 接続初期化メッセージの構造体 | ⭐ |
| `server/diagnostics.go` | デバッグログ出力 | ⭐ |
| `server/asset.go` | 静的ファイルの埋め込みデータ（自動生成、365行） | ⭐ |
| `server/run_option.go` | `Run()` のオプション（graceful shutdown 用） | ⭐ |
| `server/log_response_writer.go` | HTTP ステータスコードをログに記録するラッパー | ⭐ |
| `server/list_address.go` | サーバ起動時に IP アドレス一覧を表示する | ⭐ |
| `webtty/webtty.go` | **コアロジック**。Master↔Slave 間のデータ中継プロトコル | ⭐⭐⭐ |
| `webtty/master.go` | `Master` インタフェース (io.ReadWriter) | ⭐⭐ |
| `webtty/slave.go` | `Slave` インタフェース | ⭐⭐ |
| `webtty/message_types.go` | プロトコルのメッセージタイプ定数 | ⭐⭐⭐ |
| `webtty/option.go` | WebTTY のオプション（書込許可、固定サイズ、再接続...） | ⭐⭐ |
| `webtty/errors.go` | `ErrSlaveClosed`, `ErrMasterClosed` | ⭐ |
| `webtty/webtty_test.go` | WebTTY のテスト | ⭐⭐ |
| `backend/doc.go` | 空のパッケージ宣言のみ | - |
| `backend/localcommand/local_command.go` | **PTY の操作**。コマンド起動、Read/Write、リサイズ | ⭐⭐⭐ |
| `backend/localcommand/factory.go` | localcommand の Factory 実装 | ⭐⭐ |
| `backend/localcommand/options.go` | close-signal / close-timeout の設定 | ⭐ |
| `utils/flags.go` | 構造体タグ → CLIフラグの自動生成 / 設定ファイル読み込み | ⭐⭐ |
| `utils/default.go` | 構造体タグの `default:"..."` からデフォルト値を設定 | ⭐ |
| `pkg/homedir/expand.go` | `~` を `$HOME` に展開する | ⭐ |
| `pkg/randomstring/generate.go` | ランダムなURL文字列を生成 | ⭐ |

---

## 9. 全体の情報の流れ（コードリーディングの軸）

```
ブラウザでキー入力
  ↓
js/src/webtty.ts → WebSocket に書き込む (先頭バイト: '1' = Input)
  ↓
server/ws_wrapper.go → WebSocket から Read
  ↓
webtty/webtty.go:handleMasterReadEvent() → 先頭バイトで振り分け
  ↓  Input の場合:
local_command.go:Write() → PTY ファイルデスクリプタに書き込み
  ↓
bash が出力を生成
  ↓
local_command.go:Read() → PTY から読み出し
  ↓
webtty/webtty.go:handleSlaveReadEvent() → Base64 エンコード + 先頭バイト '1'(Output) 付与
  ↓
server/ws_wrapper.go → WebSocket に書き込み
  ↓
js/src/webtty.ts → 受け取って画面に表示
```

つまり覚えることは **「Master = WebSocket、Slave = PTY/コマンド、WebTTY = 翻訳屋さん」** の3つだけ。

# GoTTY の旧式 / 時代遅れな部分

このプロジェクト（yudai/gotty）は 2015〜2016 年ごろに書かれ、v2.0.0-alpha.3 で止まっている。
以下は現在（2026年）の Go やエコシステムの常識から見て「古い」または「良くないプラクティス」とされる箇所のリスト。

---

## 1. `io/ioutil` の使用

Go 1.16 で非推奨になった `io/ioutil` パッケージを各所で使っている。

| 該当ファイル | 使っている関数 | 置き換え先 |
|---|---|---|
| `server/server.go` | `ioutil.ReadFile()` | `os.ReadFile()` |
| `utils/flags.go` | `ioutil.ReadFile()` | `os.ReadFile()` |

---

## 2. `pkg/errors` によるエラーラッピング

`github.com/pkg/errors` の `errors.Wrapf()` / `errors.Errorf()` を使っている。
Go 1.13 以降、標準の `fmt.Errorf("...: %w", err)` で `%w` ラッピングが可能になったため、
外部パッケージは不要。

全 25 ファイル中、多数の関数で使用。

---

## 3. `go-bindata` による静的ファイル埋め込み

`server/asset.go`（365行）が `go-bindata` で自動生成されたコード。
Go 1.16 以降は `//go:embed` ディレクティブが標準機能として使える。

```go
// 現状（go-bindata）
func Asset(name string) ([]byte, error) { ... }

// 理想（//go:embed）
//go:embed static/index.html static/css/* static/js/*
var staticFS embed.FS
```

`go-bindata` を使うには専用ツールのインストールが必要で、ビルド手順が増える。

---

## 4. `unsafe` + `syscall.Syscall` による端末リサイズ

`backend/localcommand/local_command.go` でターミナルサイズ変更に生の `syscall.Syscall(SYS_IOCTL, ...)` と
`unsafe.Pointer` を使っている。

```go
// 現状
_, _, errno := syscall.Syscall(
    syscall.SYS_IOCTL,
    lcmd.pty.Fd(),
    syscall.TIOCSWINSZ,
    uintptr(unsafe.Pointer(&window)),
)
```

現在は `creack/pty` に `pty.Setsize()` があるので、そちらを使うべき。

```go
// 理想
import "github.com/creack/pty"
pty.Setsize(lcmd.pty, &pty.Winsize{Rows: height, Cols: width})
```

---

## 5. `syscall.SysProcAttr` の直指定

`local_command.go` でプロセス生成時の `SysProcAttr` を手動で設定している。

```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Setsid: true,
}
```

`creack/pty` を使う場合は `pty.Start()` が適切に設定してくれるため、
明示的に書く必要はないケースが多い。

---

## 6. 独自実装の IP アドレス列挙

`server/list_address.go` で `net.Interfaces()` から全インタフェースの
アドレスを取得して表示している。プライベートIP/Link-Local アドレスもすべて表示される。

実用的には `net.InterfaceAddrs()` を使うでもよいが、
そもそも表示が多すぎてうるさい。今書くなら `net/http/httptest` の
`Server.Listener.Addr()` だけ表示すれば十分なことが多い。

---

## 7. `fatih/structs` + リフレクションによるフラグ自動生成

`utils/flags.go` と `utils/default.go` で `github.com/fatih/structs` を使い、
構造体タグから CLI フラグとデフォルト値を自動生成している。

```go
// 現状：構造体タグにフラグ情報を埋め込む
type Options struct {
    Port string `hcl:"port" flagName:"port" flagSName:"p" flagDescribe:"..." default:"8080"`
}
```

このアプローチはリフレクションが複雑で、デバッグが難しい。
現代的なアプローチとしては `urfave/cli` v2（または `cobra`）にフラグを
手書きするほうが可読性が高い。もしくは構造体タグを使うにしても、
もっとシンプルなライブラリ（例: `jessevdk/go-flags`）に変える選択肢がある。

---

## 8. `yudai/hcl` による設定ファイル読み込み

HCL v1（HashiCorp Configuration Language の古いバージョン）のパーサである
`github.com/yudai/hcl` を fork して使っている。HCL は現在 v2 が標準。

- HCL v1 は 2019 年に EOL
- HashiCorp の公式 HCL v2 パーサ `github.com/hashicorp/hcl/v2` が存在する
- YAML や TOML で十分なユースケースも多い

---

## 9. `gziphandler` の適用方法

`github.com/NYTimes/gziphandler` で gzip 圧縮を実現しているが、
Go 1.21+ の `net/http` には標準の gzip サポートはないものの、
`net/http/httputil` やミドルウェアチェーンは改善されている。

また、このプロジェクトでは `gziphandler.GzipHandler()` で全ルートを
ラップしており、小さな静的ファイル（favicon.png など）まで圧縮しようとする。
現代的なアプローチでは `Accept-Encoding` を見ての条件付き圧縮を
よりきめ細かく制御する。

---

## 10. テスト不足

全 33 の Go ソースファイルのうち、テストファイルは `webtty/webtty_test.go` のみ。

| パッケージ | テストの有無 |
|-----------|-------------|
| `main` | ❌ |
| `server/` | ❌ |
| `webtty/` | ✅ (182行) |
| `backend/localcommand/` | ❌ |
| `utils/` | ❌ |
| `pkg/` | ❌ |

特に `server/` パッケージは HTTP ハンドラや WebSocket のロジックが複雑で、
テストが皆無なのは危険。

---

## 11. Windows 未対応

`syscall.SysProcAttr{Setsid: true}` や `TIOCSWINSZ` ioctl など、
Unix 固有のシステムコールに強く依存している。Windows でのビルドは不可能。

現代の Go では `golang.org/x/sys/windows` や `creack/pty` の Windows 対応
（`conpty` など）を使えばある程度対応できる可能性がある。

---

## 12. `ctx.Err()` による終了判定

`server/handlers.go` の `generateHandleWS()` 内で、
`webtty.ErrMasterClosed` / `webtty.ErrSlaveClosed` を返しているが、
エラーハンドリングのパターンがやや古い。

```go
err = server.processWSConn(ctx, conn)
switch err {
case ctx.Err():
    closeReason = "cancelation"
case webtty.ErrSlaveClosed:
    closeReason = server.factory.Name()
...
```

`errors.Is(err, context.Canceled)` が使われるべきだが、
`ctx.Err()` との直接比較になっている。（ただしこれは意図的な場合もある）

---

## 13. 依存関係がロックされていない / 古い

`go.sum` は存在するが、`go.mod` の各依存が最新から大きく離れている。

| 依存 | go.mod のバージョン | 最新 (2026年時点目安) |
|------|-------------------|---------------------|
| `gorilla/websocket` | v1.5.3 | 最新に近い |
| `creack/pty` | v1.1.24 | 最新に近い |
| `urfave/cli` | v1.22.17 | v2 が最新 |
| `pkg/errors` | v0.9.1 | メンテナンス終了 |
| `fatih/structs` | v1.1.0 | 最新に近いが置き換え推奨 |
| `yudai/hcl` | 2015年 | メンテナンス終了、HCL v2 が最新 |
| `NYTimes/gziphandler` | v1.1.1 | 互換性維持中 |
| `go-bindata-assetfs` | v1.0.1 | 非推奨 |

---

## 14. `--close-signal` のデフォルト値の可読性

`backend/localcommand/options.go` で `CloseSignal` のデフォルトが `1` と
数値で指定されている（SIGHUP = 1）。

```go
CloseSignal  int `hcl:"close_signal" flagName:"close-signal" ... default:"1"`
```

`syscall.SIGHUP` と書いたほうが意図が明確。

---

## 15. Makefile のレガシーなツールチェーン

`Makefile` で `go-bindata`、`gox`（クロスコンパイル）、`ghr`（GitHub Release）など、
現在ではあまり使われなくなったツールに依存している。

- `go-bindata` → `//go:embed` に置き換え可能
- `gox` → `GOOS=... GOARCH=... go build` で十分
- `ghr` → `gh` (GitHub CLI) の `gh release create` で代替可能

---

## 16. WebSocket コネクションカウンターのタイマー問題

`server/handler_atomic.go` の `newCounter()` で、`duration` が 0 のときに
`<-zeroTimer.C` をブロックしてチャネルをドレインしているが、
バッファなしタイマーチャネルで即座にドレインするこのパターンは
条件によっては直感的でない挙動を引き起こす可能性がある。

```go
func newCounter(duration time.Duration) *counter {
    zeroTimer := time.NewTimer(duration)
    if duration == 0 {
        <-zeroTimer.C  // タイマーが即発火するのを待つ
    }
    ...
}
```

より明確な方法として、`duration == 0` の場合はタイマーそのものを使わない
（`zeroTimer` を nil にする）設計のほうが安全。

---

## 17. デバッグフラグのグローバル変数的な使われ方

`server/diagnostics.go` は `server.options.Debug` を毎回チェックしているが、
デバッグフラグが構造体のフィールドとして渡り歩く設計になっている。
ログ出力のための条件分岐が各所に散らばっている。

モダンな Go では構造化ログ（`log/slog`、Go 1.21+）を使い、
ログレベルで制御するのが標準的。

---

## まとめ

優先度の高い修正候補（影響範囲が大きく、かつ作業量が少ないもの）:

1. **`io/ioutil` → `os` / `io` への置き換え**（機械的置換で完了）
2. **`pkg/errors` → `fmt.Errorf("%w")` への置き換え**（同上）
3. **`go-bindata` → `//go:embed` への移行**（メリット大、ビルド手順が簡略化）
4. **`unsafe` + `syscall.Syscall` → `pty.Setsize()`**（安全で短い）
5. **`yudai/hcl` → `hcl/v2` または YAML/TOML**（メンテナンス性向上）

# GoTTY + iPad セットアップ手順

このドキュメントは、Mac mini に GoTTY をビルド・デプロイし、iPad の Safari から Tailscale 経由でブラウザベースのターミナル（zsh）にアクセスする手順をまとめたものです。

## 前提条件

- macOS（今回は Apple Silicon Mac mini）
- Go 1.24.5 以上がインストールされていること
- Tailscale がインストールされ、同一 Tailnet に iPad が参加していること
- プロジェクトリポジトリ: `github.com/yudai/gotty`

## 1. ビルド

プロジェクトルートで以下を実行:

```bash
go build -o gotty -ldflags "-X main.Version=dev -X main.CommitID=$(git rev-parse HEAD | cut -c1-7)" .
```

ビルド成功を確認:

```bash
./gotty --version
# 出力例: gotty version dev+a85cc7c
```

## 2. 起動コマンド

zsh をブラウザで公開する場合:

```bash
./gotty -w -p 8080 zsh
```

### オプション解説

| オプション | 意味 |
|-----------|------|
| `-w` / `--permit-write` | クライアントからの入力（キー入力）を許可 |
| `-p 8080` | ポート番号を 8080 に指定 |
| `zsh` | 公開するシェル/コマンド |

### バックグラウンド実行（ログファイル出力）

```bash
./gotty -w -p 8080 zsh > /tmp/gotty.log 2>&1 &
echo $! > /tmp/gotty.pid
```

ログ確認:

```bash
tail -f /tmp/gotty.log
```

### 停止方法

```bash
kill $(cat /tmp/gotty.pid)
```

## 3. アクセス方法

### Tailscale IP の確認

```bash
/Applications/Tailscale.app/Contents/MacOS/Tailscale status
```

出力例:

```
100.127.116.118  vpss-mac-mini  duotianxiangyi@  macOS  -
```

### iPad Safari でアクセス

以下の URL を Safari で開く:

```
http://100.127.116.118:8080/
```

- `100.127.116.118` は Mac mini の Tailscale IP（各自の環境で置き換え）
- ポートは GoTTY の起動時に指定した `8080`

## 4. 注意事項

### Tailscale CLI のパス

macOS に Tailscale アプリをインストールしても、CLI (`tailscale`) はデフォルトで `PATH` に含まれない。フルパスで実行する必要がある:

```bash
# これは失敗する
which tailscale        # => not found
tailscale status       # => command not found

# フルパスで実行
/Applications/Tailscale.app/Contents/MacOS/Tailscale status
```

永続的に使いたい場合は `.zshrc` に追加:

```bash
export PATH="$PATH:/Applications/Tailscale.app/Contents/MacOS"
```

### SSH ではなく HTTP アクセス

GoTTY は Web サーバーであるため、iPad からは **ブラウザ（Safari/Chrome）** でアクセスする。SSH クライアントではなく、HTTP URL を開く必要がある。

### ファイアウォール

今回の環境では macOS のファイアウォールは無効だった。有効な場合はポート 8080 の受信を許可する必要がある。

## 5. ログ確認のコツ

GoTTY のアクセスログは標準出力/標準エラーに出力される。リアルタイムで確認するにはフォアグラウンド起動か `tail -f` を使う。

接続成功時のログ例:

```
2026/04/29 03:57:11 100.94.61.35:55351 200 GET /
2026/04/29 03:57:11 New client connected: 100.94.61.35:55358, connections: 1/0
```

- `100.94.61.35` は iPad の Tailscale IP
- `200 GET /` は HTTP 接続成功
- `New client connected` は WebSocket 接続確立

## 6. 参考: Makefile ベースのビルド

リポジトリに `Makefile` が含まれている場合、以下でもビルド可能:

```bash
make gotty
```

フロントエンドも更新する場合:

```bash
make all   # asset + gotty
```

（ただし `go-bindata` や `npm` が必要）

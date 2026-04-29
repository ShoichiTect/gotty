# HCL → JSON 設定ファイル移行計画

## 動機（Why）

- `github.com/yudai/hcl` は 2015 年のワンコミット fork。メンテ終了。Go の将来バージョンでビルドが通らなくなるリスクが高い
- `encoding/json` は標準パッケージ。依存を 1 つ削減できる
- `HtermPrefernces` にはすでに `json:"..."` タグが付いている（ブラウザに送る用）→ 流用すれば重複タグを半分消せる
- 設定ファイルを書くのは開発者だけなので、JSON の「コメントが書けない」デメリットは実害ゼロ

---

## 影響範囲マップ

```
main.go:39       utils.ApplyConfigFile() 呼び出し
                      ↓
utils/flags.go:117   hcl.Decode(object, string)   ← ここを json.Unmarshal に変える
                      ↓ 対象の構造体
server/options.go     Options{...}                ← hcl:"..." タグ (28個) → json:"..." に
                      └─ HtermPrefernces{...}     ← hcl:"..." タグ (54個) → 削除（json タグ既存）
backend/localcommand/factory.go  Options{...}     ← hcl:"..." タグ (2個) → json:"..." に
```

---

## 変更ファイル一覧

### 1. `go.mod` — 依存削除

```diff
- github.com/yudai/hcl v0.0.0-20151013225006-5fa2393b3552
```

削除後は `go mod tidy` が必要。

### 2. `utils/flags.go` — Decode ロジック差し替え

**コード変更 (1箇所 + import):**

```go
// 変更前
import "github.com/yudai/hcl"

func ApplyConfigFile(filePath string, options ...interface{}) error {
    // ...
    fileString, err := ioutil.ReadFile(filePath)  // ← ついでに os.ReadFile に
    // ...
    for _, object := range options {
        if err := hcl.Decode(object, string(fileString)); err != nil {
            return err
        }
    }
    // ...
}

// 変更後
import (
    "encoding/json"
    "os"
)

func ApplyConfigFile(filePath string, options ...interface{}) error {
    // ...
    fileBytes, err := os.ReadFile(filePath)
    // ...
    for _, object := range options {
        if err := json.Unmarshal(fileBytes, object); err != nil {
            return err
        }
    }
    // ...
}
```

**なぜ動くか:** もともと `hcl.Decode()` も `json.Unmarshal()` も、該当するキーだけ構造体にマッピングし、知らないキーは無視する。同じ設定ファイル JSON を `appOptions` と `backendOptions` の両方に同時に流し込める。

### 3. `server/options.go` — タグ整理

#### Options 本体 (28個): `hcl:"..."` → `json:"..."` に置換

```go
// 変更前
Port string `hcl:"port" flagName:"port" flagSName:"p" flagDescribe:"..." default:"8080"`

// 変更後
Port string `json:"port" flagName:"port" flagSName:"p" flagDescribe:"..." default:"8080"`
```

28 個すべて同じパターン。機械的置換で完了。

#### HtermPrefernces (54個): `hcl:"..."` を削除（`json:"..."` はすでにある）

```go
// 変更前
FontSize int `hcl:"font_size" json:"font-size,omitempty"`

// 変更後
FontSize int `json:"font-size,omitempty"`
```

54 個すべて。`hcl:"..."` だけを正規表現で消す。

### 4. `backend/localcommand/factory.go` — タグ置換 (2個)

```go
// 変更前
CloseSignal  int `hcl:"close_signal" flagName:"close-signal" ...`
CloseTimeout int `hcl:"close_timeout" flagName:"close-timeout" ...`

// 変更後
CloseSignal  int `json:"close_signal" flagName:"close-signal" ...`
CloseTimeout int `json:"close_timeout" flagName:"close-timeout" ...`
```

### 5. `.gotty` — 設定ファイルサンプルの書き換え

**変更前 (HCL):**
```hcl
// [string] Address to listen, all addresses will be used when empty
// address = ""

// [string] Port to listen
// port = "8080"

// [bool] Enable TLS/SSL
// enable_tls = false

// [string] Default TLS certificate file path
// tls_crt_file = "~/.gotty.crt"

preferences {
    // font_size = 5
    // background_color = "rgb(16, 16, 32)"
}
```

**変更後 (JSON):**
```json
{
    "_comment": "GoTTY configuration file. Remove // to enable a setting.",
    "_address": "0.0.0.0",
    "_port": "8080",
    "_enable_tls": false,
    "_tls_crt_file": "~/.gotty.crt",
    "preferences": {
        "_font_size": 5,
        "_background_color": "rgb(16, 16, 32)"
    }
}
```

JSON はコメント非対応なので `_comment` / `_xxx` プレフィックスで代用（unknown key として無視される）。実用上はコメント無しの最小構成で十分。

### 6. `README.md` — 設定ファイル例の更新

`### Config File` セクションの HCL 例を JSON に書き換える。

---

## リスクと注意点

| リスク | 深刻度 | 対策 |
|--------|--------|------|
| JSON の型が合わない | 低 | `*bool`、`*int`、`map[string]map[string]string` などすべて `encoding/json` が標準対応 |
| 既存ユーザーの `.gotty` との互換性 | ほぼゼロ | これはフォーク。既存ユーザーは yudai/gotty を使う。自分の設定ファイルを JSON に書き換えるだけ |
| `flagName` タグがなくなる | **なし** | `flagName`/`flagSName`/`flagDescribe`/`default` は CLI フラグ生成用。`hcl` タグとは完全に独立 |
| JSON のキーがフィールド名とマッチしない | 中 | `json:"..."` タグで明示するので問題なし。ケースインセンシティブマッチに頼らない |

---

## 検証チェックリスト

- [ ] `go build` が通る
- [ ] HCL の import がコードから完全に消えた (`go mod tidy` 後、go.mod に残っていない)
- [ ] `gotty --address 0.0.0.0 bash` で既存の CLI フラグが壊れていない
- [ ] 最小構成の `.gotty` JSON を作成し `gotty --config .gotty bash` で読み込める
- [ ] `preferences` ブロックが正しくパースされる
- [ ] `go vet ./...` が通る
- [ ] `webtty` のテストが通る (`go test ./webtty/...`)

---

## 作業見積もり

| 工程 | 時間 |
|------|------|
| `go.mod` 編集 + `go mod tidy` | 2分 |
| `utils/flags.go` 書き換え | 5分 |
| `server/options.go` タグ置換（82箇所） | 10分（正規表現一括置換） |
| `backend/localcommand/factory.go` タグ置換（2箇所） | 2分 |
| `.gotty` JSON化 | 10分 |
| `README.md` 更新 | 5分 |
| ビルド確認 + 動作検証 | 15分 |
| **合計** | **約50分** |

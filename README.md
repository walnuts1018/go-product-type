# go-adtgen

`go-adtgen` は、直和型や直積型、そしてそれらを扱うためのヘルパー関数を生成するツールです。

## インストール

```bash
go get -tool github.com/walnuts1018/go-adtgen
```

## 基本的な使い方

1. **生成用のファイルを作成する**
   ビルドタグ `//go:build adtgen_generate` を指定したファイル（例: `generate_types.go`）を作成し、生成したい型の定義を記述します。

   直和型を生成するには、`// +adtgen:sum`、直積型を生成するには、`// +adtgen:product`に続いて構造体名を指定します。

   ```go
   //go:build adtgen_generate

   package mypkg

   // +adtgen:sum StructA StructB
   type MySumType struct{}

   // +adtgen:product Struct1 StructB
   type MyProductType struct{}
   ```

2. **go:generate 指示語を追加する**
   パッケージ内の任意のファイル（通常のビルド対象ファイル）に以下を追記します。

   ```go
   //go:generate go tool go-adtgen
   ```

3. **コードを生成する**
   以下のコマンドを実行すると、各 `//go:build adtgen_generate` ファイルごとに、同じディレクトリへ `<source>_adtgen.go` が生成されます。たとえば `generate_types.go` からは `generate_types_adtgen.go` が生成されます。

   ```bash
   go generate ./...
   ```

## 直和型 (Sum Types) の使い方

`// +adtgen:sum <Variant1> <Variant2> ...` を使用すると、指定した構造体を候補とするインターフェースが生成されます。
候補として利用できる構造体は、同一パッケージ内で定義されている必要があります。

### 定義例

```go
type VariantA struct { Name string }
type VariantB struct { Value int }

// +adtgen:sum VariantA VariantB
type MySumType struct{}
```

### 生成される主な機能

#### パターンマッチング (`Match<Type>`)

型安全に値を処理できます。

```go
var val MySumType = ...

result := MatchMySumType(val,
    func(a VariantA) string {
        return "Aです: " + a.Name
    },
    func(b VariantB) string {
        return fmt.Sprintf("Bです: %d", b.Value)
    },
)
```

#### 安全なキャスト (`As<Variant>`)

特定の型であるかを確認し、変換します。

```go
if a, ok := val.AsVariantA(); ok {
    fmt.Println(a.Name)
}
```

#### 共通フィールドへのアクセス

全てのバリアントが共通のフィールド（埋め込み構造体など）を持っている場合、インターフェースに `Get<Field>` および `Set<Field>` メソッドが生成され、バリアントを意識せずにアクセスできます。

## 直積型 (Product Types) の使い方

`// +adtgen:product <Struct1> <Struct2> ...` を使用すると、指定した全ての構造体のフィールドを結合した新しい構造体が生成されます。

### 定義例

```go
type Base struct { ID string }
type Detail struct { Name string }

// +adtgen:product Base Detail
type Combined struct{}
```

### 生成される主な機能

#### コンストラクタ (`New<Type>`)

元の構造体群から結合された構造体を作成します。

```go
b := Base{ID: "123"}
d := Detail{Name: "Item"}
c := NewCombined(b, d)
```

#### 変換・抽出メソッド (`To<Struct>`)

結合された構造体から、元の個別の構造体を抽出します。

```go
base := c.ToBase()
detail := c.ToDetail()
```

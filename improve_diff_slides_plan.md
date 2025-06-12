# diffSlides 改善計画

## 目的

現在の `diffSlides` の実装を大幅に改訂することで、わかりやすく正確な機能を提供する

## 既存実装のまま維持しないといけないもの

- actionType
- deck.go にあるコード全て

## 前提

- 既存のdiffSlidesの実装は完全に無視します。

## アルゴリズム

以下の手順で `before` から `after` に変換するためのactionを生成します。

### 1. 初期化

`after` のスライドには *Slide と 現時点での index を使って、ユニークなキーを作成する。
`after` の全てのスライド（キー）とそのindexをトラッキングして、正しくアクションを生成するための情報を保持する（便宜上Stateと呼ぶ）

### 2. マッピング

割り当てアルゴリズムとしてハンガリアンアルゴリズムを採用する。

#### 2-1. 数調整

まず、`before` と `after` のそれぞれの `len([]*Slide)` が異なる場合はハンガリアンアルゴリズムを適用するための同数調整をする。

- `after` の `len([]*Slide)` が小さい場合は、まず、`before` の `*Slide` について `after` の全 `*Slide` との getSimilarity によるスコアの合計を算出し、スコアの小さい `before` のスライドから順に `after` の末尾にディープコピーして（ただし、`.new` を `true` にする）追加して数を同数にする。

- `before` の `len([]*Slide)` が小さい場合は、まず、`before` の各スライドについて `after` の全スライドとの getSimilarity によるスコアの合計を算出し、スコアの小さい `after` のスライドから順に `before` の末尾にディープコピーして（ただし、`.delete` を `true` にする）追加する。これで `len([]*Slide)` を同数にする。

#### 2-2. マッピング

この時点で、`before` と `after` のそれぞれの `len([]*Slide)` は同じになっている。

ここから、 `before` と `after` のスライドをそれぞれ getSimilarityForMapping で比べて、スコアの合計が最も高い形で *Slides を1:1でマッピングする。このときの割り当てアルゴリズムとしてハンガリアンアルゴリズムを採用する。

### 3. ソート

この時点で、`before` と `after` のスライドは同じ数で、かつ1:1でマッピングされている。
ここで、`before` の各スライドとマッピングされている `after` の各スライドのindexを `before` のそれと一致させるようなmoveアクションを生成し、その上でつどStateを更新する。

注意しなければならないのが、Google Slides APIを使用してmoveアクションを実施すると、つど各スライドのindexが更新されることだ。そのためのStateを設計すること。

例えば `before` がA B の2つのスライドで、`after` が B A の2つのスライドのとき、Google Slides APIを考慮すると、「Aのスライドをindex 0からindex 1へmoveするアクションを1つだけ」もしくは「Bのスライドをindex 1からindex 0へmoveするアクションを1つだけ」で済む。なぜならmove後に各スライドのindexが更新されるので、1つのmoveアクションを実行後にBもAも意図したindexになる。

### 4. 更新

`before` のスライドとのgetSimilarityによる類似度ポイントが500以上ではない `after` の各スライドに updateアクションを生成する。

## 注意点

注意しなければならないのが、Google Slides APIを使用してmoveアクションを実施すると、つど各スライドのindexが更新されることだ。そのためのStateを設計すること。

## 成功指標

1. まず既存テストケースが全て通るようになること。 `make test` で実行可能

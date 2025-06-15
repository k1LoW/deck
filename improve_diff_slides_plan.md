# diffSlides 作成計画

## 目的

`diffSlides` の実装を完成させる。

`diffSlides` に `before` と `after` を入力すると、`before` から `after` に変化させるための `[]*action` を返す。

`[]*action` を使用することで、Google Slides API を通じてスライドを操作することになる。

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

- `after` の `len([]*Slide)` が小さい場合は、まず、`before` の `*Slide` について `after` の全 `*Slide` との getSimilarity によるスコアの合計を算出し、スコアの小さい `before` のスライドから順に `after` の末尾にディープコピーして（ただし、`.delete` を `true` にする）追加して数を同数にする。

- `before` の `len([]*Slide)` が小さい場合は、まず、`before` の各スライドについて `after` の全スライドとの getSimilarity によるスコアの合計を算出し、スコアの小さい `after` のスライドから順に `before` の末尾にディープコピーして（ただし、`.new` を `true` にする）追加する。これで `len([]*Slide)` を同数にする。

#### 2-2. マッピング

この時点で、`before` と `after` のそれぞれの `len([]*Slide)` は同じになっている。

ここから、 `before` と `after` のスライドをそれぞれ getSimilarityForMapping で比べて、スコアの合計が最も高い形で *Slides を1:1でマッピングする。このときの割り当てアルゴリズムとしてハンガリアンアルゴリズムを採用する。

#### 2-3 削除マーク

この時点で、`before` と `after` のスライドは同じ数で、かつ1:1でマッピングされている。

`after` に存在する `.delete` が `true` になっている `*Slide` とマッピングされている `before` の `*Slide` も `.delete` を `true` にする

### 3. スライド操作アクション生成

この時点で、`before` と `after` のスライドは同じ数で、かつ1:1でマッピングされている。

また、 `before` の各 `*Slide` には、操作アクションとして新規作成対象であることを表す `.new` と、削除対象であることを表す `.delete` のマークがついている。

ここからここまでの結果を使用して `diffSlides` の返り値である `[]*action` を生成する

#### 3-1 内部アサーション

この時点での `before` と `after` の状態について確認する。

1. `before` と `after` の数が同じであること
2. `before` に `.new` がマークされた `*Slide` がある場合、それは末尾に連続して並んでいること
3. `after` に `.delete` がマークされた `*Slide` がある場合、それは末尾に連続して並んでいること
4. `before` の `.new` もしくは `.delete` がマークされた `*Slide` とマッピングされている `after` の `*Slide` との getSimilarity によるスコアは500であること

#### 3-2 update アクション生成

`before` の各 `*Slide` についてマッピングしている `after` の `*Slide` との getSimilarity によるスコアを確認して 500 未満の場合、update actionを生成する。

#### 3-3 append アクション生成

`before` に `.new` がマークされた `*Slide` がある場合、それらの append actionを生成する。

#### 3-4 `after` から `.delete: true` の `*Slide` を削除

この時点で `before` に必要な情報は揃ったのでハンガリアンアルゴリズムを使用するために数調整をした `after` の `.delete: true` の `*Slide` を全て削除する

#### 3-5 delete アクション生成

(1スライドごと順に作業すること)

`before` に `.delete` がマークされた `*Slide` がある場合、それらの delete actionを生成する。

そして実際に `before` に `.delete` がマークされた `*Slide` を削除する。

これにより `before` に `.delete` がマークされた `*Slide` 移行のスライドは index がつど -1 されはずだ。

#### 3-7 move アクション生成

(1スライドごと順に作業すること)

ここで、`before` の各スライドとマッピングされている `after` の各スライドのindexに `before` のそれを一致させるようなmoveアクションを生成する。

そして実際に `before` の `*Slide` を移動する。

このとき before と after のマッピングのindexも変化するはずなので更新すること。

例えば `before` がA B の2つのスライドで、`after` が B A の2つのスライドのとき、Google Slides APIを考慮すると、「Aのスライドをindex 0からindex 1へmoveするアクションを1つだけ」もしくは「Bのスライドをindex 1からindex 0へmoveするアクションを1つだけ」で済む。なぜならmove後に各スライドのindexが更新されるので、1つのmoveアクションを実行後にBもAも意図したindexになる。



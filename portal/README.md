# isucon7f-portal
ISUCON7 決勝ポータルサイトです。

## 運営アカウント

どの日も同じアカウントで入れます。

- ID: 9999
- PASS: `XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX`

※ パスワードは実際に使用したものから変更してあります。

## 環境構築

files/portalのファイルを配置

nginx.confはmax_client_body_sizeの設定がある
```sh
sudo cp files/portal.nginx.conf /etc/nginx/sites-available/default
sudo cp files/portal.day* /etc/systemd/system/
```

ビルド済のportalを持ってくる
```
ubuntu@portal:~$ tree portal
portal
|-- bin
|   |-- import
|   `-- portal
|-- data
|   |-- isu7f-servers.tsv
|   `-- isu7f-teams.tsv
`-- db
    |-- init.sh
    `-- schema.sql
```

mysql セットアップ
```
$sudo mysql -uroot
mysql>create user ubuntu@localhost identified by 'ubuntu';
mysql>grant all on *.* to ubuntu@localhost;
```

db初期化
```sh
portal/db/init.sh
```

チーム情報とサーバ情報をインポート
```sh
portal/bin/import -dsn-base "ubuntu:ubuntu@(localhost)" -target teams < portal/data/isu7f-teams.tsv
portal/bin/import -dsn-base "ubuntu:ubuntu@(localhost)" -target servers < portal/data/isu7f-servers.tsv
```


## 運用
### 起動
```sh
sudo systemctl enable portal.day1 # 1日目(本番)として起動
```

### コンテストの開閉
管理ユーザでログインし、管理ページからコンテスト状態を変更できます。

### スナップショット
終了一時間前あたりで `team_scores_snapshot` と `scores_snapshot` テーブルを作るとリーダーボードが固定されます。

```
INSERT INTO team_scores_snapshot SELECT * FROM team_scores
INSERT INTO scores_snapshot SELECT * FROM scores
```


## 開発・運用むけ情報

隠しページ

- キュー一覧 /XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX/debug/queue
- 非凍結leaderboard /XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX/debug/leaderboard
- ベンチマーク結果 /XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX/debug/result?id=123
- ベンチマークログ /XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX/debug/log?id=123
- 強制job追加 /XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX/debug/queuejob
- vars /XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX/debug/vars

管理ページ(要管理者IDでログイン)
- 管理ページ /XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX/admin
- サーバ一覧 /XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX/admin/server

※ Prefix は実際に使用したものから変更してあります。

## Portal 開発者向け情報

ベンダリングツールに [gb](https://getgb.io/) を使っています。
静的ファイルの埋め込みに [go-bindata](https://github.com/jteeuwen/go-bindata) を使っています。

```
go get -u github.com/constabulary/gb/...
go get -u github.com/jteeuwen/go-bindata/...
```

あとは Makefile を見てください。

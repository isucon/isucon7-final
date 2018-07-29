# isu7final

## ディレクトリ構成

```
db      - データベーススキーマ等
bench   - ベンチマーカー
webapp  - 各種言語実装
```

## 1VMでの実行環境構築

Ubuntu 16.04 LTS を使います。以下の手順はGCPの公式のUbuntu 16.04イメージで確認しています。

### apt install

nginx, MySQL, git, その他各言語をビルドするのに必要なパッケージをインストールしていきます。

MySQL のインストール時はrootパスワードを聞かれますが、空欄のままOKを押すと
`sudo mysql` でログインできるようになります。

```
sudo apt update
sudo apt install -y mysql-server nginx
sudo apt install -y git curl libreadline-dev pkg-config autoconf automake build-essential libmysqlclient-dev \
	libssl-dev python3 python3-dev python3-venv openjdk-8-jdk-headless libxml2-dev libcurl4-openssl-dev \
    libxslt1-dev re2c bison libbz2-dev libssl-dev gettext libgettextpo-dev libicu-dev libmhash-dev \
	libmcrypt-dev libgd-dev libtidy-dev libgmp-dev
```

### このリポジトリのチェックアウト

```
git clone https://github.com/isucon/isucon7-final cco
```

### 各言語のセットアップ

xbuild を使って各言語が以下の方法でインストールします。
ベンチマーカーのために Go は必ず必要です。それ以外の言語は省略しても構いません。


```
cd
git clone https://github.com/tagomoris/xbuild.git

mkdir local
xbuild/ruby-install   -f 2.4.2   $HOME/local/ruby
xbuild/perl-install   -f 5.26.1  $HOME/local/perl
xbuild/node-install   -f v8.9.1  $HOME/local/node
xbuild/go-install     -f 1.9.2   $HOME/local/go
xbuild/python-install -f 3.6.2   $HOME/local/python
xbuild/php-install    -f 7.1.9   $HOME/local/php -- --disable-phar --with-pcre-regex --with-zlib --enable-fpm --enable-pdo --with-mysqli=mysqlnd --with-pdo-mysql=mysqlnd --with-openssl --with-pcre-regex --with-pcre-dir --with-libxml-dir --enable-opcache --enable-bcmath --with-bz2 --enable-calendar --enable-cli --enable-shmop --enable-sysvsem --enable-sysvshm --enable-sysvmsg --enable-mbregex --enable-mbstring --with-mcrypt --enable-pcntl --enable-sockets --with-curl --enable-zip --with-pear --with-gmp
```

インストールした言語用に PATH 環境変数を設定します。次の設定を `.bashrc` などに
書いてログインし直してください。

```
export PATH=$HOME/local/go/bin:$HOME/go/bin:$PATH
export PATH=$HOME/local/ruby/bin:$PATH
export PATH=$HOME/local/python/bin:$PATH
export PATH=$HOME/local/node/bin:$PATH
export PATH=$HOME/local/perl/bin:$PATH
export PATH=$HOME/local/php/bin:$PATH
```

ベンチマーカーとGoのアプリをビルドするためには [dep](https://github.com/golang/dep)
をインストールします。

```
go get -u github.com/golang/dep/cmd/dep
```


## MySQL

データベース初期化. マスターデータの挿入:

```sh
sudo ~/cco/db/init.sh
```

appが使用するmysqlユーザを適当に作る:

```
sudo mysql -e  "GRANT ALL ON isudb.* TO local_user@localhost IDENTIFIED BY 'password'"
```

### nginx

`files/cco.nginx.conf` に nginx の設定ファイルがあります。
`root /home/isucon/webapp/public;` になっている部分を `root $HOME/cco/webapp/public;`
(`$HOME` の部分はホームディレクトリに置き換える)に書き換えてください。

```
sudo cp ~/cco/files/cco.nginx.conf /etc/nginx/sites-available
cd /etc/nginx/sites-enabled
sudo unlink *
sudo ln -s ../sites-available/cco.nginx.conf
sudo systemctl restart nginx
```

これで :80 番から localhost:5000 にリバースプロキシされるようになります。


### 参考実装(go)を動かす

ビルド:

```sh
cd ~/cco/webapp/go
make ensure
make
```

実行:

```sh
# 接続情報は必要に応じて環境変数へ
ISU_DB_HOST=localhost ISU_DB_PORT=3306 ISU_DB_USER=local_user ISU_DB_PASSWORD=password ./app
```

これで localhost:5000 でGo版のアプリが動きます。

systemd を利用して起動する場合は、 `files/` 配下にある各 service
ファイルを参照してください。アプリのあるパス、環境変数の設定、
User, Group などは適宜修正する必要があります。


### ベンチマーカー

ビルド方法:

```
cd ~/cco/bench
make ensure
make
```

実行:

```
cd ~/cco/bench
./bench
```

デフォルトでは localhost:5000 に対してベンチマークを行います。
攻撃先等の設定は `-h` でヘルプを参照のこと.


# 使用データの取得元

以下のサイトの画像データを使用させて頂きました。
各画像の取得元の URL は RESOURCES を参照してください.

- pixabay https://pixabay.com/
- jenkins https://wiki.jenkins.io/display/JENKINS/Logo (@Charles Lowell and  Frontside / CC BY-SA https://jenkins.io)
- いらすとや http://www.irasutoya.com/
- ぴぽや http://blog.pipoya.net/
- icon-rainbow http://icon-rainbow.com/
- 尾羽の小屋 http://obane.tuzikaze.com/
- なにかしらツク～ル http://nanikasiratkool.web.fc2.com/

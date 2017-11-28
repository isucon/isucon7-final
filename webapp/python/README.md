# Python 実装

## 開発環境

### セットアップ

[pipenv](http://pipenv-ja.readthedocs.io/ja/translate-ja/) を使います。

```
python -m pip install pipenv
pipenv install
```

### 実行

```
pipenv run python app.py
```

### 実行環境用の requirements.lock 作成

```
pipenv lock -r > requirements.lock
```

## 実行環境

### venv 構築

```
python -m venv .venv
.venv/bin/pip install -r requirements.lock
```

### 実行

```
.venv/bin/python app.py
```

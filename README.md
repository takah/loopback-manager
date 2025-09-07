# Loopback Manager

GitHub organization リポジトリのDocker Compose用loopback IPアドレス管理ツール

## 特徴

- 自動IP割り当て (127.0.0.10から順番)
- IP重複チェック
- 未割り当てリポジトリの自動検出
- .env ファイルの自動生成・更新
- 設定の永続化

## インストール

### バイナリダウンロード
```bash
# インストールスクリプト実行
curl -sf https://raw.githubusercontent.com/takah/loopback-manager/main/scripts/install.sh | bash
```

### ソースからビルド
```bash
git clone https://github.com/takah/loopback-manager.git
cd loopback-manager
go build -o loopback-manager
```

## 使用方法

```bash
# 全リポジトリの一覧表示
loopback-manager list

# 未割り当てリポジトリをスキャン
loopback-manager scan

# 手動でIPを割り当て
loopback-manager assign myorg myrepo

# 特定のIPを指定して割り当て
loopback-manager assign myorg myrepo --ip 127.0.0.50

# 全ての未割り当てリポジトリに自動でIP割り当て
loopback-manager auto-assign

# 重複チェック
loopback-manager check

# IP割り当てを削除
loopback-manager remove myorg myrepo
```

## 設定

デフォルトの設定ファイル: `~/.config/loopback-manager/config.yaml`

```yaml
base_dir: "~/github"
ip_range:
  base: "127.0.0"
  start: 10
  end: 254
```

環境変数での設定:
- `GITHUB_BASE_DIR`: GitHubリポジトリのベースディレクトリ

## ライセンス

MIT License

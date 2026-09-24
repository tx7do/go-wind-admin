<div align="center">

<img src="docs/brand/vortex-tile.svg" width="120" alt="GoWind 風行" />

# GoWind Admin｜風行

**すぐに使える企業級フロントエンド・バックエンド一体型管理システムスキャフォールド**

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vuedotjs)](https://vuejs.org/)
[![React](https://img.shields.io/badge/React-19.x-61DAFB?logo=react)](https://react.dev/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)](https://www.docker.com/)

[English](./README.en-US.md) | [中文](./README.md) | **日本語**

</div>

---

## プロジェクトのハイライト

- **3 フロントエンドは「3 択 1」、3 つ 1 組ではない**：`Vue3 Vben`（Ant Design Vue）、`Vue3 Element Plus`、`React19 Antd` は**同一バックエンドに対する 3 つの並列実装**です。異なるスタックのチームがそれぞれ慣れた方を選べるようにするためで、**1 チームが 1 つを取り、1 デプロイが 1 つを動かす**構成です。各エンドは独立したパッケージルートとデプロイスクリプトを持ち、相互依存はありません。選定後は他の 2 ディレクトリを削除できます（手順は [docs/adopt-one-frontend.md](./docs/adopt-one-frontend.md)・中国語参照）
- **エンタープライズ級 RBAC**：マルチテナント、マルチロール、マルチ部署、メニュー / ボタン / データレベルの権限制御（ポリシーエンジン切り替え可：Casbin / OPA）
- **セキュリティと等保コンプライアンス**：等保 2.0 の技術要求に準拠——監査ログの 180 日保持・アーカイブ、パスワードポリシー 3 点セット、TOTP MFA、パスワードのアプリケーション層暗号化、動的 RBAC とテナント分離、定時バックアップローテーション。詳しくは[セキュリティと等級保護コンプライアンス](#セキュリティと等級保護コンプライアンス等保-20) を参照
- **マイクロサービス + モノリスの自由切替**：go-kratos マイクロサービスフレームワークベースでありながら、モノリス構成での開発・デプロイもサポートし、チーム規模に柔軟に対応
- **フルスタックコード生成**：Protobuf → Go API / TypeScript クライアント、Ent Schema → ORM、ワンクリック CRUD スキャフォールド。デスクトップ GUI ジェネレーターと CLI（[go-wind-toolkit](https://github.com/tx7do/go-wind-toolkit/tree/main/gowind-uiapp)、[関連ツール](#関連ツール) 参照）を同梱
- **本番即戦力**：JWT 認証、SSE プッシュ、非同期タスクスケジューリング、Swagger ドキュメント、Docker ワンクリックデプロイ

### なぜ 3 つのフロントエンドなのか

**チームごとに使うスタックが異なるからであり、「React と Vue の両方を同時に必要とする 1 チーム」に応えるためではありません。**

バックエンドは 1 つ、API 契約は 1 つ、フロントエンド実装が 3 つ。React チームは `react`、Vue チームは `vue-vben` か `vue-element` を取ります。このスキャフォールドを使うためにスタックを乗り換える必要はありません。

3 つすべてを使える状態に保つコストは **上流（本リポジトリ側）**が負担します。採用者は選んだ 1 つだけをメンテナンスすればよく、他の 2 ディレクトリは削除できます（調整箇所は [docs/adopt-one-frontend.md](./docs/adopt-one-frontend.md)・中国語参照）。

---

## デモ

3 つの URL は同じバックエンド機能の並列デモです。**それぞれ開いて比較し、1 つだけ選べばよい**：

| フロントエンド版 | デモ |
|------------------|------|
| Vue3 Vben | <https://vben.admin.gowind.cloud> |
| Vue3 Element Plus | <https://ele.admin.gowind.cloud> |
| React | <https://react.admin.gowind.cloud> |

- バックエンド Swagger：<https://api.demo.admin.gowind.cloud/docs/>
- デフォルトアカウント：`admin` / `Abcd@1234`

---

## 技術スタック

<table>
<tr><th>レイヤー</th><th>技術</th></tr>
<tr><td><strong>バックエンドフレームワーク</strong></td><td><code>Golang</code> · <code>go-kratos v2</code> · <code>Protobuf / Buf</code></td></tr>
<tr><td><strong>ORM</strong></td><td><code>Ent</code>（主力） · <code>GORM</code>（補助） · <code>MySQL</code> · <code>PostgreSQL</code></td></tr>
<tr><td><strong>ミドルウェア</strong></td><td><code>Redis</code>（compose は <code>bitnami/redis:latest</code> を取得、バージョン固定なし。コード側は Set/Expire/Publish などの長期コマンドのみで、Redis 8 専用コマンドは不使用） · <code>MinIO</code>（S3 互換オブジェクトストレージ）</td></tr>
<tr><td><strong>認証・認可</strong></td><td><code>JWT</code> · <code>Casbin</code> · <code>OPA</code></td></tr>
<tr><td><strong>リアルタイム通信</strong></td><td><code>SSE</code>（サーバープッシュ） · <code>Asynq</code>（非同期タスク）</td></tr>
<tr><td><strong>スクリプトエンジン</strong></td><td><code>go-scripts</code> · <code>Lua</code>（gopher-lua） · <code>JavaScript</code>（goja） · 多言語 Hook プラグインシステム</td></tr>
<tr><td><strong>フロントエンド</strong></td><td><strong>3 択 1</strong> — 下の 3 行は並列の選択肢で、3 つ同時に採用するものではありません</td></tr>
<tr><td><strong>Vue Vben 版</strong></td><td><code>Vue 3</code> · <code>TypeScript</code> · <code>Vite</code> · <code>Ant Design Vue</code> · <code>Vben Admin</code></td></tr>
<tr><td><strong>Vue Element 版</strong></td><td><code>Vue 3</code> · <code>TypeScript</code> · <code>Vite</code> · <code>Element Plus</code>（軽量ピュア版）</td></tr>
<tr><td><strong>React 版</strong></td><td><code>React 19</code> · <code>TypeScript</code> · <code>Vite</code> · <code>Zustand</code> · <code>Ant Design V6</code>（UMI 不使用）</td></tr>
<tr><td><strong>デプロイ・運用</strong></td><td><code>Docker</code> · <code>Docker Compose</code> · <code>PM2</code> · <code>Swagger UI</code></td></tr>
</table>

---

## セキュリティと等級保護コンプライアンス（等保 2.0）

本プロジェクトのセキュリティ能力は、中国《ネットワークセキュリティ等級保護 2.0》（等保 2.0、レベル 2/3）の技術要求を参照して設計されており、企業の高プライバシー・プライベートデプロイシナリオにそのまま利用できます：

| 技術要求 | 実装内容 |
|---------|---------|
| **セキュリティ監査** | 6 種類の監査ログを完全網羅：ログイン / 操作 / API / データアクセス / 権限変更 / ポリシー評価。クライアント IP を記録し、ログイン / 操作 / API の 3 種は所属地も解決する。フロントエンド発行の `X-Request-ID` リクエスト ID も併記。asynq による毎日定時アーカイブ：DB 内保持は 180 日（`AUDIT_RETENTION_DAYS` で調整可能）、期限超過データは JSONL アーカイブファイルへエクスポートして痕跡を保持 |
| **本人認証** | パスワード複雑度（8 文字以上、小文字 / 大文字 / 数字 / 記号の 4 種類から 3 種類以上）、履歴パスワード再利用チェック（デフォルト直近 3 件）、パスワード有効期間（デフォルト 90 日）— しきい値は「パラメータ管理」のプラットフォームパラメータで調整（内蔵パラメータは起動時にシードされ、環境変数による設定は廃止）。TOTP 多要素認証（MFA）、画像認証コード、Redis ログイン失敗レート制限（IP + ユーザー名の 2 次元）、設定可能なログイン制限ポリシー |
| **アクセス制御** | 動的 RBAC 権限エンジン（ポリシーエンジン切替可能：Casbin / OPA）。ロール—権限—インターフェースのマッピングは DB に保存され、権限変更は即時ホットリロードで反映。メニュー / ボタンレベルの権限制御に加え、ロール単位の行レベルデータスコープ（V1 試行：役職テーブル）とフィールドレベル権限（V1 試行：ユーザーテーブル、ブラックリストフィールドはレスポンスから刈り取り）。認証判定のたびにポリシー評価ログへ記録しトレース可能 |
| **マルチテナント分離** | ent Privacy ポリシーによるコンパイルレベルのデータ分離：読み取りクエリには自動的にテナントフィルターが注入され、Create はテナント偽装を防止、Update / Delete にはテナント述語が注入される（テナント横断の変更は 0 行ヒット）。テナントリクエストは `(path, method)` により Api テーブルでフェイルクローズ検証（権限ポイント欠落は即拒否）。プランのモジュールホワイトリストと期限切れ読み取り専用ポリシー |
| **データ機密性** | ログインパスワードはアプリケーション層で AES 暗号化送信、bcrypt ハッシュで保存。機密タスク設定は AES-256-GCM で保存時暗号化（Ent Hook による透過的加復号）。JWT RS256 非対称署名、refresh token は HttpOnly Cookie。トランスポート層 TLS はデプロイ層で有効化（バックエンド `server.rest.tls` 設定、または nginx / ロードバランサー終端） |
| **データバックアップ・リカバリ** | [`scripts/backup/pg_backup.sh`](./backend/scripts/backup/pg_backup.sh) による定時フルバックアップ（pg_dump、デフォルト 30 部自動ローテーション）。Docker コンテナ / ローカル直結の双モード対応、リカバリ手順ドキュメント付き |
| **フロントエンドセキュリティ** | 3 つのフロントエンドはそれぞれ `scripts/deploy/nginx.conf` を同梱し、本番では X-Frame-Options / HSTS / Content-Security-Policy レスポンスヘッダーを下流する。react と vue-element はビルド時に `index.html` へ CSP `<meta>` も注入する（インラインスクリプトは sha256 ホワイトリスト）。web server を差し替えても一層目の防御が残る |

> **注記**：等保評価には技術要求のほか、管理制度、物理環境、人員組織などソフトウェア以外の領域が含まれます。本プロジェクトがカバーするのは技術措置の部分であり、プライベートデプロイにおける等保評価準備を直接支援しますが、完全な等保評価プロセスの代替ではありません。

---

## クイックスタート

### 環境要件

| ツール | バージョン |
|--------|-----------|
| Go | 1.26+（`backend/go.mod` に従う） |
| Node.js | `^20.19.0 \|\| >=22.12.0` — 3 系の `engines` 共通範囲は vue-element が決める（vue-vben は `>=20.10.0`、react は未宣言）。**21.x は対象外**、20.18 以下も同様 |
| pnpm | `>= 9.12.0`（下限は vue-vben の `engines.pnpm`）。vue-vben はさらに `packageManager` で `pnpm@11.18.0` を固定：corepack を有効化（`corepack enable`）すれば自動で切り替わる。有効化しない場合は `npm i -g pnpm@11.18.0` で手動インストール、少なくとも major を合わせる |
| Docker | 20.0+ |

### 環境スクリプト選択

- Linux / macOS 開発環境：`scripts/env/install_unix_dev.sh`
- Linux / macOS 本番環境：`scripts/env/install_unix_prod.sh`
- Windows 開発環境：`scripts/env/install_windows_dev.ps1`

### Docker 2つのデプロイモード

- **full_deploy 完全モード**：ミドルウェア+バックエンドアプリを同時起動、ワンクリックデモ・本番デプロイに適用。
- **libs_only 依存モード（開発推奨）**：ミドルウェアのみ起動、アプリはローカル IDE で実行・デバッグ。

### バックエンド起動

> バックエンドのコマンドは `gow` CLI 経由で統一（インストール：`go install github.com/tx7do/go-wind-toolkit/gowind/cmd/gow@latest`、詳しくは[関連ツール](#関連ツール)）。

**Linux / macOS：**

```shell
# 以下のコマンドはすべて backend/ ディレクトリで実行（scripts/ は backend/ 配下）
cd backend

# スクリプトに実行権限を付与
# scripts/ は三階層。glob では env/lib と deploy/sse が抜けるため find を使用
find ./scripts -name '*.sh' -exec chmod +x {} +

# 開発環境（推奨）
./scripts/env/install_unix_dev.sh
./scripts/docker/libs_only.sh
gow run admin

# 本番環境
./scripts/env/install_unix_prod.sh
./scripts/docker/full_deploy.sh

# PM2 プロセス管理（本番上級）
./scripts/deploy/pm2_service.sh
```

**Windows（PowerShell 管理者）：**

```powershell
# 以下のコマンドはすべて backend/ ディレクトリで実行（scripts/ は backend/ 配下）
cd backend

# スクリプト実行ポリシーの許可（初回のみ1回実行）
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser

# 環境初期化
.\scripts\env\install_windows_dev.ps1

# ローカル開発
.\scripts\docker\libs_only.ps1
gow run admin

# ワンクリック完全デプロイ
.\scripts\docker\full_deploy.ps1
```

### フロントエンド起動

フロントエンドは `frontend/admin` ディレクトリに統一配置されています。**3 つのうち 1 つを選んで**依存関係をインストールし、他の 2 つは無視して構いません：

| フロントエンド版 | ディレクトリ | 起動コマンド | ポート |
|------------------|--------------|--------------|--------|
| React | `frontend/admin/react` | `pnpm dev` | 5888 |
| Vue Element | `frontend/admin/vue-element` | `pnpm dev` | 5777 |
| Vue Vben | `frontend/admin/vue-vben` | `pnpm dev:antd` | 5666 |

```shell
# 3 つのうち 1 つを選ぶ：まずそのフロントエンドへ cd してから install → 起動。
# リポジトリ直下にも frontend/admin/ にも package.json は無いので、
# そこで `pnpm install` を実行すると ENOENT エラーになる。
cd frontend/admin/react
pnpm install
pnpm dev                    # ポート 5888

# 残り 2 系：
cd frontend/admin/vue-element && pnpm install && pnpm dev            # ポート 5777
cd frontend/admin/vue-vben   && pnpm install && pnpm dev:antd        # ポート 5666
```

> vue-vben 自体が pnpm workspace（`pnpm-workspace.yaml` + `apps/` + `packages/`）なので、依存インストールは必ず**そのルート**で行う。`pnpm dev:antd` は workspace から `@vben/web-antd` app を選んで起動するだけ。`apps/admin` で単独 install すると catalog のバージョン固定を迂回する。

---

## 機能一覧

> 各リストページ（業務データおよび監査ログ）は、現在のフィルター条件でページング集約による「エクスポート」をサポートする。形式は CSV / XLSX から選択可能（上限 1 万行）。

### 組織と権限

| 機能 | 説明 |
|------|------|
| ユーザー管理 | ユーザーの管理とクエリを行い、高度なクエリや部署に連動したユーザー検索をサポート。ユーザーの無効化 / 有効化、上司の設定 / 解除、パスワードリセット、複数ロール・複数部署・上位上司の設定、指定ユーザーでのワンクリックログインなどの機能を提供。 |
| テナント管理 | テナントの管理を行い、新規テナント追加後に自動的にテナント部署、デフォルトロール、管理者を初期化。プランの設定、無効化 / 有効化、テナント管理者でのワンクリックログイン機能をサポート。 |
| プラン・クォータ管理 | テナントのサブスクリプションプランおよびリソースクォータ（モジュールホワイトリスト、使用量上限など）を管理。プランおよびクォータ項目の CRUD をサポート。 |
| ロール管理 | ロールとロールグループの管理を行い、ロールに連動したユーザー検索、メニュー付与・データ権限スコープ（5 段階 / カスタム組織ユニット集合）・フィールドレベル権限（ブラックリストフィールド集合）の設定、従業員の一括追加・削除をサポート。 |
| 権限管理 | 権限グループ、メニュー、権限ポイントの管理を行い、ツリーリストでの表示をサポート。 |
| 組織管理 | 組織の管理を行い、ツリーリストでの表示をサポート。 |
| 役職管理 | ユーザーの役職管理を行い、役職はユーザーのタグとして使用可能。Excel インポートをサポート（クライアント側でのテンプレートダウンロード、既存の作成 API による 1 行ずつの処理と行単位のエラー報告）。**「所属組織」列を解決するのは vue-element のインポータのみ**（組織名の完全一致照合で組織ユニットを補完、不一致はその行のエラーとして報告）。react と vue-vben のインポート項目リストは `orgUnitId` を意図的に除外しており、コード中のコメントは「外部キーの名称解決は後続演进」としている。 |
| メニュー管理 | システムメニュー、操作権限、ボタン権限識別子などの設定を行い、ディレクトリ、メニュー、ボタンを含む。メニュー同期（3 フロントエンド対応）はトランザクション化された全件再構築とインクリメンタルマージの 2 モードをサポートし、マージはフルパス照合で既存メニューをその場更新して既存のメニュー ID とロール付与を保持する。 |

### システム機能

| 機能 | 説明 |
|------|------|
| インターフェース管理 | インターフェースの管理を行い、インターフェース同期機能をサポート。主に新規権限ポイント追加時のインターフェース選択に使用し、ツリーリスト表示、操作ログのリクエストパラメーターとレスポンス結果の設定をサポート。 |
| ディクショナリ管理 | データディクショナリの大分類と小分類の管理を行い、ディクショナリ大分類に連動した小分類検索、サーバー側の多列ソート、データのインポート・エクスポートをサポート。 |
| タスクスケジューリング | タスクとタスク実行ログの管理・参照を行い、タスクの新規追加、修正、削除、起動、一時停止、即時実行をサポート。 |
| ファイル管理 | ファイルアップロードの管理を行い、ファイルクエリ、OSS またはローカルへのアップロード、ダウンロード、ファイルアドレスのコピー、ファイル削除、画像の拡大表示をサポート。 |
| ログインポリシー | ログイン制限ポリシーを管理し、対象ユーザーの制限タイプ、制限方式、制限値、制限理由を設定。 |
| アカウントログイン | ユーザー名 / メールアドレス / 電話番号をアカウント識別子としてログイン可能。画像認証コード、ログインポリシー、TOTP 多要素認証と組み合わせ可能。 |
| 多要素認証（MFA） | TOTP ベースの多要素認証。ログインチャレンジ、個人センターでのバインド管理、および管理者によるユーザー MFA のレスキューリセットを含む。 |
| パスワード再設定 | バインド済みメールアドレス宛の認証コードでパスワードを再設定：コードは 10 分間・1 回のみ有効、再設定成功時に全セッションを失効。存在しないユーザーは静かに処理し、ユーザー列挙を防止。 |
| 通知チャネル | 通知チャネルの管理、タイプは 2 択：`EMAIL`（SMTP 経由、パスワードは暗号化保存・リストではマスク表示）または `WEBHOOK`（HTTP コールバック、署名スタイル 5 档：NONE / DINGTALK / FEISHU / WECOM / CUSTOM）。有効化 / 無効化とテスト送信をサポート。 |
| サーバーモニタリング | サービスのランタイム指標（CPU コア数、メモリ、goroutine 数、稼働時間など）を読み取り専用で表示し、自動更新。 |
| スクリプトシステム | スクリプトプラグインシステム（Lua / JavaScript、データベースを信頼源、管理画面の変更は即時反映）：エンティティライフサイクルフック（before は否決可 / after は非同期）、定時タスク（asynq）、HTTP 送信（ドメイン許可リスト fail-closed）、テスト実行と実行ログ。詳細は [docs/script_system.md](./docs/script_system.md) |
| パラメータ管理 | プラットフォーム全体のシステムパラメータをキー / 値で管理（業務ディクショナリとは区別）。内蔵パラメータは起動時にシードされ、削除は不可。読み取りはサービス側のキャッシュ付き accessor を経由し、マルチインスタンス構成では変更が Redis パブリッシュ / サブスクライブでブロードキャストされて各インスタンスのキャッシュが失効される。 |
| マシン資格情報（AK/SK） | テナントスコープの AccessKey / SecretKey 管理：作成時に Secret を一度だけ表示し、有効化 / 無効化・削除・シークレットのローテーションをサポート（ローテーションで旧 Secret は直ちに無効化）。AK / Secret はトークン交換エンドポイントでテナントスコープのマシン JWT（machine ロール、アクセストークンのみ）に交換可能。交換エンドポイントには IP + AK 単位の失敗レート制限が適用される。 |
| 言語管理 | システムがサポートする多言語を管理し、言語名、言語コード、ネイティブ名、有効化およびデフォルト状態を設定。 |

### メッセージとログ

| 機能 | 説明 |
|------|------|
| メッセージ分類 | メッセージ管理で選択するメッセージ分類の管理。分類は**平坦な 1 層のみ**（`sys_internal_message_categories` に parent_id 列は無く、削除も当該行のみでツリーカスケードは行わない）。 |
| メッセージ管理 | メッセージの管理を行い、送信範囲（全員 / 指定ユーザー）による送信とメッセージの取り消しをサポート。全員ブロードキャストは非同期タスクキューで配信（再開可能・冪等）。ユーザーの既読状況と既読時間の参照が可能。 |
| 内部メッセージ | 内部メッセージの管理を行い、メッセージの詳細参照、削除、既読マーク、一括既読をサポート。 |
| ログインログ | ログインログリストのクエリを行い、ユーザーのログイン成功・失敗ログを記録し、IP アドレスの所属地記録をサポート。 |
| 操作ログ | 操作ログリストのクエリを行い、ユーザーの操作正常・異常ログを記録。IP アドレスの所属地記録とリソースオブジェクトの特定、操作ログの詳細参照をサポート。 |
| APIログ | API 監査ログリストのクエリを行い、API リクエストの操作者、パス、メソッド、成功状態を記録し、IP アドレスの所属地記録をサポート。 |
| データログ | データアクセス監査ログリストのクエリを行い、SQL は字句マスキング、対象テーブル名とデータ分類を自動抽出。 |
| 権限ログ | 権限変更監査ログリストのクエリを行い、操作者、対象オブジェクト、理由を記録し、リクエストスナップショットを保持。 |
| ポリシー評価ログ | ポリシー評価監査ログリストのクエリを行い、認可判定ごとの結果と評価コンテキストを記録。trace_id 関連のトラブルシューティングをサポート。 |
| Redisキャッシュモニター | 読み取り専用の Redis キャッシュモニタリングで、INFO、DBSIZE、スローログデータを表示し、書き込み操作は実行しない。 |

### 個人センター

| 機能 | 説明 |
|------|------|
| マイページ | 個人情報の表示・修正、最終ログイン情報の参照、パスワードの変更、メールアドレスのバインド / 再バインド（認証コード検証）などの機能を提供。 |

---

## プロジェクト構成

```
go-wind-admin/
├── backend/                        # バックエンドプロジェクト
│   ├── api/                        # Protobuf API 定義と生成コード
│   │   ├── protos/                 # .proto ソースファイル（ドメイン別）
│   │   └── gen/go/                 # buf が生成する Go コード
│   ├── app/admin/service/          # Admin サービスアプリケーション
│   │   ├── cmd/server/             # エントリポイント（main.go、wiring_ent.go 依存性組み立て）
│   │   ├── configs/                # 設定ファイル（YAML）
│   │   └── internal/               # ビジネスコア（data/service/server）
│   ├── pkg/                        # 共通パッケージ
│   │   ├── scripting/              # 多言語スクリプトエンジン（Lua + JavaScript）
│   │   ├── oss/                    # オブジェクトストレージ（MinIO）
│   │   ├── eventbus/               # イベントバス
│   │   └── ...                     # その他ユーティリティパッケージ
│   ├── scripts/                    # デプロイ・バックアップスクリプト（env/docker/deploy/backup）
│   └── sql/                        # デモデータ SQL（デフォルトデータはサービス起動時に自動シード）
├── frontend/admin/                 # フロントエンドプロジェクト（3 択 1、選んだ 1 つだけを維持すればよい）
│   ├── react/                      # React 19 + Ant Design V6
│   ├── vue-element/                # Vue 3 + Element Plus
│   └── vue-vben/                   # Vue 3 + Ant Design Vue + Vben Admin
└── docs/                           # プロジェクトドキュメント
```

---

## スクリーンショット

<table>
<tr>
<td><img src="./docs/images/admin_login_page.png" alt="バックエンドユーザーログイン画面"/></td>
<td><img src="./docs/images/admin_dashboard.png" alt="バックエンド分析画面"/></td>
</tr>
<tr>
<td><img src="./docs/images/admin_user_list.png" alt="バックエンドユーザーリスト画面"/></td>
<td><img src="./docs/images/admin_user_create.png" alt="バックエンドユーザー作成画面"/></td>
</tr>
<tr>
<td><img src="./docs/images/admin_tenant_list.png" alt="バックエンドテナントリスト画面"/></td>
<td><img src="./docs/images/admin_tenant_create.png" alt="バックエンドテナント作成画面"/></td>
</tr>
<tr>
<td><img src="./docs/images/admin_org_unit_list.png" alt="組織単位リスト画面"/></td>
<td><img src="./docs/images/admin_org_unit_create.png" alt="組織単位作成画面"/></td>
</tr>
<tr>
<td><img src="./docs/images/admin_position_list.png" alt="バックエンド役職リスト画面"/></td>
<td><img src="./docs/images/admin_position_create.png" alt="バックエンド役職作成画面"/></td>
</tr>
<tr>
<td><img src="./docs/images/admin_role_list.png" alt="バックエンドロールリスト画面"/></td>
<td><img src="./docs/images/admin_role_create.png" alt="バックエンドロール作成画面"/></td>
</tr>
<tr>
<td><img src="./docs/images/admin_permission_list.png" alt="バックエンド権限リスト画面"/></td>
<td><img src="./docs/images/admin_permission_create.png" alt="バックエンド権限作成画面"/></td>
</tr>
<tr>
<td><img src="./docs/images/admin_menu_list.png" alt="バックエンドディレクトリリスト画面"/></td>
<td><img src="./docs/images/admin_menu_create.png" alt="バックエンドディレクトリ作成画面"/></td>
</tr>
<tr>
<td><img src="./docs/images/admin_task_list.png" alt="バックエンドスケジューリングタスクリスト画面"/></td>
<td><img src="./docs/images/admin_task_create.png" alt="バックエンドスケジューリングタスク作成画面"/></td>
</tr>
<tr>
<td><img src="./docs/images/admin_dict_list.png" alt="バックエンドデータディクショナリリスト画面"/></td>
<td><img src="./docs/images/admin_dict_entry_create.png" alt="バックエンドデータディクショナリエントリ作成画面"/></td>
</tr>
<tr>
<td><img src="./docs/images/admin_internal_message_list.png" alt="バックエンド内部メッセージリスト画面"/></td>
<td><img src="./docs/images/admin_internal_message_publish.png" alt="バックエンド内部メッセージ発行画面"/></td>
</tr>
<tr>
<td><img src="./docs/images/admin_login_policy_list.png" alt="ログインポリシーリスト画面"/></td>
<td><img src="./docs/images/admin_login_policy_create.png" alt="ログインポリシー作成画面"/></td>
</tr>
<tr>
<td><img src="./docs/images/admin_login_audit_log_list.png" alt="バックエンドログインログ画面"/></td>
<td><img src="./docs/images/admin_api_audit_log_list.png" alt="バックエンド操作ログ画面"/></td>
</tr>
<tr>
<td><img src="./docs/images/admin_api_list.png" alt="APIリスト画面"/></td>
<td><img src="./docs/images/api_swagger_ui.png" alt="バックエンド内蔵Swagger UI画面"/></td>
</tr>
</table>

## 関連ツール

- **[go-wind-toolkit / gowind-uiapp](https://github.com/tx7do/go-wind-toolkit/tree/main/gowind-uiapp)** — クロスプラットフォームのデスクトップ型コードジェネレーター（Go + Wails）。SQL のインポートまたはデータベーステーブル（MySQL / PostgreSQL / SQLite / SQL Server / Oracle）への接続から、gRPC / RESTful などのテンプレートでサーバーサイド・フロントエンドのコード（簡易フォームを含む）を自動生成。非対話・JSON 出力の CLI（`gowind-cli`）も同梱しており、スクリプトや AI エージェントからの呼び出しに便利です。
- **[gow — GoWind CLI](https://github.com/tx7do/go-wind-toolkit/tree/main/gowind)** — 本プロジェクトの推奨コマンドライン入口：`gow run admin` でサービス起動、`gow ent` / `gow api` でコード生成、`gow generate` でデータベース DSN から CRUD マイクロサービスを生成、`gow extract` でマイクロサービスのモジュール分割を行います。`backend/` 配下で実行すると `app/*/service` を自動検出します。日常開発では Makefile より優先的に使用してください。

## コミュニティとコントリビューション

GoWind Admin コミュニティへの参加を歓迎します。以下のドキュメントで、コードの貢献方法、Issue の報告、セキュリティ脆弱性の開示について説明しています：

- [コントリビューションガイド](./CONTRIBUTING.md) — 開発環境、コード生成規約、コミット規約と PR フロー
- [行動規範](./.github/CODE_OF_CONDUCT.md) — コミュニティでの交流に関する方針
- [セキュリティポリシー](./SECURITY.md) — 脆弱性報告フローと適用範囲
- [更新履歴](./CHANGELOG.md) — バージョン変更記録
- Issue テンプレート：[バグ報告](./.github/ISSUE_TEMPLATE/bug_report.md) · [機能リクエスト](./.github/ISSUE_TEMPLATE/feature_request.md)
- [PR テンプレート](./.github/PULL_REQUEST_TEMPLATE.md)

## お問い合わせ

- WeChat 個人アカウント：`yang_lin_bo`（備考：`go-wind-admin`）
- 掘金コラム：[go-wind-admin](https://juejin.cn/column/7541283508041826367)

## 謝辞

[![JetBrains](https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.svg)](https://jb.gg/OpenSource)

JetBrains により無料の GoLand & WebStorm オープンソースライセンスを提供いただいています。心より感謝申し上げます。

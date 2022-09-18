# gmail-url-dl

Gmail APIを用いて、指定された検索条件に合致するメールに記載されたURLの先のファイルをダウンロードします。

# 認証情報ファイルの作成

- 実行環境はGmail APIの[Go quickstart](https://developers.google.com/gmail/api/quickstart/go)と同じです。
- [こちらの記事](https://developers.google.com/workspace/guides/create-credentials)のOAuth クライアント ID
  の認証情報のデスクトップアプリから認証情報を作成し、credentials.jsonという名前でプロジェクト直下に保存してください。
- 事前にGmail APIを有効化する必要があります。

# 設定ファイルの作成

settings/sample_settings.goを参考にsettings/settings.goを作成してください。



# mich-chosei

Discord鯖における日程調整botです。

## 開発環境構築

- Dockerコンテナのビルド

```
docker build -t mich-chosei .
```

- Dockerコンテナの実行

```
docker run --rm --env-file .env -p 8080:8080 mich-chosei
```

- 開発用cron実行間隔

```
*/2 * * * *
```

- 本番用cron実行間隔

```
0 9 * * 1
```

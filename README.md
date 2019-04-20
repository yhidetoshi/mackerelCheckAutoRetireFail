# mackerelCheckAutoRetireFail

### 目的
- mackerelのauto-retirementに失敗し、ホストリストに存在するがメトリクスを取得できていないインスタンスを検出して通知させる

### 実装
- mackerel-client-goを利用
  - https://github.com/mackerelio/mackerel-client-go
  
- 登録されて5分以上経過して、メモリのメトリクスが取得できていないインスタンスが存在する場合にOrg名と登録名をSlackに通知する
  - Statusが `Working` でメトリクス取得に失敗している場合を検知するために、今回はメモリのメトリクスが取得できない場合という条件にした。


- 通知結果例

![Alt Text](https://github.com/yhidetoshi/Pictures/raw/master/mackerel/mackerel-slack-notice.png)


### 実行方法

- 環境変数として、セットする。
```
username = os.Getenv("USERNAME")
slackURL = os.Getenv("SLACKURL")
mkrKey   = os.Getenv("MKRKEY")
```

AWS LambdaにGoランタイムで実行する。定期実行はCloudWatch Eventと連携してcron実行させる。

goctl-gin api go -api ./doc/api.api -dir ./
goctl-gin model mysql ddl --src ./deploy/sql/user.sql --dir ./internal/model

go test -v -bench="函数名正则" \
  -benchmem \
  -benchtime="测试秒数"s \
  -count="测试次数" \
  -cpuprofile=cpu.pprof \
  -memprofile=mem.pprof \
  "文件或包路径"

go tool pprof -http=:8080 mem.pprof
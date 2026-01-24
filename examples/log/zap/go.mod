module github.com/zzguang83325/eorm/examples/log/zap

go 1.24.0

replace github.com/zzguang83325/eorm => ../../../

require (
	github.com/mattn/go-sqlite3 v1.14.33
	github.com/zzguang83325/eorm v0.0.0
	go.uber.org/zap v1.26.0
)

require (
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

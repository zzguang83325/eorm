module batch_exec_example

go 1.24.0

require (
	github.com/go-sql-driver/mysql v1.7.1
	github.com/zzguang83325/eorm v0.0.0
)

require golang.org/x/text v0.14.0 // indirect

replace github.com/zzguang83325/eorm => ../../

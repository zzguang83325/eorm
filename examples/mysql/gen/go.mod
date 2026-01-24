module github.com/zzguang83325/eorm/examples/mysql

go 1.24.0

replace github.com/zzguang83325/eorm => ../../../

replace github.com/zzguang83325/eorm/drivers/mysql => ../../../drivers/mysql

require (
	github.com/go-sql-driver/mysql v1.7.1
	github.com/zzguang83325/eorm v0.0.0-00010101000000-000000000000
)

require golang.org/x/text v0.14.0 // indirect

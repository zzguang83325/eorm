module github.com/zzguang83325/eorm/examples/sql_template

go 1.24.0

replace github.com/zzguang83325/eorm => ../../

replace github.com/zzguang83325/eorm/drivers/mysql => ../../drivers/mysql

require (
	github.com/zzguang83325/eorm v0.0.0-00010101000000-000000000000
	github.com/zzguang83325/eorm/drivers/mysql v0.0.0-00010101000000-000000000000
)

require (
	github.com/go-sql-driver/mysql v1.7.1 // indirect
	golang.org/x/text v0.14.0 // indirect
)

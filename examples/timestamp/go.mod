module timestamp_example

go 1.24.0

replace github.com/zzguang83325/eorm => ../..

replace github.com/zzguang83325/eorm/drivers/sqlite => ../../drivers/sqlite

require (
	github.com/zzguang83325/eorm v0.0.0-00010101000000-000000000000
	github.com/zzguang83325/eorm/drivers/sqlite v0.0.0-00010101000000-000000000000
)

require (
	github.com/mattn/go-sqlite3 v1.14.18 // indirect
	golang.org/x/text v0.14.0 // indirect
)

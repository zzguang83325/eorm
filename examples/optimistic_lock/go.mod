module optimistic_lock_example

go 1.24.0

require (
	github.com/mattn/go-sqlite3 v1.14.33
	github.com/zzguang83325/eorm v0.0.0
)

require golang.org/x/text v0.14.0 // indirect

replace github.com/zzguang83325/eorm => ../..

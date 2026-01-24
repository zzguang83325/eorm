module gen

go 1.24.0

require (
	github.com/go-sql-driver/mysql v1.8.1
	github.com/zzguang83325/eorm v0.0.0
)

replace github.com/zzguang83325/eorm => ../../..

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

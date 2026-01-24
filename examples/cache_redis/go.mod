module cache_redis

go 1.25.5

replace github.com/zzguang83325/eorm => ../../

replace github.com/zzguang83325/eorm/drivers/mysql => ../../drivers/mysql

replace github.com/zzguang83325/eorm/redis => ../../redis

require (
	github.com/zzguang83325/eorm v0.0.0
	github.com/zzguang83325/eorm/drivers/mysql v0.0.0
	github.com/zzguang83325/eorm/redis v0.0.0-20260108114422-749c9f8196ef
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-redis/redis/v8 v8.11.5 // indirect
	github.com/go-sql-driver/mysql v1.9.3 // indirect
	golang.org/x/text v0.14.0 // indirect
)

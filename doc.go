/*
Package eorm provides a high-performance, easy-to-use ORM framework for Go.

EORM (Easy ORM) is designed to be simple and intuitive, allowing database operations without defining structs for every table.
It supports multiple databases including MySQL, PostgreSQL, SQLite, SQL Server, and Oracle.

Key Features:
  - Record-based CRUD: Operate on data using the dynamic Record object, inspired by JFinal.
  - Multi-Database Support: Manage multiple database connections seamlessly.
  - SQL Security: Built-in validator to prevent SQL injection attacks.
  - Smart Caching: Include result caching (Memory/Redis) and statement caching (LRU) for high performance.
  - Connection Monitoring: Auto-reconnect and health checks for database connections.
  - Soft Delete & Concurrency Control: Native support for soft deletes and optimistic locking.

Basic Usage:

	// Initialize
	db, err := eorm.OpenDatabase(eorm.MySQL, "user:pass@tcp(localhost:3306)/dbname", 10)
	if err != nil {
		log.Fatal(err)
	}

	// Query
	users, err := eorm.Query("SELECT * FROM users WHERE age > ?", 18)

	// Insert using Record
	user := eorm.NewRecord().Set("name", "John").Set("age", 25)
	id, err := eorm.InsertRecord("users", user)

For more detailed guides, see the README.md or the doc/ directory.
*/
package eorm

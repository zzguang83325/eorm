package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/zzguang83325/eorm"
	_ "github.com/zzguang83325/eorm/drivers/sqlserver"
	"github.com/zzguang83325/eorm/examples/sqlserver/models"
)

func main() {
	dsn := "sqlserver://sa:123456@192.168.10.44:1433?database=test"
	_, err := eorm.OpenDatabaseWithDBName("sqlserver", eorm.SQLServer, dsn, 25)
	if err != nil {
		log.Fatalf("SQL Server数据库连接失败: %v", err)
	}
	eorm.SetDebugMode(true)

	setupTable()
	prepareData()
	demoRecordOperations()
	demoDbModelOperations()
	demoChainOperations()
	demoCacheOperations()
	demoUpdateDeleteOperations()
}

func setupTable() {
	sql := `
	IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'demo')
	CREATE TABLE demo (
		id INT IDENTITY(1,1) PRIMARY KEY,
		name NVARCHAR(100),
		age INT,
		salary DECIMAL(10, 2),
		is_active BIT,
		birthday DATE,
		created_at DATETIME DEFAULT GETDATE(),
		metadata NVARCHAR(MAX)
	)`
	eorm.Use("sqlserver").Exec(sql)
}

func prepareData() {
	count, _ := eorm.Use("sqlserver").Count("demo", "")
	if count >= 100 {
		fmt.Printf("SQL Server: Already has %d rows, skipping data preparation.\n", count)
		return
	}
	fmt.Println("SQL Server: Inserting 110 rows of data...")
	records := make([]*eorm.Record, 0, 110)
	for i := 1; i <= 110; i++ {
		record := eorm.NewRecord().
			Set("name", fmt.Sprintf("MSSQL_User_%d", i)).
			Set("age", 18+rand.Intn(40)).
			Set("salary", 3000.0+float64(i)).
			Set("is_active", true).
			Set("birthday", time.Now()).
			Set("metadata", "MSSQL Meta")
		records = append(records, record)
	}
	eorm.Use("sqlserver").BatchInsertRecord("demo", records, 100)
	fmt.Println("SQL Server: Data preparation complete.")
}

func demoRecordOperations() {
	fmt.Println("\n--- MSSQL Record Operations ---")
	records, err := eorm.Use("sqlserver").Query("SELECT TOP 5 * FROM demo WHERE age > ?", 30)
	if err != nil {
		log.Printf("MSSQL Query failed: %v", err)
		return
	}
	fmt.Printf("Query returned %d records\n", len(records))
}

func demoDbModelOperations() {
	fmt.Println("\n--- SQL Server DbModel CRUD Operations ---")
	model := &models.Demo{}

	// 1. Insert
	newUser := &models.Demo{
		Name:     ptrString("ModelUser"),
		Age:      ptrInt64(28),
		Salary:   ptrFloat64(7500.00),
		IsActive: ptrBoolean(true),
		Birthday: ptrDateTime(time.Now().AddDate(-28, 0, 0)),
		Metadata: ptrString(`{"role": "admin"}`),
	}
	id, err := newUser.Insert()
	if err != nil {
		log.Printf("SQL Server DbModel Insert failed: %v", err)
		return
	}
	fmt.Printf("SQL Server DbModel Insert: ID = %d\n", id)
	newUser.ID = id

	// 2. FindFirst (Read)
	foundUser, err := model.FindFirst("name = ?", "New_MSSQL_User")
	if err != nil {
		log.Printf("SQL Server DbModel FindFirst failed: %v", err)
	} else if foundUser != nil {
		fmt.Printf("SQL Server DbModel FindFirst: Found user %s, Age: %d\n", foundUser.Name, foundUser.Age)
	}

	// 3. Update
	foundUser.Age = ptrInt64(38)
	foundUser.Salary = ptrFloat64(10500.75)
	affected, err := foundUser.Update()
	if err != nil {
		log.Printf("SQL Server DbModel Update failed: %v", err)
	} else {
		fmt.Printf("SQL Server DbModel Update: %d rows affected\n", affected)
	}

	// 4. Find (Read)
	results, err := model.Find("age >= ?", "id DESC", 30)
	if err != nil {
		log.Printf("SQL Server DbModel Find failed: %v", err)
	} else {
		fmt.Printf("SQL Server DbModel Find: %d results, first user: %s\n", len(results), results[0].Name)
	}

	// 5. Paginate (Read)
	page, err := model.Paginate(1, 10, "select * from demo where age > ? order by id ASC", 20)
	if err != nil {
		log.Printf("SQL Server DbModel Paginate failed: %v", err)
	} else {
		fmt.Printf("SQL Server DbModel Paginate: Total %d rows, current page size %d\n", page.TotalRow, len(page.List))
	}

	// 6. Delete
	affected, err = foundUser.Delete()
	if err != nil {
		log.Printf("SQL Server DbModel Delete failed: %v", err)
	} else {
		fmt.Printf("SQL Server DbModel Delete: %d rows affected\n", affected)
	}
}

func demoChainOperations() {
	fmt.Println("\n--- MSSQL Chain Operations ---")
	page, err := eorm.Use("sqlserver").Table("demo").Where("age > ?", 20).OrderBy("id").Paginate(1, 10)
	if err != nil {
		log.Printf("MSSQL Chain Paginate failed: %v", err)
		return
	}
	fmt.Printf("MSSQL Chain Paginate: Total %d rows, current page size %d\n", page.TotalRow, len(page.List))
}

func demoCacheOperations() {
	fmt.Println("\n--- SQL Server Cache Operations ---")
	var results []models.Demo
	// First call - should hit DB and save to cache
	start := time.Now()
	err := eorm.Use("sqlserver").Cache("mssql_demo_cache", 60).Table("demo").Where("age > ?", 35).FindToDbModel(&results)
	if err != nil {
		log.Printf("SQL Server Cache Find (1st) failed: %v", err)
	} else {
		fmt.Printf("SQL Server Cache Find (1st): %d results, took %v\n", len(results), time.Since(start))
	}

	// Second call - should hit cache
	start = time.Now()
	err = eorm.Use("sqlserver").Cache("mssql_demo_cache", 60).Table("demo").Where("age > ?", 35).FindToDbModel(&results)
	if err != nil {
		log.Printf("SQL Server Cache Find (2nd) failed: %v", err)
	} else {
		fmt.Printf("SQL Server Cache Find (2nd): %d results, took %v (from cache)\n", len(results), time.Since(start))
	}
}

func demoUpdateDeleteOperations() {
	fmt.Println("\n--- MSSQL Update/Delete Operations ---")
	// Update
	affected, err := eorm.Use("sqlserver").Table("demo").Where("name = ?", "MSSQL_User_1").Update(eorm.NewRecord().Set("age", 99))
	if err != nil {
		log.Printf("MSSQL Update failed: %v", err)
	} else {
		fmt.Printf("MSSQL Update: %d rows affected\n", affected)
	}

	// Delete
	affected, err = eorm.Use("sqlserver").Table("demo").Where("name = ?", "MSSQL_User_2").Delete()
	if err != nil {
		log.Printf("MSSQL Delete failed: %v", err)
	} else {
		fmt.Printf("MSSQL Delete: %d rows affected\n", affected)
	}
}

func ptrString(s string) *string         { return &s }
func ptrInt64(i int64) *int64            { return &i }
func ptrFloat64(f float64) *float64      { return &f }
func ptrDateTime(f time.Time) *time.Time { return &f }
func ptrBoolean(f bool) *bool            { return &f }

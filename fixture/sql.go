package fixture

import "strings"

const (
	CreateTable1   = "ceate_table_1"
	CreateTable2   = "ceate_table_2"
	DropTable      = "drop_table"
	AutoMigrate    = "automigrate"
	SaveSQL        = "sqve_sql"
	UpdateSQL      = "update_sql"
	SingularTable  = "singular_table"
	FirstSQL1      = "firsr_sql_1"
	FirstSQL2      = "firsr_sql_2"
	LastSQL1       = "last_sql_1"
	LastSQL2       = "last_sql_2"
	FindSQL1       = "find_sql_1"
	FindSQL2       = "find_sql_2"
	AddIndexSQL    = "add_index_sql"
	DeleteSQL      = "delete_sql"
	AddUniqueIndex = "add_unique_index"
)

var samples map[string]map[string]string

func init() {
	samples = make(map[string]map[string]string)
	samples["ql-mem"] = sampleQL()
	samples["postgres"] = samplePG()
}

func sampleQL() map[string]string {
	o := make(map[string]string)
	s := `
BEGIN TRANSACTION; 
	CREATE TABLE foos (id int,stuff string ) ;
COMMIT;`
	o[CreateTable1] = s
	s = `
BEGIN TRANSACTION; 
	CREATE TABLE users (id int64,age int64,user_num int64,name string,email string,birthday time,created_at time,updated_at time,billing_address_id int64,shipping_address_id int64,latitude float64,company_id int,role string,password_hash blob,sequence uint ) ;
	CREATE TABLE user_languages (user_id int64,language_id int64 ) ;
COMMIT;`
	o[CreateTable2] = s
	s = `
BEGIN TRANSACTION; 
	DROP TABLE foos;
	DROP TABLE users;
COMMIT;`
	o[DropTable] = s
	s = `
BEGIN TRANSACTION;
	CREATE TABLE users (id int64,age int64,user_num int64,name string,email string,birthday time,created_at time,updated_at time,billing_address_id int64,shipping_address_id int64,latitude float64,company_id int,role string,password_hash blob,sequence uint ) ;
	CREATE TABLE user_languages (user_id int64,language_id int64 ) ;
	CREATE TABLE emails (id int16,user_id int,email string,created_at time,updated_at time ) ;
	CREATE TABLE languages (id int64,created_at time,updated_at time,deleted_at time,name string ) ;
	CREATE INDEX idx_languages_deleted_at ON languages(deleted_at);
	CREATE TABLE companies (id int64,name string ) ;
	CREATE TABLE credit_cards (id int8,number string,user_id int64,created_at time NOT NULL,updated_at time,deleted_at time ) ;
	CREATE TABLE addresses (id int,address1 string,address2 string,post string,created_at time,updated_at time,deleted_at time ) ;
COMMIT;`
	o[AutoMigrate] = s
	s = `
BEGIN TRANSACTION;
	UPDATE foos SET stuff = $1  WHERE id = $2;
COMMIT;`
	o[SaveSQL] = s
	s = `
BEGIN TRANSACTION;
	UPDATE foos SET stuff = $1  WHERE id = $2;
COMMIT;`
	o[UpdateSQL] = s
	s = `
BEGIN TRANSACTION;
	INSERT INTO foo (stuff) VALUES ($1);
COMMIT;`
	o[SingularTable] = s
	s = `SELECT * FROM users   ORDER BY id ASC LIMIT 1`
	o[FirstSQL1] = s
	s = `SELECT * FROM users  WHERE (id = $1) ORDER BY id ASC LIMIT 1`
	o[FirstSQL2] = s
	s = `SELECT * FROM users   ORDER BY id DESC LIMIT 1`
	o[LastSQL1] = s
	s = `SELECT * FROM users  WHERE (id = $1) ORDER BY id DESC LIMIT 1`
	o[LastSQL2] = s
	s = `SELECT * FROM users`
	o[FindSQL1] = s
	s = `SELECT * FROM users   LIMIT 2`
	o[FindSQL2] = s
	s = `CREATE INDEX _idx_foo_stuff ON foos(stuff) `
	o[AddIndexSQL] = s
	s = `
BEGIN TRANSACTION;
	DELETE FROM foos  WHERE id = $1 ;
COMMIT;
`
	o[DeleteSQL] = s
	s = `CREATE UNIQUE INDEX idx_foo_stuff ON foos(stuff) `
	o[AddUniqueIndex] = s
	return o
}

func samplePG() map[string]string {
	o := make(map[string]string)
	s := `
	CREATE TABLE "foos" ("id" serial,"stuff" text ) ;
`
	o[CreateTable1] = s
	s = `
BEGIN TRANSACTION; 
 	CREATE TABLE "users" ("id" bigserial,"age" bigint,"user_num" bigint,"name" varchar(255),"email" text,"birthday" times
tamp with time zone,"created_at" timestamp with time zone,"updated_at" timestamp with time zone,"billing_address_id" bigint,"shipping_address
_id" bigint,"latitude" numeric,"company_id" integer,"role" varchar(256),"password_hash" bytea,"sequence" serial ) ;
COMMIT;`
	o[CreateTable2] = s
	s = `
	DROP TABLE "foos";
	DROP TABLE "users";
`
	o[DropTable] = s
	s = `
	CREATE TABLE "users" ("id" bigserial,"age" bigint,"user_num" bigint,"name" varchar(255),"email" text,"birthday" timestamp with time zone,"created_at" timestamp with time zone,"updated_at" timestamp with time zone,"billing_address_id" bigint,"shipping_address_id" bigint,"latitude" numeric,"company_id" integer,"role" varchar(256),"password_hash" bytea,"sequence" serial ) ;
	CREATE TABLE "emails" ("id" serial,"user_id" integer,"email" varchar(100),"created_at" timestamp with time zone,"updated_at" timestamp with time zone ) ;
	CREATE TABLE "languages" ("id" bigserial,"created_at" timestamp with time zone,"updated_at" timestamp with time zone,"deleted_at" timestamp with time zone,"name" text ) ;
	CREATE INDEX idx_languages_deleted_at ON "languages"(deleted_at);
	CREATE TABLE "companies" ("id" bigserial,"name" text ) ;
	CREATE TABLE "credit_cards" ("id" serial,"number" text,"user_id" bigint,"created_at" timestamp with time zone NOT NULL,"updated_at" timestamp with time zone,"deleted_at" timestamp with time zone ) ;
	CREATE TABLE "addresses" ("id" serial,"address1" text,"address2" text,"post" text,"created_at" timestamp with time zone,"updated_at" timestamp with time zone,"deleted_at" timestamp with time zone ) ;
`
	o[AutoMigrate] = s
	s = `
BEGIN TRANSACTION;
	UPDATE foos SET stuff = $1  WHERE id = $2;
COMMIT;`
	o[SaveSQL] = s
	s = `
BEGIN TRANSACTION;
	UPDATE foos SET stuff = $1  WHERE id = $2;
COMMIT;`
	o[UpdateSQL] = s
	s = `
BEGIN TRANSACTION;
	INSERT INTO foo (stuff) VALUES ($1);
COMMIT;`
	o[SingularTable] = s
	s = `
SELECT * FROM "users"   ORDER BY "users"."id" ASC LIMIT 1
`
	o[FirstSQL1] = s
	s = `
SELECT * FROM "users"  WHERE ("users"."id" = $1) ORDER BY "users"."id" ASC LIMIT 1
`
	o[FirstSQL2] = s
	s = `
SELECT * FROM "users"   ORDER BY "users"."id" DESC LIMIT 1
`
	o[LastSQL1] = s
	s = `
SELECT * FROM "users"  WHERE ("users"."id" = $1) ORDER BY "users"."id" DESC LIMIT 1
`
	o[LastSQL2] = s
	s = `SELECT * FROM "users"`
	o[FindSQL1] = s
	s = `SELECT * FROM "users"   LIMIT 2`
	o[FindSQL2] = s
	s = `CREATE INDEX _idx_foo_stuff ON foos(stuff) `
	o[AddIndexSQL] = s
	s = `
BEGIN TRANSACTION;
	DELETE FROM foos  WHERE id = $1 ;
COMMIT;
`
	o[DeleteSQL] = s
	s = `CREATE UNIQUE INDEX idx_foo_stuff ON foos(stuff) `
	o[AddUniqueIndex] = s
	return o
}
func GetSQL(dialect string, key string) string {
	if d, ok := samples[dialect]; ok {
		return strings.TrimSpace(d[key])
	}
	return ""
}

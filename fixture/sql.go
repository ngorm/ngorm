package fixture

import "strings"

const (
	CreateTable1  = "ceate_table_1"
	CreateTable2  = "ceate_table_2"
	DropTable     = "drop_table"
	AutoMigrate   = "automigrate"
	SaveSQL       = "sqve_sql"
	UpdateSQL     = "update_sql"
	SingularTable = "singular_table"
	FirstSQL1     = "firsr_sql_1"
	FirstSQL2     = "firsr_sql_2"
)

var samples map[string]map[string]string

func init() {
	samples = make(map[string]map[string]string)
	samples["ql-mem"] = sampleQL()
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
	s = `SELECT * FROM users   ORDER BY id ASC`
	o[FirstSQL1] = s
	s = `SELECT * FROM users  WHERE (id = $1) ORDER BY id ASC`
	o[FirstSQL2] = s
	return o
}

func GetSQL(dialect string, key string) string {
	if d, ok := samples[dialect]; ok {
		return strings.TrimSpace(d[key])
	}
	return ""
}

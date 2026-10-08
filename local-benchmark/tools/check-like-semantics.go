package main
import (
 "database/sql"
 "fmt"
 "log"
 "github.com/go-sql-driver/mysql"
)
func main() {
 c:=mysql.NewConfig(); c.Net="tcp"; c.Addr="127.0.0.1:3306"; c.User="isucon"; c.Passwd="isucon"; c.DBName="isupipe"; c.ParseTime=true
 db,e:=sql.Open("mysql",c.FormatDSN()); if e!=nil {log.Fatal(e)}; defer db.Close()
 var conn,param string
 e=db.QueryRow("SELECT @@collation_connection, COLLATION(?)", "AbC").Scan(&conn,&param); if e!=nil {log.Fatal(e)}
 fmt.Printf("connection=%s parameter=%s\n",conn,param)
 for _,p:=range [][2]string{{"ABC","abc"},{"café","cafe"},{"abc","a_c"},{"abc","%"},{"a%b",`a\%b`},{"a_b",`a\_b`},{"日本語","本"},{"", ""}} {
  var original,replacement int
  e=db.QueryRow("SELECT (SELECT COUNT(*) FROM (SELECT ? AS text) texts INNER JOIN (SELECT CONCAT('%', ?, '%') AS pattern) patterns ON texts.text LIKE patterns.pattern), CONVERT(? USING utf8mb4) COLLATE utf8mb4_general_ci LIKE CONCAT('%',CONVERT(? USING utf8mb4) COLLATE utf8mb4_general_ci,'%')",p[0],p[1],p[0],p[1]).Scan(&original,&replacement)
  if e!=nil {log.Fatal(e)}; if original!=replacement {log.Fatalf("mismatch %q original=%d replacement=%d",p,original,replacement)}
  fmt.Printf("%q match=%d\n",p,original)
 }
}

package datatable

import (
	"fmt"
	"strings"
)

/*
func main() {
	conn, err := gohive.Connect("47.93.244.185:10000", gohive.DefaultOptions)
	fmt.Println(conn)
	if err != nil {
		fmt.Errorf("Connect error %v", err)
	}

	var c1 string
		rs, err := conn.Query( "alter table u_user set location" + "\"alluxio://172.16.0.118:19998/user/hive/warehouse/u_user\"")
		if err != nil {
			fmt.Errorf("Connection.Query error: %v", err)
		}
		for rs.Next() {
			rs.Scan(&c1)
			fmt.Println(c1)
		}

	rs, err := conn.Query("show create table u_user")
	if rs.Next() {
		rs.Scan(&c1)
		fmt.Println(c1)
	}
	conn.Close()
}*/

func main() {
	var a = "ds-schemaName-tableName-partitionName"
	strs := strings.Split(a, "-")
	if len(strs) == 4 {
		fmt.Print(strs[1])
	}

}

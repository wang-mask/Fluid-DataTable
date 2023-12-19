package main

import (
	"errors"
	"fmt"

	"strings"

	"github.com/dazheng/gohive"
)

func main() {

	str := "expforpartition-part_catalog_returns-cr_returned_date_sk=2450838"
	list := strings.SplitN(str, "_", 3)
	for _, item := range list {
		println(item)
	}
	println(list)

}

func TableLocationNotChange(conn *gohive.Connection, tableName string) (bool, string, error) {
	rs, err := conn.Query("desc formatted " + tableName)
	if err != nil {
		println(err)
	}
	var resultLine string
	for rs.Next() {
		_ = rs.Scan(&resultLine)
		fmt.Println(len(resultLine))
		if strings.Contains(resultLine, "Location11:") {
			fmt.Println("11111111")
			datatableLocation := strings.SplitN(resultLine, "\\s+", 2)[1]
			fmt.Println(datatableLocation)
			if strings.Contains(datatableLocation, "hdfs") {
				return false, datatableLocation, nil
			} else {
				return true, "", nil
			}
		}
	}
	return true, "", errors.New("fail to get the location")
}

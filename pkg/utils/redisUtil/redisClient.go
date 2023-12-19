package redisUtil

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"strconv"
)

/**
Connect the redisUtil.
*/
/*https://www.tizi365.com/archives/304.html  可以查询的api*/

var ctx context.Context
var zSetKey = "tableScore"

func ConnectRedis() *redis.Client {
	cli := redis.NewClient(&redis.Options{
		Addr:     "210.28.132.16:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return cli
}

/**
Get the table which has the minimum score.
*/
func IsinZset(cli *redis.Client, member string) bool {
	_, err := cli.ZRank(cli.Context(), zSetKey, member).Result()
	if err != nil {
		return false
	}
	return true

}

/**
获取redis zset中所有文件的列表，并且是按照score由小到大进行排序好的
*/
func GetValue(cli *redis.Client) []string {
	re, _ := cli.ZRange(cli.Context(), zSetKey, 0, -1).Result()
	fmt.Println(re)
	return re
}

/** 判断表是否缓存在Alluxio集群中
1 表示在缓存中， 0 表示不在缓存中
*/
func IsExistInALluxio(cli *redis.Client, tableName string) bool {
	re, _ := cli.Exists(cli.Context(), tableName).Result()
	if re == 1 {
		return true
	} else {
		return false
	}
}

/**
redis中存储kv对，k表示表名，v表示表的大小
*/
func GetSize(cli *redis.Client, tableName string) int64 {
	re, _ := cli.Get(cli.Context(), tableName).Result()

	size, _ := strconv.ParseInt(re, 10, 64)
	return size
}

// 添加键值对
func AddKV(cli *redis.Client, key string, value string) error {
	err := cli.Set(cli.Context(), key, value, 0).Err()
	return err

}

// 删除键值对
func RemoveKV(cli *redis.Client, key string) error {
	err := cli.Del(cli.Context(), key).Err()
	return err
}

func main() {
	cli := ConnectRedis()
	// GetValue(cli)
	// fmt.Println(IsinZset(cli,"tpcds_date_dim"))
	fmt.Println(IsExistInALluxio(cli, "tpcds_date_dim"))

}

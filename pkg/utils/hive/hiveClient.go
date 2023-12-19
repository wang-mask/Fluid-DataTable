package hive

import (
	datav1alpha1 "Fluid-Datatable/api/v1alpha1"
	"Fluid-Datatable/pkg/utils"
	datatableUtils "Fluid-Datatable/pkg/utils/datatable"
	execAlluxio "Fluid-Datatable/pkg/utils/exec"
	re "Fluid-Datatable/pkg/utils/redisUtil"
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/dazheng/gohive"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"
	cli "sigs.k8s.io/controller-runtime/pkg/client"
)

var mutex = sync.Mutex{}                    // 类变量
var alluxio_master_pod_ip = "210.28.132.18" // todo
var hdfs_master_ip = "210.28.132.15"        // todo

func ConnectHiveMetaStore(hostPort string) (conn *gohive.Connection, err error) {
	//	utils.Log.Info("show hostPost", hostPort)
	conn, err = gohive.Connect(hostPort, gohive.DefaultOptions)
	if err != nil {
		utils.Log.Error(err, "fail to connect the hive serivce", "hostPort", hostPort)
		return
	}
	return conn, err
}

func CloseHiveMetasStore(conn *gohive.Connection) (err error) {
	err = conn.Close()
	if err != nil {
		utils.Log.Error(err, "fail to close the hive serivce")
	}
	return err
}

// GetTableSize gets the size of the table
func GetTableSize(conn *gohive.Connection, schemaName string, tableName string) (size int64, err error) {
	_, err = conn.Query("use " + schemaName)
	if err != nil {
		utils.Log.Error(err, "fail to change the schema of hive")
		return -1, err
	}

	rs, err := conn.Query("desc formatted " + tableName)
	if err != nil {
		utils.Log.Error(err, "fail to execute the sql", "sql", "desc formatted", "tableName", tableName)
	}
	utils.Log.Info("desc formatted", "schemaName", schemaName, "tableName", tableName)
	var resultLine string
	for rs.Next() {
		err = rs.Scan(&resultLine)
		if err != nil {
			utils.Log.Error(err, "fail to execute the sql")
			return -1, err
		}
		utils.Log.Info("get the line", "resultline", resultLine)

	}
	return -1, err
}

// ChangeSchemaURL change the table location for alluxio in the specific schema.
func ChangeSchemaURLForCache(client cli.Client, datatable datav1alpha1.DataTable, conn *gohive.Connection, schema datav1alpha1.Schema) ([]string, error) {
	var labelToUpdate []string // store the label of the table

	_, err := conn.Query("use " + schema.SchemaName)
	if err != nil {
		utils.Log.Error(err, "fail to change the schema of hive")
		return nil, err
	}

	// 如果是schema中没有针对表的任何信息进行填写，那么就将挂载整个schema的数据
	if len(schema.Tables) == 0 {
		schema.Tables, err = GetTablesFromSchema(conn, schema.SchemaName)
		if err != nil {
			utils.Log.Error(err, "fail to execute the sql")
		}
	}

	// operate each table
	for _, table := range schema.Tables {
		if len(table.ColumnName) == 0 && len(table.PartitionColumn) == 0 {
			// 如果没有column和partition column，那么就是对整张表的缓存。
			err = ChangeTableURL(client, conn, schema.SchemaName, table.TableName, true)
			labelToUpdate = append(labelToUpdate, datatableUtils.GetDatatableCommonLabel(schema.SchemaName, table.TableName))
			if err != nil {
				utils.Log.Error(err, "fail to change table URL.")
				return nil, err
			}
		} else if len(table.ColumnName) == 0 && len(table.PartitionColumn) != 0 {
			// 缓存分区表
			partitionNameArrs := table.PartitionColumn
			// get the suitable partition url
			suitablePartitionURLs, err := GetSuitablePartition(conn, schema.SchemaName, table.TableName, partitionNameArrs)
			if err != nil {
				utils.Log.Error(err, "fail to get suitable partition url.")
				return nil, err
			}

			for _, suitablePartitionURL := range suitablePartitionURLs {
				err = ChangePartitionTableURL(client, conn, schema.SchemaName, table.TableName, suitablePartitionURL, true)
				labelToUpdate = append(labelToUpdate, datatableUtils.GetPartitionDatatableCommonLabel(schema.SchemaName, table.TableName, suitablePartitionURL))
			}
			if err != nil {
				utils.Log.Error(err, "fail to change partition table URL.")
				return nil, err
			}
		} else if len(table.ColumnName) != 0 && len(table.PartitionColumn) == 0 {
			// 缓存列
			//todo 缓存列
		} else if len(table.ColumnName) != 0 && len(table.PartitionColumn) != 0 {
			// 直接报错
		}
	}
	return labelToUpdate, err
}

// GetTablesFromSchema gets all the tables of the specific schema.
func GetTablesFromSchema(conn *gohive.Connection, schemaName string) (tables []datav1alpha1.Table, err error) {
	_, err = conn.Query("use " + schemaName)
	if err != nil {
		utils.Log.Error(err, "fail to execute the sql", "sql", "use "+schemaName, "schema", schemaName)
		return nil, err
	}

	rs, err := conn.Query("show tables")
	if err != nil {
		utils.Log.Error(err, "fail to execute the sql", "sql", "show tables", "schema", schemaName)
	}

	var resultLine string
	for rs.Next() {
		_ = rs.Scan(&resultLine)
		table := datav1alpha1.Table{TableName: resultLine}
		tables = append(tables, table)
	}

	return tables, nil
}

// ChangeSchemaURLForRecover change the table location for recover
func ChangeSchemaURLForRecover(client cli.Client, datatable datav1alpha1.DataTable, conn *gohive.Connection) error {

	mutex.Lock()
	defer mutex.Unlock()

	datatableLabels := datatable.Labels
	//for _, table := range schema.Tables { // operate each table
	for labelKey, labelValue := range datatableLabels {
		if labelValue != "true" {
			continue
		}
		datatableLabelSelector, err := labels.Parse(fmt.Sprintf("%s=true", labelKey))
		if err != nil {
			utils.Log.Error(err, "fail to create the datatable label selector.")
			return err
		}

		// check
		datatableList := datav1alpha1.DataTableList{}
		err = client.List(context.TODO(), &datatableList, &cli.ListOptions{
			LabelSelector: datatableLabelSelector,
		})
		if err != nil {
			utils.Log.Error(err, "fail to list the datatable list.")
		}
		utils.Log.Info("get datatable list", "tablelist", datatableList)
		// 只有执行过delete之后，才会给添加上时间戳，此时再进行list的时候就已经不存在该datatable了。
		// 要想实现最后一个datatable修改回原来的url，就必须要在list的时候发现该datatable已经为空，才可以执行。
		if len(datatableList.Items) <= 1 { // todo 可能需要修改
			//执行删除操作
			schemaName, tableName, partitionName := datatableUtils.GetTableInfoFromLabel(labelKey)
			if partitionName == "" {
				err = ChangeTableURL(client, conn, schemaName, tableName, false)
			} else {
				err = ChangePartitionTableURL(client, conn, schemaName, tableName, partitionName, false)
			}
			if err != nil {
				utils.Log.Error(err, "fail to change table URL.")
				return err
			}
		}
	}

	// suppose more than one datatable here for deletion
	// 每个都觉得自己不是最后一个，这时候两个都更新完之后，hive metastore中的内容还是没有更新,必须要加锁。
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {

		dt, err := datatableUtils.GetDatatable(client, datatable.Name, datatable.Namespace)
		if err != nil {
			utils.Log.Error(err, "fail to get datatable.")
		}

		dtToUpdate := dt.DeepCopy()

		// 删除该标签
		for labelKey, _ := range datatableLabels {
			dtToUpdate.Labels[labelKey] = "false"
		}
		if !reflect.DeepEqual(dt, dtToUpdate) {
			err = client.Update(context.TODO(), dtToUpdate)
			if err != nil {
				return err
			}
		}
		return err
	})
	if err != nil {
		utils.Log.Error(err, "fail to update the label ")
	}

	return nil
}

var redisclient *redis.Client

// Change the location of the table.
func ChangeTableURL(client cli.Client, conn *gohive.Connection, schemaName string, tableName string, toCache bool) (err error) {
	if redisclient == nil {
		redisclient = re.ConnectRedis()
	}

	_, err = conn.Query("use " + schemaName)
	if err != nil {
		utils.Log.Error(err, "fail to change the schema of hive")
		return err
	}

	var changeLocationSQL string
	// 在default目录下 /user/hive/warehouse
	// 在其他schema目录下  /user/hive/warehouse/schemaName.db/<table-name>
	schemaDir := ""
	if !(schemaName == "default" || schemaName == "") {
		schemaDir = schemaName + ".db/"
	}

	if toCache {
		changeLocationSQL = "alter table " + tableName + " set location " + " \"alluxio://" + alluxio_master_pod_ip + ":19998/user/hive/warehouse/" + schemaDir + tableName + "\""
		changeLocationSQL = strings.TrimSpace(changeLocationSQL)
		err = execAlluxio.ExecAlluxioTablemount(client, schemaName, tableName)
		if err != nil {
			utils.Log.Error(err, "fail to execute mount ")
			return err
		}

		tableSize, err := execAlluxio.ExecTableSize(client, schemaName, tableName)
		if err != nil {
			utils.Log.Error(err, "fail to get table size")
			return err
		}

		err = re.AddKV(redisclient, schemaName+"_"+tableName, strconv.FormatInt(tableSize, 10)) // int64 to string
		if err != nil {
			utils.Log.Error(err, "fail to set the k-v in redis")
			return err
		}
	} else {
		changeLocationSQL = "alter table " + tableName + " set location " + " \"hdfs://" + hdfs_master_ip + ":9000/user/hive/warehouse/" + schemaDir + tableName + "\""
		execAlluxio.ExecAlluxioTableUnmountWithoutInfos(schemaName, tableName) // 执行解挂载操作
		err = re.RemoveKV(redisclient, schemaName+"_"+tableName)
		if err != nil {
			utils.Log.Error(err, "fail to del the k-v in redis")
			return err
		}
	}

	utils.Log.Info(changeLocationSQL)
	_, err = conn.Query(changeLocationSQL)
	if err != nil {
		utils.Log.Error(err, "fail to update the table location")
		return err
	}

	return
}

// Check user partition is correct in the metastore
func GetSuitablePartition(conn *gohive.Connection, schemaName string, tableName string, partitionNameArr []map[string]string) ([]string, error) {
	// use schemaName
	_, err := conn.Query("use " + schemaName)
	if err != nil {
		utils.Log.Error(err, "fail to execute the sql", "sql", "use "+schemaName, "schema", schemaName)
		return nil, err
	}

	var suitablePartitionURL []string
	// show partitions table-name
	rs, err := conn.Query("show partitions " + tableName)
	if err != nil {
		utils.Log.Error(err, "fail to execute the sql", "sql", "show tables", "schema", schemaName)
		return nil, err
	}

	var resultLine string
	for rs.Next() {

		_ = rs.Scan(&resultLine)
		resultLine = strings.TrimSpace(resultLine)
		utils.Log.Info(resultLine + "end")
		// utils.Log.Error(err, "fail to execute the sql", "sql", "show tables", "schema", schemaName)
		// 对每一个需要检查的map映射
		flag := false
		for _, partitionName := range partitionNameArr { // 对每个分区表（包含多个k/v）
			// 该映射中的每个键值对
			for k, v := range partitionName {
				// 如果不包含，则继续
				if !strings.Contains(resultLine, k+"="+v) {
					flag = true
					break
				}
			}
			if !flag {
				// 当前这个分区表对应resultLine，将resultline加入到最后的结果中，并开始读取下一行。
				suitablePartitionURL = append(suitablePartitionURL, resultLine)
			}
			break
		}
	}

	// return the result
	return suitablePartitionURL, nil
}

// Change the location of the partition table.
func ChangePartitionTableURL(client cli.Client, conn *gohive.Connection, schemaName string, tableName string, partitionName string, toCache bool) (err error) {
	_, err = conn.Query("use " + schemaName)
	if err != nil {
		utils.Log.Error(err, "fail to change the schema of hive")
		return err
	}
	var changeLocationSQL string
	// 在default目录下 /user/hive/warehouse
	// 在其他schema目录下  /user/hive/warehouse/schemaName.db/<table-name>
	schemaDir := ""
	if !(schemaName == "default" || schemaName == "") {
		schemaDir = schemaName + ".db/"
	}

	partitions := strings.Split(partitionName, "/") //  table=.../add=beijing
	partitionFilter := "partition("
	for index, partition := range partitions {
		partitionKV := strings.Split(partition, "=")
		partitionFilter = partitionFilter + partitionKV[0] + "='" + partitionKV[1] + "'"
		if index != len(partitions)-1 {
			partitionFilter = partitionFilter + ","
		}
	}
	partitionFilter = partitionFilter + ")"
	utils.Log.Info(partitionFilter)

	// for each suitable partition table, change the url.
	if toCache {
		changeLocationSQL = "alter table " + tableName + " " + partitionFilter + " set location " + " \"alluxio://" + alluxio_master_pod_ip + ":19998/user/hive/warehouse/" + schemaDir + tableName + "/" + partitionName + "\""
	} else {
		changeLocationSQL = "alter table " + tableName + " " + partitionFilter + " set location " + " \"hdfs://" + hdfs_master_ip + ":9000/user/hive/warehouse/" + schemaDir + tableName + "/" + partitionName + "\""
	}

	utils.Log.Info(changeLocationSQL)
	_, err = conn.Query(changeLocationSQL)

	// ExecAlluxioTablemount (podName string,  containerName string, namespace string, schemaName string, tableName string)

	if err != nil {
		utils.Log.Error(err, "fail to update the table location")
		return err
	}
	if toCache {
		execAlluxio.ExecAlluxioPartitionTablemount(client, schemaName, tableName, partitionName)
	} else {
		execAlluxio.ExecAlluxioPartitionTableUnmount(schemaName, tableName, partitionName)
	}
	return
}

// TableLocationNotChange checks the table is changed or not.
func TableLocationNotChange(conn *gohive.Connection, tableName string) (bool, string, error) {
	rs, err := conn.Query("desc formatted " + tableName)
	if err != nil {
		utils.Log.Error(err, "fail to query", "sql", "desc formatted "+tableName)
	}
	var resultLine string
	for rs.Next() {
		_ = rs.Scan(&resultLine)
		if strings.Contains(resultLine, "Location:") {
			datatableLocation := strings.SplitN(resultLine, "\\s+", 2)[1]
			if strings.Contains(datatableLocation, "hdfs") {
				return false, datatableLocation, nil
			} else {
				return true, "", nil
			}
		}
	}
	return true, "", errors.New("fail to get the location")
}

package exec

import (
	datav1alpha1 "Fluid-Datatable/api/v1alpha1"
	"Fluid-Datatable/pkg/utils"
	"Fluid-Datatable/pkg/utils/kubeclient"
	"context"
	"github.com/pkg/errors"
	cli "sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
)

/*
var (

	execpodName       string
	execcontainerName string
	execnamespace     string

)
*/
var (
	execpodName       = "demo1-master-0"
	execnamespace     = "default"
	execcontainerName = "alluxio-master"
)
var hdfs_master_ip = "210.28.132.15" // todo

func initVal(client cli.Client) error {
	var alluxioRuntimeList datav1alpha1.AlluxioRuntimeList
	if err := client.List(context.TODO(), &alluxioRuntimeList); err != nil {
		utils.Log.Error(err, "fail to get the alluxioruntime")
		return err
	}

	if len(alluxioRuntimeList.Items) == 0 {
		// 不存在Alluxio 集群就直接报错
		utils.Log.Error(errors.New("fail to get the alluxio runtime, please run the alluxio runtime first"), "fail to get the alluxio runtime")
		return errors.New("fail to get the alluxio runtime")
	}

	execpodName = alluxioRuntimeList.Items[0].Name + "-master-0"
	execnamespace = alluxioRuntimeList.Items[0].Namespace
	execcontainerName = "alluxio-master"

	return nil
}

// alluxio fs getUsedBytes
// alluxio fs getCapacityBytes
func ExecAlluxioFreeSize(podName string, containerName string, namespace string) (result int64, err error) {
	execpodName = podName
	execcontainerName = containerName
	execnamespace = namespace

	command := []string{"alluxio", "fs", "getUsedBytes"}

	var stdout string
	var stderr string
	stdout, stderr, err = exec(command, podName, containerName, namespace)
	if err != nil {
		utils.Log.Error(err, "fail to get the used bytes with the stderr ", stderr)
		return
	}
	usedSize, err := strconv.ParseInt(stdout, 10, 64)
	if err != nil {
		utils.Log.Error(err, "fail to convert to the int64 ", stderr)
		return
	}

	command = []string{"alluxio", "fs", "getCapacityBytes"}
	stdout, stderr, err = exec(command, podName, containerName, namespace)
	if err != nil {
		utils.Log.Error(err, "fail to get the used bytes with the stderr ", stderr)
		return
	}
	totalSize, err := strconv.ParseInt(stdout, 10, 64)
	if err != nil {
		utils.Log.Error(err, "fail to convert to the int64 ", stderr)
		return
	}

	return totalSize - usedSize, err
}
func ExecAlluxioTableUnmountWithoutInfos(schemaName string, tableName string) (res string, err error) {
	return ExecAlluxioTableUnmount(execpodName, execcontainerName, execnamespace, schemaName, tableName)
}

func ExecAlluxioTableUnmount(podName string, containerName string, namespace string, schemaName string, fileName string) (res string, err error) {
	execpodName = podName
	execcontainerName = containerName
	execnamespace = namespace
	var schemaDir string
	if !(schemaName == "default" || schemaName == "") {
		schemaDir = schemaName + ".db/"
	}
	command := []string{"alluxio", "fs", "unmount", "/user/hive/warehouse/" + schemaDir + fileName}

	var stdout string
	var stderr string
	stdout, stderr, err = exec(command, podName, containerName, namespace)
	if err != nil {
		utils.Log.Error(err, "fail to umount ", stderr)
		return
	}

	return stdout, err
}

// mount table
func ExecAlluxioTablemount(client cli.Client, schemaName string, tableName string) error {

	if execpodName == "" {
		err := initVal(client)
		return err
	}

	var schemaDir string
	if !(schemaName == "default" || schemaName == "") {
		schemaDir = schemaName + ".db/"
	}

	// 判断是否已经挂载
	commandPre := []string{"alluxio", "fs", "ls", "/user/hive/warehouse/" + schemaDir + tableName}
	out, _, _ := exec(commandPre, execpodName, execcontainerName, execnamespace)
	if !strings.Contains(out, "not") {
		utils.Log.Info("the table already in the cache system", "schemaName", schemaDir, "tableName", tableName, "out", out)
		return nil
	}

	// 判断是否还没创建文件夹
	commandPre = []string{"alluxio", "fs", "ls", "/user/hive/warehouse/" + schemaDir}
	out, _, _ = exec(commandPre, execpodName, execcontainerName, execnamespace)
	if strings.Contains(out, "not") {
		commandPre = []string{"alluxio", "fs", "mkdir", "/user/hive/warehouse/" + schemaDir}
		exec(commandPre, execpodName, execcontainerName, execnamespace)
	}

	command := []string{"alluxio", "fs", "mount", "/user/hive/warehouse/" + schemaDir + tableName, "hdfs://" + hdfs_master_ip + ":9000/user/hive/warehouse/" + schemaDir + tableName}

	_, stderr, err := exec(command, execpodName, execcontainerName, execnamespace)
	if err != nil {
		utils.Log.Error(err, "fail to mount", stderr)
		return err
	}

	return err
}

func ExecAlluxioPartitionTableUnmount(schemaName string, tableName string, partitionName string) (res string, err error) {
	var schemaDir string
	if !(schemaName == "default" || schemaName == "") {
		schemaDir = schemaName + ".db/"
	}

	command := []string{"alluxio", "fs", "unmount", "/user/hive/warehouse/" + schemaDir + tableName + "/" + partitionName}

	var stdout string
	var stderr string
	stdout, stderr, err = exec(command, execpodName, execcontainerName, execnamespace)
	if err != nil {
		utils.Log.Error(err, "fail to mount", stderr)
		return
	}

	return stdout, err
}

func ExecAlluxioPartitionTablemount(client cli.Client, schemaName string, tableName string, partitionName string) (err error) {

	if execpodName == "" {
		err := initVal(client)
		return err
	}

	var schemaDir string
	if !(schemaName == "default" || schemaName == "") {
		schemaDir = schemaName + ".db/"
	}

	// Step1 : 判断是否已经挂载
	commandPre := []string{"alluxio", "fs", "ls", "/user/hive/warehouse/" + schemaDir + tableName + "/" + partitionName}
	out, _, _ := exec(commandPre, execpodName, execcontainerName, execnamespace)
	if !strings.Contains(out, "not") {
		utils.Log.Info("the table already in the cache system", "schemaName", schemaDir, "tableName", tableName, "out", out)
		return nil
	}

	// Step2 : 创建父目录
	// 判断是否还没创建文件夹
	commandPre = []string{"alluxio", "fs", "ls", "/user/hive/warehouse/" + schemaDir + tableName}
	out, _, _ = exec(commandPre, execpodName, execcontainerName, execnamespace)
	if strings.Contains(out, "not") {
		commandPre = []string{"alluxio", "fs", "mkdir", "/user/hive/warehouse/" + schemaDir + tableName}
		exec(commandPre, execpodName, execcontainerName, execnamespace)
	}

	// Step3 :挂载
	command := []string{"alluxio", "fs", "mount", "/user/hive/warehouse/" + schemaDir + tableName + "/" + partitionName, "hdfs://" + hdfs_master_ip + ":9000/user/hive/warehouse/" + schemaDir + tableName + "/" + partitionName}

	// var stdout string
	var stderr string
	_, stderr, err = exec(command, execpodName, execcontainerName, execnamespace)
	if err != nil {
		utils.Log.Error(err, "fail to mount", stderr)
		return
	}

	return err
}

func exec(command []string, podName string, containerName string, namespace string) (stdout string, stderr string, err error) {
	execpodName = podName
	execcontainerName = containerName
	execnamespace = namespace

	stdout, stderr, err = kubeclient.ExecCommandInContainer(podName, containerName, namespace, command)
	if err != nil {
		utils.Log.Info("Stdout", "Command", command, "Stdout", stdout)
		utils.Log.Error(err, "Failed", "Command", command, "FailedReason", stderr)
		return
	}

	return
}

func ExecTableSize(client cli.Client, schemaName string, tableName string) (res int64, err error) {
	if execpodName == "" {
		err := initVal(client)
		if err != nil {
			return -1, err
		}
	}
	var schemaDir string
	if !(schemaName == "default" || schemaName == "") {
		schemaDir = schemaName + ".db/"
	}
	command := []string{"alluxio", "fs", "du", "/user/hive/warehouse/" + schemaDir + tableName}

	var stdout string
	var stderr string
	stdout, stderr, err = exec(command, execpodName, execcontainerName, execnamespace)
	if err != nil {
		utils.Log.Error(err, "fail to mount", stderr)
		return
	}

	str := strings.Split(stdout, "\n")
	if len(str) == 1 {
		return -1, err
	}

	var tableSize int64
	tableSize = 0
	for index, s := range str {
		if index == 0 {
			continue
		}
		data := strings.Fields(s)
		size, err := strconv.ParseInt(data[0], 10, 64)
		if err != nil {
			return -1, err
		}
		tableSize = tableSize + size
	}
	return tableSize, err
}

package datatable

import (
	datav1alpha1 "Fluid-Datatable/api/v1alpha1"
	"context"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (

	// prefix of the label for datatable
	DATATABLEPRE = "datatable"
)

func GetDatatableCommonLabel(schemaName string, tableName string) string {
	return DATATABLEPRE + "-" + schemaName + "-" + tableName
}

func GetPartitionDatatableCommonLabel(schemaName string, tableName string, partitionName string) string {
	if strings.Contains(partitionName, "=") {
		partitionName = strings.ReplaceAll(partitionName, "=", ".")
	}

	return GetDatatableCommonLabel(schemaName, tableName) + "-" + partitionName
}

func GetTableInfoFromLabel(label string) (schemaName string, tableName string, partitionName string) {
	infos := strings.Split(label, "-")
	if len(infos) == 4 {
		infos[3] = strings.ReplaceAll(infos[3], ".", "=")
		return infos[1], infos[2], infos[3]
	}

	return infos[1], infos[2], ""
}

func GetDatatable(client client.Client, name, namespace string) (*datav1alpha1.DataTable, error) {
	key := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	var datatable datav1alpha1.DataTable
	if err := client.Get(context.TODO(), key, &datatable); err != nil {
		return nil, err
	}
	return &datatable, nil
}

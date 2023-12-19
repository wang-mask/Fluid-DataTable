package utils

import (
	"fmt"

	"Fluid-Datatable/pkg/common"
)

// GetExclusiveKey gets exclusive key
func GetExclusiveKey() string {
	return common.FluidExclusiveKey
}

// GetExclusiveValue gets exclusive value
func GetExclusiveValue(namespace, name string) string {
	return fmt.Sprintf("%s_%s", namespace, name)
}

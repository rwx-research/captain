package config

import "fmt"

type PartitionNodes struct {
	Total int
	Index int
}

func (pn PartitionNodes) String() string {
	return fmt.Sprintf("%d/%d", pn.Index, pn.Total)
}

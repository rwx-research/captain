package testing

import (
	"fmt"
	"time"
)

type TestPartition struct {
	RemainingCapacity time.Duration
	Index             int
	TestFilePaths     []string
	TotalCapacity     time.Duration
}

func (p TestPartition) Add(matchedTiming FileTimingMatch) TestPartition {
	p = p.AddFilePath(matchedTiming.ClientFilepath)
	p.RemainingCapacity -= matchedTiming.Duration()
	return p
}

func (p TestPartition) AddFilePath(filepath string) TestPartition {
	p.TestFilePaths = append(p.TestFilePaths, filepath)
	return p
}

func (p TestPartition) String() string {
	percent := 100 - (float64(p.RemainingCapacity) / float64(p.TotalCapacity) * 100)
	return fmt.Sprintf("[PART %d (%0.2f)]", p.Index, percent)
}

package testing

import (
	"fmt"
	"time"
)

type TestPartition struct {
	Index         int
	Runtime       time.Duration
	TestFilePaths []string
}

func (p TestPartition) Add(matchedTiming FileTimingMatch) TestPartition {
	p = p.AddFilePath(matchedTiming.ClientFilepath)
	p.Runtime += matchedTiming.Duration()
	return p
}

func (p TestPartition) AddFilePath(filepath string) TestPartition {
	p.TestFilePaths = append(p.TestFilePaths, filepath)
	return p
}

func (p TestPartition) String() string {
	return fmt.Sprintf("[PART %d (%0.2fs)]", p.Index, p.Runtime.Seconds())
}

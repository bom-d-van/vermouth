package verparser

import (
	"testing"
	. "launchpad.net/gocheck"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type MainSuite struct{}

var _ = Suite(&MainSuite{})

func (s *MainSuite) TestParse(c *C) {
	changes, err := Parse("github.com/bom-d-van/vermouth/verparser/testapis/v1", "github.com/bom-d-van/vermouth/verparser/testapis/v2")
	if err != nil {
		c.Fatal(err)
	}

	c.Check(changes.GenDoc(), Equals, `The New API is NOT backward compatible.
Structs:
Deprecated: OldEntry
Modified:
Entry:
	Deprecated Fields:
		OldField float64
	Type Changes:
		Id: string -> int
	New Fields:
		NewField string
New: EntryItem
========
Interfaces:
Deprecated: RemovedInterface
Modified:
Methods:
	Deprecated Methods:
		TemporalSession() (session string)
	Signature Changes:
		OldApi(name string, id string) (session string)
		-> OldApi(name string, id int) (session string)
	New Methods:
		NewApi(id int) (session string)
New: NewInterface
`)
}

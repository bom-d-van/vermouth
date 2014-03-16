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

	Name string

	Modified Types:

	Id: string -> int

	New Fields:

	Type string

New:

========

Interfaces:

Deprecated:

Modified:

Methods:

	Deprecated:

	TemporalSession() (session string)

	Modified:

	OldApi(name string, id string) (session string)
	->
	OldApi(name string, id int) (session string)

	New:

	NewApi(id int) (session string)

New:
`)
}

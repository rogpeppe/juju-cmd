package main_test

import (
	. "launchpad.net/gocheck"
	main "launchpad.net/juju/go/cmd/jujud"
)

type UnitSuite struct{}

var _ = Suite(&UnitSuite{})

func (s *UnitSuite) TestParseSuccess(c *C) {
	create := func() main.AgentCommand { return main.NewUnitCommand() }
	uc := CheckAgentCommand(c, create, []string{"--unit-name", "w0rd-pre55/1"})
	c.Assert(uc.(*main.UnitCommand).Agent.Name, Equals, "w0rd-pre55/1")
}

func (s *UnitSuite) TestParseMissing(c *C) {
	uc := main.NewUnitCommand()
	err := ParseAgentCommand(uc, []string{})
	c.Assert(err, ErrorMatches, "--unit-name option must be set")
}

func (s *UnitSuite) TestParseNonsense(c *C) {
	for _, args := range [][]string{
		[]string{"--unit-name", "wordpress"},
		[]string{"--unit-name", "wordpress/seventeen"},
		[]string{"--unit-name", "wordpress/-32"},
		[]string{"--unit-name", "wordpress/wild/9"},
		[]string{"--unit-name", "20/20"},
	} {
		err := ParseAgentCommand(main.NewUnitCommand(), args)
		c.Assert(err, ErrorMatches, "--unit-name option expects <service-name>/<non-negative integer>")
	}
}

func (s *UnitSuite) TestParseUnknown(c *C) {
	uc := main.NewUnitCommand()
	err := ParseAgentCommand(uc, []string{"--unit-name", "wordpress/1", "thundering typhoons"})
	c.Assert(err, ErrorMatches, `unrecognised args: \[thundering typhoons\]`)
}
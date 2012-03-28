package main_test

import (
	"io/ioutil"
	. "launchpad.net/gocheck"
	"launchpad.net/juju/go/cmd"
	"launchpad.net/juju/go/juju"
	main "launchpad.net/juju/go/cmd/juju"
	"launchpad.net/juju/go/environs/dummy"
	"os"
	"path/filepath"
	"reflect"
)

type cmdSuite struct {
	home string
}

var _ = Suite(&cmdSuite{})

var config = `
default:
    peckham
environments:
    peckham:
        type: dummy
        zookeeper: false
    walthamstow:
        type: dummy
        zookeeper: false
`

func (s *cmdSuite) SetUpTest(c *C) {
	// Arrange so that the "home" directory points
	// to a temporary directory containing the config file.
	s.home = os.Getenv("HOME")
	dir := c.MkDir()
	os.Setenv("HOME", dir)
	err := os.Mkdir(filepath.Join(dir, ".juju"), 0777)
	c.Assert(err, IsNil)
	err = ioutil.WriteFile(filepath.Join(dir, ".juju", "environments.yaml"), []byte(config), 0666)
	c.Assert(err, IsNil)
}

func (s *cmdSuite) TearDownTest(c *C) {
	os.Setenv("HOME", s.home)

	dummy.Reset(nil)
}

var cmdTests = []struct {
	cmd      cmd.Command
	args     []string
	ops      []dummy.Operation
	parseErr string
	runErr   string
} {
	{
		cmd: &main.BootstrapCommand{},
		args: []string{"hotdog"},
		parseErr: `unrecognised args: \[hotdog\]`,
	}, {
		cmd: &main.BootstrapCommand{},
		ops: envOps("peckham", dummy.OpBootstrap),
	}, {
		cmd: &main.DestroyCommand{},
		ops: envOps("peckham", dummy.OpDestroy),
	},
}

// newCommand makes a new Command of the same
// type as the old one.
func newCommand(old cmd.Command) cmd.Command {
	v := reflect.New(reflect.TypeOf(old).Elem())
	return v.Interface().(cmd.Command)
}

func testParse(c *C, com cmd.Command, args []string, errPat string) cmd.Command {
	com = newCommand(com)
	err := cmd.Parse(com, args)
	if err != nil {
		c.Assert(err, ErrorMatches, errPat)
	} else {
		c.Assert(err, IsNil)
	}
	return com
}

func testConn(c *C, com cmd.Command, name string) {
	v := reflect.ValueOf(com).Elem().FieldByName("Conn")
	c.Assert(v.IsValid(), Equals, true)
	conn := v.Interface().(*juju.Conn)
	c.Assert(dummy.EnvironName(conn.Environ), Equals, name)
}

// All members of genericTests are tested for the -environment and -e
// flags, and that extra arguments will cause parsing to fail.
var genericParseTests = []struct {
	cmd cmd.Command
	args []string
	allowExtraArgs bool
} {
	{
		cmd: &main.BootstrapCommand{},
	}, {
		cmd: &main.DestroyCommand{},
	},
}

func (*cmdSuite) TestGenericBehaviour(c *C) {
	for _, t := range genericParseTests {
		com := testParse(c, t.cmd, t.args, "")
		testConn(c, com, "peckham")

		com = testParse(c, t.cmd, append([]string{"-e", "walthamstow"}, t.args...), "")
		testConn(c, com, "walthamstow")

		com = testParse(c, t.cmd, append([]string{"-environment", "walthamstow"}, t.args...), "")
		testConn(c, com, "walthamstow")

		testParse(c, t.cmd, append([]string{"-e", "unknown"}, t.args...), "some error")

		if !t.allowExtraArgs {
			testParse(c, t.cmd, append(t.args, "hotdog"), "no args allowed")
		}
	}
}

func (*cmdSuite) TestCommands(c *C) {
	for i, t := range cmdTests {
		c.Logf("test %d", i)
		err := cmd.Parse(t.cmd, t.args)
		checkError(c, "parse", err, t.parseErr)
	
		// gather operations as they happen
		opc := make(chan dummy.Operation)
		dummy.Reset(opc)
		done := make(chan bool)
		var ops []dummy.Operation
		go func() {
			for op := range opc {
				ops = append(ops, op)
			}
			done <- true
		}()
		c.Logf("running %T %#v\n", t.cmd, t.cmd)
		err = t.cmd.Run()
		checkError(c, "run", err, t.runErr)
	
		// signal that we're done with this listener channel.
		dummy.Reset(nil)
		<-done
		c.Check(ops, DeepEquals, t.ops)
	}
}

func checkError(c *C, kind string, err error, expect string) {
	switch {
	case err != nil && expect == "":
		c.Fatalf("unexpected %s error: %v", kind, err)
	case err != nil && expect != "":
		c.Assert(err, ErrorMatches, expect)
	case err == nil && expect != "":
		c.Fatalf("unexpected %s success: expected %q", kind, expect)
	}
}

// envOps returns a slice of expected operations on a given
// environment name.
func envOps(name string, events ...dummy.OperationKind) []dummy.Operation {
	ops := make([]dummy.Operation, len(events))
	for i, e := range events {
		ops[i] = dummy.Operation{
			EnvironName: name,
			Kind: e,
		}
	}
	return ops
}
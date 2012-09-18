package main

import (
	"errors"
	"launchpad.net/gnuflag"
	"launchpad.net/juju-core/charm"
	"launchpad.net/juju-core/cmd"
	"launchpad.net/juju-core/juju"
	"os"
)

type DeployCommand struct {
	EnvName      string
	CharmName    string
	ServiceName  string
	Config       cmd.FileVar
	NumUnits     int // defaults to 1
	BumpRevision bool
	RepoPath     string // defaults to JUJU_REPOSITORY
}

const deployDoc = `
<charm name> can be a charm URL, or an unambiguously condensed form of it;
assuming a current default series of "precise", the following forms will be
accepted.

For cs:precise/mysql
  mysql
  precise/mysql

For cs:~user/precise/mysql
  cs:~user/mysql

For local:precise/mysql
  local:mysql

In all cases, a versioned charm URL will be expanded as expected (for example,
mysql-33 becomes cs:precise/mysql-33).

<service name>, if omitted, will be derived from <charm name>.
`

func (c *DeployCommand) Info() *cmd.Info {
	return &cmd.Info{
		"deploy", "<charm name> [<service name>]", "deploy a new service", deployDoc,
	}
}

func (c *DeployCommand) Init(f *gnuflag.FlagSet, args []string) error {
	addEnvironFlags(&c.EnvName, f)
	f.IntVar(&c.NumUnits, "n", 1, "number of service units to deploy for principal charms")
	f.IntVar(&c.NumUnits, "num-units", 1, "")
	f.BoolVar(&c.BumpRevision, "u", false, "increment local charm directory revision")
	f.BoolVar(&c.BumpRevision, "upgrade", false, "")
	f.Var(&c.Config, "config", "path to yaml-formatted service config")
	f.StringVar(&c.RepoPath, "repository", os.Getenv("JUJU_REPOSITORY"), "local charm repository")
	// TODO --constraints
	if err := f.Parse(true, args); err != nil {
		return err
	}
	args = f.Args()
	switch len(args) {
	case 2:
		c.ServiceName = args[1]
		fallthrough
	case 1:
		c.CharmName = args[0]
	case 0:
		return errors.New("no charm specified")
	default:
		return cmd.CheckEmpty(args[2:])
	}
	if c.NumUnits < 1 {
		return errors.New("must deploy at least one unit")
	}
	return nil
}

func (c *DeployCommand) Run(ctx *cmd.Context) error {
	conn, err := juju.NewConnFromName(c.EnvName)
	if err != nil {
		return err
	}
	defer conn.Close()
	conf, err := conn.State.EnvironConfig()
	if err != nil {
		return err
	}
	curl, err := charm.InferURL(c.CharmName, conf.DefaultSeries())
	if err != nil {
		return err
	}
	repo, err := charm.InferRepository(curl, ctx.AbsPath(c.RepoPath))
	if err != nil {
		return err
	}
	ch, err := conn.PutCharm(curl, repo, c.BumpRevision)
	if err != nil {
		return err
	}
	if c.Config.Path != nil {
		// TODO many dependencies :(
		return errors.New("state.Service.SetConfig not implemented (format 2...)")
	}
	svc, err := conn.AddService(c.ServiceName, ch)
	if err != nil {
		return err
	}
	if ch.Meta().Subordinate {
		return nil
	}
	_, err = conn.AddUnits(svc, c.NumUnits)
	return err
}

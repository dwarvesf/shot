package main

import (
	"os"

	"github.com/dwarvesf/shot/config"
	"github.com/dwarvesf/shot/dflog"
	"github.com/dwarvesf/shot/ssh"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

// Usage
// $ shot init --target=<target>  // --> /opt/shot/port
// $ shot deploy --file=feature__login.yml
// $ shot down --file=feature__login.yml

var (
	l   = dflog.New()
	cfg *config.Config

	app = kingpin.New("shot", "Automation deployment inside the fortress")

	setup     = app.Command("setup", "Setup all given servers")
	setupPath = setup.Flag("config", "Path to configuration file").Short('c').String()

	deploy     = app.Command("deploy", "Deploy given git branches to targeted servers")
	deployPath = deploy.Flag("config", "Path to configuration file").Short('c').String()

	down     = app.Command("down", "Put down all the targeted servers")
	downPath = down.Flag("config", "Path to configuration file").Short('c').String()
)

func init() {
	app.Version("1.0")
	app.Author("dev@dwarvesf.com")
}

func main() {

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {

	case setup.FullCommand():
		SetupTarget(*setupPath)

	case deploy.FullCommand():

	case down.FullCommand():

	default:
		l.Error("Command not found.")
	}
}

func SetupTarget(configFile string) {

	if configFile == "" {
		l.Fatal("Configuration file not found.")
	}

	cfg, err := config.Init(configFile)
	if err != nil {
		l.Fatal(err)
	}

	for _, target := range *cfg.Targets {
		c := ssh.Credential{
			User: target.User,
			Host: target.Host,
			Port: target.Port,
		}
		ssh.Run("mkdir ~/hellox/", []ssh.Credential{c})
	}
}

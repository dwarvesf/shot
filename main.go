package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
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
		logrus.Info("Reading config file...")
		conf, err := config.Init(*deployPath)
		if err != nil {
			logrus.Fatal(err)
		}

		for _, t := range conf.Targets {
			for _, b := range t.Branches {
				gco := fmt.Sprintf("git checkout %s", b)
				image := fmt.Sprintf("%s/%s:%s", conf.Registry, conf.Project.Name, strings.Replace(b, "/", "-", -1))
				dBuild := fmt.Sprintf("docker build -t %s .", image)
				dPush := fmt.Sprintf("docker push %s", image)

				// run cmd in local to build and push docker
				cmds := []string{gco, dBuild, dPush}
				for _, cmd := range cmds {
					execCMD(cmd)
				}

				availablePort, err := ssh.Run("cat /opt/shot/port", ssh.Credential{
					User: t.User,
					Host: t.Host,
					Port: t.Port,
				})
				if err != nil {
					logrus.WithError(err).Error("Cannot read file port from server")
					continue
				}
				port, err := strconv.Atoi(strings.TrimSpace(availablePort))
				if err != nil {
					logrus.WithError(err).Error("Cannot convert port to int")
					continue
				}

				dPull := fmt.Sprintf("docker pull %s", image)
				dRun := fmt.Sprintf("docker run -d -p %s:%d %s", availablePort, conf.Project.Port, image)
				cmds = []string{dPull, dRun}
				for _, cmd := range cmds {
					_, err := ssh.Run(cmd, ssh.Credential{
						User: t.User,
						Host: t.Host,
						Port: t.Port,
					})
					if err != nil {
						logrus.WithError(err).Error("Cannot run command on target server")
						break
					}
				}

				// copy new port to target and remove file port from localhost
				p := []byte(strconv.Itoa(port + 1))
				err = ioutil.WriteFile("port", p, 0644)
				if err != nil {
					logrus.WithError(err).Error("Cannot write port to file")
					continue
				}
				execCMD(fmt.Sprintf("scp port %s@%s:/opt/shot/port", t.User, t.Host))
				err = os.Remove("port")
				if err != nil {
					logrus.WithError(err).Error("Cannot remove file port")
				}
			}
		}

	case down.FullCommand():

	default:
		l.Error("Command not found.")
	}
}

// SetupTarget ...
func SetupTarget(configFile string) {

	// if configFile == "" {
	// 	l.Fatal("Configuration file not found.")
	// }

	// cfg, err := config.Init(configFile)
	// if err != nil {
	// 	l.Fatal(err)
	// }

	// for _, target := range *cfg.Targets {
	// 	c := ssh.Credential{
	// 		User: target.User,
	// 		Host: target.Host,
	// 		Port: target.Port,
	// 	}
	// 	ssh.Run("mkdir ~/hellox/", []ssh.Credential{c})
	// }
}

func execCMD(cmdLine string) {
	c := strings.Split(cmdLine, " ")
	var args []string
	for i := 1; i < len(c); i++ {
		args = append(args, c[i])
	}

	cmd := exec.Command(c[0], args...)
	stdout, err := cmd.Output()

	if err != nil {
		logrus.WithError(err).Error("Cannot run command: ", cmdLine)
		return
	}

	logrus.Info(string(stdout))
}

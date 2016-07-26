package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/dwarvesf/shot/config"
	"github.com/dwarvesf/shot/dflog"
	"github.com/dwarvesf/shot/ssh"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

// Usage
// $ shot setup --file=feature__login.yml  // --> /opt/shot/port
// $ shot deploy --file=feature__login.yml
// $ shot down --file=feature__login.yml

var (
	l = dflog.New()

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
		Setup(*setupPath)

	case deploy.FullCommand():
		Deploy(*deployPath)

	case down.FullCommand():
		Down(*downPath)

	default:
		l.Error("Command not found.")
	}
}

// Setup creates needed files in target servers
func Setup(configFile string) {

	cfg, err := config.Init(configFile)
	if err != nil {
		l.WithError(err).Fatal("Configuration file not found")
	}

	for _, target := range cfg.Targets {
		c := ssh.Credential{
			User: target.User,
			Host: target.Host,
			Port: target.Port,
		}

		// Silently create log file
		_, err = ssh.Run(`touch /var/log/shot.log || exit`, c)
		if err != nil {
			l.Error(err)
			continue
		}

		// Check if available port file is existed or not
		res, err := ssh.Run(`if test -f "/opt/shot/port"; then echo "Found";fi`, c)
		if err != nil {
			l.Error(err)
		}

		if strings.TrimSpace(res) == "Found" {
			l.Warn("Skipped. Port file already existed.")
			continue
		}

		// res, err = ssh.Run(`mkdir -p /opt/shot && touch /opt/shot/port && echo "8900" > /opt/shot/port || exit`, c)
		res, err = ssh.Run(`echo 8900 > /opt/shot/port || exit`, c)
		if err != nil {
			l.Error(err)
		}
	}
}

// Deploy
func Deploy(configFile string) {

	cfg, err := config.Init(configFile)
	if err != nil {
		l.WithError(err).Fatal("Configuration file not found")
	}

	for _, t := range cfg.Targets {
		for _, b := range t.Branches {

			gco := fmt.Sprintf("git checkout %s", b)
			imageName := fmt.Sprintf("%s/%s:%s", cfg.Registry, cfg.Project.Name, strings.Replace(b, "/", "-", -1))
			dockerBuildCmd := fmt.Sprintf("docker build -t %s .", imageName)
			dockerPushCmd := fmt.Sprintf("docker push %s", imageName)

			// Dockerize all containers
			cmds := []string{gco, dockerBuildCmd, dockerPushCmd}
			for _, cmd := range cmds {
				_, _ = execCmd(cmd)
			}

			c := ssh.Credential{
				User: t.User,
				Host: t.Host,
				Port: t.Port,
			}
			availablePort, err := ssh.Run("cat /opt/shot/port", c)
			if err != nil {
				l.WithError(err).Error("Cannot read file port from server")
				continue
			}

			port, err := strconv.Atoi(strings.TrimSpace(availablePort))
			if err != nil {
				l.WithError(err).Error("Cannot convert port to int")
				continue
			}

			// Pull and run containers
			dockerPullCmd := fmt.Sprintf("docker pull %s", imageName)
			dockerRunCmd := fmt.Sprintf("docker run -d -p %s:%d %s", availablePort, cfg.Project.Port, imageName)
			cmds = []string{dockerPullCmd, dockerRunCmd}
			for _, cmd := range cmds {
				_, err := ssh.Run(cmd, ssh.Credential{
					User: t.User,
					Host: t.Host,
					Port: t.Port,
				})
				if err != nil {
					l.WithError(err).Error("Cannot run command on target server")
					break
				}
			}

			// Increase port count and replace to the file
			port = port + 1

			// Copy new port to target and remove file port from localhost
			p := []byte(strconv.Itoa(port + 1))
			err = ioutil.WriteFile("port", p, 0644)
			if err != nil {
				l.WithError(err).Error("Cannot write port to file")
				continue
			}

			_, _ = execCmd(fmt.Sprintf("scp port %s@%s:/opt/shot/port", t.User, t.Host))
			err = os.Remove("port")
			if err != nil {
				l.WithError(err).Error("Cannot remove file port")
			}
		}
	}
}

func Down(configFile string) {

	cfg, err := config.Init(configFile)
	if err != nil {
		l.WithError(err).Fatal("Configuration file not found")
	}

	for _, target := range cfg.Targets {
		for _, branch := range target.Branches {

			c := ssh.Credential{
				User: target.User,
				Host: target.Host,
				Port: target.Port,
			}

			// Remove related docker containers
			dockerRemoveCmd := fmt.Sprintf("docker rm -f $(docker ps -a | grep %s)", branch)
			_, err := ssh.Run(dockerRemoveCmd, c)
			if err != nil {
				l.WithError(err).Error("Cannot execute commands")
			}

			// Send notification
			sendEmail()

			// Post to Slack
			postToSlack()
		}
	}
}

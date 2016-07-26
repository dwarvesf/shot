package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/dwarvesf/shot/config"
	"github.com/dwarvesf/shot/dflog"
	"github.com/dwarvesf/shot/ssh"
	"github.com/dwarvesf/shot/utils"
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

// Deploy ...
func Deploy(configFile string) {
	cfg, err := config.Init(configFile)
	if err != nil {
		l.WithError(err).Fatal("Configuration file not found")
	}

	for _, t := range cfg.Targets {
		lf := dflog.Fields{"target": t.Host}
		c := ssh.Credential{
			User: t.User,
			Host: t.Host,
			Port: t.Port,
		}
		availablePort, err := ssh.Run("cat /opt/shot/port", c)
		if err != nil {
			l.WithFields(lf).Error(err)
		}
		port, err := strconv.Atoi(strings.TrimSpace(availablePort))
		if err != nil {
			l.WithFields(lf).WithError(err).Error("Cannot read port from file /opt/shot/port on server")
			continue
		}

		checkContainerExists := fmt.Sprintf("docker ps -a | awk '{ print $1,$2 }' | grep %s/%s | awk '{print $2 }'", cfg.Registry, cfg.Project.Name)
		res, err := ssh.Run(checkContainerExists, c)
		if err != nil {
			l.WithFields(lf).WithError(err).Error("Cannot execute commands")
		}
		if res == "" {
			// this means no container is running, reset port to 8900
			_, err := ssh.Run(`echo 8900 > /opt/shot/port || exit`, c)
			if err != nil {
				l.WithFields(lf).WithError(err).Error("Cannot write into /opt/shot/port")
			}
			port = 8900
		}

		for _, b := range t.Branches {
			lf := dflog.Fields{"target": t.Host, "branch": b}
			gco := fmt.Sprintf("git checkout %s", b)
			imageName := fmt.Sprintf("%s/%s:%s", cfg.Registry, cfg.Project.Name, strings.Replace(b, "/", "-", -1))
			dockerBuildCmd := fmt.Sprintf("docker build -t %s .", imageName)
			dockerPushCmd := fmt.Sprintf("docker push %s", imageName)

			// Dockerize all containers
			cmds := []string{gco, dockerBuildCmd, dockerPushCmd}
			var err error
			for _, cmd := range cmds {
				_, err = utils.ExecCmd(cmd)
				if err != nil {
					l.WithError(err).Error("Cannot run command: ", cmd)
					break
				}
			}
			if err != nil {
				l.WithFields(lf).WithError(err).Error("Cannot continue due to unexpected err")
				continue
			}

			// Pull and run containers
			dockerPullCmd := fmt.Sprintf("docker pull %s", imageName)
			dockerRunCmd := fmt.Sprintf("docker run -d -p %d:%d --name %s %s", port, cfg.Project.Port, fmt.Sprintf("%s-%s", strings.Replace(cfg.Project.Name, "/", "-", -1), strings.Replace(b, "/", "-", -1)), imageName)
			cmds = []string{dockerPullCmd, dockerRunCmd}
			var cErr error
			for _, cmd := range cmds {
				_, cErr = ssh.Run(cmd, ssh.Credential{
					User: t.User,
					Host: t.Host,
					Port: t.Port,
				})
				if cErr != nil {
					l.WithError(cErr).Error("Cannot run command on target server")
					break
				}
			}
			if cErr != nil {
				l.WithFields(lf).WithError(cErr).Error("Cannot run docker due to unexpected err")
				continue
			}

			// Rewrite port into file
			port = port + 1
			_, err = ssh.Run(fmt.Sprintf(`echo %d > /opt/shot/port || exit`, port), c)
			if err != nil {
				l.WithFields(lf).WithError(err).Error("Cannot rewrite port into /opt/shot/port on server")
			}
		}
	}
}

// Down ...
func Down(configFile string) {

	cfg, err := config.Init(configFile)
	if err != nil {
		l.WithError(err).Fatal("Configuration file not found")
	}

	for _, target := range cfg.Targets {
		for _, branch := range target.Branches {

			// Remove related docker containers
			dockerRemoveCmd := fmt.Sprintf("docker rm -f $(docker ps -a | grep %s)", branch)
			_, err := ssh.Run(dockerRemoveCmd, ssh.Credential{
				User: target.User,
				Host: target.Host,
				Port: target.Port,
			})
			if err != nil {
				l.WithError(err).Error("Cannot execute commands")
			}

			// Send notification
			if cfg.Notification.Email.Enable {
				for _, r := range cfg.Notification.Email.Recipients {
					utils.SendEmail(r)
				}
			}

			// Post to Slack
			if cfg.Notification.Slack.Enable {
				for _, c := range cfg.Notification.Slack.Channels {
					utils.PostToSlack(c)
				}
			}
		}
	}
}

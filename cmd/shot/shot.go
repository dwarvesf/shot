package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

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

	app   = kingpin.New("shot", "Automation deployment inside the fortress")
	debug = app.Flag("debug", "enable debug mode").Default("false").Short('d').Bool()

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
		setDebugMode()
		Setup(*setupPath)

	case deploy.FullCommand():
		setDebugMode()
		Deploy(*deployPath)

	case down.FullCommand():
		setDebugMode()
		Down(*downPath)

	default:
		l.Error("Command not found.")
	}
}

// Setup creates needed files in target servers
func Setup(configFile string) {
	cfg, err := config.Init(configFile)
	if err != nil {
		l.Log(dflog.FatalLevel, "Configuration file not found", err, nil)
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
			l.Log(dflog.ErrorLevel, "Cannot run command on server", err, nil)
			continue
		}

		// Check if available port file is existed or not
		res, err := ssh.Run(`if test -f "/opt/shot/port"; then echo "Found";fi`, c)
		if err != nil {
			l.Log(dflog.ErrorLevel, "Cannot run command on server", err, nil)
			continue
		}

		if strings.TrimSpace(res) == "Found" {
			l.Log(dflog.WarnLevel, "Skipped. Port file already existed.", nil, nil)
			continue
		}

		res, err = ssh.Run(`mkdir -p /opt/shot/ && touch /opt/shot/port && echo 8900 > /opt/shot/port || exit`, c)
		if err != nil {
			l.Log(dflog.ErrorLevel, "Cannot run command on server", err, nil)
		}
	}
}

// Deploy ...
func Deploy(configFile string) {
	cfg, err := config.Init(configFile)
	if err != nil {
		l.Log(dflog.FatalLevel, "Configuration file not found", err, nil)
	}

	var wgT sync.WaitGroup
	wgT.Add(len(cfg.Targets))
	for _, e := range cfg.Targets {
		t := e
		go func() {
			defer wgT.Done()
			lf := dflog.Fields{"target": t.Host}
			c := ssh.Credential{
				User: t.User,
				Host: t.Host,
				Port: t.Port,
			}

			// Check if available port file is existed or not
			res, err := ssh.Run(`if test -f "/opt/shot/port"; then echo "Found";fi`, c)
			if err != nil {
				l.Log(dflog.ErrorLevel, "Cannot run command on server", err, nil)
				return
			}
			if strings.TrimSpace(res) != "Found" {
				l.Log(dflog.ErrorLevel, "Cannot read port from file /opt/shot/port on server", nil, nil)
				return
			}

			availablePort, err := ssh.Run("cat /opt/shot/port", c)
			if err != nil {
				l.Log(dflog.ErrorLevel, "Cannot run command on server", err, nil)
				return
			}
			port, err := strconv.Atoi(strings.TrimSpace(availablePort))
			if err != nil {
				l.Log(dflog.ErrorLevel, "Cannot read port from server", err, nil)
				return
			}

			checkContainerExists := fmt.Sprintf(`docker ps -a --filter="name=%s__%s" -q`, cfg.Project.Name, cfg.Registry)
			res, err = ssh.Run(checkContainerExists, c)
			if err != nil {
				l.Log(dflog.ErrorLevel, "Cannot execute commands", err, lf)
			}
			if res == "" {
				// this means no container is running, reset port to 8900
				_, err := ssh.Run(`echo 8900 > /opt/shot/port || exit`, c)
				if err != nil {
					l.Log(dflog.ErrorLevel, "Cannot write into /opt/shot/port", err, lf)
				} else {
					port = 8900
				}
			}

			var wgB sync.WaitGroup
			wgB.Add(len(t.Branches))
			for _, v := range t.Branches {
				b := v
				go func() {
					defer wgB.Done()
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
							l.Log(dflog.ErrorLevel, fmt.Sprintf("Cannot run command: %s", cmd), err, lf)
							break
						}
					}
					if err != nil {
						l.Log(dflog.ErrorLevel, "Cannot continue deploy due to unexpected error", err, lf)
						return
					}

					// Pull and run containers
					dockerPullCmd := fmt.Sprintf("docker pull %s", imageName)
					dockerRunCmd := fmt.Sprintf("docker run -d -p %d:%d --name %s %s", port, cfg.Project.Port, fmt.Sprintf("%s__%s", strings.Replace(cfg.Project.Name, "/", "-", -1), strings.Replace(b, "/", "-", -1)), imageName)
					cmds = []string{dockerPullCmd, dockerRunCmd}
					var cErr error
					for _, cmd := range cmds {
						res, cErr = ssh.Run(cmd, ssh.Credential{
							User: t.User,
							Host: t.Host,
							Port: t.Port,
						})
						if cErr != nil {
							l.Log(dflog.ErrorLevel, "Cannot run command on server", cErr, lf)
							break
						}
						if strings.Contains(res, "docker: Error response from daemon") {
							cErr = errors.New(res)
							l.Log(dflog.ErrorLevel, "Cannot run command on server", cErr, lf)
							break
						}
					}
					if cErr != nil {
						l.Log(dflog.ErrorLevel, "Cannot use 'docker run' due to unexpected error", cErr, lf)
						return
					}

					// Send notification
					message := fmt.Sprintf("Deployed (%s:%s) to server %s:%d", cfg.Project.Name, b, t.Host, port)
					if cfg.Notification.Email.Enable {
						var wgM sync.WaitGroup
						wgM.Add(len(cfg.Notification.Email.Recipients))
						for _, v := range cfg.Notification.Email.Recipients {
							r := v
							go func() {
								defer wgM.Done()
								l.Info("Sending mail to ", r)
								err := utils.SendMail(r, fmt.Sprintf("Deployed %s to server with PR %s", cfg.Project.Name, b), message, cfg)
								if err != nil {
									l.Log(dflog.ErrorLevel, fmt.Sprintf("Cannot send mail to %s", r), err, lf)
								}
							}()
						}
						wgM.Wait()
					}

					// Post to Slack
					if cfg.Notification.Slack.Enable {
						var wgS sync.WaitGroup
						wgS.Add(len(cfg.Notification.Slack.Channels))
						for _, v := range cfg.Notification.Slack.Channels {
							c := v
							go func() {
								defer wgS.Done()
								l.Info("Posting to Slack channel ", c)
								err := utils.PostToSlack(c, message)
								if err != nil {
									l.Log(dflog.ErrorLevel, fmt.Sprintf("Cannot post to channel %s", c), err, lf)
								}
							}()
						}
						wgS.Wait()
					}

					// Rewrite port into file
					port = port + 1
					_, err = ssh.Run(fmt.Sprintf(`echo %d > /opt/shot/port || exit`, port), c)
					if err != nil {
						l.Log(dflog.ErrorLevel, "Cannot rewrite port into /opt/shot/port on server", err, lf)
					}
				}()
			}
			wgB.Wait()
		}()
	}
	wgT.Wait()
	l.Log(dflog.InfoLevel, "Done", nil, nil)
}

// Down ...
func Down(configFile string) {
	cfg, err := config.Init(configFile)
	if err != nil {
		l.Log(dflog.FatalLevel, "Configuration file not found", err, nil)
	}

	var wgT sync.WaitGroup
	wgT.Add(len(cfg.Targets))
	for _, e := range cfg.Targets {
		t := e
		go func() {
			defer wgT.Done()
			var wgB sync.WaitGroup
			wgB.Add(len(t.Branches))
			for _, v := range t.Branches {
				b := v
				go func() {
					defer wgB.Done()
					lf := dflog.Fields{"target": t.Host, "branch": b}
					// Remove related docker containers
					containerName := fmt.Sprintf("%s__%s", strings.Replace(cfg.Project.Name, "/", "-", -1), strings.Replace(b, "/", "-", -1))
					dockerRemoveCmd := fmt.Sprintf(`docker rm -f $(docker ps -a --filter="name=%s" -q)`, containerName)
					_, err := ssh.Run(dockerRemoveCmd, ssh.Credential{
						User: t.User,
						Host: t.Host,
						Port: t.Port,
					})
					if err != nil {
						l.Log(dflog.ErrorLevel, "Cannot execute commands", err, lf)
					}

					// Send notification
					// Send mail
					message := fmt.Sprintf("Shutdown (%s:%s) from server %s", cfg.Project.Name, b, t.Host)
					if cfg.Notification.Email.Enable {
						var wgM sync.WaitGroup
						wgM.Add(len(cfg.Notification.Email.Recipients))
						for _, v := range cfg.Notification.Email.Recipients {
							r := v
							go func() {
								defer wgM.Done()
								l.Info("Sending mail to ", r)
								err := utils.SendMail(r, fmt.Sprintf("Shutdown (%s:%s) from server %s", cfg.Project.Name, b, t.Host), message, cfg)
								if err != nil {
									l.Log(dflog.ErrorLevel, fmt.Sprintf("Cannot send mail to %s", r), err, lf)
								}
							}()
						}
						wgM.Wait()
					}

					// Post to Slack
					if cfg.Notification.Slack.Enable {
						var wgS sync.WaitGroup
						wgS.Add(len(cfg.Notification.Slack.Channels))
						for _, v := range cfg.Notification.Slack.Channels {
							c := v
							go func() {
								defer wgS.Done()
								l.Info("Posting to Slack channel ", c)
								err := utils.PostToSlack(c, message)
								if err != nil {
									l.Log(dflog.ErrorLevel, fmt.Sprintf("Cannot post to channel %s", c), err, lf)
								}
							}()
						}
						wgS.Wait()
					}
				}()
			}
			wgB.Wait()
		}()
	}
	wgT.Wait()
	l.Log(dflog.InfoLevel, "Done", nil, nil)
}

func setDebugMode() {
	if *debug {
		l.DebugMode = true
	}
}

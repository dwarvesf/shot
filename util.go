package main

import (
	"os/exec"
	"strings"
)

// execCmd receives cmdLine as input and helps to run it in shell env
func execCmd(cmdLine string) (string, error) {
	c := strings.Split(cmdLine, " ")
	var args []string
	for i := 1; i < len(c); i++ {
		args = append(args, c[i])
	}

	cmd := exec.Command(c[0], args...)
	stdout, err := cmd.Output()

	if err != nil {
		l.WithError(err).Error("Cannot run command: ", cmdLine)
	}

	l.Info(string(stdout))

	return string(stdout), err
}

// sendEmail will help to send an email to recepients
func sendEmail(to string) {

}

// postToSlack will help to send notification messages to Slack channel
func postToSlack(channel string) {

}

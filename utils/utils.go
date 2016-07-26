package utils

import (
	"os/exec"
	"strings"

	"github.com/dwarvesf/shot/dflog"
)

var l = dflog.New()

// ExecCmd receives cmdLine as input and helps to run it in shell env
func ExecCmd(cmdLine string) (string, error) {
	c := strings.Split(cmdLine, " ")
	var args []string
	for i := 1; i < len(c); i++ {
		args = append(args, c[i])
	}

	cmd := exec.Command(c[0], args...)
	stdout, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(stdout), err
}

// SendEmail will help to send an email to recepients
func SendEmail(to string) {

}

// PostToSlack will help to send notification messages to Slack channel
func PostToSlack(channel string) {

}

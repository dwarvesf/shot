package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"os/exec"
	"strings"

	"github.com/dwarvesf/shot/config"
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

// PostToSlack will help to send notification messages to Slack channel
func PostToSlack(channel, text string) error {
	mJSON, err := json.Marshal(map[string]interface{}{
		"text": text,
	})
	if err != nil {
		return err
	}

	body := bytes.NewReader(mJSON)
	req, err := http.NewRequest("POST", channel, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	_, err = client.Do(req)
	if err != nil {
		return err
	}

	return nil
}

// SendMail will help to send an email to recepients
func SendMail(to, subject, body string, config *config.Config) error {
	msg := "From: " + config.Notification.Email.SMTP.User + "\n" +
		"To: " + to + "\n" +
		"Subject: " + subject + "\n\n" +
		body

	err := smtp.SendMail(fmt.Sprintf("%s:%d", config.Notification.Email.SMTP.Host, config.Notification.Email.SMTP.Port),
		smtp.PlainAuth("", config.Notification.Email.SMTP.User, config.Notification.Email.SMTP.Pass, config.Notification.Email.SMTP.Host),
		config.Notification.Email.SMTP.User, []string{to}, []byte(msg))

	if err != nil {
		return err
	}

	return nil
}

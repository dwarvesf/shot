package ssh

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// Credential ...
type Credential struct {
	User string
	Host string
	Port int
}

func executeCmd(cmd string, hostname string, port int, config *ssh.ClientConfig) (string, error) {
	logrus.WithField("func", "ssh.executeCmd").Info("connecting to server ", hostname)
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", hostname, port), config)
	if err != nil {
		return "", err
	}

	session, _ := conn.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	logrus.Info(hostname + ": " + cmd)
	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Run(cmd)

	if len(stdoutBuf.String()) != 0 {
		logrus.Info(fmt.Sprintf(hostname + ": " + stdoutBuf.String()))
	}

	return stdoutBuf.String(), nil
}

// Run executes shell commands or given host
func Run(command string, c Credential) (string, error) {
	config, err := ClientConfig(c)
	if err != nil {
		return "", err
	}

	// Exec commands
	response, err := executeCmd(command, c.Host, c.Port, config)
	if err != nil {
		return "", err
	}

	return response, nil
}

// ClientConfig ...
func ClientConfig(c Credential) (*ssh.ClientConfig, error) {
	// Get SSH key
	key, err := ioutil.ReadFile(os.Getenv("HOME") + "/.ssh/id_rsa")
	if err != nil {
		return nil, err
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User: c.User,
		Auth: []ssh.AuthMethod{
			// Use the PublicKeys method for remote authentication.
			ssh.PublicKeys(signer),
		},
	}, nil
}

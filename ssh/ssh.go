package ssh

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type Credential struct {
	User string
	Host string
	Port int
}

func executeCmd(cmd, hostname string, port int, config *ssh.ClientConfig) {
	conn, _ := ssh.Dial("tcp", fmt.Sprintf("%s:%d", hostname, port), config)
	session, _ := conn.NewSession()
	defer session.Close()

	logrus.Info(hostname + ": " + cmd)
	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Run(cmd)

	if len(stdoutBuf.String()) != 0 {
		fmt.Sprintf(hostname + ": " + stdoutBuf.String())
	}
}

// Run executes shell commands or given host
func Run(command string, credentials []Credential) {

	// Get SSH key
	key, err := ioutil.ReadFile(os.Getenv("HOME") + "/.ssh/id_rsa")
	if err != nil {
		logrus.Error(err)
		return
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		logrus.Error(err)
		return
	}

	for _, c := range credentials {

		config := &ssh.ClientConfig{
			User: c.User,
			Auth: []ssh.AuthMethod{
				// Use the PublicKeys method for remote authentication.
				ssh.PublicKeys(signer),
			},
		}

		// Exec commands
		executeCmd(command, c.Host, c.Port, config)
	}
}

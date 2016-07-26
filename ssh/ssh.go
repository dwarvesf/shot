package ssh

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/dwarvesf/shot/dflog"
	"golang.org/x/crypto/ssh"
)

var l = dflog.New()

// Credential ...
type Credential struct {
	User string
	Host string
	Port int
}

func executeCmd(cmd string, hostname string, port int, config *ssh.ClientConfig) (string, error) {
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", hostname, port), config)
	if err != nil {
		return "", err
	}

	session, _ := conn.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	_ = session.Run(cmd)

	return stdoutBuf.String(), nil
}

// 2015-06-10 20:10:08.123456
func getTime() string {
	var buf [30]byte
	b := buf[:0]
	t := time.Now()
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	nsec := t.Nanosecond()

	itoa(&b, year, 4)
	b = append(b, '-')
	itoa(&b, int(month), 2)
	b = append(b, '-')
	itoa(&b, day, 2)
	b = append(b, ' ')
	itoa(&b, hour, 2)
	b = append(b, ':')
	itoa(&b, min, 2)
	b = append(b, ':')
	itoa(&b, sec, 2)
	b = append(b, '.')
	itoa(&b, nsec/1e3, 6)

	return string(b)
}

// Taken from stdlib "log".
//
// Cheap integer to fixed-width decimal ASCII.  Give a negative width to avoid zero-padding.
func itoa(buf *[]byte, i int, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

// Run executes shell commands or given host
func Run(command string, c Credential) (string, error) {
	config, err := ClientConfig(c)
	if err != nil {
		return "", err
	}

	// Attemp to write command to log file
	logCmd := fmt.Sprintf(`echo %s: "%s" >> /var/log/shot.log`, getTime(), command)
	_, _ = executeCmd(logCmd, c.Host, c.Port, config)

	// Exec commands and write its output to log file
	l.Info(c.Host + ": " + command)
	response, err := executeCmd(command+" 2>&1 | tee -a var/log/shot.log", c.Host, c.Port, config)
	if err != nil {
		return "", err
	}

	// Print out response
	if len(response) != 0 {
		l.Info(fmt.Sprintf(c.Host + ": " + strings.TrimSpace(response)))
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

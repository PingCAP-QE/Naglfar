package ssh

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	*ssh.Client
}

func NewSSHClient(username string, keyPath string, host string, port int) (*Client, error) {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			func() ssh.AuthMethod {
				key, err := ioutil.ReadFile(keyPath)
				if err != nil {
					panic(err)
				}
				signer, err := ssh.ParsePrivateKey(key)
				if err != nil {
					panic(err)
				}
				return ssh.PublicKeys(signer)
			}(),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		return nil, err
	}
	return &Client{conn}, nil
}

func (c *Client) RunCommand(command string) (stdout string, stderr string, err error) {
	session, err := c.NewSession()
	if err != nil {
		return "", "", err
	}
	defer session.Close()
	var bStd bytes.Buffer
	var bErr bytes.Buffer
	session.Stdout = &bStd
	session.Stderr = &bErr
	session.Run(command)
	return bStd.String(), bErr.String(), err
}

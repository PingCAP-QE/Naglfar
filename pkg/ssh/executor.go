package ssh

import (
	"strconv"
	"time"

	"github.com/appleboy/easyssh-proxy"
)

func MakeSSHConfig(username string,
	password string,
	host string,
	port int,
	timeout time.Duration) *easyssh.MakeConfig {

	return &easyssh.MakeConfig{
		User:     username,
		Password: password,
		Server:   host,
		Port:     strconv.Itoa(port),
		Timeout:  timeout,
	}
}

func MakeSSHKeyConfig(username string,
	keyPath string,
	host string,
	port int,
) *easyssh.MakeConfig {
	return &easyssh.MakeConfig{
		User:    username,
		Server:  host,
		KeyPath: keyPath,
		Port:    strconv.Itoa(port),
	}
}

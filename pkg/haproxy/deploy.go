package haproxy

import (
	"net/http"
	"strconv"
	"strings"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	"github.com/PingCAP-QE/Naglfar/pkg/container"
)

const (
	BaseImageName = "docker.io/library/haproxy:"
	SourceMount   = "/var/naglfar/lib/haproxy.cfg"
	TargetMount   = "/usr/local/etc/haproxy/haproxy.cfg"
	FileName      = "haproxy.cfg"
)

func generateFrontend(port int) string {
	return "\nfrontend haproxy\n" +
		"\tbind *:" + strconv.Itoa(port) + "\n" +
		"\tdefault_backend tidbs\n"
}

func generateBackend(tidbs []string) string {
	backend := "\nbackend tidbs\n"
	for i := 0; i < len(tidbs); i++ {
		backend += "\tserver tidb" + strconv.Itoa(i) + " " + tidbs[i] + " maxconn 64" + "\n"
	}
	return backend
}

func GenerateHAProxyConfig(tct *naglfarv1.TiDBCluster, clusterIPMaps map[string]string) (bool, string) {
	config := tct.HAProxy.Config + "\n"
	config += generateFrontend(tct.HAProxy.Port)
	var tidbs []string
	for i := 0; i < len(tct.TiDB); i++ {
		var port int
		if tct.TiDB[i].Port == 0 {
			port = 4000
		} else {
			port = tct.TiDB[i].Port
		}
		ip := clusterIPMaps[tct.TiDB[i].Host]
		if ip == "" {
			return true, ""
		}
		tidbs = append(tidbs, ip+":"+strconv.Itoa(port))
	}
	config += generateBackend(tidbs)
	return false, config
}

func WriteConfigToMachine(machine string, name string, config string) error {
	req, err := http.NewRequest("POST", "http://"+machine+":"+strconv.Itoa(container.UploadDaemonExternalPort)+"/upload", strings.NewReader(config))
	if err != nil {
		return err
	}
	req.Header.Set("fileName", name)
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func DeleteConfigFromMachine(machine string, name string) error {
	req, err := http.NewRequest("POST", "http://"+machine+":"+strconv.Itoa(container.UploadDaemonExternalPort)+"/delete", nil)
	if err != nil {
		return err
	}
	req.Header.Set("fileName", name)
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

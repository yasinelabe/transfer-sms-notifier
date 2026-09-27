package oracle

import (
	"fmt"
	"strconv"
	"strings"

	go_ora "github.com/sijms/go-ora/v2"
)

// buildDSN parses ORACLE_CONN_STRING (host:port/service, same as autosupport) into a go-ora URL.
func buildDSN(user, password, connString string) (string, error) {
	connString = strings.TrimSpace(connString)
	if strings.HasPrefix(connString, "oracle://") {
		return connString, nil
	}

	host, port, service, err := parseConnString(connString)
	if err != nil {
		return "", err
	}
	return go_ora.BuildUrl(host, port, service, user, password, nil), nil
}

func parseConnString(connString string) (host string, port int, service string, err error) {
	if connString == "" {
		return "", 0, "", fmt.Errorf("empty connection string")
	}
	slash := strings.Index(connString, "/")
	if slash <= 0 {
		return "", 0, "", fmt.Errorf("connection string must be host:port/service")
	}
	service = strings.TrimSpace(connString[slash+1:])
	hostPort := strings.TrimSpace(connString[:slash])

	colon := strings.LastIndex(hostPort, ":")
	if colon <= 0 {
		host = hostPort
		port = 1521
	} else {
		host = hostPort[:colon]
		port, err = strconv.Atoi(hostPort[colon+1:])
		if err != nil || port <= 0 {
			return "", 0, "", fmt.Errorf("invalid port in %q", connString)
		}
	}
	if host == "" || service == "" {
		return "", 0, "", fmt.Errorf("invalid connection string %q", connString)
	}
	return host, port, service, nil
}

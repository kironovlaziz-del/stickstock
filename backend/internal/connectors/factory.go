package connectors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func New(kind string, dsn string) (Connector, error) {
	switch Kind(kind) {
	case KindPostgres, KindFile:
		return NewPostgresConnector(dsn)
	case KindMySQL:
		return NewMySQLConnector(dsn)
	case KindMongoDB:
		return NewMongoConnector(dsn)
	case KindREST:
		return NewRESTConnector(dsn)
	default:
		return nil, fmt.Errorf("connector kind %q is not implemented yet", kind)
	}
}

func IsSQLKind(kind string) bool {
	switch Kind(kind) {
	case KindPostgres, KindMySQL, KindFile:
		return true
	default:
		return false
	}
}

// NewWithSSH creates a connector with SSH tunnel support.
// If sshHost is empty, behaves like New.
func NewWithSSH(kind, dsn string, sshHost string, sshPort int, sshUser, sshPassword, sshPrivateKey string) (Connector, error) {
	if sshHost == "" || sshUser == "" {
		return New(kind, dsn)
	}

	remoteHost, remotePort, err := parseDSN(kind, dsn)
	if err != nil {
		return nil, fmt.Errorf("parse DSN: %w", err)
	}

	localPort, err := establishTunnel(sshHost, sshPort, sshUser, sshPassword, sshPrivateKey, remoteHost, remotePort)
	if err != nil {
		return nil, fmt.Errorf("establish tunnel: %w", err)
	}
	if localPort == 0 {
		return New(kind, dsn)
	}
	newDSN, err := replaceHostPort(dsn, "localhost", localPort)
	if err != nil {
		return nil, fmt.Errorf("replace DSN host/port: %w", err)
	}
	return New(kind, newDSN)
}

func establishTunnel(host string, port int, user, password, privateKey, remoteHost string, remotePort int) (int, error) {
	payload := map[string]interface{}{
		"host":        host,
		"port":        port,
		"user":        user,
		"password":    password,
		"private_key": privateKey,
		"remote_host": remoteHost,
		"remote_port": remotePort,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post("http://ssh-tunnel-service:8081/tunnel", "application/json", bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("SSH tunnel service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		return 0, fmt.Errorf("tunnel service error: %s", errResp["error"])
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("invalid response from tunnel service")
	}
	if portVal, ok := result["local_port"].(float64); ok {
		return int(portVal), nil
	}
	return 0, fmt.Errorf("local_port not returned")
}

func parseDSN(kind, dsn string) (string, int, error) {
	switch kind {
	case "postgres", "mysql", "clickhouse":
		u, err := url.Parse(dsn)
		if err != nil {
			return "", 0, err
		}
		host := u.Hostname()
		portStr := u.Port()
		if portStr == "" {
			switch kind {
			case "postgres":
				portStr = "5432"
			case "mysql":
				portStr = "3306"
			default:
				portStr = "9000"
			}
		}
		port, _ := strconv.Atoi(portStr)
		return host, port, nil
	case "mongodb":
		u, err := url.Parse(dsn)
		if err != nil {
			return "", 0, err
		}
		host := u.Hostname()
		portStr := u.Port()
		if portStr == "" {
			portStr = "27017"
		}
		port, _ := strconv.Atoi(portStr)
		return host, port, nil
	default:
		return "", 0, fmt.Errorf("unsupported kind for SSH: %s", kind)
	}
}

func replaceHostPort(dsn, newHost string, newPort int) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	u.Host = newHost + ":" + strconv.Itoa(newPort)
	return u.String(), nil
}

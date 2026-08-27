package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
)

type TunnelRequest struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
	RemoteHost string `json:"remote_host"`
	RemotePort int    `json:"remote_port"`
}

func main() {
	http.HandleFunc("/tunnel", handleTunnel)
	log.Println("SSH Tunnel Service listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func handleTunnel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req TunnelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Host == "" || req.User == "" || req.RemoteHost == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	// Устанавливаем SSH-туннель
	closeFunc, err := startTunnel(req)
	if err != nil {
		http.Error(w, "tunnel failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer closeFunc()

	// Отвечаем, что туннель установлен
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "tunnel established"})
}

func startTunnel(req TunnelRequest) (func() error, error) {
	var authMethods []ssh.AuthMethod
	if req.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(req.PrivateKey))
		if err != nil {
			return nil, err
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}
	if req.Password != "" {
		authMethods = append(authMethods, ssh.Password(req.Password))
	}
	if len(authMethods) == 0 {
		return nil, nil
	}
	cfg := &ssh.ClientConfig{
		User:            req.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	addr := net.JoinHostPort(req.Host, strconv.Itoa(req.Port))
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, err
	}
	// Пробрасываем локальный порт (выбираем случайный)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		client.Close()
		return nil, err
	}
	localPort := listener.Addr().(*net.TCPAddr).Port
	log.Printf("Tunnel established: localhost:%d -> %s:%d", localPort, req.RemoteHost, req.RemotePort)

	done := make(chan struct{})
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					continue
				}
			}
			go handleConn(conn, client, req.RemoteHost, req.RemotePort)
		}
	}()
	closeFunc := func() error {
		listener.Close()
		client.Close()
		<-done
		return nil
	}
	return closeFunc, nil
}

func handleConn(local net.Conn, client *ssh.Client, remoteHost string, remotePort int) {
	defer local.Close()
	remote, err := client.Dial("tcp", net.JoinHostPort(remoteHost, strconv.Itoa(remotePort)))
	if err != nil {
		return
	}
	defer remote.Close()
	go io.Copy(remote, local)
	io.Copy(local, remote)
}

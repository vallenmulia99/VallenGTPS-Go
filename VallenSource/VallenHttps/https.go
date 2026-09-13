package https

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type ServerData struct {
	Server   string
	Port     int
	Type     int
	Type2    int
	Maint    string
	LoginURL string
	Meta     string
}

type HTTPSServer struct {
	port       int
	serverData *ServerData
}

func NewHTTPSServer(port int) *HTTPSServer {
	return &HTTPSServer{
		port: port,
		serverData: &ServerData{
			Server:   "127.0.0.1",
			Port:     17091,
			Type:     1,
			Type2:    1,
			Maint:    "Server under maintenance. Please try again later.",
			LoginURL: "",
			Meta:     "gurotopia",
		},
	}
}

func (s *HTTPSServer) LoadServerData() error {
	file, err := os.Open("VallenSetting/server_data.php")
	if err != nil {
		return s.createDefaultConfig()
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "RTENDMARKERBS1001" {
			continue
		}

		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "server":
			s.serverData.Server = value
		case "port":
			if port, err := strconv.Atoi(value); err == nil {
				s.serverData.Port = port
			}
		case "type":
			if t, err := strconv.Atoi(value); err == nil {
				s.serverData.Type = t
			}
		case "type2":
			if t, err := strconv.Atoi(value); err == nil {
				s.serverData.Type2 = t
			}
		case "maint", "#maint":
			s.serverData.Maint = value
		case "loginurl":
			s.serverData.LoginURL = value
		case "meta":
			s.serverData.Meta = value
		}
	}

	log.Printf("[HTTPS] Loaded server_data.php: %s:%d (loginurl: %s)", s.serverData.Server, s.serverData.Port, s.serverData.LoginURL)
	return scanner.Err()
}

func (s *HTTPSServer) createDefaultConfig() error {
	content := fmt.Sprintf(
		"server|%s\n"+
			"port|%d\n"+
			"type|%d\n"+
			"type2|%d\n"+
			"#maint|%s\n"+
			"loginurl|%s\n"+
			"meta|%s\n"+
			"RTENDMARKERBS1001",
		s.serverData.Server,
		s.serverData.Port,
		s.serverData.Type,
		s.serverData.Type2,
		s.serverData.Maint,
		s.serverData.LoginURL,
		s.serverData.Meta,
	)

	return os.WriteFile("VallenSetting/server_data.php", []byte(content), 0644)
}

func (s *HTTPSServer) buildResponse() string {
	content := fmt.Sprintf(
		"server|%s\n"+
			"port|%d\n"+
			"type|%d\n"+
			"type2|%d\n"+
			"#maint|%s\n"+
			"loginurl|%s\n"+
			"meta|%s\n"+
			"RTENDMARKERBS1001",
		s.serverData.Server,
		s.serverData.Port,
		s.serverData.Type,
		s.serverData.Type2,
		s.serverData.Maint,
		s.serverData.LoginURL,
		s.serverData.Meta,
	)

	return fmt.Sprintf(
		"HTTP/1.1 200 OK\r\n"+
			"Content-Type: text/plain\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: close\r\n\r\n"+
			"%s",
		len(content),
		content,
	)
}

func (s *HTTPSServer) Start() error {
	if err := s.LoadServerData(); err != nil {
		return fmt.Errorf("failed to load server data: %w", err)
	}

	certFile := "VallenSetting/resources/ctx/server.crt"
	keyFile := "VallenSetting/resources/ctx/server.key"

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("failed to load certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to bind to port %d: %w", s.port, err)
	}

	tlsListener := tls.NewListener(listener, tlsConfig)
	defer tlsListener.Close()

	log.Printf("[HTTPS] Server listening on port %d", s.port)
	log.Printf("[HTTPS] Serving: %s:%d", s.serverData.Server, s.serverData.Port)

	for {
		conn, err := tlsListener.Accept()
		if err != nil {
			continue
		}

		go s.handleConnection(conn)
	}
}

func (s *HTTPSServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		return
	}

	request := string(buf[:n])
	remoteAddr := conn.RemoteAddr().String()

	if strings.Contains(request, "POST /growtopia/server_data.php") || strings.Contains(request, "/growtopia/server_data.php") {
		log.Printf("[HTTPS] Valid server_data request from %s", remoteAddr)
		response := s.buildResponse()
		conn.Write([]byte(response))
	} else {
		log.Printf("[HTTPS] Unknown request from %s", remoteAddr)
	}
}

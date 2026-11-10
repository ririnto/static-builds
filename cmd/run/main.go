package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

const malformedResponse = "malformed response\r\n"

type prerequisiteError string

func (err prerequisiteError) Error() string { return string(err) }

var samplePattern = regexp.MustCompile(`(?P<sample>,backup="(?:true|false)"(?:,result="[A-Z0-9]+")?} [-+0-9.eE]+)$`)

var results = []string{"UNKNOWN", "SOCKERR", "L4OK", "L4TOUT", "L4CON", "L6OK", "L6TOUT", "L6RSP", "L7OK", "L7TOUT", "L7RSP", "L7STS"}

type fixture struct {
	command *exec.Cmd
	done    chan error
}

type suite struct {
	ctx       context.Context
	directory string
	binary    string
	fixtures  []*fixture
	metrics   string
	headers   http.Header
	client    *http.Client
}

func (s *suite) start(log string, name string, arguments ...string) (*fixture, error) {
	command := exec.Command(name, arguments...)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if log != "" {
		output, err := os.Create(filepath.Join(s.directory, log))
		if err != nil {
			return nil, err
		}
		defer output.Close()
		command.Stdout, command.Stderr = output, output
	}
	if err := command.Start(); err != nil {
		return nil, err
	}
	owned := &fixture{command: command, done: make(chan error, 1)}
	go func() { owned.done <- command.Wait() }()
	s.fixtures = append(s.fixtures, owned)
	return owned, nil
}

func groupRunning(pid int) bool {
	paths, _ := filepath.Glob("/proc/[0-9]*/stat")
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			fields := strings.Fields(string(data)[strings.LastIndex(string(data), ") ")+2:])
			if len(fields) >= 3 && fields[2] == fmt.Sprint(pid) && fields[0] != "Z" {
				return true
			}
		}
	}
	return false
}

func (s *suite) stop(owned *fixture) {
	_ = syscall.Kill(-owned.command.Process.Pid, syscall.SIGTERM)
	deadline := time.Now().Add(3 * time.Second)
	for groupRunning(owned.command.Process.Pid) && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	if groupRunning(owned.command.Process.Pid) {
		_ = syscall.Kill(-owned.command.Process.Pid, syscall.SIGKILL)
	}
	<-owned.done
	retained := s.fixtures[:0]
	for _, item := range s.fixtures {
		if item != owned {
			retained = append(retained, item)
		}
	}
	s.fixtures = retained
}

func (s *suite) execute(arguments ...string) error {
	command := exec.CommandContext(s.ctx, s.binary, arguments...)
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	return command.Run()
}

func (s *suite) write(name, contents string, mode os.FileMode) error {
	return os.WriteFile(filepath.Join(s.directory, name), []byte(contents), mode)
}

func (s *suite) render(name string) error {
	data, err := os.ReadFile(filepath.Join("proxy/tests/fixtures", name))
	if err != nil {
		return err
	}
	return s.write(strings.TrimSuffix(name, ".template"), strings.ReplaceAll(string(data), "@FIXTURE_DIR@", s.directory), 0644)
}

func (s *suite) certificates() error {
	now := time.Now()
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	ca := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "healthcheck-suite-ca"}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(72 * time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign}
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caKey.PublicKey, caKey)
	if err != nil {
		return err
	}
	if err := s.write("ca.pem", string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caBytes})), 0644); err != nil {
		return err
	}
	for index, name := range []string{"wrong-ca", "valid", "expired", "future"} {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return err
		}
		certificate := &x509.Certificate{SerialNumber: big.NewInt(int64(index + 2)), Subject: pkix.Name{CommonName: "backend.local"}, DNSNames: []string{"backend.local"}, NotBefore: now.Add(-5 * time.Minute), NotAfter: now.Add(48 * time.Hour), KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
		parent, signer := ca, caKey
		if name == "expired" {
			certificate.NotBefore, certificate.NotAfter = now.Add(-48*time.Hour), now.Add(-24*time.Hour)
		}
		if name == "future" {
			certificate.NotBefore, certificate.NotAfter = now.Add(time.Hour), now.Add(2*time.Hour)
		}
		if name == "wrong-ca" {
			certificate.Subject.CommonName = "healthcheck-suite-wrong-ca"
			certificate.IsCA = true
			certificate.BasicConstraintsValid = true
			certificate.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageCRLSign
			parent, signer = certificate, key
		}
		data, err := x509.CreateCertificate(rand.Reader, certificate, parent, &key.PublicKey, signer)
		if err != nil {
			return err
		}
		if err := s.write(name+".pem", string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: data})), 0644); err != nil {
			return err
		}
		if err := s.write(name+".key", string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})), 0600); err != nil {
			return err
		}
	}
	return nil
}

func serveMalformed(listener net.Listener) func() {
	var workers sync.WaitGroup
	connections := make(map[net.Conn]struct{})
	var lock sync.Mutex
	workers.Add(1)
	go func() {
		defer workers.Done()
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			lock.Lock()
			connections[connection] = struct{}{}
			lock.Unlock()
			workers.Add(1)
			go func() {
				defer workers.Done()
				defer func() { connection.Close(); lock.Lock(); delete(connections, connection); lock.Unlock() }()
				_ = connection.SetDeadline(time.Now().Add(5 * time.Second))
				_, _ = io.WriteString(connection, malformedResponse)
			}()
		}
	}()
	return func() {
		listener.Close()
		lock.Lock()
		for connection := range connections {
			connection.Close()
		}
		lock.Unlock()
		workers.Wait()
	}
}

func (s *suite) fetch(path string) (string, error) {
	request, err := http.NewRequestWithContext(s.ctx, http.MethodGet, "http://127.0.0.1:19081/"+path, nil)
	if err != nil {
		return "", err
	}
	response, err := s.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if response.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d", response.StatusCode)
	}
	s.headers = response.Header
	return string(data), err
}

func (s *suite) metric(family, upstream, peer, result string) (string, error) {
	matched := 0
	value := ""
	for _, line := range strings.Split(s.metrics, "\n") {
		if strings.HasPrefix(line, family+"{") {
			if !samplePattern.MatchString(line) {
				return "", fmt.Errorf("invalid quoted peer labels in %s", family)
			}
			labels := "," + strings.SplitN(strings.TrimPrefix(line, family+"{"), "}", 2)[0] + ","
			if strings.Contains(labels, `,upstream="`+upstream+`",`) && strings.Contains(labels, `,peer="`+peer+`",`) && (result == "" || strings.Contains(labels, `,result="`+result+`",`)) {
				matched++
				value = line[strings.LastIndex(line, " ")+1:]
			}
		}
	}
	if matched != 1 {
		return "", fmt.Errorf("expected one %s series for %s %s %s, got %d\n%s", family, upstream, peer, result, matched, s.metrics)
	}
	return value, nil
}

func (s *suite) poll(family, upstream, peer, expected string, oneHot bool) error {
	var err error
	for attempt := 0; attempt < 120; attempt++ {
		s.metrics, err = s.fetch("metrics")
		if err == nil {
			if oneHot {
				for _, result := range results {
					value, metricErr := s.metric(family, upstream, peer, result)
					wanted := "0"
					if result == expected {
						wanted = "1"
					}
					if metricErr != nil || value != wanted {
						err = fmt.Errorf("%s %s %s result %s: %s expected %s: %v", family, upstream, peer, result, value, wanted, metricErr)
					}
				}
			} else {
				value, metricErr := s.metric(family, upstream, peer, "")
				if metricErr != nil || value != expected {
					err = fmt.Errorf("%s %s %s readiness %s expected %s: %v", family, upstream, peer, value, expected, metricErr)
				}
			}
		}
		if err == nil {
			return nil
		}
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	return err
}

func streamResponse(host, port, expected string) error {
	connection, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 3*time.Second)
	if err != nil {
		return err
	}
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return err
	}
	if port == "19083" {
		if _, err := io.WriteString(connection, "PING\r\n"); err != nil {
			return err
		}
	}
	response, err := bufio.NewReader(connection).ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.Contains(strings.TrimSuffix(response, "\r\n"), expected) {
		return fmt.Errorf("unexpected stream response %q", response)
	}
	return nil
}

func (s *suite) run() error {
	if err := s.certificates(); err != nil {
		return err
	}
	if err := s.write("hosts", "127.0.0.2 backend.local\n127.0.0.3 backend.local\n", 0644); err != nil {
		return err
	}
	dns, err := s.start("", "dnsmasq", "--keep-in-foreground", "--no-resolv", "--no-hosts", "--addn-hosts="+filepath.Join(s.directory, "hosts"), "--port=15353", "--listen-address=127.0.0.1", "--bind-interfaces", "--pid-file="+filepath.Join(s.directory, "dnsmasq.pid"))
	if err != nil {
		return err
	}
	for _, item := range []struct{ port, log string }{{"19443", "tls-stall.log"}, {"19086", "http-stall.log"}} {
		if _, err := s.start(item.log, "socat", "TCP-LISTEN:"+item.port+",bind=127.0.0.1,reuseaddr,fork", "EXEC:sleep 5"); err != nil {
			return err
		}
	}
	certificate, err := tls.LoadX509KeyPair(filepath.Join(s.directory, "valid.pem"), filepath.Join(s.directory, "valid.key"))
	if err != nil {
		return err
	}
	listener, err := tls.Listen("tcp", "127.0.0.1:19455", &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12})
	if err != nil {
		return err
	}
	defer serveMalformed(listener)()
	for _, name := range []string{"nginx.conf.template", "http-missing-name.conf.template", "http-missing-ca.conf.template", "stream-missing-name.conf.template", "stream-missing-ca.conf.template"} {
		if err := s.render(name); err != nil {
			return err
		}
	}
	for _, kind := range []string{"http", "stream"} {
		for _, missing := range []string{"name", "ca"} {
			command := exec.CommandContext(s.ctx, s.binary, "-t", "-p", s.directory, "-c", filepath.Join(s.directory, kind+"-missing-"+missing+".conf"))
			output, err := command.CombinedOutput()
			message := "TLS healthcheck requires healthcheck_tls_name and healthcheck_trusted_certificate"
			if kind == "http" {
				message = "HTTPS healthcheck requires healthcheck_tls_name and healthcheck_trusted_certificate"
			}
			if err == nil || !strings.Contains(string(output), message) {
				return fmt.Errorf("invalid configuration acceptance: %v\n%s", err, output)
			}
			fmt.Print(string(output))
		}
	}
	for _, path := range []string{filepath.Join("proxy/tests/fixtures", "implicit-http.conf"), filepath.Join("proxy/tests/fixtures", "implicit-stream.conf"), filepath.Join(s.directory, "nginx.conf")} {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if err := s.execute("-t", "-p", s.directory, "-c", absolute); err != nil {
			return err
		}
	}
	if _, err := s.start("", s.binary, "-p", s.directory, "-c", filepath.Join(s.directory, "nginx.conf"), "-g", "daemon off;"); err != nil {
		return err
	}
	for _, test := range resultCases {
		if err := s.poll(test.family, test.upstream, test.peer, test.expected, true); err != nil {
			return err
		}
	}
	for _, test := range []struct{ family, upstream, peer string }{{"nginx_healthcheck_peer_up", "dynamic_http", "127.0.0.2:19080"}, {"nginx_stream_healthcheck_peer_up", "dynamic_tcp", "127.0.0.2:19082"}, {"nginx_healthcheck_peer_up", "ipv6_http", "[::1]:19084"}} {
		if err := s.poll(test.family, test.upstream, test.peer, "1", false); err != nil {
			return err
		}
	}
	for _, test := range []struct{ family, upstream, expected string }{{"nginx_healthcheck_check_status", "tls_only", "1"}, {"nginx_healthcheck_check_code", "tls_only", "0"}, {"nginx_healthcheck_check_status", "https_success", "1"}, {"nginx_healthcheck_check_code", "https_success", "200"}, {"nginx_healthcheck_check_status", "https_status", "0"}, {"nginx_healthcheck_check_code", "https_status", "503"}} {
		s.metrics, err = s.fetch("metrics")
		if err != nil {
			return err
		}
		peer := "127.0.0.1:19454"
		if test.upstream == "tls_only" {
			peer = "127.0.0.1:19444"
		}
		value, err := s.metric(test.family, test.upstream, peer, "")
		if err != nil || value != test.expected {
			return fmt.Errorf("gauge %s %s got %q expected %q: %v", test.family, test.upstream, value, test.expected, err)
		}
	}
	application, err := os.ReadFile(filepath.Join(s.directory, "tls-application.log"))
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimSuffix(string(application), "\n"), "\n")
	if len(lines) < 2 {
		return fmt.Errorf("both TLS-only checks were not observed")
	}
	for _, line := range lines {
		if line != "0" {
			return fmt.Errorf("TLS-only probe sent application bytes: %q", line)
		}
	}
	if err := s.route("127.0.0.2"); err != nil {
		return err
	}
	if err := streamResponse("::1", "19084", "IPv6 ready"); err != nil {
		return err
	}
	traffic, err := s.fetch("traffic")
	if err != nil {
		return err
	}
	found := false
	for _, line := range strings.Split(traffic, "\n") {
		found = found || strings.HasPrefix(line, "nginx_vts_")
	}
	if !found {
		return fmt.Errorf("VTS endpoint did not export a metric")
	}
	firstWorker := ""
	observed := false
	for attempt := 0; attempt < 120 && !observed; attempt++ {
		if err := s.poll("nginx_healthcheck_check_result", "dynamic_http", "127.0.0.2:19080", "L7OK", true); err != nil {
			return err
		}
		worker := s.headers.Get("X-Worker")
		if worker != "" {
			if firstWorker == "" {
				firstWorker = worker
			} else if worker != firstWorker {
				observed = true
				fmt.Println("Shared health metrics served by workers", firstWorker, "and", worker)
			}
		}
		if !observed {
			time.Sleep(50 * time.Millisecond)
		}
	}
	if !observed {
		return fmt.Errorf("did not observe shared metrics from two workers")
	}
	for _, test := range []struct{ family, upstream, peer string }{{"nginx_healthcheck_peer_up", "dynamic_http", "127.0.0.3:19080"}, {"nginx_stream_healthcheck_peer_up", "dynamic_tcp", "127.0.0.3:19082"}} {
		if err := s.poll(test.family, test.upstream, test.peer, "0", false); err != nil {
			return err
		}
	}
	if err := s.write("hosts", "127.0.0.4 backend.local\n127.0.0.3 backend.local\n", 0644); err != nil {
		return err
	}
	if err := dns.command.Process.Signal(syscall.SIGHUP); err != nil {
		return err
	}
	for _, test := range []struct{ family, upstream, peer string }{{"nginx_healthcheck", "dynamic_http", "127.0.0.4:19080"}, {"nginx_stream_healthcheck", "dynamic_tcp", "127.0.0.4:19082"}} {
		if err := s.poll(test.family+"_check_result", test.upstream, test.peer, "L7OK", true); err != nil {
			return err
		}
		if err := s.poll(test.family+"_peer_up", test.upstream, test.peer, "1", false); err != nil {
			return err
		}
	}
	if err := s.route("127.0.0.4"); err != nil {
		return err
	}
	s.metrics, err = s.fetch("metrics")
	if err != nil {
		return err
	}
	for _, line := range strings.Split(s.metrics, "\n") {
		if strings.Contains(line, `upstream="dynamic_http"`) && strings.Contains(line, `peer="127.0.0.2:19080"`) || strings.Contains(line, `upstream="dynamic_tcp"`) && strings.Contains(line, `peer="127.0.0.2:19082"`) {
			return fmt.Errorf("removed DNS peer remains: %s", line)
		}
	}
	if err := s.write("recovered-http", "", 0644); err != nil {
		return err
	}
	if err := s.poll("nginx_healthcheck_peer_up", "dynamic_http", "127.0.0.3:19080", "1", false); err != nil {
		return err
	}
	for _, upstream := range []string{"dynamic_http", "dynamic_tcp"} {
		found := false
		for _, line := range strings.Split(s.metrics, "\n") {
			found = found || strings.Contains(line, `upstream="`+upstream+`"`) && strings.Contains(line, `server="127.0.0.1:19999"`) && strings.Contains(line, `peer="127.0.0.1:19999"`) && strings.Contains(line, `backup="true"`)
		}
		if !found {
			return fmt.Errorf("backup labels missing for %s", upstream)
		}
	}
	recovery, err := s.start("recovery-server.log", "openssl", "s_server", "-accept", "127.0.0.1:19445", "-cert", filepath.Join(s.directory, "wrong-ca.pem"), "-key", filepath.Join(s.directory, "wrong-ca.key"), "-quiet", "-ign_eof")
	if err != nil {
		return err
	}
	for _, test := range []struct{ family, upstream string }{{"nginx_healthcheck", "tls_recovery"}, {"nginx_stream_healthcheck", "stream_tls_recovery"}} {
		if err := s.poll(test.family+"_check_result", test.upstream, "127.0.0.1:19445", "L6RSP", true); err != nil {
			return err
		}
		if err := s.poll(test.family+"_peer_up", test.upstream, "127.0.0.1:19445", "0", false); err != nil {
			return err
		}
	}
	s.stop(recovery)
	if _, err := s.start("recovery-server.log", "openssl", "s_server", "-accept", "127.0.0.1:19445", "-cert", filepath.Join(s.directory, "valid.pem"), "-key", filepath.Join(s.directory, "valid.key"), "-quiet", "-ign_eof"); err != nil {
		return err
	}
	for _, test := range []struct{ family, upstream string }{{"nginx_healthcheck_peer_up", "tls_recovery"}, {"nginx_stream_healthcheck_peer_up", "stream_tls_recovery"}} {
		if err := s.poll(test.family, test.upstream, "127.0.0.1:19445", "1", false); err != nil {
			return err
		}
	}
	fmt.Println("L4/L6/L7 classification, DNS replacement, recovery, IPv6, multi-worker visibility, and VTS checks passed.")
	return nil
}

func (s *suite) route(peer string) error {
	output, err := s.fetch("route")
	if err != nil {
		return err
	}
	if strings.TrimSuffix(output, "\n") != peer {
		return fmt.Errorf("HTTP route got %q expected %s", output, peer)
	}
	return streamResponse("127.0.0.1", "19083", peer+" PONG")
}

func run(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return prerequisiteError("usage: run /path/to/linux/nginx")
	}
	if runtime.GOOS != "linux" {
		return prerequisiteError("the functional suite requires Linux, use test-healthcheck-docker on macOS")
	}
	binary, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	info, err := os.Stat(binary)
	if err != nil || info.IsDir() || info.Mode()&0111 == 0 {
		return prerequisiteError(fmt.Sprintf("NGINX binary is not executable: %s", binary))
	}
	for _, name := range []string{"dnsmasq", "openssl", "socat", "sleep"} {
		if _, err := exec.LookPath(name); err != nil {
			return prerequisiteError(err.Error())
		}
	}
	directory, err := os.MkdirTemp("/tmp", "healthcheck-suite.")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	if err := os.Chmod(directory, 0711); err != nil {
		return err
	}
	if err := os.Mkdir(filepath.Join(directory, "logs"), 0755); err != nil {
		return err
	}
	transport := &http.Transport{DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	s := &suite{ctx: ctx, directory: directory, binary: binary, client: &http.Client{Timeout: 2 * time.Second, Transport: transport}}
	defer func() {
		for len(s.fixtures) > 0 {
			s.stop(s.fixtures[len(s.fixtures)-1])
		}
	}()
	return s.run()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	received := make(chan os.Signal, 1)
	signal.Notify(received, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(received)
	status := make(chan int, 1)
	go func() {
		select {
		case value := <-received:
			status <- 128 + int(value.(syscall.Signal))
			cancel()
		case <-ctx.Done():
		}
	}()
	err := run(ctx, os.Args[1:])
	cancel()
	select {
	case code := <-status:
		os.Exit(code)
	default:
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		if _, ok := err.(prerequisiteError); ok {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

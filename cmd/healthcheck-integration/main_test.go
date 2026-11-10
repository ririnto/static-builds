package main

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestMalformedResponseWire(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	stop := serveMalformed(listener)
	defer stop()
	connection, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	response, err := io.ReadAll(connection)
	if err != nil {
		t.Fatal(err)
	}
	if string(response) != "malformed response\r\n" {
		t.Fatalf("malformed fixture wire bytes changed: %q", response)
	}
}

func TestMetricSamples(t *testing.T) {
	s := suite{metrics: `nginx_healthcheck_check_result{upstream="dynamic_http",server="backend.local:19080",peer="127.0.0.2:19080",backup="false",result="L7OK"} 1`}
	if value, err := s.metric("nginx_healthcheck_check_result", "dynamic_http", "127.0.0.2:19080", "L7OK"); err != nil || value != "1" {
		t.Fatalf("value=%q err=%v", value, err)
	}
	s.metrics += "\n" + s.metrics
	if _, err := s.metric("nginx_healthcheck_check_result", "dynamic_http", "127.0.0.2:19080", "L7OK"); err == nil {
		t.Fatal("duplicate series accepted")
	}
	s.metrics = `nginx_healthcheck_check_result{upstream="dynamic_http",peer="127.0.0.2:19080",backup=false,result="L7OK"} 1`
	if _, err := s.metric("nginx_healthcheck_check_result", "dynamic_http", "127.0.0.2:19080", "L7OK"); err == nil {
		t.Fatal("unquoted backup label accepted")
	}
}

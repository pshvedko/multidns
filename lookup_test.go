package multidns

import (
	"fmt"
	"log"
	"net"
	"reflect"
	"testing"
)

func TestLookupIP(t *testing.T) {
	type args struct {
		host string
		via  net.IP
	}
	tests := []struct {
		name    string
		args    args
		wantIp  []net.IP
		wantErr bool
	}{
		// TODO: Add test cases.
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIp, err := LookupIP(tt.args.host, tt.args.via)
			if (err != nil) != tt.wantErr {
				t.Errorf("LookupIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotIp, tt.wantIp) {
				t.Errorf("LookupIP() gotIp = %v, want %v", gotIp, tt.wantIp)
			}
		})
	}
}

func TestLookupIPv4(t *testing.T) {
	type args struct {
		host string
		via  net.IP
	}
	tests := []struct {
		name    string
		args    args
		wantIp  []net.IP
		wantErr bool
	}{
		// TODO: Add test cases.
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIp, err := LookupIPv4(tt.args.host, tt.args.via)
			if (err != nil) != tt.wantErr {
				t.Errorf("LookupIPv4() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotIp, tt.wantIp) {
				t.Errorf("LookupIPv4() gotIp = %v, want %v", gotIp, tt.wantIp)
			}
		})
	}
}

func TestLookupIPv6(t *testing.T) {
	type args struct {
		host string
		via  net.IP
	}
	tests := []struct {
		name    string
		args    args
		wantIp  []net.IP
		wantErr bool
	}{
		// TODO: Add test cases.
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIp, err := LookupIPv6(tt.args.host, tt.args.via)
			if (err != nil) != tt.wantErr {
				t.Errorf("LookupIPv6() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotIp, tt.wantIp) {
				t.Errorf("LookupIPv6() gotIp = %v, want %v", gotIp, tt.wantIp)
			}
		})
	}
}

func ExampleLookupIPv6() {
	ips, err := LookupIPv6("google.com", net.IP{8, 8, 8, 8})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ips)
	// Output:
	// [2a00:1450:4026:803::200e]
}

func ExampleLookupIPv4() {
	ips, err := LookupIPv4("google.com", net.IP{8, 8, 8, 8})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ips)
	// Output:
	// [216.58.209.206]
}

func ExampleLookupIP() {
	ips, err := LookupIP("google.com", net.IP{8, 8, 8, 8})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ips)
	// Output:
	// [216.58.209.206 2a00:1450:4026:803::200e]
}

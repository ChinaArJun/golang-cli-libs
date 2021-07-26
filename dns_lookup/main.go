package main

// https://medium.com/@owlwalks/build-a-dns-server-in-golang-fec346c42889

import (
	"context"
	"fmt"
	"net"
	"os"
)

func main() {
	var resolver *net.Resolver

	resolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, "udp", net.JoinHostPort("127.0.0.1", "53535"))
		},
	}

	cname, srv, err := resolver.LookupSRV(context.Background(), "", "", "dns.service.discover.tld.")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not get IPs: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("cname: %v\n", cname)
	for _, s := range srv {
		fmt.Printf("srv IN A Target: %s, Port: %v, Priority: %v, Weight: %v\n", s.Target, s.Port, s.Priority, s.Weight)
	}

	cname, srv, err = resolver.LookupSRV(context.Background(), "", "", "http.service.discover.tld.")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not get IPs: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n\ncname: %v\n", cname)
	for _, s := range srv {
		fmt.Printf("srv IN A Target: %s, Port: %v, Priority: %v, Weight: %v\n", s.Target, s.Port, s.Priority, s.Weight)
	}
}

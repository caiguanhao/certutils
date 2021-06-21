package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/caiguanhao/certutils/dns"
)

const (
	ymdhmsFormat = "2006-01-02 15:04:05"

	textChecking = "checking..."

	colorReset = "\x1b[0m"
)

const (
	colorCyan = iota
	colorGreen
	colorRed
	colorYellow
)

var (
	dialer = &net.Dialer{Timeout: 10 * time.Second}
	config = &tls.Config{InsecureSkipVerify: true}

	colors = []string{
		/* colorCyan   */ "\x1b[96m",
		/* colorGreen  */ "\x1b[92m",
		/* colorRed    */ "\x1b[31m",
		/* colorYellow */ "\x1b[33m",
	}
)

func main() {
	dnsType := flag.String("dns", "alidns", "can be alidns, cloudflare")
	flag.Usage = func() {
		fmt.Println("Usage of chkcert [OPTIONS] [PATTERNS...]")
		fmt.Println(`
This utility makes TLS connections to all your domains, checks the
certificates' expiration dates and lists how many days left until expiration
date.

PATTERNS: Optional. Only check domains contains one of specific strings.

OPTIONS:`)
		flag.PrintDefaults()
	}
	flag.Parse()

	var client dns.DNS
	if *dnsType == "alidns" {
		client = dns.Alidns{}
	} else if *dnsType == "cloudflare" {
		client = dns.Cloudflare{}
	} else {
		log.Fatal("bad dns type")
	}

	patterns := flag.Args()
	match := func(name string) bool {
		if len(patterns) == 0 {
			return true
		}
		for _, pattern := range patterns {
			if strings.Contains(name, pattern) {
				return true
			}
		}
		return false
	}

	domains := client.GetListOfDomains()
	for _, domain := range domains {
		for _, record := range client.GetRecords(domain) {
			if !match(record.FullName) || record.Type != "A" {
				continue
			}
			fmt.Printf("%40s  %s", record.FullName, colorize(textChecking, colorCyan))
			result, color := getExpiry(record.FullName)
			if len(result) < len(textChecking) {
				result += strings.Repeat(" ", len(textChecking)-len(result))
			}
			fmt.Print("\r")
			fmt.Printf("%40s  %s", record.FullName, colorize(result, color))
			time.Sleep(300 * time.Millisecond)
			fmt.Print("\n")
		}
	}
}

func getExpiry(host string) (string, int) {
	if strings.LastIndex(host, ":") == -1 {
		host = host + ":443"
	}
	conn, err := tls.DialWithDialer(dialer, "tcp", host, config)
	if err != nil {
		return err.Error(), colorYellow
	}
	defer conn.Close()
	certs := conn.ConnectionState().PeerCertificates
	now := time.Now()
	var daysMin *int
	for _, cert := range certs {
		if !now.Before(cert.NotAfter) || !now.After(cert.NotBefore) {
			return fmt.Sprintf("expired! (%s - %s)",
				cert.NotBefore.Format(ymdhmsFormat),
				cert.NotAfter.Format(ymdhmsFormat)), colorRed
		}
		days := int(time.Until(cert.NotAfter).Hours() / 24)
		if daysMin == nil || days < *daysMin {
			daysMin = &days
		}
	}
	if daysMin == nil {
		return "ok", colorGreen
	}
	return fmt.Sprintf("ok (%d days left)", *daysMin), colorGreen
}

func colorize(str string, color int) string {
	return colors[color] + string(str) + colorReset
}

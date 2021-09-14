package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/caiguanhao/certutils/dns"
)

var (
	debug  bool
	dryRun bool
	email  string

	secondsToWait int
	shouldClean   bool
)

func main() {
	flag.BoolVar(&debug, "debug", false, "show more info")
	dnsType := flag.String("dns", "alidns", "can be alidns, cloudflare")
	flag.IntVar(&secondsToWait, "wait", 10, "seconds to wait for dns record to take effect")
	flag.BoolVar(&dryRun, "dry-run", false, "dry-run certbot, but dns records will still be modified")
	flag.StringVar(&email, "email", "a@a.com", "email for certbot")
	flag.BoolVar(&shouldClean, "clean", false, "remove acme challenge txt records for domain and exit")
	flag.Usage = func() {
		fmt.Println("Usage of mkcert [OPTIONS] [NAMES...]")
		fmt.Println(`
This utility obtains certbot's (Let's Encrypt) wildcard certificates by
updating DNS TXT records and answering stupid certbot questions for you.

NAMES: Provide at least one domain. All domain names must start with "*.".

NOTE: You may be [rate-limited](https://letsencrypt.org/docs/rate-limits/)
if you are going to make many certs with the same IP address.

NOTE: Certbot runs in a Docker container, the certificate files
("example.com.cert" and "example.com.key") will be copied from the container to
the working directory (will be overwritten without prompt if same file exists).
If you still need your old certificate files, please backup first.

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
		log.Fatal("Error: bad dns type")
	}

	targets := flag.Args()

	if len(targets) == 0 {
		log.Fatal("please provide wildcard domain name like this: *.example.com")
	}

	for i, target := range targets {
		if strings.Count(target, "*") == 0 {
			targets[i] = "*." + target
			fmt.Fprintf(os.Stderr, `Did you mean "%s"? (Y/n) `, targets[i])
			var answer string
			fmt.Scanln(&answer)
			answer = strings.ToLower(strings.TrimSpace(answer))
			if answer != "" && answer != "y" {
				log.Fatal("Aborted")
				return
			}
		} else if strings.Count(target, "*") > 1 || !strings.HasPrefix(target, "*.") {
			log.Fatalf("Error: domain name %s must start with one '*.'", target)
		}
	}

	for i, target := range targets {
		if i > 0 {
			log.Println(strings.Repeat("=", 40))
		}
		get(client, target)
	}
}

func get(client dns.DNS, target string) {
	log.Println("processing", target)
	targetWithoutWildcard := strings.TrimPrefix(target, "*.")
	acme := strings.Replace(target, "*", "_acme-challenge", 1)
	domains := client.GetListOfDomains()
	root := ""
	for _, domain := range domains {
		if strings.HasSuffix(target, domain) {
			root = domain
			break
		}
	}
	if root == "" {
		log.Fatalln("Error: you don't have root domain for", target)
	}
	acmeWithoutRoot := strings.TrimSuffix(strings.TrimSuffix(acme, root), ".")

	log.Println("root domain:", root)

	if shouldClean {
		for _, id := range client.GetRecordIdsFor(root, acmeWithoutRoot, "TXT") {
			log.Println("deleting TXT record with id", id)
			client.DeleteRecord(root, id)
		}
		return
	}

	containerId := newContainer(target)
	containerId = containerId[:8]
	log.Println("created container:", containerId)

	log.Println("finding TXT records for", acmeWithoutRoot)
	ids := client.GetRecordIdsFor(root, acmeWithoutRoot, "TXT")
	if len(ids) == 0 {
		log.Println("no TXT records for", acmeWithoutRoot, "yet!")
	} else {
		log.Println("found", len(ids), "TXT records for", acmeWithoutRoot)
		for _, id := range ids {
			log.Println("deleting TXT record with id", id)
			client.DeleteRecord(root, id)
		}
	}

	c := newCertbot()
	go c.start(containerId, acme)

	log.Println("waiting acme challenge")
	challenges := []string{}
	for challenge := range c.acmeChallengeChan {
		challenges = append(challenges, challenge)
	}
	for _, challenge := range challenges {
		log.Println("received certbot's acme challenge:", challenge)
		log.Println("creating new TXT record")
		id := client.AddNewRecord(root, acmeWithoutRoot, "TXT", challenge)
		log.Println("new record has been created, id:", id)
	}
	log.Println("wait", secondsToWait, "seconds for dns records to take effect")
	time.Sleep(time.Duration(secondsToWait) * time.Second)
	c.continueChan <- true
	if !dryRun {
		log.Println("waiting cert files")
		cert := copyFileFromContainer(containerId, <-c.pemFileChan)
		writeFile(targetWithoutWildcard+".cert", cert)
		key := copyFileFromContainer(containerId, <-c.keyFileChan)
		writeFile(targetWithoutWildcard+".key", key)
	}
	<-c.doneChan
	removeContainer(containerId)
	log.Println("done:", target)
}

func writeFile(file string, content []byte) {
	if len(content) == 0 {
		log.Fatal(file, "is empty")
	}
	err := ioutil.WriteFile(file, content, 0644)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("written file", file)
}

func newContainer(domain string) string {
	domainWithoutWildcard := strings.TrimPrefix(domain, "*.")
	command := []string{
		"docker", "create", "-i", "certbot/certbot:v1.10.0", "certonly", "--manual",
		"--preferred-challenges=dns", "--email", email,
		"--server", "https://acme-v02.api.letsencrypt.org/directory",
		"--agree-tos", "-d", domain, "-d", domainWithoutWildcard,
	}
	if dryRun {
		command = append(command, "--dry-run")
	}
	if debug {
		log.Println("running", command)
	}
	cmd := exec.Command(command[0], command[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(string(out))
	}
	return strings.TrimSpace(string(out))
}

func removeContainer(containerId string) {
	log.Println("removing container", containerId)
	cmd := exec.Command("docker", "rm", "-fv", containerId)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func copyFileFromContainer(containerId, file string) []byte {
	log.Println("copying", file, "from", containerId)
	cmd := exec.Command("docker", "cp", "--follow-link", containerId+":"+file, "-")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	tr := tar.NewReader(stdout)
	var buf bytes.Buffer
	for {
		_, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			log.Fatal(err)
		}
		if _, err := io.Copy(&buf, tr); err != nil {
			log.Fatal(err)
		}
	}
	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}

type certbot struct {
	acmeChallengeChan, pemFileChan, keyFileChan chan string
	continueChan, doneChan                      chan bool
}

func newCertbot() *certbot {
	return &certbot{
		acmeChallengeChan: make(chan string),
		pemFileChan:       make(chan string),
		keyFileChan:       make(chan string),
		continueChan:      make(chan bool),
		doneChan:          make(chan bool),
	}
}

func (c *certbot) start(containerId, acme string) {
	cmd := exec.Command("docker", "start", "-ai", containerId)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	defer stdin.Close()
	go func() {
		for {
			select {
			case <-c.continueChan:
				log.Println("pressing enter to certbot, waiting for response...")
				io.WriteString(stdin, "\n")
			}
		}
	}()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(stdout)
	mode, success, acmeCount := 0, false, 0
	for scanner.Scan() {
		t := scanner.Text()
		if debug {
			log.Println("certbot:", t)
		}
		switch mode {
		case 1:
			if strings.Contains(t, acme) {
				mode = 2
			}
		case 2:
			if t == "" {
				continue
			}
			c.acmeChallengeChan <- t
			acmeCount += 1
			if acmeCount == 1 {
				c.continueChan <- true
			} else if acmeCount == 2 {
				close(c.acmeChallengeChan)
			}
			mode = 3
		default:
			if strings.Contains(t, "deploy a DNS TXT record") {
				mode = 1
			} else if strings.Contains(t, "successful") || strings.Contains(t, "Congratulations") {
				success = true
			} else if strings.Contains(t, "fullchain.pem") {
				c.pemFileChan <- strings.TrimSpace(t)
			} else if strings.Contains(t, "privkey.pem") {
				c.keyFileChan <- strings.TrimSpace(t)
			}
		}
	}
	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}
	if success {
		log.Println("successfully generated certificates")
	} else {
		log.Println("failed to generate certificates")
	}
	c.doneChan <- true
}

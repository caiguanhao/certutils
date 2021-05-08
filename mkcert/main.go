package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/caiguanhao/certutils/dns"
)

var (
	debug  bool
	dryRun bool
	email  string
)

func main() {
	flag.BoolVar(&debug, "debug", false, "show more info")
	dnsType := flag.String("dns", "alidns", "can be alidns, cloudflare")
	wait := flag.Int("wait", 10, "seconds to wait for dns record to take effect")
	flag.BoolVar(&dryRun, "dry-run", false, "dry-run certbot, but dns records will still be modified")
	flag.StringVar(&email, "email", "a@a.com", "email for certbot")
	clean := flag.Bool("clean", false, "remove acme challenge txt records for domain and exit")
	flag.Parse()

	if flag.NArg() == 0 {
		log.Fatal("please provide wildcard domain name like this: *.example.com")
	}

	target := flag.Arg(0)
	if strings.Count(target, "*") != 1 || !strings.HasPrefix(target, "*.") {
		log.Fatal("domain name should contain one *. prefix")
	}

	var client dns.DNS
	if *dnsType == "alidns" {
		client = dns.Alidns{}
	} else if *dnsType == "cloudflare" {
		client = dns.Cloudflare{}
	} else {
		log.Fatal("bad dns type")
	}

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
		log.Fatalln("you don't have root domain for", target)
	}
	acmeWithoutRoot := strings.TrimSuffix(strings.TrimSuffix(acme, root), ".")

	log.Println("root domain:", root)

	if *clean {
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
	log.Println("wait", *wait, "seconds for dns records to take effect")
	time.Sleep(time.Duration(*wait) * time.Second)
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
	log.Println("done")
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
		"docker", "create", "-i", "certbot/certbot", "certonly", "--manual",
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

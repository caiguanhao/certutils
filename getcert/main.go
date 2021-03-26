package main

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/caiguanhao/ossslim"
)

var (
	encryptionKey      string
	ossAccessKeyId     string
	ossAccessKeySecret string
	ossPrefix          string
	ossBucket          string

	force     bool
	showDates bool

	suffixes = []string{".cert", ".key"}

	client ossslim.Client
)

const (
	certsDir = "certs/"
)

func main() {
	flag.BoolVar(&force, "f", false, "overwrite existing file")
	flag.BoolVar(&showDates, "d", false, "display expiration dates")
	flag.Parse()
	client = ossslim.Client{
		AccessKeyId:     ossAccessKeyId,
		AccessKeySecret: ossAccessKeySecret,
		Prefix:          ossPrefix,
		Bucket:          ossBucket,
	}
	targets := flag.Args()
	if len(targets) == 0 {
		log.Println("getting list of certs")
		result, err := client.List(certsDir, false)
		if err != nil {
			panic(err)
		}
		names := []string{}
		combined := map[string][]string{}
		for _, f := range result.Files {
			for _, s := range suffixes {
				if strings.HasSuffix(f.Name, s) {
					name := strings.TrimSuffix(f.Name[len(certsDir):], s)
					if _, ok := combined[name]; !ok {
						names = append(names, name)
					}
					combined[name] = append(combined[name], s)
				}
			}
		}
		sort.Strings(names)

		var notAfters *sync.Map
		if showDates {
			notAfters = getNotAfters(names)
		}

		printTo := func(w io.Writer) {
			for i, name := range names {
				var extra string
				if notAfters != nil {
					if notAfter, ok := notAfters.Load(name); ok {
						extra = " - " + notAfter.(string)
					}
				}
				fmt.Fprintf(w, "%d. %s{%s}%s\n", i+1, name, strings.Join(combined[name], ","), extra)
			}
		}
		cmd := exec.Command("column")
		cmd.Stdout = os.Stdout
		stdin, err := cmd.StdinPipe()
		if err == nil {
			go func() {
				defer stdin.Close()
				printTo(stdin)
			}()
			err = cmd.Run()
		}
		if err != nil {
			printTo(os.Stdout)
		}
		var selected []int
		for len(selected) == 0 {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter numbers (separated by comma) to choose files: ")
			input, err := reader.ReadString('\n')
			if err != nil {
				panic(err)
			}
			input = strings.TrimSpace(input)
			numbers := strings.Split(input, ",")
		a:
			for _, n := range numbers {
				num, err := strconv.Atoi(n)
				if err != nil {
					continue
				}
				if num < 1 || num > len(names) {
					continue
				}
				for _, s := range selected {
					if s == num {
						continue a
					}
				}
				selected = append(selected, num)
			}
		}
		for _, s := range selected {
			for _, suffix := range combined[names[s-1]] {
				targets = append(targets, names[s-1]+suffix)
			}
		}
	}
	for _, t := range targets {
		if !strings.HasPrefix(t, certsDir) {
			t = certsDir + t
		}
		file := filepath.Base(t)
		if !canWrite(file) {
			continue
		}
		log.Println("downloading", file)
		var buffer bytes.Buffer
		_, err := client.Download(t, &buffer)
		if err != nil {
			log.Println(err)
			continue
		}
		content, err := decrypt(buffer.Bytes())
		if err != nil {
			log.Println(err)
			continue
		}
		err = ioutil.WriteFile(file, content, 0600)
		if err != nil {
			log.Println(err)
		}
		log.Println("written", file)
	}
}

func canWrite(path string) bool {
	if force {
		return true
	}
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return true
	}
	if err != nil {
		return false
	}
	reader := bufio.NewReader(os.Stdin)
	var input string
	for input != "y" && input != "n" {
		fmt.Print(path, " already exists. Overwrite? (y/N): ")
		input, err = reader.ReadString('\n')
		if err != nil {
			return false
		}
		input = strings.ToLower(strings.TrimSpace(input))
		if input == "" {
			input = "n"
		}
	}
	return input == "y"
}

func decrypt(content []byte) ([]byte, error) {
	block, err := aes.NewCipher([]byte(encryptionKey))
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesgcm.NonceSize()
	nonce, ciphertext := content[:nonceSize], content[nonceSize:]

	return aesgcm.Open(nil, nonce, ciphertext, nil)
}

func getNotAfter(name string) (string, error) {
	var buffer bytes.Buffer
	_, err := client.Download(certsDir+name+".cert", &buffer)
	if err != nil {
		return "", err
	}
	content, err := decrypt(buffer.Bytes())
	if err != nil {
		return "", err
	}
	block, _ := pem.Decode(content)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", err
	}
	days := int(time.Until(cert.NotAfter).Hours() / 24)
	return fmt.Sprintf("%s (%d days)", cert.NotAfter.Format("2006-01-02"), days), nil
}

func getNotAfters(names []string) *sync.Map {
	var notAfters sync.Map
	log.Println("getting expiration dates of certs")
	jobs := make(chan string)
	go func() {
		defer close(jobs)
		for _, name := range names {
			jobs <- name
		}
	}()
	concurrency := 5
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			for name := range jobs {
				notAfter, err := getNotAfter(name)
				if err != nil {
					log.Println(err)
					continue
				}
				notAfters.Store(name, notAfter)
			}
		}()
	}
	wg.Wait()
	return &notAfters
}

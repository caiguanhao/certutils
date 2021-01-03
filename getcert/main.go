package main

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/caiguanhao/ossslim"
)

var (
	encryptionKey      string
	ossAccessKeyId     string
	ossAccessKeySecret string
	ossPrefix          string
	ossBucket          string

	force bool
)

const (
	certsDir = "certs/"
)

func main() {
	flag.BoolVar(&force, "f", false, "overwrite existing file")
	flag.Parse()
	targets := flag.Args()
	client := ossslim.Client{
		AccessKeyId:     ossAccessKeyId,
		AccessKeySecret: ossAccessKeySecret,
		Prefix:          ossPrefix,
		Bucket:          ossBucket,
	}
	if len(targets) == 0 {
		log.Println("getting list of certs")
		result, err := client.List(certsDir, false)
		if err != nil {
			panic(err)
		}
		for i, f := range result.Files {
			fmt.Printf("%d. %s\n", i+1, f.Name[len(certsDir):])
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
				if num < 1 || num > len(result.Files) {
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
			targets = append(targets, result.Files[s-1].Name)
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
		err = ioutil.WriteFile(file, content, 0400)
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

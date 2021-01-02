package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/caiguanhao/ossslim"
)

var (
	encryptionKey      string
	ossAccessKeyId     string
	ossAccessKeySecret string
	ossPrefix          string
	ossBucket          string
)

func main() {
	flag.Parse()
	files := flag.Args()
	if len(files) == 0 {
		panic("no files")
	}
	client := ossslim.Client{
		AccessKeyId:     ossAccessKeyId,
		AccessKeySecret: ossAccessKeySecret,
		Prefix:          ossPrefix,
		Bucket:          ossBucket,
	}
	for _, file := range files {
		f, err := ioutil.ReadFile(file)
		if err != nil {
			panic(err)
		}
		b, err := encrypt(f)
		if err != nil {
			panic(err)
		}
		file = filepath.Base(file)
		_, err = client.Upload("/certs/"+file, bytes.NewReader(b), nil, "")
		if err != nil {
			panic(err)
		}
		log.Println("uploaded", file)
	}
}

func encrypt(content []byte) ([]byte, error) {
	block, err := aes.NewCipher([]byte(encryptionKey))
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return aesgcm.Seal(nonce, nonce, content, nil), nil
}

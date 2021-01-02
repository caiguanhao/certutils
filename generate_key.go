package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	key, err := hex.DecodeString(os.Getenv("ENCRYPTION_KEY"))
	if err != nil {
		panic(err)
	}
	if len(key) == 0 {
		key = make([]byte, 32)
		_, err := rand.Read(key)
		if err != nil {
			panic(err)
		}
	}
	akid, aks, err := getAccessKey()
	if err != nil {
		akid = os.Getenv("OSS_ACCESS_KEY_ID")
		aks = os.Getenv("OSS_ACCESS_KEY_SECRET")
	}
	region := os.Getenv("OSS_REGION")
	if region == "" {
		reader := bufio.NewReader(os.Stdin)
		defaultRegion := "cn-hongkong"
		fmt.Print("Enter region (" + defaultRegion + "): ")
		region, err = reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		region = strings.TrimSpace(region)
		if region == "" {
			region = defaultRegion
		}
	}
	bucket := os.Getenv("OSS_BUCKET")
	if bucket == "" {
		reader := bufio.NewReader(os.Stdin)
		for bucket == "" {
			fmt.Print("Enter bucket: ")
			bucket, err = reader.ReadString('\n')
			bucket = strings.TrimSpace(bucket)
			if err != nil {
				panic(err)
			}
		}
	}
	content := `package main

import (
	"strings"
)

func init() {
	encryptionKey = strings.Join([]string{`
	for i, c := range key {
		if i%8 == 0 {
			content += "\n\t\t"
		} else if i > 0 {
			content += " "
		}
		content += fmt.Sprintf(`"\x%02X",`, c)
	}
	content += `
	}, "")

	ossAccessKeyId = strings.Join([]string{`
	for i, c := range akid {
		if i%8 == 0 {
			content += "\n\t\t"
		} else if i > 0 {
			content += " "
		}
		content += fmt.Sprintf(`"%c",`, c)
	}
	content += `
	}, "")

	ossAccessKeySecret = strings.Join([]string{`
	for i, c := range aks {
		if i%8 == 0 {
			content += "\n\t\t"
		} else if i > 0 {
			content += " "
		}
		content += fmt.Sprintf(`"%c",`, c)
	}
	content += `
	}, "")

	ossPrefix = "https://` + bucket + `.oss-` + region + `.aliyuncs.com"

	ossBucket = "` + bucket + `"
}
`
	writeFile("upcert/key.go", content)
	writeFile("getcert/key.go", content)
}

func writeFile(file, content string) {
	err := ioutil.WriteFile(file, []byte(content), 0644)
	if err != nil {
		panic(err)
	}
	log.Println("written", file)
}

func getAccessKey() (akid string, aks string, err error) {
	var home string
	home, err = os.UserHomeDir()
	if err != nil {
		return
	}
	file := filepath.Join(home, ".aliyun", "config.json")
	var fileContent []byte
	fileContent, err = ioutil.ReadFile(file)
	if err != nil {
		return
	}
	var config struct {
		Profiles []struct {
			AccessKeyId     string `json:"access_key_id"`
			AccessKeySecret string `json:"access_key_secret"`
		} `json:"profiles"`
	}
	err = json.Unmarshal(fileContent, &config)
	if err != nil {
		return
	}
	if len(config.Profiles) > 0 {
		akid = config.Profiles[0].AccessKeyId
		aks = config.Profiles[0].AccessKeySecret
	}
	return
}

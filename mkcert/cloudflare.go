package main

import (
	"encoding/json"
	"log"
	"os/exec"
	"strings"
)

type (
	cloudflare struct{}
)

func (_ cloudflare) getListOfDomains() []string {
	cmd := exec.Command("cloudflare", "--raw", "ls")
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	var result []struct {
		Name string `json:"name"`
	}
	err = json.Unmarshal(out, &result)
	if err != nil {
		log.Fatal(err)
	}

	domains := []string{}
	for _, d := range result {
		domains = append(domains, d.Name)
	}
	return domains
}

func (_ cloudflare) getRecordIdsFor(domain, dname, dtype string) []string {
	cmd := exec.Command("cloudflare", "--raw", "records", domain)
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	var result []struct {
		Id   string `json:"id"`
		Type string `json:"type"`
		Name string `json:"name"`
	}
	err = json.Unmarshal(out, &result)
	if err != nil {
		log.Fatal(err)
	}
	ids := []string{}
	for _, d := range result {
		withoutRoot := strings.TrimSuffix(strings.TrimSuffix(d.Name, domain), ".")
		if withoutRoot == dname && d.Type == dtype {
			ids = append(ids, d.Id)
		}
	}
	return ids
}

func (_ cloudflare) addNewRecord(domain, dname, dtype, dvalue string) string {
	cmd := exec.Command("cloudflare", "--raw", "addrecord", domain, dname, dtype, dvalue)
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	var result struct {
		Result struct {
			Id string `json:"id"`
		} `json:"result"`
	}
	err = json.Unmarshal(out, &result)
	if err != nil {
		log.Fatal(err)
	}
	return result.Result.Id
}

func (_ cloudflare) deleteRecord(domain, id string) {
	cmd := exec.Command("cloudflare", "delrecord", domain, id)
	_, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
}

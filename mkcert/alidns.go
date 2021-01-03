package main

import (
	"encoding/json"
	"log"
	"os/exec"
)

type (
	alidns struct{}
)

func (_ alidns) getListOfDomains() []string {
	cmd := exec.Command("aliyun", "alidns", "DescribeDomains")
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	var result struct {
		Domains struct {
			Domain []struct {
				DomainName string
			}
		}
	}
	err = json.Unmarshal(out, &result)
	if err != nil {
		log.Fatal(err)
	}

	domains := []string{}
	for _, d := range result.Domains.Domain {
		domains = append(domains, d.DomainName)
	}
	return domains
}

func (_ alidns) getRecordIdsFor(domain, dname, dtype string) []string {
	cmd := exec.Command("aliyun", "alidns", "DescribeDomainRecords", "--DomainName", domain)
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	var result struct {
		DomainRecords struct {
			Record []struct {
				RecordId string
				RR       string
				Type     string
			}
		}
	}
	err = json.Unmarshal(out, &result)
	if err != nil {
		log.Fatal(err)
	}
	ids := []string{}
	for _, d := range result.DomainRecords.Record {
		if d.RR == dname && d.Type == dtype {
			ids = append(ids, d.RecordId)
		}
	}
	return ids
}

func (_ alidns) addNewRecord(domain, dname, dtype, dvalue string) string {
	cmd := exec.Command("aliyun", "alidns", "AddDomainRecord", "--DomainName", domain,
		"--RR", dname, "--Type", dtype, "--Value", dvalue)
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	var result struct {
		RecordId string
	}
	err = json.Unmarshal(out, &result)
	if err != nil {
		log.Fatal(err)
	}
	return result.RecordId
}

func (_ alidns) deleteRecord(domain, id string) {
	cmd := exec.Command("aliyun", "alidns", "DeleteDomainRecord", "--RecordId", id)
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	var result struct {
		RecordId string
	}
	err = json.Unmarshal(out, &result)
	if err != nil {
		log.Fatal(err)
	}
	if result.RecordId != id {
		log.Fatal(string(out))
	}
}

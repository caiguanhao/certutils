package dns

import (
	"encoding/json"
	"log"
	"os/exec"
	"strconv"
)

type (
	Alidns struct{}
)

var _ DNS = (*Alidns)(nil)

func (_ Alidns) GetListOfDomains() []string {
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

func (a Alidns) GetRecords(domain string) []Record {
	return a.getRecords(domain, 1)
}

func (a Alidns) getRecords(domain string, page int) (records []Record) {
	cmd := exec.Command("aliyun", "alidns", "DescribeDomainRecords",
		"--DomainName", domain, "--PageNumber", strconv.Itoa(page))
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
				Value    string
			}
		}
		PageNumber int
		PageSize   int
		TotalCount int
	}
	err = json.Unmarshal(out, &result)
	if err != nil {
		log.Fatal(err)
	}
	for _, d := range result.DomainRecords.Record {
		fullName := domain
		if d.RR != "@" {
			fullName = d.RR + "." + fullName
		}
		records = append(records, Record{
			Id:       d.RecordId,
			Type:     d.Type,
			Name:     d.RR,
			FullName: fullName,
			Content:  d.Value,
		})
	}
	totalPages := result.TotalCount/result.PageSize + 1
	if result.PageNumber < totalPages {
		records = append(records, a.getRecords(domain, result.PageNumber+1)...)
	}
	return
}

func (_ Alidns) GetRecordIdsFor(domain, dname, dtype string) []string {
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

func (_ Alidns) AddNewRecord(domain, dname, dtype, dvalue string) string {
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

func (_ Alidns) DeleteRecord(domain, id string) {
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

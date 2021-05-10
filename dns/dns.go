package dns

type (
	DNS interface {
		GetListOfDomains() []string
		GetRecords(domain string) []Record
		GetRecordIdsFor(domain, dname, dtype string) []string
		AddNewRecord(domain, dname, dtype, dvalue string) string
		DeleteRecord(domain, id string)
	}

	Record struct {
		Id       string
		Type     string
		Name     string
		FullName string
		Content  string
	}
)

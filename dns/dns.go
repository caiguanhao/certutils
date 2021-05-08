package dns

type (
	DNS interface {
		GetListOfDomains() []string
		GetRecordIdsFor(domain, dname, dtype string) []string
		AddNewRecord(domain, dname, dtype, dvalue string) string
		DeleteRecord(domain, id string)
	}
)

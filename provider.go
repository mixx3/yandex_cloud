package yandex_cloud

import (
	"context"
	"strings"

	"github.com/libdns/libdns"
)

type Provider struct {
	AuthAPIToken string `json:"auth_api_token"`
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	records, err := getAllRecords(ctx, p.AuthAPIToken, unFQDN(zone))
	if err != nil {
		return nil, err
	}

	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var appendedRecords []libdns.Record

	for _, record := range records {
		newRecord, err := updateRecord(ctx, p.AuthAPIToken, unFQDN(zone), record, "ADD")
		if err != nil {
			return nil, err
		}
		appendedRecords = append(appendedRecords, newRecord)
	}

	return appendedRecords, nil
}

// DeleteRecords deletes the records from the zone.
func (p *Provider) DeleteRecords(ctx context.Context, _ string, records []libdns.Record) ([]libdns.Record, error) {
	for _, record := range records {
		_, err := updateRecord(ctx, p.AuthAPIToken, "", record, "DELETE")
		if err != nil {
			return nil, err
		}
	}

	return records, nil
}

// SetRecords sets the records in the zone, either by updating existing records
// or creating new ones. It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var setRecords []libdns.Record

	for _, record := range records {
		setRecord, err := upsertRecord(ctx, p.AuthAPIToken, unFQDN(zone), record, "REPLACE")
		if err != nil {
			return setRecords, err
		}
		setRecords = append(setRecords, setRecord)
	}

	return setRecords, nil
}

// unFQDN trims any trailing "." from fqdn. Hetzner's API does not use FQDNs.
func unFQDN(fqdn string) string {
	return strings.TrimSuffix(fqdn, ".")
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)

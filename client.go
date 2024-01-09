package yandex_cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/libdns/libdns"
)

type getAllZonesResponse struct {
	DnsZones []zone `json:"dnsZones"`
}

type getAllRecordsResponse struct {
	Records []record `json:"recordSets"`
}

type createRecordResponse struct {
	ID          string `json:"id,omitempty"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	CreatedBy   string `json:"createdBy"`
	ModifiedAt  string `json:"modifiedAt"`
	IsDone      bool   `json:"done"`
}

type updateRecordResponse struct {
	ID          string `json:"id,omitempty"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	CreatedBy   string `json:"createdBy"`
	ModifiedAt  string `json:"modifiedAt"`
	IsDone      bool   `json:"done"`
}

type upsertRecordsBody struct {
	Deletions    []record `json:"deletions"`
	Replacements []record `json:"replacements"`
	Merges       []record `json:"merges"`
}

type updateRecordsBody struct {
	Deletions []record `json:"deletions"`
	Additions []record `json:"additions"`
}

type record struct {
	ID   string   `json:"id,omitempty"`
	Type string   `json:"type"`
	Name string   `json:"name"`
	Data []string `json:"data"`
	TTL  int      `json:"ttl"`
}

type zone struct {
	ID       string `json:"id,omitempty"`
	FolderId string `json:"folderId,omitempty"`
	Zone     string `json:"zone,omitempty"`
	Type     string `json:"type"`
	Name     string `json:"name"`
}

func doRequest(token string, request *http.Request) ([]byte, error) {
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("%s (%d)", http.StatusText(response.StatusCode), response.StatusCode)
	}

	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func getZoneID(ctx context.Context, token string, zone string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://dns.api.cloud.yandex.net/dns/v1/zones?name=%s", url.QueryEscape(zone)), nil)
	data, err := doRequest(token, req)
	if err != nil {
		return "", err
	}

	result := getAllZonesResponse{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	if len(result.DnsZones) > 1 {
		return "", errors.New("zone is ambiguous")
	}

	return result.DnsZones[0].ID, nil
}

func getAllRecords(ctx context.Context, token string, zone string) ([]libdns.Record, error) {
	zoneID, err := getZoneID(ctx, token, zone)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://dns.api.cloud.yandex.net/dns/v1/zones/%s:listRecordSets", zoneID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)

	data, err := doRequest(token, req)
	if err != nil {
		return nil, err
	}

	result := getAllRecordsResponse{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	records := []libdns.Record{}
	for _, r := range result.Records {
		records = append(records, libdns.Record{
			ID:    r.ID,
			Type:  r.Type,
			Name:  r.Name,
			Value: r.Data[0],
			TTL:   time.Duration(r.TTL) * time.Second,
		})
	}

	return records, nil
}

func upsertRecord(ctx context.Context, token string, zone string, r libdns.Record, method string) (libdns.Record, error) {
	zoneID, err := getZoneID(ctx, token, zone)
	if err != nil {
		return libdns.Record{}, err
	}

	reqData := upsertRecordsBody{
		Replacements: []record{},
		Deletions:    []record{},
		Merges:       []record{},
	}
	recordData := record{
		Type: r.Type,
		Name: normalizeRecordName(r.Name, zone),
		Data: []string{r.Value},
		TTL:  int(r.TTL.Seconds()),
	}

	if method == "DELETE" {
		reqData.Replacements = append(reqData.Replacements, recordData)
	}
	if method == "REPLACE" {
		reqData.Deletions = append(reqData.Deletions, recordData)
	}
	if method == "MERGE" {
		reqData.Merges = append(reqData.Merges, recordData)
	}

	reqBuffer, err := json.Marshal(reqData)
	if err != nil {
		return libdns.Record{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("https://dns.api.cloud.yandex.net/dns/v1/zones/%s:upsertRecordSets", zoneID), bytes.NewBuffer(reqBuffer))
	data, err := doRequest(token, req)
	if err != nil {
		return libdns.Record{}, err
	}

	result := createRecordResponse{}
	if err := json.Unmarshal(data, &result); err != nil {
		return libdns.Record{}, err
	}

	return libdns.Record{
		ID:    result.ID,
		Type:  r.Type,
		Name:  r.Name,
		Value: r.Value,
		TTL:   time.Duration(r.TTL) * time.Second,
	}, nil
}

func updateRecord(ctx context.Context, token string, zone string, r libdns.Record, method string) (libdns.Record, error) {
	zoneID, err := getZoneID(ctx, token, zone)
	if err != nil {
		return libdns.Record{}, err
	}

	reqData := updateRecordsBody{
		Additions: []record{},
		Deletions: []record{},
	}
	recordData := record{
		Type: r.Type,
		Name: normalizeRecordName(r.Name, zone),
		Data: []string{r.Value},
		TTL:  int(r.TTL.Seconds()),
	}

	if method == "DELETE" {
		reqData.Deletions = append(reqData.Deletions, recordData)
	}
	if method == "ADD" {
		reqData.Additions = append(reqData.Additions, recordData)
	}

	reqBuffer, err := json.Marshal(reqData)
	if err != nil {
		return libdns.Record{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("https://dns.api.cloud.yandex.net/dns/v1/zones/%s:updateRecordSets", zoneID), bytes.NewBuffer(reqBuffer))
	data, err := doRequest(token, req)
	if err != nil {
		return libdns.Record{}, err
	}

	result := createRecordResponse{}
	if err := json.Unmarshal(data, &result); err != nil {
		return libdns.Record{}, err
	}

	return libdns.Record{
		ID:    result.ID,
		Type:  r.Type,
		Name:  r.Name,
		Value: r.Value,
		TTL:   time.Duration(r.TTL) * time.Second,
	}, nil
}

func normalizeRecordName(recordName string, zone string) string {
	// Workaround for https://github.com/caddy-dns/hetzner/issues/3
	// Can be removed after https://github.com/libdns/libdns/issues/12
	normalized := unFQDN(recordName)
	normalized = strings.TrimSuffix(normalized, unFQDN(zone))
	return unFQDN(normalized)
}

package yandex_cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strconv"
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
	Response 	upsertRecordsBody `json:"response"`
	CreatedAt   string `json:"createdAt"`
	CreatedBy   string `json:"createdBy"`
	ModifiedAt  string `json:"modifiedAt"`
	IsDone      bool   `json:"done"`
}

type updateRecordResponse struct {
	ID          string `json:"id,omitempty"`
	Description string `json:"description"`
	Response updateRecordsBody `json:"response"`
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
	TTL  string      `json:"ttl"`
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
	reqDump, err := httputil.DumpRequestOut(request, true)
	
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	resDump, err := httputil.DumpResponse(response, true)
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("%s\n%s", string(resDump) , string(reqDump))
	}

	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func getZoneName(ctx context.Context, token string, zoneID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://dns.api.cloud.yandex.net/dns/v1/zones/%s", zoneID), nil)
	data, err := doRequest(token, req)
	if err != nil {
		return "", err
	}

	result := zone{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	return result.Name, nil
}

func getAllRecords(ctx context.Context, token string, zoneID string) ([]libdns.Record, error) {
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
		intTtl, err := strconv.Atoi(r.TTL)
		if err != nil{
			return []libdns.Record{}, err
		}
		records = append(records, libdns.Record{
			ID:    r.ID,
			Type:  r.Type,
			Name:  r.Name,
			Value: r.Data[0],
			TTL:   time.Duration(intTtl) * time.Second,
		})
	}

	return records, nil
}

func upsertRecords(ctx context.Context, token string, zoneID string, rs []libdns.Record, method string) ([]libdns.Record, error) {
	// zoneID, err := getZoneID(ctx, token, zone)
	// if err != nil {
	// 	return libdns.Record{}, err
	// }

	reqData := upsertRecordsBody{
		Replacements: []record{},
		Deletions:    []record{},
		Merges:       []record{},
	}
	zoneName, err := getZoneName(ctx, token, zoneID)
	if err != nil{
		return []libdns.Record{}, err
	}
	for _, r := range rs{
		recordData := record{
		Type: r.Type,
		Name: normalizeRecordName(r.Name, zoneName),
		Data: []string{r.Value},
		TTL:  fmt.Sprint(r.TTL.Seconds()),
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
	}
	reqBuffer, err := json.Marshal(reqData)
	if err != nil {
		return []libdns.Record{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("https://dns.api.cloud.yandex.net/dns/v1/zones/%s:upsertRecordSets", zoneID), bytes.NewBuffer(reqBuffer))
	data, err := doRequest(token, req)
	if err != nil {
		return []libdns.Record{}, err
	}

	result := createRecordResponse{}
	if err := json.Unmarshal(data, &result); err != nil {
		return []libdns.Record{}, err
	}
	resultList := []record{}
	if method == "DELETE" {
		resultList = result.Response.Deletions
	}
	if method == "REPLACE" {
		resultList = result.Response.Replacements
	}
	if method == "MERGE" {
		resultList = result.Response.Merges
	}
	res := make([]libdns.Record, 0)
	for _, r := range resultList{
		intTtl, _ := strconv.Atoi(r.TTL)
		res = append(res, libdns.Record{
		ID:    result.ID,
		Type:  r.Type,
		Name:  r.Name,
		Value: r.Data[0],
		TTL:   time.Duration(intTtl) * time.Second,
		})
	}
	return res, nil
}

func updateRecords(ctx context.Context, token string, zoneID string, rs []libdns.Record, method string) ([]libdns.Record, error) {
	zoneName, err := getZoneName(ctx, token, zoneID)
	if err != nil {
		return []libdns.Record{}, err
	}

	reqData := updateRecordsBody{
		Additions: []record{},
		Deletions: []record{},
	}

	for _, r := range rs{
		recordData := record{
			Type: r.Type,
			Name: normalizeRecordName(r.Name, zoneName),
			Data: []string{r.Value},
			TTL:  fmt.Sprint(r.TTL.Seconds()),
		}
		if method == "DELETE" {
			reqData.Deletions = append(reqData.Deletions, recordData)
		}
		if method == "ADD" {
			reqData.Additions = append(reqData.Additions, recordData)
		}
	}

	reqBuffer, err := json.Marshal(reqData)
	if err != nil {
		return []libdns.Record{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("https://dns.api.cloud.yandex.net/dns/v1/zones/%s:updateRecordSets", zoneID), bytes.NewBuffer(reqBuffer))
	data, err := doRequest(token, req)
	if err != nil {
		return []libdns.Record{}, err
	}

	result := updateRecordResponse{}
	if err := json.Unmarshal(data, &result); err != nil {
		return []libdns.Record{}, err
	}
	resultList := []record{}
	if method == "DELETE" {
		resultList = result.Response.Deletions
	}
	if method == "ADD" {
		resultList = result.Response.Additions
	}
	res := make([]libdns.Record, 0)
	for _, r := range resultList{
		intTtl, _ := strconv.Atoi(r.TTL)
		res = append(res, libdns.Record{
		ID:    result.ID,
		Type:  r.Type,
		Name:  r.Name,
		Value: r.Data[0],
		TTL:   time.Duration(intTtl) * time.Second,
		})
	}
	return res, nil
}

func normalizeRecordName(recordName string, zone string) string {
	// Workaround for https://github.com/caddy-dns/hetzner/issues/3
	// Can be removed after https://github.com/libdns/libdns/issues/12
	normalized := unFQDN(recordName)
	normalized = strings.TrimSuffix(normalized, unFQDN(zone))
	return unFQDN(normalized)
}

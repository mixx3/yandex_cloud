package yandex_cloud_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
	"github.com/libdns/libdns"
	"github.com/mixx3/yandex_cloud"
)

var (
	envToken = ""
	envZoneID  = ""
	envZoneName = ""
	ttl      = time.Duration(120 * time.Second)
)

type testRecordsCleanup = func()

func setupTestRecords(t *testing.T, p *yandex_cloud.Provider) ([]libdns.Record, testRecordsCleanup) {
	testRecords := []libdns.Record{
		{
			Type:  "TXT",
			Name:  "test1",
			Value: "test1",
			TTL:   ttl,
		}, {
			Type:  "TXT",
			Name:  "test2",
			Value: "test2",
			TTL:   ttl,
		}, {
			Type:  "TXT",
			Name:  "test3",
			Value: "test3",
			TTL:   ttl,
		},
	}

	records, err := p.AppendRecords(context.TODO(), envZoneID, testRecords)
	if err != nil {
		t.Fatal(err)
		return nil, func() {}
	}

	return records, func() {
		cleanupRecords(t, p, records)
	}
}

func cleanupRecords(t *testing.T, p *yandex_cloud.Provider, r []libdns.Record) {
	_, err := p.DeleteRecords(context.TODO(),  envZoneID, r)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
}

func TestMain(m *testing.M) {
	envToken = os.Getenv("IAM_TOKEN")
	envZoneName = os.Getenv("ZONE_NAME")
	envZoneID = os.Getenv("ZONE_ID")

	if len(envToken) == 0 || len(envZoneName) == 0 || len(envZoneID) == 0 {
		fmt.Println(`Please notice that this test runs agains the public Hetzner DNS Api, so you sould
never run the test with a zone, used in production.
To run this test, you have to specify 'LIBDNS_HETZNER_TEST_TOKEN' and 'LIBDNS_HETZNER_TEST_ZONE'.
Example: "IAM_TOKEN="123" LIBDNS_HETZNER_TEST_ZONE="my-domain.com" go test ./... -v`)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func Test_AppendRecords(t *testing.T) {
	p := &yandex_cloud.Provider{
		AuthAPIToken: envToken,
	}

	testCases := []struct {
		records  []libdns.Record
		expected []libdns.Record
	}{
		{
			// relative name
			records: []libdns.Record{
				{Type: "TXT", Name: "123.test", Value: "123", TTL: ttl},
			},
			expected: []libdns.Record{
				{Type: "TXT", Name: "123.test", Value: "123", TTL: ttl},
			},
		},
	}

	for _, c := range testCases {
		func() {
			result, err := p.AppendRecords(context.TODO(), envZoneID, c.records)
			if err != nil {
				t.Fatal(err)
			}
			defer cleanupRecords(t, p, result)

			if len(result) != len(c.records) {
				t.Fatalf("len(resilt) != len(c.records) => %d != %d", len(c.records), len(result))
			}

			for k, r := range result {
				if len(result[k].ID) == 0 {
					t.Fatalf("len(result[%d].ID) == 0", k)
				}
				if r.Type != c.expected[k].Type {
					t.Fatalf("r.Type != c.exptected[%d].Type => %s != %s", k, r.Type, c.expected[k].Type)
				}
				if r.Name != c.expected[k].Name {
					t.Fatalf("r.Name != c.exptected[%d].Name => %s != %s", k, r.Name, c.expected[k].Name)
				}
				if r.Value != c.expected[k].Value {
					t.Fatalf("r.Value != c.exptected[%d].Value => %s != %s", k, r.Value, c.expected[k].Value)
				}
				if r.TTL != c.expected[k].TTL {
					t.Fatalf("r.TTL != c.exptected[%d].TTL => %s != %s", k, r.TTL, c.expected[k].TTL)
				}
			}
		}()
	}
}

func Test_DeleteRecords(t *testing.T) {
	p := &yandex_cloud.Provider{
		AuthAPIToken: envToken,
	}

	testRecords, cleanupFunc := setupTestRecords(t, p)
	defer cleanupFunc()

	records, err := p.GetRecords(context.TODO(), envZoneID)
	if err != nil {
		t.Fatal(err)
	}

	if len(records) < len(testRecords) {
		t.Fatalf("len(records) < len(testRecords) => %d < %d", len(records), len(testRecords))
	}

	for _, testRecord := range testRecords {
		var foundRecord *libdns.Record
		for _, record := range records {
			if testRecord.Name + "." + envZoneName == record.Name {
				foundRecord = &testRecord
			}
		}

		if foundRecord == nil {
			t.Fatalf("Record not found => %s", testRecord.Name)
		}
	}
}

func Test_GetRecords(t *testing.T) {
	p := &yandex_cloud.Provider{
		AuthAPIToken: envToken,
	}

	testRecords, cleanupFunc := setupTestRecords(t, p)
	defer cleanupFunc()

	records, err := p.GetRecords(context.TODO(), envZoneID)
	if err != nil {
		t.Fatal(err)
	}

	if len(records) < len(testRecords) {
		t.Fatalf("len(records) < len(testRecords) => %d < %d", len(records), len(testRecords))
	}

	for _, testRecord := range testRecords {
		var foundRecord *libdns.Record
		for _, record := range records {
			if testRecord.Name + "." + envZoneName == record.Name {
				foundRecord = &testRecord
			}
		}

		if foundRecord == nil {
			t.Fatalf("Record not found => %s", testRecord.Name)
		}
	}
}

func Test_SetRecords(t *testing.T) {
	p := &yandex_cloud.Provider{
		AuthAPIToken: envToken,
	}

	existingRecords, _ := setupTestRecords(t, p)
	newTestRecords := []libdns.Record{
		{
			Type:  "TXT",
			Name:  "new_test1",
			Value: "new_test1",
			TTL:   ttl,
		},
		{
			Type:  "TXT",
			Name:  "new_test2",
			Value: "new_test2",
			TTL:   ttl,
		},
	}

	allRecords := append(existingRecords, newTestRecords...)

	records, err := p.SetRecords(context.TODO(), envZoneID, allRecords)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupRecords(t, p, allRecords)

	if len(records) != len(allRecords) {
		t.Fatalf("len(records) != len(allRecords) => %d != %d", len(records), len(allRecords))
	}
}

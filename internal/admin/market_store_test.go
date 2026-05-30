package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMarketStoreSyncOfficialRegistry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v0.1/servers" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"servers": [
				{
					"server": {
						"name": "io.example/filesystem",
						"title": "Filesystem",
						"description": "File access for tests",
						"version": "1.0.0",
						"packages": [
							{
								"registryType": "npm",
								"identifier": "@example/mcp-filesystem",
								"version": "1.0.0",
								"packageArguments": [
									"--root",
									{
										"name": "/tmp"
									}
								],
								"runtimeArguments": [
									{
										"name": "--inspect"
									}
								],
								"environmentVariables": [
									{
										"name": "FILESYSTEM_ROOT",
										"description": "Root path",
										"isRequired": true
									}
								]
							}
						]
					},
					"_meta": {
						"io.modelcontextprotocol.registry/official": {
							"status": "active",
							"isLatest": true
						}
					}
				}
			],
			"metadata": {
				"count": 1
			}
		}`))
	}))
	defer srv.Close()

	store := newMarketStore()
	store.registerAdapter(&officialRegistryAdapter{
		sourceID: "official",
		baseURL:  srv.URL,
		client:   srv.Client(),
	})

	job, err := store.syncSource(context.Background(), "official")
	if err != nil {
		t.Fatalf("sync source: %v", err)
	}
	if job.Status != "success" {
		t.Fatalf("expected success job, got %s", job.Status)
	}

	items := store.listPackages("io.example/filesystem", "official", "", "", false)
	if len(items) == 0 {
		t.Fatal("expected synced package")
	}
	if items[0].CanonicalName != "io.example/filesystem" {
		t.Fatalf("unexpected canonical name: %s", items[0].CanonicalName)
	}
	if items[0].Installability != installabilityConfigRequired {
		t.Fatalf("expected config-required package, got %s", items[0].Installability)
	}
	if len(items[0].InstallOptions) == 0 || items[0].InstallOptions[0].Type != "npx" {
		t.Fatalf("expected npx install option: %#v", items[0].InstallOptions)
	}
	if len(items[0].InstallOptions[0].RequiredEnv) != 1 {
		t.Fatalf("expected required env option: %#v", items[0].InstallOptions[0].RequiredEnv)
	}
	if got := items[0].InstallOptions[0].Args; len(got) != 4 || got[2] != "--root" || got[3] != "/tmp" {
		t.Fatalf("expected object package args to be normalized: %#v", got)
	}
}

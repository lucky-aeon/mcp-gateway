package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func defaultMarketAdapters(client *http.Client) []MarketSourceAdapter {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return []MarketSourceAdapter{
		&officialRegistryAdapter{
			sourceID: "official",
			baseURL:  "https://registry.modelcontextprotocol.io",
			client:   client,
		},
	}
}

type officialRegistryAdapter struct {
	sourceID string
	baseURL  string
	client   *http.Client
}

func (a *officialRegistryAdapter) SourceID() string { return a.sourceID }
func (a *officialRegistryAdapter) Kind() string     { return "official_registry" }

func (a *officialRegistryAdapter) FetchPage(ctx context.Context, req MarketFetchRequest) (MarketFetchPage, error) {
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	u, err := url.Parse(a.baseURL + "/v0.1/servers")
	if err != nil {
		return MarketFetchPage{}, err
	}
	q := u.Query()
	q.Set("limit", fmt.Sprintf("%d", limit))
	if req.Cursor != "" {
		q.Set("cursor", req.Cursor)
	}
	if req.UpdatedSince != nil && !req.UpdatedSince.IsZero() {
		q.Set("updated_since", req.UpdatedSince.UTC().Format(time.RFC3339Nano))
	}
	u.RawQuery = q.Encode()

	var payload officialRegistryListResponse
	if err := fetchJSON(ctx, a.client, u.String(), &payload); err != nil {
		return MarketFetchPage{}, err
	}
	items := make([]MarketPackage, 0, len(payload.Servers))
	for _, item := range payload.Servers {
		pkg, ok := officialRegistryToMarketPackage(item, a.sourceID)
		if ok {
			items = append(items, pkg)
		}
	}
	return MarketFetchPage{
		Items:      items,
		NextCursor: payload.Metadata.NextCursor,
		HasMore:    payload.Metadata.NextCursor != "",
	}, nil
}

func (a *officialRegistryAdapter) FetchDetail(ctx context.Context, externalID string) (*MarketPackage, error) {
	if externalID == "" {
		return nil, fmt.Errorf("external id is required")
	}
	escapedName := url.PathEscape(externalID)
	endpoint := fmt.Sprintf("%s/v0.1/servers/%s/versions/latest", a.baseURL, escapedName)
	var payload officialRegistryServerEnvelope
	if err := fetchJSON(ctx, a.client, endpoint, &payload); err != nil {
		return nil, err
	}
	pkg, ok := officialRegistryToMarketPackage(payload, a.sourceID)
	if !ok {
		return nil, fmt.Errorf("package is not active")
	}
	return &pkg, nil
}

type officialRegistryListResponse struct {
	Servers  []officialRegistryServerEnvelope `json:"servers"`
	Metadata struct {
		NextCursor string `json:"nextCursor"`
		Count      int    `json:"count"`
	} `json:"metadata"`
}

type officialRegistryServerEnvelope struct {
	Server officialRegistryServer `json:"server"`
	Meta   map[string]interface{} `json:"_meta"`
}

type officialRegistryServer struct {
	Name        string                         `json:"name"`
	Title       string                         `json:"title"`
	Description string                         `json:"description"`
	Version     string                         `json:"version"`
	Packages    []officialRegistryPackageEntry `json:"packages"`
	Remotes     []officialRegistryRemote       `json:"remotes"`
	Repository  struct {
		URL string `json:"url"`
	} `json:"repository"`
	Homepage string   `json:"homepage"`
	License  string   `json:"license"`
	Tags     []string `json:"tags"`
}

type officialRegistryPackageEntry struct {
	RegistryType string                 `json:"registryType"`
	Identifier   string                 `json:"identifier"`
	Version      string                 `json:"version"`
	RuntimeHint  string                 `json:"runtimeHint"`
	Transport    map[string]interface{} `json:"transport"`
	RuntimeArgs  json.RawMessage        `json:"runtimeArguments"`
	PackageArgs  json.RawMessage        `json:"packageArguments"`
	Env          json.RawMessage        `json:"environmentVariables"`
}

type officialRegistryRemote struct {
	Type      string                            `json:"type"`
	URL       string                            `json:"url"`
	Variables map[string]officialRemoteVariable `json:"variables"`
}

type officialRemoteVariable struct {
	Description string      `json:"description"`
	IsRequired  bool        `json:"isRequired"`
	Default     interface{} `json:"default"`
	IsSecret    bool        `json:"isSecret"`
}

func officialRegistryToMarketPackage(item officialRegistryServerEnvelope, sourceID string) (MarketPackage, bool) {
	status := officialRegistryStatus(item.Meta)
	if status == "deleted" {
		return MarketPackage{}, false
	}
	if latest, ok := officialRegistryIsLatest(item.Meta); ok && !latest {
		return MarketPackage{}, false
	}
	server := item.Server
	if server.Name == "" {
		return MarketPackage{}, false
	}
	options := make([]MarketInstallOption, 0, len(server.Remotes)+len(server.Packages))
	for _, remote := range server.Remotes {
		if remote.URL == "" {
			continue
		}
		options = append(options, MarketInstallOption{
			Type:        "remote",
			URL:         remote.URL,
			Transport:   normalizeRemoteTransport(remote.Type),
			RequiredEnv: remoteVariablesToEnv(remote.Variables),
			SourceID:    sourceID,
			Confidence:  "high",
		})
	}
	for _, p := range server.Packages {
		opt, ok := packageInstallOption(p, sourceID)
		if ok {
			options = append(options, opt)
		}
	}
	title := server.Title
	if title == "" {
		title = server.Name
	}
	repo := server.Repository.URL
	return MarketPackage{
		CanonicalName:  server.Name,
		Name:           lastNamePart(server.Name),
		Title:          title,
		Description:    server.Description,
		Version:        server.Version,
		Tags:           server.Tags,
		Repository:     repo,
		Homepage:       server.Homepage,
		License:        server.License,
		Verified:       true,
		InstallOptions: options,
		SourceRefs: []MarketSourceRef{{
			SourceID:   sourceID,
			ExternalID: server.Name,
			Version:    server.Version,
			Meta:       item.Meta,
		}},
		RawMeta: map[string]interface{}{
			"registry_status": status,
		},
	}, true
}

func packageInstallOption(p officialRegistryPackageEntry, sourceID string) (MarketInstallOption, bool) {
	if p.Identifier == "" {
		return MarketInstallOption{}, false
	}
	env, requiredEnv := parseOfficialPackageEnv(p.Env)
	runtimeArgs := parseOfficialPackageArgs(p.RuntimeArgs)
	packageArgs := parseOfficialPackageArgs(p.PackageArgs)
	switch strings.ToLower(p.RegistryType) {
	case "npm":
		args := []string{"-y", p.Identifier}
		args = append(args, packageArgs...)
		return MarketInstallOption{
			Type:        "npx",
			Command:     "npx",
			Args:        args,
			Env:         env,
			RequiredEnv: requiredEnv,
			PackageName: p.Identifier,
			SourceID:    sourceID,
			Confidence:  "high",
		}, true
	case "pypi":
		args := []string{p.Identifier}
		args = append(args, packageArgs...)
		args = append(runtimeArgs, args...)
		return MarketInstallOption{
			Type:        "uvx",
			Command:     "uvx",
			Args:        args,
			Env:         env,
			RequiredEnv: requiredEnv,
			PackageName: p.Identifier,
			SourceID:    sourceID,
			Confidence:  "high",
		}, true
	case "oci", "docker":
		return MarketInstallOption{
			Type:        "docker",
			Command:     "docker",
			Args:        []string{"run", "--rm", "-i", p.Identifier},
			Image:       p.Identifier,
			PackageName: p.Identifier,
			SourceID:    sourceID,
			Confidence:  "high",
		}, true
	default:
		return MarketInstallOption{
			Type:        "manual",
			PackageName: p.Identifier,
			SourceID:    sourceID,
			Confidence:  "low",
			Raw: map[string]interface{}{
				"registry_type": p.RegistryType,
			},
		}, true
	}
}

func parseOfficialPackageArgs(raw json.RawMessage) []string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}

	var stringsOnly []string
	if err := json.Unmarshal(raw, &stringsOnly); err == nil {
		return stringsOnly
	}

	var mixed []interface{}
	if err := json.Unmarshal(raw, &mixed); err != nil {
		return nil
	}
	out := make([]string, 0, len(mixed))
	for _, item := range mixed {
		switch v := item.(type) {
		case string:
			out = append(out, v)
		case map[string]interface{}:
			for _, key := range []string{"value", "default", "name"} {
				if s, ok := v[key].(string); ok && s != "" {
					out = append(out, s)
					break
				}
			}
		}
	}
	return out
}

func parseOfficialPackageEnv(raw json.RawMessage) (map[string]string, []MarketEnvVarSpec) {
	if len(raw) == 0 || string(raw) == "null" {
		return map[string]string{}, nil
	}

	var envMap map[string]interface{}
	if err := json.Unmarshal(raw, &envMap); err == nil {
		return asStringMap(envMap), nil
	}

	var envList []struct {
		Name        string      `json:"name"`
		Description string      `json:"description"`
		IsRequired  bool        `json:"isRequired"`
		Required    bool        `json:"required"`
		Default     interface{} `json:"default"`
		IsSecret    bool        `json:"isSecret"`
		Secret      bool        `json:"secret"`
	}
	if err := json.Unmarshal(raw, &envList); err != nil {
		return map[string]string{}, nil
	}

	required := make([]MarketEnvVarSpec, 0, len(envList))
	defaults := make(map[string]string)
	for _, item := range envList {
		if item.Name == "" {
			continue
		}
		defaultValue := ""
		if item.Default != nil {
			defaultValue = fmt.Sprintf("%v", item.Default)
			defaults[item.Name] = defaultValue
		}
		required = append(required, MarketEnvVarSpec{
			Name:        item.Name,
			Description: item.Description,
			Required:    item.IsRequired || item.Required,
			Default:     defaultValue,
			Secret:      item.IsSecret || item.Secret,
		})
	}
	return defaults, required
}

func officialRegistryStatus(meta map[string]interface{}) string {
	raw, _ := meta["io.modelcontextprotocol.registry/official"].(map[string]interface{})
	status, _ := raw["status"].(string)
	return status
}

func officialRegistryIsLatest(meta map[string]interface{}) (bool, bool) {
	raw, _ := meta["io.modelcontextprotocol.registry/official"].(map[string]interface{})
	latest, ok := raw["isLatest"].(bool)
	return latest, ok
}

func remoteVariablesToEnv(vars map[string]officialRemoteVariable) []MarketEnvVarSpec {
	if len(vars) == 0 {
		return nil
	}
	out := make([]MarketEnvVarSpec, 0, len(vars))
	for name, spec := range vars {
		defaultValue := ""
		if spec.Default != nil {
			defaultValue = fmt.Sprintf("%v", spec.Default)
		}
		out = append(out, MarketEnvVarSpec{
			Name:        name,
			Description: spec.Description,
			Required:    spec.IsRequired,
			Default:     defaultValue,
			Secret:      spec.IsSecret,
		})
	}
	return out
}

func normalizeRemoteTransport(t string) string {
	switch strings.ToLower(t) {
	case "streamable-http", "streamhttp":
		return "streamhttp"
	default:
		return strings.ToLower(t)
	}
}

type smitheryAdapter struct {
	sourceID string
	baseURL  string
	client   *http.Client
}

func (a *smitheryAdapter) SourceID() string { return a.sourceID }
func (a *smitheryAdapter) Kind() string     { return "smithery" }

func (a *smitheryAdapter) FetchPage(ctx context.Context, req MarketFetchRequest) (MarketFetchPage, error) {
	u, err := url.Parse(a.baseURL + "/servers")
	if err != nil {
		return MarketFetchPage{}, err
	}
	q := u.Query()
	if req.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", req.Limit))
	}
	if req.Cursor != "" {
		q.Set("cursor", req.Cursor)
	}
	u.RawQuery = q.Encode()
	var payload smitheryListResponse
	if err := fetchJSON(ctx, a.client, u.String(), &payload); err != nil {
		return MarketFetchPage{}, err
	}
	items := make([]MarketPackage, 0, len(payload.Servers))
	for _, server := range payload.Servers {
		items = append(items, smitheryPackage(server, a.sourceID))
	}
	return MarketFetchPage{Items: items, HasMore: false}, nil
}

func (a *smitheryAdapter) FetchDetail(ctx context.Context, externalID string) (*MarketPackage, error) {
	if externalID == "" {
		return nil, fmt.Errorf("external id is required")
	}
	endpoint := a.baseURL + "/servers/" + strings.TrimPrefix(externalID, "/")
	var payload smitheryServer
	if err := fetchJSON(ctx, a.client, endpoint, &payload); err != nil {
		return nil, err
	}
	pkg := smitheryPackage(payload, a.sourceID)
	return &pkg, nil
}

type smitheryListResponse struct {
	Servers []smitheryServer `json:"servers"`
}

type smitheryServer struct {
	ID            string  `json:"id"`
	QualifiedName string  `json:"qualifiedName"`
	Namespace     string  `json:"namespace"`
	Slug          string  `json:"slug"`
	DisplayName   string  `json:"displayName"`
	Description   string  `json:"description"`
	IconURL       string  `json:"iconUrl"`
	Verified      bool    `json:"verified"`
	UseCount      int64   `json:"useCount"`
	Remote        bool    `json:"remote"`
	IsDeployed    bool    `json:"isDeployed"`
	Homepage      string  `json:"homepage"`
	BySmithery    bool    `json:"bySmithery"`
	Score         float64 `json:"score"`
}

func smitheryPackage(server smitheryServer, sourceID string) MarketPackage {
	name := server.QualifiedName
	if name == "" {
		name = server.ID
	}
	title := server.DisplayName
	if title == "" {
		title = name
	}
	options := []MarketInstallOption{{
		Type:       "manual",
		SourceID:   sourceID,
		Confidence: "low",
	}}
	if server.Remote && server.IsDeployed {
		options = []MarketInstallOption{{
			Type:       "manual",
			SourceID:   sourceID,
			Confidence: "medium",
			Raw: map[string]interface{}{
				"remote":      server.Remote,
				"is_deployed": server.IsDeployed,
			},
		}}
	}
	return MarketPackage{
		CanonicalName:  name,
		Name:           lastNamePart(name),
		Title:          title,
		Description:    server.Description,
		Homepage:       server.Homepage,
		Verified:       server.Verified || server.BySmithery,
		UseCount:       server.UseCount,
		Rating:         server.Score,
		InstallOptions: options,
		SourceRefs: []MarketSourceRef{{
			SourceID:   sourceID,
			ExternalID: name,
			URL:        "https://smithery.ai/servers/" + name,
			Meta: map[string]interface{}{
				"id":           server.ID,
				"remote":       server.Remote,
				"is_deployed":  server.IsDeployed,
				"by_smithery":  server.BySmithery,
				"icon_url":     server.IconURL,
				"namespace":    server.Namespace,
				"slug":         server.Slug,
				"smithery_url": "https://smithery.ai/servers/" + name,
			},
		}},
		RawMeta: map[string]interface{}{
			"icon_url":     server.IconURL,
			"by_smithery":  server.BySmithery,
			"remote":       server.Remote,
			"is_deployed":  server.IsDeployed,
			"smithery_url": "https://smithery.ai/servers/" + name,
		},
	}
}

func fetchJSON(ctx context.Context, client *http.Client, endpoint string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "mcp-gateway-market-sync/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GET %s returned %s", endpoint, resp.Status)
	}
	dec := json.NewDecoder(io.LimitReader(resp.Body, 16<<20))
	if err := dec.Decode(out); err != nil {
		return err
	}
	return nil
}

func lastNamePart(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, "/")
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

package admin

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
)

const (
	localMarketSourceID          = "local"
	installabilityInstallable    = "installable"
	installabilityConfigRequired = "config_required"
	installabilityManual         = "manual"
	installabilityUnsupported    = "unsupported"
)

type MarketSource struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Kind       string    `json:"kind"`
	URL        string    `json:"url"`
	Trusted    bool      `json:"trusted"`
	Enabled    bool      `json:"enabled"`
	Priority   int       `json:"priority"`
	AuthType   string    `json:"auth_type,omitempty"`
	Status     string    `json:"status"`
	LastSynced time.Time `json:"last_synced,omitempty"`
	LastError  string    `json:"last_error,omitempty"`
	TotalItems int       `json:"total_items"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	SyncCursor string `json:"-"`
}

type MarketSourceRef struct {
	SourceID   string                 `json:"source_id"`
	ExternalID string                 `json:"external_id"`
	URL        string                 `json:"url,omitempty"`
	Version    string                 `json:"version,omitempty"`
	UpdatedAt  time.Time              `json:"updated_at,omitempty"`
	Meta       map[string]interface{} `json:"meta,omitempty"`
}

type MarketEnvVarSpec struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
	Secret      bool   `json:"secret,omitempty"`
}

type MarketInstallOption struct {
	Type        string                 `json:"type"`
	Command     string                 `json:"command,omitempty"`
	Args        []string               `json:"args,omitempty"`
	Env         map[string]string      `json:"env,omitempty"`
	URL         string                 `json:"url,omitempty"`
	Transport   string                 `json:"transport,omitempty"`
	Image       string                 `json:"image,omitempty"`
	PackageName string                 `json:"package_name,omitempty"`
	RequiredEnv []MarketEnvVarSpec     `json:"required_env,omitempty"`
	Auth        *MarketAuthSpec        `json:"auth,omitempty"`
	SourceID    string                 `json:"source_id"`
	Confidence  string                 `json:"confidence"`
	Raw         map[string]interface{} `json:"raw,omitempty"`
}

type MarketAuthSpec struct {
	Type             string `json:"type"`
	AuthorizationURL string `json:"authorization_url,omitempty"`
	Instructions     string `json:"instructions,omitempty"`
}

type MarketToolSpec struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema,omitempty"`
}

type MarketPackage struct {
	ID             string                 `json:"id"`
	CanonicalName  string                 `json:"canonical_name"`
	Name           string                 `json:"name"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Author         string                 `json:"author,omitempty"`
	Version        string                 `json:"version,omitempty"`
	Tags           []string               `json:"tags"`
	Category       string                 `json:"category,omitempty"`
	Repository     string                 `json:"repository,omitempty"`
	Homepage       string                 `json:"homepage,omitempty"`
	License        string                 `json:"license,omitempty"`
	Verified       bool                   `json:"verified"`
	Rating         float64                `json:"rating,omitempty"`
	Downloads      int64                  `json:"downloads,omitempty"`
	UseCount       int64                  `json:"use_count,omitempty"`
	Installability string                 `json:"installability"`
	InstallOptions []MarketInstallOption  `json:"install_options"`
	Tools          []MarketToolSpec       `json:"tools"`
	EnvSchema      map[string]interface{} `json:"env_schema,omitempty"`
	SourceRefs     []MarketSourceRef      `json:"source_refs"`
	RawMeta        map[string]interface{} `json:"raw_meta,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`

	canonicalKey string
}

type MarketFetchRequest struct {
	Cursor       string
	UpdatedSince *time.Time
	Limit        int
}

type MarketFetchPage struct {
	Items      []MarketPackage
	NextCursor string
	HasMore    bool
}

type MarketSourceAdapter interface {
	SourceID() string
	Kind() string
	FetchPage(ctx context.Context, req MarketFetchRequest) (MarketFetchPage, error)
	FetchDetail(ctx context.Context, externalID string) (*MarketPackage, error)
}

type marketSyncJob struct {
	ID            string     `json:"id"`
	SourceID      string     `json:"source_id"`
	Status        string     `json:"status"`
	Cursor        string     `json:"cursor,omitempty"`
	StartedAt     time.Time  `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	ItemsFetched  int        `json:"items_fetched"`
	ItemsUpserted int        `json:"items_upserted"`
	ErrorMessage  string     `json:"error_message,omitempty"`
}

type marketStore struct {
	mu       sync.RWMutex
	sources  map[string]*MarketSource
	packages map[string]*MarketPackage
	byKey    map[string]string
	adapters map[string]MarketSourceAdapter
	jobs     map[string]*marketSyncJob
}

func newMarketStore() *marketStore {
	now := time.Now().UTC()
	s := &marketStore{
		sources:  make(map[string]*MarketSource),
		packages: make(map[string]*MarketPackage),
		byKey:    make(map[string]string),
		adapters: make(map[string]MarketSourceAdapter),
		jobs:     make(map[string]*marketSyncJob),
	}
	for _, src := range defaultAPIMarketSources(now) {
		cp := src
		s.sources[src.ID] = &cp
	}
	for _, pkg := range defaultMarketPackages {
		s.upsertLocked(marketPackageFromSeed(pkg, now))
	}
	return s
}

func defaultAPIMarketSources(now time.Time) []MarketSource {
	return []MarketSource{
		{
			ID:         localMarketSourceID,
			Name:       "Gateway Local Market",
			Kind:       "local_market",
			URL:        "local://gateway-market",
			Trusted:    true,
			Enabled:    true,
			Priority:   0,
			Status:     "healthy",
			CreatedAt:  now,
			UpdatedAt:  now,
			TotalItems: len(defaultMarketPackages),
		},
		{
			ID:         "official",
			Name:       "Official MCP Registry",
			Kind:       "official_registry",
			URL:        "https://registry.modelcontextprotocol.io",
			Trusted:    true,
			Enabled:    true,
			Priority:   1,
			Status:     "ready",
			CreatedAt:  now,
			UpdatedAt:  now,
			TotalItems: 0,
		},
	}
}

func (s *marketStore) registerAdapter(adapter MarketSourceAdapter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.adapters[adapter.SourceID()] = adapter
}

func (s *marketStore) listSources() []MarketSource {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]MarketSource, 0, len(s.sources))
	for _, src := range s.sources {
		cp := *src
		items = append(items, cp)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Priority == items[j].Priority {
			return items[i].ID < items[j].ID
		}
		return items[i].Priority < items[j].Priority
	})
	return items
}

func (s *marketStore) listPackages(q, sourceID, category, installability string, verifiedOnly bool) []MarketPackage {
	q = strings.ToLower(strings.TrimSpace(q))
	category = strings.TrimSpace(category)
	sourceID = strings.TrimSpace(sourceID)
	installability = strings.TrimSpace(installability)

	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]MarketPackage, 0, len(s.packages))
	for _, pkg := range s.packages {
		if q != "" && !strings.Contains(strings.ToLower(pkg.Name+" "+pkg.Title+" "+pkg.Description+" "+pkg.CanonicalName), q) {
			continue
		}
		if category != "" && category != "全部" && pkg.Category != category {
			continue
		}
		if sourceID != "" && !pkg.hasSource(sourceID) {
			continue
		}
		if installability != "" && pkg.Installability != installability {
			continue
		}
		if verifiedOnly && !pkg.Verified {
			continue
		}
		items = append(items, pkg.clone())
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Verified != items[j].Verified {
			return items[i].Verified
		}
		iScore := items[i].UseCount + items[i].Downloads
		jScore := items[j].UseCount + items[j].Downloads
		if iScore != jScore {
			return iScore > jScore
		}
		return items[i].Title < items[j].Title
	})
	return items
}

func (s *marketStore) getPackage(id string) (*MarketPackage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pkg, ok := s.packages[id]
	if !ok {
		return nil, false
	}
	cp := pkg.clone()
	return &cp, true
}

func (s *marketStore) createLocalPackage(pkg MarketPackage) MarketPackage {
	now := time.Now().UTC()
	if pkg.ID == "" {
		pkg.ID = "local_" + stableMarketID(fmt.Sprintf("%s:%d", valueOrDefault(pkg.Name, pkg.Title), now.UnixNano()))[4:]
	}
	pkg.SourceRefs = []MarketSourceRef{{
		SourceID:   localMarketSourceID,
		ExternalID: pkg.ID,
		UpdatedAt:  now,
	}}
	pkg.RawMeta = map[string]interface{}{"local_market": true}
	pkg.CreatedAt = now
	pkg.UpdatedAt = now
	for i := range pkg.InstallOptions {
		pkg.InstallOptions[i].SourceID = localMarketSourceID
		if pkg.InstallOptions[i].Confidence == "" {
			pkg.InstallOptions[i].Confidence = "high"
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.upsertLocked(pkg)
	s.refreshSourceCountLocked(localMarketSourceID)
	cp := s.packages[pkg.ID].clone()
	return cp
}

func (s *marketStore) updateLocalPackage(id string, pkg MarketPackage) (MarketPackage, bool) {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.packages[id]
	if !ok || !existing.hasSource(localMarketSourceID) {
		return MarketPackage{}, false
	}
	existing.Name = pkg.Name
	existing.Title = pkg.Title
	existing.Description = pkg.Description
	existing.Author = pkg.Author
	existing.Version = pkg.Version
	existing.Tags = append([]string(nil), pkg.Tags...)
	existing.Category = pkg.Category
	existing.Repository = pkg.Repository
	existing.Homepage = pkg.Homepage
	existing.License = pkg.License
	existing.Verified = pkg.Verified
	existing.Tools = append([]MarketToolSpec(nil), pkg.Tools...)
	existing.InstallOptions = append([]MarketInstallOption(nil), pkg.InstallOptions...)
	for i := range existing.InstallOptions {
		existing.InstallOptions[i].SourceID = localMarketSourceID
		if existing.InstallOptions[i].Confidence == "" {
			existing.InstallOptions[i].Confidence = "high"
		}
	}
	existing.Installability = computeInstallability(existing.InstallOptions)
	existing.UpdatedAt = now
	for i := range existing.SourceRefs {
		if existing.SourceRefs[i].SourceID == localMarketSourceID {
			existing.SourceRefs[i].UpdatedAt = now
			break
		}
	}
	s.refreshSourceCountLocked(localMarketSourceID)
	return existing.clone(), true
}

func (s *marketStore) deleteLocalPackage(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	pkg, ok := s.packages[id]
	if !ok || !pkg.hasSource(localMarketSourceID) {
		return false
	}
	delete(s.packages, id)
	for key, packageID := range s.byKey {
		if packageID == id {
			delete(s.byKey, key)
		}
	}
	s.refreshSourceCountLocked(localMarketSourceID)
	return true
}

func (s *marketStore) syncSource(ctx context.Context, sourceID string) (*marketSyncJob, error) {
	s.mu.Lock()
	src, ok := s.sources[sourceID]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("market source not found")
	}
	if !src.Enabled {
		s.mu.Unlock()
		return nil, fmt.Errorf("market source is disabled")
	}
	adapter, ok := s.adapters[sourceID]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("market source adapter not found")
	}
	job := &marketSyncJob{
		ID:        fmt.Sprintf("sync_%s_%d", sourceID, time.Now().UnixNano()),
		SourceID:  sourceID,
		Status:    "running",
		Cursor:    src.SyncCursor,
		StartedAt: time.Now().UTC(),
	}
	s.jobs[job.ID] = job
	s.mu.Unlock()

	cursor := job.Cursor
	totalFetched := 0
	totalUpserted := 0
	var syncErr error
	for page := 0; page < 20; page++ {
		resp, err := adapter.FetchPage(ctx, MarketFetchRequest{Cursor: cursor, Limit: 100})
		if err != nil {
			syncErr = err
			break
		}
		totalFetched += len(resp.Items)
		s.mu.Lock()
		for _, item := range resp.Items {
			item = s.preparePackage(item, sourceID)
			if s.upsertLocked(item) {
				totalUpserted++
			}
		}
		s.mu.Unlock()
		if !resp.HasMore || resp.NextCursor == "" {
			cursor = resp.NextCursor
			break
		}
		cursor = resp.NextCursor
	}

	finished := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	job.ItemsFetched = totalFetched
	job.ItemsUpserted = totalUpserted
	job.Cursor = cursor
	job.FinishedAt = &finished
	src.UpdatedAt = finished
	src.TotalItems = s.countSourceLocked(sourceID)
	if syncErr != nil {
		job.Status = "failed"
		job.ErrorMessage = syncErr.Error()
		src.Status = "unhealthy"
		src.LastError = syncErr.Error()
		return job, syncErr
	}
	job.Status = "success"
	src.Status = "healthy"
	src.LastError = ""
	src.LastSynced = finished
	src.SyncCursor = cursor
	return job, nil
}

func (s *marketStore) preparePackage(pkg MarketPackage, sourceID string) MarketPackage {
	now := time.Now().UTC()
	if pkg.CreatedAt.IsZero() {
		pkg.CreatedAt = now
	}
	if pkg.UpdatedAt.IsZero() {
		pkg.UpdatedAt = now
	}
	if pkg.ID == "" {
		pkg.ID = stableMarketID(canonicalKey(pkg))
	}
	if pkg.CanonicalName == "" {
		pkg.CanonicalName = pkg.Name
	}
	if pkg.Title == "" {
		pkg.Title = strings.Title(strings.ReplaceAll(pkg.Name, "-", " "))
	}
	if pkg.Name == "" {
		pkg.Name = pkg.Title
	}
	if len(pkg.SourceRefs) == 0 {
		pkg.SourceRefs = []MarketSourceRef{{SourceID: sourceID, ExternalID: pkg.CanonicalName}}
	}
	for i := range pkg.InstallOptions {
		if pkg.InstallOptions[i].SourceID == "" {
			pkg.InstallOptions[i].SourceID = sourceID
		}
	}
	pkg.canonicalKey = canonicalKey(pkg)
	pkg.Installability = computeInstallability(pkg.InstallOptions)
	if pkg.RawMeta == nil {
		pkg.RawMeta = map[string]interface{}{}
	}
	return pkg
}

func (s *marketStore) upsertLocked(pkg MarketPackage) bool {
	pkg = s.preparePackage(pkg, firstSourceID(pkg.SourceRefs))
	key := pkg.canonicalKey
	id := pkg.ID
	if existingID, ok := s.byKey[key]; ok {
		id = existingID
	}
	existing, exists := s.packages[id]
	if !exists {
		cp := pkg.clone()
		cp.ID = id
		s.packages[id] = &cp
		s.byKey[key] = id
		return true
	}
	mergeMarketPackage(existing, pkg)
	s.byKey[key] = existing.ID
	return true
}

func (s *marketStore) countSourceLocked(sourceID string) int {
	total := 0
	for _, pkg := range s.packages {
		if pkg.hasSource(sourceID) {
			total++
		}
	}
	return total
}

func (s *marketStore) refreshSourceCountLocked(sourceID string) {
	if src, ok := s.sources[sourceID]; ok {
		src.TotalItems = s.countSourceLocked(sourceID)
		src.UpdatedAt = time.Now().UTC()
	}
}

func (pkg MarketPackage) clone() MarketPackage {
	cp := pkg
	cp.Tags = append([]string(nil), pkg.Tags...)
	cp.InstallOptions = append([]MarketInstallOption(nil), pkg.InstallOptions...)
	cp.Tools = append([]MarketToolSpec(nil), pkg.Tools...)
	cp.SourceRefs = append([]MarketSourceRef(nil), pkg.SourceRefs...)
	if pkg.RawMeta != nil {
		cp.RawMeta = make(map[string]interface{}, len(pkg.RawMeta))
		for k, v := range pkg.RawMeta {
			cp.RawMeta[k] = v
		}
	}
	return cp
}

func (pkg MarketPackage) hasSource(sourceID string) bool {
	for _, ref := range pkg.SourceRefs {
		if ref.SourceID == sourceID {
			return true
		}
	}
	return false
}

func mergeMarketPackage(dst *MarketPackage, src MarketPackage) {
	if shouldPreferSource(src, *dst) {
		if src.Name != "" {
			dst.Name = src.Name
		}
		if src.Title != "" {
			dst.Title = src.Title
		}
		if src.Version != "" {
			dst.Version = src.Version
		}
		if src.Repository != "" {
			dst.Repository = src.Repository
		}
		if src.Homepage != "" {
			dst.Homepage = src.Homepage
		}
		if src.License != "" {
			dst.License = src.License
		}
	}
	if len(src.Description) > len(dst.Description) {
		dst.Description = src.Description
	}
	if src.Author != "" && dst.Author == "" {
		dst.Author = src.Author
	}
	dst.Verified = dst.Verified || src.Verified
	if src.Rating > dst.Rating {
		dst.Rating = src.Rating
	}
	if src.Downloads > dst.Downloads {
		dst.Downloads = src.Downloads
	}
	if src.UseCount > dst.UseCount {
		dst.UseCount = src.UseCount
	}
	dst.Tags = mergeStrings(dst.Tags, src.Tags)
	if dst.Category == "" {
		dst.Category = src.Category
	}
	dst.InstallOptions = mergeInstallOptions(dst.InstallOptions, src.InstallOptions)
	dst.Tools = mergeTools(dst.Tools, src.Tools)
	dst.SourceRefs = mergeSourceRefs(dst.SourceRefs, src.SourceRefs)
	dst.Installability = computeInstallability(dst.InstallOptions)
	if src.UpdatedAt.After(dst.UpdatedAt) {
		dst.UpdatedAt = src.UpdatedAt
	}
}

func shouldPreferSource(src, dst MarketPackage) bool {
	if src.hasSource("official") && !dst.hasSource("official") {
		return true
	}
	if src.Verified && !dst.Verified {
		return true
	}
	return false
}

func mergeStrings(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, item := range append(a, b...) {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}

func mergeSourceRefs(a, b []MarketSourceRef) []MarketSourceRef {
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]MarketSourceRef, 0, len(a)+len(b))
	for _, ref := range append(a, b...) {
		key := ref.SourceID + ":" + ref.ExternalID
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, ref)
	}
	return out
}

func mergeInstallOptions(a, b []MarketInstallOption) []MarketInstallOption {
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]MarketInstallOption, 0, len(a)+len(b))
	for _, opt := range append(a, b...) {
		key := opt.Type + ":" + opt.SourceID + ":" + opt.Command + ":" + opt.URL + ":" + strings.Join(opt.Args, "\x00")
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, opt)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return installOptionRank(out[i]) < installOptionRank(out[j])
	})
	return out
}

func mergeTools(a, b []MarketToolSpec) []MarketToolSpec {
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]MarketToolSpec, 0, len(a)+len(b))
	for _, tool := range append(a, b...) {
		if tool.Name == "" || seen[tool.Name] {
			continue
		}
		seen[tool.Name] = true
		out = append(out, tool)
	}
	return out
}

func installOptionRank(opt MarketInstallOption) int {
	if opt.Type == "docker" && opt.Confidence == "high" {
		return 1
	}
	if opt.Type == "remote" && opt.Confidence == "high" {
		return 2
	}
	if opt.SourceID == "official" {
		return 3
	}
	if opt.SourceID == "smithery" {
		return 4
	}
	if opt.Type == "manual" {
		return 99
	}
	return 10
}

func computeInstallability(options []MarketInstallOption) string {
	if len(options) == 0 {
		return installabilityManual
	}
	best := installabilityManual
	for _, opt := range options {
		if opt.Type == "manual" {
			continue
		}
		if opt.Type == "unsupported" {
			if best == installabilityManual {
				best = installabilityUnsupported
			}
			continue
		}
		if len(opt.RequiredEnv) > 0 {
			if best != installabilityInstallable {
				best = installabilityConfigRequired
			}
			continue
		}
		return installabilityInstallable
	}
	return best
}

func marketPackageFromSeed(pkg marketPackage, now time.Time) MarketPackage {
	sourceID := localMarketSourceID
	title := pkg.Name
	if title == "" {
		title = strings.Title(strings.ReplaceAll(pkg.ID, "-", " "))
	}
	option := MarketInstallOption{
		Type:       normalizeInstallType(pkg.Install.Type),
		Command:    pkg.Install.Command,
		Args:       append([]string(nil), pkg.Install.Args...),
		Env:        copyStringMap(pkg.Install.Env),
		Auth:       pkg.Install.Auth,
		SourceID:   sourceID,
		Confidence: "high",
	}
	if option.Type == "remote" {
		option.URL = "https://example.com/" + pkg.ID
		option.Command = ""
		option.Args = nil
	}
	tools := make([]MarketToolSpec, 0, len(pkg.Tools))
	for _, tool := range pkg.Tools {
		tools = append(tools, MarketToolSpec{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}
	return MarketPackage{
		ID:             pkg.ID,
		CanonicalName:  pkg.ID,
		Name:           pkg.ID,
		Title:          title,
		Description:    pkg.Description,
		Author:         pkg.Author,
		Version:        pkg.Version,
		Tags:           append([]string(nil), pkg.Tags...),
		Category:       pkg.Category,
		Verified:       pkg.Verified,
		Rating:         pkg.Rating,
		Downloads:      int64(pkg.Downloads),
		InstallOptions: []MarketInstallOption{option},
		Tools:          tools,
		SourceRefs: []MarketSourceRef{{
			SourceID:   sourceID,
			ExternalID: pkg.ID,
			Version:    pkg.Version,
			UpdatedAt:  now,
		}},
		RawMeta:   map[string]interface{}{"seed": true, "readme": pkg.Readme, "versions": pkg.Versions},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func marketPackageToServiceConfig(pkg MarketPackage, optionIndex int, workspaceID string, env map[string]string) (config.MCPServerConfig, error) {
	if optionIndex < 0 {
		optionIndex = 0
	}
	if optionIndex >= len(pkg.InstallOptions) {
		return config.MCPServerConfig{}, fmt.Errorf("install option not found")
	}
	opt := pkg.InstallOptions[optionIndex]
	if opt.Type == "manual" || opt.Type == "unsupported" {
		return config.MCPServerConfig{}, fmt.Errorf("install option is not installable")
	}
	cfg := config.MCPServerConfig{
		Workspace: workspaceID,
		Args:      append([]string(nil), opt.Args...),
		Env:       copyStringMap(opt.Env),
	}
	for _, spec := range opt.RequiredEnv {
		if spec.Required && env[spec.Name] == "" && cfg.Env[spec.Name] == "" && spec.Default == "" {
			return config.MCPServerConfig{}, fmt.Errorf("required env %s is missing", spec.Name)
		}
		if spec.Default != "" && cfg.Env[spec.Name] == "" {
			cfg.Env[spec.Name] = spec.Default
		}
	}
	switch opt.Type {
	case "remote":
		cfg.URL = opt.URL
	case "npx", "uvx", "command", "docker":
		cfg.Command = opt.Command
		if cfg.Command == "" {
			cfg.Command = opt.Type
		}
	default:
		return config.MCPServerConfig{}, fmt.Errorf("unsupported install option type %q", opt.Type)
	}
	for k, v := range env {
		cfg.Env[k] = v
	}
	return cfg, nil
}

func marketInstallOptionAuth(pkg MarketPackage, optionIndex int) *MarketAuthSpec {
	if optionIndex < 0 {
		optionIndex = 0
	}
	if optionIndex >= len(pkg.InstallOptions) {
		return nil
	}
	auth := pkg.InstallOptions[optionIndex].Auth
	if auth == nil || strings.ToLower(strings.TrimSpace(auth.Type)) != "oauth2" {
		return nil
	}
	cp := *auth
	return &cp
}

func normalizeInstallType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "url", "remote", "sse", "streamable-http", "streamhttp":
		return "remote"
	case "npx", "uvx", "docker", "command":
		return strings.ToLower(t)
	default:
		if t == "" {
			return "manual"
		}
		return strings.ToLower(t)
	}
}

func canonicalKey(pkg MarketPackage) string {
	if pkg.hasSource(localMarketSourceID) {
		return "source:" + localMarketSourceID + ":" + pkg.ID
	}
	if pkg.Repository != "" {
		return "repo:" + normalizeRepoURL(pkg.Repository)
	}
	if pkg.CanonicalName != "" {
		return "name:" + strings.ToLower(pkg.CanonicalName)
	}
	if pkg.Name != "" {
		return "name:" + strings.ToLower(pkg.Name)
	}
	if len(pkg.SourceRefs) > 0 {
		ref := pkg.SourceRefs[0]
		return "source:" + ref.SourceID + ":" + strings.ToLower(ref.ExternalID)
	}
	return "unknown:" + pkg.ID
}

func stableMarketID(key string) string {
	sum := sha1.Sum([]byte(key))
	return "pkg_" + hex.EncodeToString(sum[:])[:16]
}

func normalizeRepoURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return strings.ToLower(strings.TrimSuffix(raw, ".git"))
	}
	host := strings.ToLower(u.Host)
	path := strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git")
	if host == "www.github.com" {
		host = "github.com"
	}
	return "https://" + host + "/" + strings.ToLower(path)
}

func firstSourceID(refs []MarketSourceRef) string {
	if len(refs) == 0 {
		return ""
	}
	return refs[0].SourceID
}

func copyStringMap(src map[string]string) map[string]string {
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

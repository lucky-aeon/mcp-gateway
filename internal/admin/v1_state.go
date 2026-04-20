package admin

import (
	"sync"
	"time"
)

type workspaceMeta struct {
	ID             string
	Name           string
	Description    string
	CreatedAt      time.Time
	LastActivityAt time.Time
}

type serviceMeta struct {
	Name          string
	WorkspaceID   string
	SourceType    string
	SourceRef     string
	Version       string
	CreatedAt     time.Time
	InstalledFrom string
}

type activityItem struct {
	At            time.Time
	Type          string
	WorkspaceID   string
	WorkspaceName string
	ServiceName   string
	SessionID     string
	Message       string
}

type controlPlaneState struct {
	mu         sync.RWMutex
	workspaces map[string]*workspaceMeta
	services   map[string]map[string]*serviceMeta
	activities []activityItem
}

func newControlPlaneState() *controlPlaneState {
	return &controlPlaneState{
		workspaces: make(map[string]*workspaceMeta),
		services:   make(map[string]map[string]*serviceMeta),
		activities: make([]activityItem, 0, 64),
	}
}

func (s *controlPlaneState) ensureWorkspace(id string) *workspaceMeta {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()

	if meta, ok := s.workspaces[id]; ok {
		if meta.LastActivityAt.IsZero() {
			meta.LastActivityAt = now
		}
		return meta
	}

	meta := &workspaceMeta{
		ID:             id,
		Name:           id,
		CreatedAt:      now,
		LastActivityAt: now,
	}
	s.workspaces[id] = meta
	return meta
}

func (s *controlPlaneState) getWorkspace(id string) (*workspaceMeta, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	meta, ok := s.workspaces[id]
	return meta, ok
}

func (s *controlPlaneState) listWorkspaces() []*workspaceMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]*workspaceMeta, 0, len(s.workspaces))
	for _, item := range s.workspaces {
		cp := *item
		items = append(items, &cp)
	}
	return items
}

func (s *controlPlaneState) upsertWorkspace(id, name, description string) *workspaceMeta {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()

	meta, ok := s.workspaces[id]
	if !ok {
		meta = &workspaceMeta{
			ID:        id,
			CreatedAt: now,
		}
		s.workspaces[id] = meta
	}
	if meta.Name == "" {
		meta.Name = id
	}
	if name != "" {
		meta.Name = name
	}
	meta.Description = description
	meta.LastActivityAt = now
	return meta
}

func (s *controlPlaneState) deleteWorkspace(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.workspaces, id)
	delete(s.services, id)
}

func (s *controlPlaneState) touchWorkspace(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if meta, ok := s.workspaces[id]; ok {
		meta.LastActivityAt = time.Now().UTC()
	}
}

func (s *controlPlaneState) upsertService(workspaceID string, meta serviceMeta) {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.services[workspaceID]; !ok {
		s.services[workspaceID] = make(map[string]*serviceMeta)
	}
	existing, ok := s.services[workspaceID][meta.Name]
	if !ok {
		if meta.CreatedAt.IsZero() {
			meta.CreatedAt = now
		}
		cp := meta
		s.services[workspaceID][meta.Name] = &cp
	} else {
		if meta.SourceType != "" {
			existing.SourceType = meta.SourceType
		}
		if meta.SourceRef != "" {
			existing.SourceRef = meta.SourceRef
		}
		if meta.Version != "" {
			existing.Version = meta.Version
		}
		if meta.InstalledFrom != "" {
			existing.InstalledFrom = meta.InstalledFrom
		}
	}
	if ws, ok := s.workspaces[workspaceID]; ok {
		ws.LastActivityAt = now
	}
}

func (s *controlPlaneState) getService(workspaceID, name string) (*serviceMeta, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	services, ok := s.services[workspaceID]
	if !ok {
		return nil, false
	}
	meta, ok := services[name]
	if !ok {
		return nil, false
	}
	cp := *meta
	return &cp, true
}

func (s *controlPlaneState) listServices(workspaceID string) []*serviceMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	services := s.services[workspaceID]
	items := make([]*serviceMeta, 0, len(services))
	for _, item := range services {
		cp := *item
		items = append(items, &cp)
	}
	return items
}

func (s *controlPlaneState) deleteService(workspaceID, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if services, ok := s.services[workspaceID]; ok {
		delete(services, name)
	}
	if ws, ok := s.workspaces[workspaceID]; ok {
		ws.LastActivityAt = time.Now().UTC()
	}
}

func (s *controlPlaneState) appendActivity(item activityItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item.At.IsZero() {
		item.At = time.Now().UTC()
	}
	s.activities = append([]activityItem{item}, s.activities...)
	if len(s.activities) > 100 {
		s.activities = s.activities[:100]
	}
	if ws, ok := s.workspaces[item.WorkspaceID]; ok {
		ws.LastActivityAt = item.At
	}
}

func (s *controlPlaneState) listActivities(limit int) []activityItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.activities) {
		limit = len(s.activities)
	}
	out := make([]activityItem, limit)
	copy(out, s.activities[:limit])
	return out
}

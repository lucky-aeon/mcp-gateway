package identity

import (
	"context"
	"strings"
	"time"

	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
)

type MongoStore struct {
	client        *qmgo.Client
	db            *qmgo.Database
	accounts      *qmgo.Collection
	refreshTokens *qmgo.Collection
	members       *qmgo.Collection
	apiKeys       *qmgo.Collection
	auditLogs     *qmgo.Collection
	workspaces    *qmgo.Collection
	mcpServers    *qmgo.Collection
}

func OpenMongoStore(ctx context.Context, uri, dbName string) (*MongoStore, error) {
	client, err := qmgo.NewClient(ctx, &qmgo.Config{Uri: uri})
	if err != nil {
		return nil, err
	}
	db := client.Database(dbName)
	return &MongoStore{
		client:        client,
		db:            db,
		accounts:      db.Collection("accounts"),
		refreshTokens: db.Collection("refresh_tokens"),
		members:       db.Collection("workspace_members"),
		apiKeys:       db.Collection("api_keys"),
		auditLogs:     db.Collection("audit_logs"),
		workspaces:    db.Collection("workspaces"),
		mcpServers:    db.Collection("mcp_servers"),
	}, nil
}

func (s *MongoStore) Close(ctx context.Context) error {
	return s.client.Close(ctx)
}

func (s *MongoStore) UpsertAdmin(ctx context.Context, email, displayName, password string) (*Account, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	account, err := s.FindAccountByEmail(ctx, email)
	if err == nil {
		updates := bson.M{
			"$set": bson.M{
				"display_name":    displayName,
				"is_system_admin": true,
				"status":          "active",
				"updated_at":      time.Now().UTC(),
			},
		}
		if strings.TrimSpace(password) != "" {
			hash, hashErr := HashPassword(password)
			if hashErr != nil {
				return nil, hashErr
			}
			updates["$set"].(bson.M)["password_hash"] = hash
		}
		if err := s.accounts.UpdateOne(ctx, bson.M{"id": account.ID}, updates); err != nil {
			return nil, err
		}
		return s.FindAccountByID(ctx, account.ID)
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	account = &Account{
		ID:            newID(),
		Email:         email,
		PasswordHash:  hash,
		DisplayName:   displayName,
		Status:        "active",
		IsSystemAdmin: true,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	return account, s.CreateAccount(ctx, account)
}

func (s *MongoStore) CreateAccount(ctx context.Context, account *Account) error {
	account.Email = strings.ToLower(strings.TrimSpace(account.Email))
	_, err := s.accounts.InsertOne(ctx, account)
	return err
}

func (s *MongoStore) FindAccountByEmail(ctx context.Context, email string) (*Account, error) {
	item := &Account{}
	err := s.accounts.Find(ctx, bson.M{"email": strings.ToLower(strings.TrimSpace(email))}).One(item)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s *MongoStore) FindAccountByID(ctx context.Context, id string) (*Account, error) {
	item := &Account{}
	err := s.accounts.Find(ctx, bson.M{"id": id}).One(item)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s *MongoStore) UpsertWorkspaceMember(ctx context.Context, member *WorkspaceMember) error {
	member.Role = NormalizeWorkspaceRole(member.Role)
	member.UpdatedAt = time.Now().UTC()
	existing, err := s.GetWorkspaceMember(ctx, member.WorkspaceID, member.AccountID)
	if err == nil {
		return s.members.UpdateOne(ctx, bson.M{"id": existing.ID}, bson.M{
			"$set": bson.M{
				"role":       member.Role,
				"updated_at": member.UpdatedAt,
			},
		})
	}
	if member.CreatedAt.IsZero() {
		member.CreatedAt = member.UpdatedAt
	}
	_, err = s.members.InsertOne(ctx, member)
	return err
}

func (s *MongoStore) GetWorkspaceMember(ctx context.Context, workspaceID, accountID string) (*WorkspaceMember, error) {
	item := &WorkspaceMember{}
	err := s.members.Find(ctx, bson.M{"workspace_id": workspaceID, "account_id": accountID}).One(item)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s *MongoStore) ListWorkspaceMembersByAccount(ctx context.Context, accountID string) ([]WorkspaceMember, error) {
	var items []WorkspaceMember
	err := s.members.Find(ctx, bson.M{"account_id": accountID}).All(&items)
	return items, err
}

func (s *MongoStore) ListWorkspaceMembersByWorkspace(ctx context.Context, workspaceID string) ([]WorkspaceMember, error) {
	var items []WorkspaceMember
	err := s.members.Find(ctx, bson.M{"workspace_id": workspaceID}).All(&items)
	return items, err
}

func (s *MongoStore) DeleteWorkspaceMembers(ctx context.Context, workspaceID string) error {
	_, err := s.members.RemoveAll(ctx, bson.M{"workspace_id": workspaceID})
	return err
}

func (s *MongoStore) CreateWorkspace(ctx context.Context, ws *Workspace) error {
	_, err := s.workspaces.InsertOne(ctx, ws)
	return err
}

func (s *MongoStore) GetWorkspace(ctx context.Context, id string) (*Workspace, error) {
	item := &Workspace{}
	err := s.workspaces.Find(ctx, bson.M{"id": id}).One(item)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s *MongoStore) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	var items []Workspace
	err := s.workspaces.Find(ctx, bson.M{}).All(&items)
	return items, err
}

func (s *MongoStore) DeleteWorkspace(ctx context.Context, id string) error {
	err := s.workspaces.Remove(ctx, bson.M{"id": id})
	return err
}

func (s *MongoStore) CreateMCPServer(ctx context.Context, server *MCPServer) error {
	_, err := s.mcpServers.InsertOne(ctx, server)
	return err
}

func (s *MongoStore) GetMCPServer(ctx context.Context, workspaceID, name string) (*MCPServer, error) {
	item := &MCPServer{}
	err := s.mcpServers.Find(ctx, bson.M{"workspace_id": workspaceID, "name": name}).One(item)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s *MongoStore) ListMCPServers(ctx context.Context, workspaceID string) ([]MCPServer, error) {
	var items []MCPServer
	err := s.mcpServers.Find(ctx, bson.M{"workspace_id": workspaceID}).All(&items)
	return items, err
}

func (s *MongoStore) DeleteMCPServer(ctx context.Context, workspaceID, name string) error {
	err := s.mcpServers.Remove(ctx, bson.M{"workspace_id": workspaceID, "name": name})
	return err
}

func (s *MongoStore) CreateAPIKey(ctx context.Context, key *APIKey) error {
	_, err := s.apiKeys.InsertOne(ctx, key)
	return err
}

func (s *MongoStore) ListAPIKeysByAccount(ctx context.Context, accountID string) ([]APIKey, error) {
	var items []APIKey
	err := s.apiKeys.Find(ctx, bson.M{"account_id": accountID}).Sort("-created_at").All(&items)
	return items, err
}

func (s *MongoStore) FindAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	item := &APIKey{}
	err := s.apiKeys.Find(ctx, bson.M{"key_hash": keyHash, "status": apiKeyStatusActive}).One(item)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s *MongoStore) UpdateAPIKeyUsage(ctx context.Context, keyID string, when time.Time) error {
	return s.apiKeys.UpdateOne(ctx, bson.M{"id": keyID}, bson.M{"$set": bson.M{"last_used_at": when}})
}

func (s *MongoStore) RevokeAPIKey(ctx context.Context, keyID, accountID string) error {
	return s.apiKeys.UpdateOne(ctx, bson.M{"id": keyID, "account_id": accountID}, bson.M{"$set": bson.M{"status": apiKeyStatusRevoked}})
}

func (s *MongoStore) CreateRefreshToken(ctx context.Context, token *RefreshToken) error {
	_, err := s.refreshTokens.InsertOne(ctx, token)
	return err
}

func (s *MongoStore) FindRefreshTokenByHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	item := &RefreshToken{}
	err := s.refreshTokens.Find(ctx, bson.M{"token_hash": tokenHash}).One(item)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s *MongoStore) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	return s.refreshTokens.Remove(ctx, bson.M{"token_hash": tokenHash})
}

func (s *MongoStore) AppendAuditLog(ctx context.Context, item *AuditLog) error {
	_, err := s.auditLogs.InsertOne(ctx, item)
	return err
}

func (s *MongoStore) ListAuditLogs(ctx context.Context, workspaceID string, limit int) ([]AuditLog, error) {
	query := bson.M{}
	if workspaceID != "" {
		query["workspace_id"] = workspaceID
	}
	if limit <= 0 {
		limit = 50
	}
	var items []AuditLog
	err := s.auditLogs.Find(ctx, query).Sort("-created_at").Limit(int64(limit)).All(&items)
	return items, err
}

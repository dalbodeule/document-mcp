package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"document-mdp/ent"
	"document-mdp/ent/documentacl"
	"document-mdp/ent/group"
	"document-mdp/ent/refreshtoken"
	"document-mdp/ent/schema"
	"document-mdp/ent/user"
	"document-mdp/ent/usergroup"
	"document-mdp/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	ent *ent.Client
	cfg config.Config
}

func NewService(entClient *ent.Client, cfg config.Config) *Service {
	return &Service{ent: entClient, cfg: cfg}
}

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type AccessClaims struct {
	jwt.RegisteredClaims
	Type     string   `json:"typ"`
	UserID   string   `json:"uid"`
	GroupIDs []string `json:"gids"`
}

type RefreshClaims struct {
	jwt.RegisteredClaims
	Type   string `json:"typ"`
	UserID string `json:"uid"`
	JTI    string `json:"jti"`
}

func (s *Service) Register(ctx context.Context, email, password, name string) (*ent.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	u, err := s.ent.User.Create().
		SetEmail(email).
		SetPasswordHash(string(hash)).
		SetName(name).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (Tokens, error) {
	u, err := s.ent.User.Query().Where(user.EmailEQ(email)).Only(ctx)
	if err != nil {
		return Tokens{}, fmt.Errorf("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return Tokens{}, fmt.Errorf("invalid credentials")
	}
	return s.issueTokens(ctx, u.ID, userAgent, ip)
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ip string) (Tokens, error) {
	claims, err := s.ParseRefreshToken(refreshToken)
	if err != nil {
		return Tokens{}, err
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return Tokens{}, fmt.Errorf("invalid token")
	}
	h := hashToken(refreshToken)
	rt, err := s.ent.RefreshToken.Query().
		Where(refreshtoken.TokenHashEQ(h)).
		Where(refreshtoken.UserIDEQ(userID)).
		Only(ctx)
	if err != nil {
		return Tokens{}, fmt.Errorf("invalid token")
	}
	if rt.RevokedAt != nil {
		return Tokens{}, fmt.Errorf("token revoked")
	}
	if time.Now().After(rt.ExpiresAt) {
		return Tokens{}, fmt.Errorf("token expired")
	}
	// rotate
	_, err = s.ent.RefreshToken.UpdateOneID(rt.ID).SetRevokedAt(time.Now()).Save(ctx)
	if err != nil {
		return Tokens{}, fmt.Errorf("revoke token: %w", err)
	}
	return s.issueTokens(ctx, userID, userAgent, ip)
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.ParseRefreshToken(refreshToken)
	if err != nil {
		return err
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return fmt.Errorf("invalid token")
	}
	h := hashToken(refreshToken)
	_, err = s.ent.RefreshToken.Update().
		Where(refreshtoken.TokenHashEQ(h)).
		Where(refreshtoken.UserIDEQ(userID)).
		Where(refreshtoken.RevokedAtIsNil()).
		SetRevokedAt(time.Now()).
		Save(ctx)
	return err
}

func (s *Service) ParseAccessToken(tokenString string) (AccessClaims, error) {
	key := []byte(s.cfg.JWTAccessSecret)
	tok, err := jwt.ParseWithClaims(tokenString, &AccessClaims{}, func(token *jwt.Token) (any, error) {
		return key, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return AccessClaims{}, err
	}
	claims, ok := tok.Claims.(*AccessClaims)
	if !ok || !tok.Valid {
		return AccessClaims{}, fmt.Errorf("invalid token")
	}
	if claims.Type != "access" {
		return AccessClaims{}, fmt.Errorf("invalid token type")
	}
	return *claims, nil
}

func (s *Service) ParseRefreshToken(tokenString string) (RefreshClaims, error) {
	key := []byte(s.cfg.JWTRefreshSecret)
	tok, err := jwt.ParseWithClaims(tokenString, &RefreshClaims{}, func(token *jwt.Token) (any, error) {
		return key, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return RefreshClaims{}, err
	}
	claims, ok := tok.Claims.(*RefreshClaims)
	if !ok || !tok.Valid {
		return RefreshClaims{}, fmt.Errorf("invalid token")
	}
	if claims.Type != "refresh" {
		return RefreshClaims{}, fmt.Errorf("invalid token type")
	}
	return *claims, nil
}

func (s *Service) issueTokens(ctx context.Context, userID uuid.UUID, userAgent, ip string) (Tokens, error) {
	groupIDs, err := s.userGroupIDs(ctx, userID)
	if err != nil {
		return Tokens{}, err
	}

	now := time.Now()
	accessExp := now.Add(time.Duration(s.cfg.AccessTTLSeconds) * time.Second)
	refreshExp := now.Add(time.Duration(s.cfg.RefreshTTLSeconds) * time.Second)

	accessClaims := AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
		Type:     "access",
		UserID:   userID.String(),
		GroupIDs: groupIDs,
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.cfg.JWTAccessSecret))
	if err != nil {
		return Tokens{}, err
	}

	jti := uuid.New().String()
	refreshClaims := RefreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExp),
			ID:        jti,
		},
		Type:   "refresh",
		UserID: userID.String(),
		JTI:    jti,
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(s.cfg.JWTRefreshSecret))
	if err != nil {
		return Tokens{}, err
	}

	_, err = s.ent.RefreshToken.Create().
		SetUserID(userID).
		SetTokenHash(hashToken(refreshToken)).
		SetExpiresAt(refreshExp).
		SetUserAgent(userAgent).
		SetIP(ip).
		Save(ctx)
	if err != nil {
		return Tokens{}, err
	}

	return Tokens{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (s *Service) userGroupIDs(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := s.ent.UserGroup.Query().
		Where(usergroup.UserIDEQ(userID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.GroupID.String())
	}
	return ids, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// --- Authorization helpers ---

func (s *Service) CanReadDocument(ctx context.Context, userID uuid.UUID, groupIDs []string, documentID uuid.UUID) (bool, error) {
	return s.canDocument(ctx, userID, groupIDs, documentID, false)
}

func (s *Service) CanWriteDocument(ctx context.Context, userID uuid.UUID, groupIDs []string, documentID uuid.UUID) (bool, error) {
	return s.canDocument(ctx, userID, groupIDs, documentID, true)
}

func (s *Service) canDocument(ctx context.Context, userID uuid.UUID, groupIDs []string, documentID uuid.UUID, wantWrite bool) (bool, error) {
	// owner shortcut
	doc, err := s.ent.Document.Get(ctx, documentID)
	if err != nil {
		return false, err
	}
	if doc.CreatedByUserID == userID {
		return true, nil
	}

	if len(groupIDs) == 0 {
		return false, nil
	}

	gids := make([]uuid.UUID, 0, len(groupIDs))
	for _, gidStr := range groupIDs {
		gid, err := uuid.Parse(gidStr)
		if err == nil {
			gids = append(gids, gid)
		}
	}
	if len(gids) == 0 {
		return false, nil
	}

	rules, err := s.ent.DocumentACL.Query().
		Where(documentacl.DocumentIDEQ(documentID)).
		Where(documentacl.GroupIDIn(gids...)).
		All(ctx)
	if err != nil {
		return false, err
	}
	effects := make([]schema.ACLEffect, 0, len(rules))
	for _, r := range rules {
		effects = append(effects, r.Effect)
	}
	deny, allowRead, allowWrite := evaluateEffects(effects)
	if deny {
		return false, nil
	}
	if wantWrite {
		return allowWrite, nil
	}
	return allowRead, nil
}

// evaluateEffects applies precedence for ACL effects.
// deny > read_write > read.
func evaluateEffects(effects []schema.ACLEffect) (deny bool, allowRead bool, allowWrite bool) {
	for _, e := range effects {
		if e == schema.ACLEffectDeny {
			return true, false, false
		}
	}
	for _, e := range effects {
		if e == schema.ACLEffectWrite {
			return false, true, true
		}
	}
	for _, e := range effects {
		if e == schema.ACLEffectRead {
			return false, true, false
		}
	}
	return false, false, false
}

func (s *Service) SetDocumentACL(ctx context.Context, documentID, groupID uuid.UUID, effect string) error {
	var eff schema.ACLEffect
	switch effect {
	case string(schema.ACLEffectDeny):
		eff = schema.ACLEffectDeny
	case string(schema.ACLEffectRead):
		eff = schema.ACLEffectRead
	case string(schema.ACLEffectWrite):
		eff = schema.ACLEffectWrite
	default:
		return fmt.Errorf("invalid effect")
	}

	return s.ent.DocumentACL.Create().
		SetDocumentID(documentID).
		SetGroupID(groupID).
		SetEffect(eff).
		OnConflictColumns(documentacl.FieldDocumentID, documentacl.FieldGroupID).
		UpdateNewValues().
		Exec(ctx)
}

func (s *Service) CreateGroup(ctx context.Context, name string) (*ent.Group, error) {
	return s.ent.Group.Create().SetName(name).Save(ctx)
}

func (s *Service) AddUserToGroup(ctx context.Context, userID, groupID uuid.UUID) error {
	return s.ent.UserGroup.Create().
		SetUserID(userID).
		SetGroupID(groupID).
		OnConflictColumns(usergroup.FieldUserID, usergroup.FieldGroupID).
		Ignore().
		Exec(ctx)
}

func (s *Service) ListUserGroups(ctx context.Context, userID uuid.UUID) ([]*ent.Group, error) {
	// groups joined by user_groups
	ug, err := s.ent.UserGroup.Query().
		Where(usergroup.UserIDEQ(userID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(ug) == 0 {
		return []*ent.Group{}, nil
	}
	ids := make([]uuid.UUID, 0, len(ug))
	for _, r := range ug {
		ids = append(ids, r.GroupID)
	}
	return s.ent.Group.Query().Where(group.IDIn(ids...)).All(ctx)
}

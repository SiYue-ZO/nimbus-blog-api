package user

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	_defaultAccessTTL  = 15 * time.Minute
	_defaultRefreshTTL = 7 * 24 * time.Hour
)

type TokenSigner interface {
	SignAccess(id int64, meta map[string]string) (string, error)
	SignRefresh(id int64) (string, error)
	ParseRefresh(string) (int64, error)
	ParseRefreshClaims(string) (*RefreshTokenClaims, error)
	ParseAccess(string) (*AccessTokenClaims, error)
	AccessTTL() time.Duration
	RefreshTTL() time.Duration
}

type signer struct {
	accessSecret  string
	refreshSecret string
	accessTTL     time.Duration
	refreshTTL    time.Duration
	issuer        string
}

func NewTokenSigner(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration, issuer string) (TokenSigner, error) {
	if accessSecret == "" || refreshSecret == "" {
		return nil, errors.New("auth.user.NewTokenSigner: secrets empty")
	}
	if accessTTL == 0 {
		accessTTL = _defaultAccessTTL
	}
	if refreshTTL == 0 {
		refreshTTL = _defaultRefreshTTL
	}
	if issuer == "" {
		issuer = "nimbus-blog-api"
	}
	return &signer{accessSecret: accessSecret, refreshSecret: refreshSecret, accessTTL: accessTTL, refreshTTL: refreshTTL, issuer: issuer}, nil
}

func (s *signer) AccessTTL() time.Duration  { return s.accessTTL }
func (s *signer) RefreshTTL() time.Duration { return s.refreshTTL }

type AccessTokenClaims struct {
	UserID string            `json:"sub"`
	Meta   map[string]string `json:"meta,omitempty"`
	jwt.RegisteredClaims
}

type RefreshTokenClaims struct {
	UserID string `json:"sub"`
	Type   string `json:"type"`
	jwt.RegisteredClaims
}

func NewAccessClaims(id int64, meta map[string]string, issuer string, ttl time.Duration) *AccessTokenClaims {
	now := time.Now()
	c := &AccessTokenClaims{
		UserID: strconv.FormatInt(id, 10),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	if meta != nil {
		c.Meta = meta
	}
	return c
}

func NewRefreshClaims(id int64, issuer string, ttl time.Duration) *RefreshTokenClaims {
	now := time.Now()
	return &RefreshTokenClaims{
		UserID: strconv.FormatInt(id, 10),
		Type:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
}

func (c *AccessTokenClaims) UserIDInt() (int64, error)  { return strconv.ParseInt(c.UserID, 10, 64) }
func (c *RefreshTokenClaims) UserIDInt() (int64, error) { return strconv.ParseInt(c.UserID, 10, 64) }
func (c *RefreshTokenClaims) IsRefresh() bool           { return c.Type == "refresh" }

type ctxKeyAccess struct{}
type ctxKeyRefresh struct{}

var accessClaimsContextKey = ctxKeyAccess{}
var refreshClaimsContextKey = ctxKeyRefresh{}

func WithAccessClaims(ctx context.Context, claims *AccessTokenClaims) context.Context {
	return context.WithValue(ctx, accessClaimsContextKey, claims)
}

func AccessClaimsFromContext(ctx context.Context) (*AccessTokenClaims, bool) {
	v := ctx.Value(accessClaimsContextKey)
	if v == nil {
		return nil, false
	}
	c, ok := v.(*AccessTokenClaims)
	return c, ok
}

func WithRefreshClaims(ctx context.Context, claims *RefreshTokenClaims) context.Context {
	return context.WithValue(ctx, refreshClaimsContextKey, claims)
}

func RefreshClaimsFromContext(ctx context.Context) (*RefreshTokenClaims, bool) {
	v := ctx.Value(refreshClaimsContextKey)
	if v == nil {
		return nil, false
	}
	c, ok := v.(*RefreshTokenClaims)
	return c, ok
}

func (s *signer) SignAccess(id int64, meta map[string]string) (string, error) {
	claims := NewAccessClaims(id, meta, s.issuer, s.accessTTL)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.accessSecret))
}

func (s *signer) SignRefresh(id int64) (string, error) {
	claims := NewRefreshClaims(id, s.issuer, s.refreshTTL)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.refreshSecret))
}

func (s *signer) ParseRefreshClaims(token string) (*RefreshTokenClaims, error) {
	claims := &RefreshTokenClaims{}
	t, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.refreshSecret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	if !t.Valid || !claims.IsRefresh() {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

func (s *signer) ParseRefresh(token string) (int64, error) {
	claims, err := s.ParseRefreshClaims(token)
	if err != nil {
		return 0, err
	}
	uid, err := claims.UserIDInt()
	if err != nil {
		return 0, ErrTokenInvalid
	}
	return uid, nil
}

func (s *signer) ParseAccess(token string) (*AccessTokenClaims, error) {
	claims := &AccessTokenClaims{}
	t, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return []byte(s.accessSecret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	if !t.Valid || claims.RegisteredClaims.Issuer != s.issuer {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

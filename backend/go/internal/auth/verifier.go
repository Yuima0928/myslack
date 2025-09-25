package auth

import (
	"context"
	"time"

	keyfunc "github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type Config struct {
	Domain   string // e.g. your-tenant.us.auth0.com
	Audience string // e.g. https://api.myslack.local
}

type Verifier struct {
	cfg Config
	kf  keyfunc.Keyfunc
}

func NewVerifier(ctx context.Context, cfg Config) (*Verifier, error) {
	kf, err := keyfunc.NewDefaultCtx(ctx, []string{
		"https://" + cfg.Domain + "/.well-known/jwks.json",
	})
	if err != nil {
		return nil, err
	}
	return &Verifier{cfg: cfg, kf: kf}, nil
}

type Claims struct {
	Sub   string
	Email string
	Name  string
}

func (v *Verifier) Verify(raw string) (*Claims, error) {
	tok, err := jwt.Parse(raw, v.kf.Keyfunc,
		jwt.WithAudience(v.cfg.Audience),
		jwt.WithIssuer("https://"+v.cfg.Domain+"/"),
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil || !tok.Valid {
		return nil, err
	}
	mc, _ := tok.Claims.(jwt.MapClaims)
	sub, _ := mc["sub"].(string)
	email, _ := mc["email"].(string)
	name, _ := mc["name"].(string)
	if sub == "" {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return &Claims{Sub: sub, Email: email, Name: name}, nil
}

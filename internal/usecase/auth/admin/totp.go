package admin

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"

	"github.com/pquerna/otp/totp"
)

type TOTP interface {
	Generate(issuer, account string) (secret string, qrBase64 string, err error)
	Validate(code, secret string) bool
}

type TOTPConfig struct {
	QRWidth  int
	QRHeight int
}

type totpProvider struct {
	cfg TOTPConfig
}

func NewTOTPProvider() TOTP {
	return &totpProvider{cfg: TOTPConfig{QRWidth: 200, QRHeight: 200}}
}

func NewTOTPProviderWithConfig(cfg TOTPConfig) TOTP {
	if cfg.QRWidth <= 0 {
		cfg.QRWidth = 200
	}
	if cfg.QRHeight <= 0 {
		cfg.QRHeight = 200
	}
	return &totpProvider{cfg: cfg}
}

func (p *totpProvider) Generate(issuer, account string) (string, string, error) {
	key, err := totp.Generate(totp.GenerateOpts{Issuer: issuer, AccountName: account})
	if err != nil {
		return "", "", err
	}
	img, err := key.Image(p.cfg.QRWidth, p.cfg.QRHeight)
	if err != nil {
		return "", "", err
	}
	b64 := encodeImageToBase64PNG(img)
	return key.Secret(), b64, nil
}

func (p *totpProvider) Validate(code, secret string) bool {
	return totp.Validate(code, secret)
}

func encodeImageToBase64PNG(img image.Image) string {
	buf := &bytes.Buffer{}
	_ = png.Encode(buf, img)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

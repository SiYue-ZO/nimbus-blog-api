package captcha

import (
	"context"
	"errors"
	"fmt"

	"github.com/scc749/nimbus-blog-api/internal/repo"
)

var (
	// ErrGenerate Generate 错误哨兵。
	ErrGenerate = errors.New("captcha generate")
	// ErrStore Store 错误哨兵。
	ErrStore = errors.New("captcha store")
)

// UseCase 验证码用例。
type UseCase struct {
	store repo.CaptchaStore
	gen   Generator
}

// New 创建 UseCase。
func New(store repo.CaptchaStore, gen Generator) *UseCase {
	return &UseCase{store: store, gen: gen}
}

// Generate 生成验证码并返回 id 与 base64 图片。
func (u *UseCase) Generate(ctx context.Context) (string, string, error) {
	id, b64s, answer, err := u.gen.Generate()
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrGenerate, err)
	}
	if err := u.store.Set(id, answer); err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrStore, err)
	}
	return id, b64s, nil
}

// Verify 校验验证码并在读取后清除缓存值。
func (u *UseCase) Verify(ctx context.Context, id, answer string) (bool, error) {
	stored := u.store.Get(id, true)
	if stored == "" {
		return false, nil
	}
	return stored == answer, nil
}

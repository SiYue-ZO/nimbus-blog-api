package email

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/usecase"
)

var (
	// ErrCodeGeneration CodeGeneration 错误哨兵。
	ErrCodeGeneration = errors.New("generate code")
	// ErrEmailSend EmailSend 错误哨兵。
	ErrEmailSend = errors.New("send email")
	// ErrCodeStore CodeStore 错误哨兵。
	ErrCodeStore = errors.New("store code")
)

// UseCase 邮件用例。
type UseCase struct {
	emailSender repo.EmailSender
	codeStore   repo.EmailCodeStore
}

// New 创建 Email UseCase。
func New(emailSender repo.EmailSender, codeStore repo.EmailCodeStore) usecase.Email {
	return &UseCase{emailSender: emailSender, codeStore: codeStore}
}

// SendCode 发送验证码邮件。
func (u *UseCase) SendCode(ctx context.Context, to string) error {
	code, err := generateNumericCode(6)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCodeGeneration, err)
	}

	subject := "Your verification code"
	body := fmt.Sprintf("Your verification code is %s. It expires in 10 minutes.", code)

	if err := u.emailSender.Send(to, subject, body); err != nil {
		return fmt.Errorf("%w: %v", ErrEmailSend, err)
	}

	if u.codeStore != nil {
		if err := u.codeStore.Set(to, code); err != nil {
			return fmt.Errorf("%w: %v", ErrCodeStore, err)
		}
	}
	return nil
}

// VerifyCode 校验验证码。
func (u *UseCase) VerifyCode(ctx context.Context, to string, code string) (bool, error) {
	if u.codeStore == nil {
		return false, fmt.Errorf("%w: code store not initialized", ErrCodeStore)
	}
	ok := u.codeStore.Verify(to, code, false)
	return ok, nil
}

func generateNumericCode(n int) (string, error) {
	const digits = "0123456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := 0; i < n; i++ {
		b[i] = digits[int(b[i])%10]
	}
	return string(b), nil
}

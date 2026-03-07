package auth

import "github.com/scc749/nimbus-blog-api/internal/usecase"

type useCase struct {
	adminAuth usecase.AdminAuth
	userAuth  usecase.UserAuth
}

func New(adminAuth usecase.AdminAuth, userAuth usecase.UserAuth) usecase.Auth {
	return &useCase{adminAuth: adminAuth, userAuth: userAuth}
}

func (uc *useCase) Admin() usecase.AdminAuth {
	return uc.adminAuth
}

func (uc *useCase) User() usecase.UserAuth {
	return uc.userAuth
}

package output

type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	TokenType        string
	ExpiresIn        int64
	RefreshExpiresIn int64
}

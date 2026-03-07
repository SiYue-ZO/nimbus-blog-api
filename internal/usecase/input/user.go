package input

type ListUsers struct {
	PageParams
	Keyword *KeywordParams
	Sort    *SortParams
	Status  StringFilterParam
}

type UpdateProfile struct {
	Name            string
	Bio             string
	Region          string
	BlogURL         string
	ShowFullProfile bool
}

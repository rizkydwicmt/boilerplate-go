package enum

type EnvEnum string

const (
	DEVELOPMENT EnvEnum = "development"
	PRODUCTION  EnvEnum = "production"
	STAGING     EnvEnum = "staging"
)

func (e EnvEnum) ToString() string {
	switch e {
	case DEVELOPMENT:
		return "development"
	case PRODUCTION:
		return "production"
	case STAGING:
		return "staging"
	}
	return ""
}

func (e EnvEnum) IsValid() bool {
	switch e {
	case DEVELOPMENT, PRODUCTION, STAGING:
		return true
	}
	return false
}

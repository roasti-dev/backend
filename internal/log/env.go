package log

type Env string

const (
	EnvProduction  Env = "production"
	EnvDevelopment Env = "development"
	EnvStaging     Env = "staging"
)

func (e Env) IsValid() bool {
	switch e {
	case EnvProduction, EnvDevelopment, EnvStaging:
		return true
	}
	return false
}

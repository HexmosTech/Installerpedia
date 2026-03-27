package endpoints


var IsLocalDev = false


// ServiceConfig holds the toggle and the two possible URLs
type ServiceConfig struct {
	UseLocal bool
	Local    string
	Prod     string
}

// Get returns the active URL based on the UseLocal flag
func (s ServiceConfig) Get() string {
    // If master is on OR individual is on, return Local
    if IsLocalDev || s.UseLocal {
        return s.Local
    }
    return s.Prod
}

// Endpoints acts as your central dictionary for API URLs
var Endpoints = struct {
	AddEntry     ServiceConfig
	GenerateRepo ServiceConfig
	GenerateRepoMethod ServiceConfig
	AutoIndex    ServiceConfig
	Featured     ServiceConfig
	UpdateEntry  ServiceConfig
	CheckRepoUpdates ServiceConfig
}{
	AddEntry: ServiceConfig{
		UseLocal: false,
		Local:    "http://localhost:4321/freedevtools/api/installerpedia/add-entry",
		Prod:     "https://hexmos.com/freedevtools/api/installerpedia/add-entry",
	},
	GenerateRepo: ServiceConfig{
		UseLocal: false,
		Local:    "http://localhost:4321/freedevtools/api/installerpedia/generate_ipm_repo",
		Prod:     "https://hexmos.com/freedevtools/api/installerpedia/generate_ipm_repo",
	},
	GenerateRepoMethod: ServiceConfig{
		UseLocal: false,
		Local:    "http://localhost:4321/freedevtools/api/installerpedia/generate_ipm_repo_method",
		Prod:     "https://hexmos.com/freedevtools/api/installerpedia/generate_ipm_repo_method",
	},
	AutoIndex: ServiceConfig{
		UseLocal: false,
		Local:    "http://localhost:4321/freedevtools/api/installerpedia/auto_index",
		Prod:     "https://hexmos.com/freedevtools/api/installerpedia/auto_index",
	},
	Featured: ServiceConfig{
        UseLocal: false,
        Local:    "http://localhost:4321/freedevtools/api/installerpedia/featured",
        Prod:     "https://hexmos.com/freedevtools/api/installerpedia/featured",
    },
	UpdateEntry: ServiceConfig{
		UseLocal: false,
		Local:    "http://localhost:4321/freedevtools/api/installerpedia/update-entry",
        Prod:     "https://hexmos.com/freedevtools/api/installerpedia/update-entry",
	},
	CheckRepoUpdates: ServiceConfig{
		UseLocal: false,
		Local:    "http://localhost:4321/freedevtools/api/installerpedia/check_ipm_repo_updates",
        Prod:     "https://hexmos.com/freedevtools/api/installerpedia/check_ipm_repo_updates",
	},
}
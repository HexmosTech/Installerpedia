package types

type Instruction struct {
	Command  string `json:"command"`
	Meaning  string `json:"meaning,omitempty"`
	Optional bool   `json:"optional,omitempty"`
}

type InstallMethod struct {
	Title        string        `json:"title"`
	Instructions []Instruction `json:"instructions"`
}

type Prerequisite struct {
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Optional    bool     `json:"optional"`
	AppliesTo   []string `json:"applies_to"`
}

type RepoDocumentFull struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	RepoType            string          `json:"repo_type"`
	Stars               int             `json:"stars"`
	Description         string          `json:"description"`
	Note                string          `json:"note,omitempty"`
	Prerequisites       []Prerequisite  `json:"prerequisites,omitempty"`
	InstallationMethods []InstallMethod `json:"installation_methods,omitempty"`
	PostInstallation    []string        `json:"post_installation,omitempty"`
}

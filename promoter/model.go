package promoter

// Repos what promotionChecker looks for.
type Repos struct {
	RepoImage string
	Repo      string
	Image     string
	Tags      []string
}

// Item the config values that the app check
type Item struct {
	Repo    string
	Image   string
	Webhook string
}

// Items config file struct
type Items struct {
	Containers        []Item `yaml:"containers"`
	ArtifactoryURL    string `yaml:"artifactoryURL"`
	ArtifactoryAPIkey string `yaml:"artifactoryAPIkey"`
	ArtifactoryUSER   string `yaml:"artifactoryUSER"`
	PollTime          int    `yaml:"pollTime"`
	HTTPtimeout       int    `yaml:"httpTimeout"`
	HTTPinsecure      bool   `yaml:"httpInsecure"`
	WebhookSecret     string `yaml:"webhookSecret"`
	DBType            string `yaml:"dbType"`
	EndpointPort      int    `yaml:"endpointPort"`
}

// Tags artifactory output from rest call
type Tags struct {
	Repo         string
	Path         string
	Created      string
	CreatedBy    string
	LastModified string
	ModifiedBy   string
	LastUpdated  string
	Children     []Children
	URI          string
}

// Children artifactory output for all tags
type Children struct {
	URI    string
	Folder bool
}

package promoter

type RedirectRepository interface {
	Store(repoImage string, artrepo string, image string, tags []string) (*Repos, error)
	Read(repoImage string) ([]string, error)
	UpdateTags(repoImage string, repo string, image string, newTags []string) error
}

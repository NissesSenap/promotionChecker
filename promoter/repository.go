package promoter

type RedirectRepository interface {
	StartupUpdate(repoImage string, repo string, image string, tags []string) (*Repos, error)
	Store(repoImage string, artrepo string, image string, tags []string) (*Repos, error)
	Read(repoImage string) ([]string, error)
}

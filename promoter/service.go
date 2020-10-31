package promoter

type RedirectService interface {
	Store(repoImage string, artrepo string, image string, tags []string) error
	Read(repoImage string) ([]string, error)
	UpdateTags(repoImage string, repo string, image string, newTags []string) error
}

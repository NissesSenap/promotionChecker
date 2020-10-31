package promoter

type redirectService struct {
	redirectRepo RedirectRepository
}

func NewRedirectService(redirectRepo RedirectRepository) RedirectService {
	return &redirectService{
		redirectRepo,
	}
}

func (r *redirectService) UpdateTags(repoImage string, repo string, image string, newTags []string) error {
	return r.redirectRepo.UpdateTags(repoImage, repo, image, newTags)
}

func (r *redirectService) Store(repoImage string, artrepo string, image string, tags []string) error {

	return r.redirectRepo.Store(repoImage, artrepo, image, tags)
}

func (r *redirectService) Read(repoImage string) ([]string, error) {
	return r.redirectRepo.Read(repoImage)
}

// AppendIfMissing takes a list and adds unique values to that list
func AppendIfMissing(slice []string, i string) []string {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

// StringInSlice checks if a string is in the existing list
func StringNotInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return false
		}
	}
	return true
}

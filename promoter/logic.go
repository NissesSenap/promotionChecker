package promoter

import (
	"fmt"
)

type redirectService struct {
	redirectRepo RedirectRepository
}

func NewRedirectService(redirectRepo RedirectRepository) RedirectService {
	return &redirectService{
		redirectRepo,
	}
}
func UpdateTags(repoImage string, repo string, image string, newTags []string, hmemdbRepo RedirectRepository) error {
	currentTags, err := hmemdbRepo.Read(repoImage)
	if err != nil {
		fmt.Println("Unable to find any current repoImage")
	}
	fmt.Printf("Here is the current tags %v", currentTags)

	// newTags will allways only contain 1 value
	realTag := appendIfMissing(currentTags, newTags[0])
	fmt.Println(realTag)

	// Update/Create the tags in repoImage
	_, err = hmemdbRepo.Store(repoImage, repo, image, realTag)
	if err != nil {
		return err
	}
	return nil
}

func (r *redirectService) StartupUpdate(repoImage string, repo string, image string, tags []string) (*Repos, error) {

	return r.redirectRepo.StartupUpdate(repoImage, repo, image, tags)
}

func (r *redirectService) Store(repoImage string, artrepo string, image string, tags []string) (*Repos, error) {

	return r.redirectRepo.Store(repoImage, artrepo, image, tags)
}

func (r *redirectService) Read(repoImage string) ([]string, error) {
	return r.redirectRepo.Read(repoImage)
}

// appendIfMissing takes a list and adds unique values to that list
func appendIfMissing(slice []string, i string) []string {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

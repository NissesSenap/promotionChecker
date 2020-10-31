package hmemdb

import (
	"fmt"
	"time"

	"github.com/NissesSenap/promotionChecker/promoter"
	"github.com/hashicorp/go-memdb"
)

type memDBRepository struct {
	client    *memdb.MemDB
	tableName string
	timeout   time.Duration
}

func newMemDBClient(schema *memdb.DBSchema, txnTimeout int) (*memdb.MemDB, error) {
	// Create a new data base
	db, err := memdb.NewMemDB(schema)
	if err != nil {
		panic(err)
	}
	return db, nil
}

func NewMemDBRepository(tableName string, timeout int) (promoter.RedirectRepository, error) {

	// Promotion schema
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			tableName: &memdb.TableSchema{
				Name: tableName,
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "RepoImage"},
					},
					"image": &memdb.IndexSchema{
						Name:    "image",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "Image"},
					},
				},
			},
		},
	}

	client, _ := newMemDBClient(schema, 3)
	repo := &memDBRepository{
		timeout:   time.Duration(3 * time.Second),
		tableName: tableName,
	}
	repo.client = client
	return repo, nil
}

func (r *memDBRepository) Store(repoImage string, artrepo string, image string, tags []string) (*promoter.Repos, error) {
	// Create a write transaction
	txn := r.client.Txn(true)

	// Insert some repo
	repo := &promoter.Repos{repoImage, artrepo, image, tags}

	if err := txn.Insert(r.tableName, repo); err != nil {
		panic(err)
	}

	// Commit the transaction
	txn.Commit()

	// Create read-only transaction
	txn = r.client.Txn(false)
	defer txn.Abort()

	// Lookup by email
	raw, err := txn.First(r.tableName, "id", repoImage)
	if err != nil {
		panic(err)
	}

	// Say hi!
	fmt.Printf("Hello %s!\n", raw.(*promoter.Repos).Tags)

	return nil, nil
}

// Read returns the tags currently existing in repoImage
func (r *memDBRepository) Read(repoImage string) ([]string, error) {
	txn := r.client.Txn(false)
	defer txn.Abort()

	fmt.Println("I'm in read")
	// Lookup by repoImage
	raw, err := txn.First(r.tableName, "id", repoImage)
	if err != nil {
		return nil, err
	}

	// Say hi!
	fmt.Printf("This is my tags! %v\n", raw.(*promoter.Repos).Tags)

	return raw.(*promoter.Repos).Tags, nil

}

func (r *memDBRepository) UpdateTags(repoImage string, repo string, image string, newTags []string) error {
	fmt.Println("I'm in update")
	currentTags, err := r.Read(repoImage)
	if err != nil {
		fmt.Println("Unable to find any current repoImage")
	}
	fmt.Printf("Here is the current tags %v", currentTags)

	// newTags will allways only contain 1 value
	realTag := promoter.AppendIfMissing(currentTags, newTags[0])
	fmt.Println(realTag)

	// Update/Create the tags in repoImage
	_, err = r.Store(repoImage, repo, image, realTag)
	if err != nil {
		return err
	}
	return nil
}

/*
		// List all the people
		it, err := txn.Get("person", "id")
		if err != nil {
			panic(err)
		}

		fmt.Println("All the people:")
		for obj := it.Next(); obj != nil; obj = it.Next() {
			p := obj.(*Person)
			fmt.Printf("  %s\n", p.Name)
		}

		// Range scan over people with ages between 25 and 35 inclusive
		it, err = txn.LowerBound("person", "age", 25)
		if err != nil {
			panic(err)
		}

		fmt.Println("People aged 25 - 35:")
		for obj := it.Next(); obj != nil; obj = it.Next() {
			p := obj.(*Person)
			if p.Age > 35 {
				break
			}
			fmt.Printf("  %s is aged %d\n", p.Name, p.Age)
		}
	}
*/

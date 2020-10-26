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
	/*
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(mongoTimeout)*time.Second)
		defer cancel()
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
		if err != nil {
			return nil, err
		}
		err = client.Ping(ctx, readpref.Primary())
		if err != nil {
			return nil, err
		}
	*/
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

func (r *memDBRepository) StartupUpdate(repoImage string, artrepo string, image string, tags []string) (*promoter.Repos, error) {

	// Insert multiple things at the same time.
	// Create a write transaction
	txn := r.client.Txn(true)

	// Insert some repo
	repo := []*promoter.Repos{
		&promoter.Repos{"repo1/app1", "repo1", "app1", []string{"v1.0.0", "v2.0.0"}},
		&promoter.Repos{"repo2/app2", "repo2", "app2", []string{"SNAPSHOT-1", "123456"}},
		&promoter.Repos{"repo2/app3", "repo2", "app3", []string{"SNAPSHOT-2", "567890"}},
	}

	for _, r := range repo {
		if err := txn.Insert("repo", r); err != nil {
			panic(err)
		}
	}

	// Commit the transaction
	txn.Commit()

	// Create read-only transaction
	txn = r.client.Txn(false)
	defer txn.Abort()

	// Lookup by email
	raw, err := txn.First("repo", "id", "repo2/app2")
	if err != nil {
		panic(err)
	}

	// Say hi!
	fmt.Printf("Hello %s!\n", raw.(*promoter.Repos).Image)

	return nil, nil
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

	// Lookup by repoImage
	raw, err := txn.First(r.tableName, "id", repoImage)
	if err != nil {
		panic(err)
	}

	// Say hi!
	fmt.Printf("This is my tags! %v\n", raw.(*promoter.Repos).Tags)

	return raw.(*promoter.Repos).Tags, nil

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

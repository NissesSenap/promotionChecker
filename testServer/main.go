package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type Tags struct {
	Repo         string
	Path         string
	Created      string
	CreatedBy    string
	LastModified string
	ModifiedBy   string
	LastUpdated  string
	Children     *[]Children
	URI          string
}

type Children struct {
	URI    string
	Folder bool
}

var Mylist []Children

func main() {
	// Start with a base list
	startList()
	fmt.Println("The app is starting")

	// Create the handlers
	http.HandleFunc("/api/storage/repo1/app1", tags)
	http.HandleFunc("/api/storage/repo2/app2", tags2)
	http.HandleFunc("/update", update)
	http.HandleFunc("/uploads", updateUploads)
	http.HandleFunc("/webhook", webhook)
	http.ListenAndServe(":8081", nil)
}

func webhook(w http.ResponseWriter, r *http.Request) {
	// Don't care what comes in I just return ok and see what request we got.
	fmt.Println("We are inside the webhook")

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}
	bodyString := string(bodyBytes)
	fmt.Println(bodyString)

	fmt.Println("###### HEADER INFO################")
	fmt.Println(r.Header)
	// Return a ok in normal text
	js := []byte("ok")
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func update(w http.ResponseWriter, r *http.Request) {
	// Don't care what comes in I just return ok and see what request we got.
	fmt.Println("We are inside the /UPDATE")

	postTags()
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}
	bodyString := string(bodyBytes)
	fmt.Println(bodyString)

	// Return a ok in normal text
	js := []byte("ok")
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func postTags() {
	Mylist = []Children{
		{URI: "/1.0.1-SNAPSHOT", Folder: true},
		{URI: "/884b988", Folder: true},
		{URI: "/MyNewTAG", Folder: true},
	}
	fmt.Println("I have now updated Mylist")

}

func updateUploads(w http.ResponseWriter, r *http.Request) {
	// Don't care what comes in I just return ok and see what request we got.
	fmt.Println("We are inside the /updateUploads")

	postTagsUploads()
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}
	bodyString := string(bodyBytes)
	fmt.Println(bodyString)

	// Return a ok in normal text
	js := []byte("ok")
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func postTagsUploads() {
	Mylist = []Children{
		{URI: "/_uploads", Folder: true},
		{URI: "/1.0.1-SNAPSHOT", Folder: true},
		{URI: "/884b988", Folder: true},
		{URI: "/MyNewTAG", Folder: true},
	}
	fmt.Println("I have now updated Mylist with /_uploads")

}

func startList() {
	Mylist = []Children{
		{URI: "/1.0.1-SNAPSHOT", Folder: true},
		{URI: "/884b988", Folder: true},
	}
}
func tags(w http.ResponseWriter, r *http.Request) {

	myTags := Tags{
		Repo:         "repo1",
		Path:         "/app1",
		Created:      "2020-05-28T10:32:09.490+02:00",
		CreatedBy:    "user1",
		LastModified: "2020-05-28T10:32:09.490+02:00",
		ModifiedBy:   "user1",
		LastUpdated:  "2020-05-28T10:32:09.490+02:00",
		Children:     &Mylist,
		URI:          "http://localhost:8081/api/storage/repo1/app1",
	}

	fmt.Println("This is MyList#########################")
	fmt.Println(Mylist)

	fmt.Println(myTags)

	fmt.Println(r.Header)
	js, err := json.Marshal(myTags)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func tags2(w http.ResponseWriter, r *http.Request) {
	myTags := Tags{
		Repo:         "repo2",
		Path:         "/app2",
		Created:      "2020-05-28T10:32:09.490+02:00",
		CreatedBy:    "user1",
		LastModified: "2020-05-28T10:32:09.490+02:00",
		ModifiedBy:   "user1",
		LastUpdated:  "2020-05-28T10:32:09.490+02:00",
		Children: &[]Children{
			{URI: "/v1.0.0", Folder: true},
			{URI: "/123456", Folder: true},
		},
		URI: "http://localhost:8081/api/storage/repo2/app2",
	}

	fmt.Println(myTags)

	js, err := json.Marshal(myTags)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

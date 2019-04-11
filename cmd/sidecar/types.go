package main

//Event data from secret/configmap
type Event struct {
	entry     []Entry
	action    string
	cmid      string
	namespace string
}

//Entry single entry from configmap/secret (data/strintgData)
type Entry struct {
	name string
	data string
}

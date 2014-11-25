package main

import (
	"net/http"
	"fmt"
	"io/ioutil"
	"github.com/russross/blackfriday"
	"html/template"
	"github.com/pcwizz/xattr"
)

func main (){
	http.HandleFunc("/", markdownServer)
	http.Handle("/css/", http.FileServer(http.Dir(".www/css/")))
	http.Handle("/js/", http.FileServer(http.Dir(".www/js/")))
	http.Handle("/img/", http.FileServer(http.Dir(".www/img/")))
	http.ListenAndServe(":8080", nil)
}

type Page struct {
	Title string
	Content template.HTML 
	Tags string
}

func markdownServer(w http.ResponseWriter, r *http.Request){
	path := r.URL.Path[1:]
	//Place to do special routes
	switch path{
		case "":
			path = "index"
	}
	file, err := ioutil.ReadFile("www/" + path + ".md")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	content := blackfriday.MarkdownCommon(file)//TODO: Cache rendered templates.
	t, err := template.ParseFiles("templates/main.html")
	if err != nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//Read tags from extended attribute
	tags, err := xattr.Getxattr("www/" + path + ".md", "tags")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	page := Page{Title: path, Content: template.HTML(fmt.Sprintf("%s", content)), Tags: string(tags)}
	t.Execute(w,page)
}

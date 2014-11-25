/*
	Markdown server, compiles markdown and templates on request.
    Copyright (C) 2014  Morgan Hill <morgan@pcwizzltd.com>

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <http://www.gnu.org/licenses/>.

*/

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

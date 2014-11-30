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
	"strings"
	"fmt"
	"io/ioutil"
	"github.com/russross/blackfriday"
	"html/template"
	"github.com/pcwizz/xattr"
	"github.com/gorilla/feeds"
	"time"
	"os"
)

func main (){
	http.HandleFunc("/", markdownServer)
	http.HandleFunc("/feed", feedServer)
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
	//Read title from extended attribute
	title, err := xattr.Getxattr("www/" + path + ".md", "title")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	page := Page{Title: string(title), Content: template.HTML(fmt.Sprintf("%s", content)), Tags: string(tags)}
	t.Execute(w,page)
}

func feedServer (w http.ResponseWriter, r *http.Request){
	Domain := "http://localhost:8080"
	WebRoot := "www"
	FeedAuthorName := "Morgan Hill"
	FeedAuthorEmail := "morgan@pcwizzltd.com"
	//Will grab this information from a config file at some point.
	feed := &feeds.Feed{
		Title:		"Markdown server",
		Link: 		&feeds.Link{Href: Domain},
		Description:"Not a real feed; unless it is real",
		Author:		&feeds.Author{FeedAuthorName, FeedAuthorEmail},
		Created:	time.Now(),
	}
	entries, err := exploreDirectory("www", []string {"www/css", "www/js", "www/img"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	var Items []*feeds.Item
	for _, entry := range entries {
		//Title
		TitleRaw, err := xattr.Getxattr(entry, "title")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		Title := string(TitleRaw)
		//Time
		entryInfo, err := os.Stat(entry)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		Time := entryInfo.ModTime()
		//Author
		var Author string
		AuthorRaw, err := xattr.Getxattr(entry, "author")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if AuthorRaw == nil {
			Author = FeedAuthorName
		} else {
			Author = string(AuthorRaw)
		}
		//Description
		DescriptionRaw, err := xattr.Getxattr(entry, "description")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		Description := string(DescriptionRaw)
		//Email
		var Email string
		EmailRaw, err := xattr.Getxattr(entry, "Email")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if EmailRaw == nil {
			Email = FeedAuthorEmail
		} else {
			Email = string(EmailRaw)
		}
		//Link
		Link := strings.TrimPrefix(entry, WebRoot + "/")
		Link = strings.TrimSuffix(Link, ".md")
		Link = Domain + "/" + Link
		//Add object
		Items = append(Items, &feeds.Item{
			Title: Title,
			Link: &feeds.Link{Href: Link},
			Description: Description,
			Author: &feeds.Author{Author, Email},
			Created: Time,
		})
	}
	feed.Items = Items
	atom, err := feed.ToAtom()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	fmt.Fprintf(w, "%v\n", atom)
}

func exploreDirectory (path string, excludes []string) ([]string, error){
	var output []string
	//Get entries
	info, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for i, _ := range info {
		//Using a closure, to solve an issue with nested for loops.
		skip := func () bool {
			 for _, exclude := range excludes {
				if exclude == path + "/" + info[i].Name() {
					return true		
				}
			}
			return false
		}()
		if skip {
			continue
		}
		if !info[i].IsDir() {
			//Output file
			output = append(output, path + "/" + info[i].Name())
		} else {
			//Explore it; it's a directory.
			entries, err := exploreDirectory(path + "/"  + info[i].Name(), excludes)
			if err != nil {
				return nil, err
			}
			output = append(output, entries...)
		}
	}
	return output, nil
}

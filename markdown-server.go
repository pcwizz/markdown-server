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
	"encoding/json"
	"net/http/httputil"
	"net/url"
	"log"
)

type templateCacheObj struct{
	Expiry int64
	Obj *template.Template
}

type pageCacheObj struct{
	Expiry int64
	Obj Page
}

var templateCache map[string]templateCacheObj
var pageCache map[string]pageCacheObj

//Configuration
var config Config
type Redirect struct {
	Begin string
	End string
	Silent bool
}

type Author struct {
	Name string
	Email string
}

type Feed struct {
	Title string
	Root string
	Path string
	Excludes []string
	Description string
	Author Author
}

type Config struct {
	WebRoot string
	Domain string
	Author Author
	InternalRedirects []Redirect
	ExternalRedirects []Redirect
	Statics []struct {
		PathInternal string
		PathExternal string
	}
	Feeds []Feed
	ContentExpiry int64// Time period in seconds; controls cache.
}

//Load configuration from file
func loadConfig (){
	File, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Panic(err)
	}
	err = json.Unmarshal(File, &config)
	if err != nil {
		log.Panic(err)
	}
}

func main (){
	templateCache = make(map[string]templateCacheObj)
	pageCache = make(map[string]pageCacheObj)
	loadConfig()
	http.HandleFunc("/", markdownServer)
	//Set feed handlers
	for _, feed := range config.Feeds {
		http.HandleFunc("/" + feed.Path, feed.feedServer)
	}
	//Statics
	for _, static := range config.Statics { 
		http.Handle("/" + static.PathExternal, http.FileServer(http.Dir("." + 
			config.WebRoot + "/" + static.PathInternal)))
	}
	//Internal redirects
	for _, internalRedirect := range config.InternalRedirects {
		if internalRedirect.Silent {
			url, err := url.Parse(config.Domain + "/" + internalRedirect.End)
			if err != nil {
				fmt.Print(err)
			}
			rp := httputil.NewSingleHostReverseProxy(url)
			http.HandleFunc("/" + internalRedirect.Begin, reverseProxyHandler(rp))
		} else {
			http.Handle("/" + internalRedirect.Begin,
				 http.RedirectHandler(internalRedirect.End, 302))
		} 
	}
	//External redirects
	for _, externalRedirect := range config.ExternalRedirects {
		if externalRedirect.Silent {
			url, err := url.Parse(externalRedirect.End)
			if err != nil {
				fmt.Print(err)
			}
			rp := httputil.NewSingleHostReverseProxy(url)
			http.HandleFunc("/" + externalRedirect.Begin, reverseProxyHandler(rp))
		} else {
			http.Handle("/" + externalRedirect.Begin,
				 http.RedirectHandler(externalRedirect.End, 301))
		} 
	}
	http.ListenAndServe(":8080", nil)
}

// checkTemplateCacheExpiry: returns true if object is cached and not expired
func (obj templateCacheObj) CheckTemplateExpiry () bool{
	//Check expiry
	if obj.Expiry > time.Now().Unix() {
		return true
	}
	return false
}

// checkPageCacheExpiry: returns true if object is cached and not expired
func (obj pageCacheObj) CheckPageExpiry () bool{
	//Check expiry
	if obj.Expiry > time.Now().Unix() {
		return true
	}
	return false
}

//retriveTemplate: Get template from cache if valid, otherwise compiles template and add it to cache
func retriveTemplate (templateName string) (*template.Template, error) {
	if _, ok := templateCache[templateName]; ok {
		if templateCache[templateName].CheckTemplateExpiry() {
			return templateCache[templateName].Obj, nil
		}
	}
	t, err := template.ParseFiles("templates/" + templateName)
	if err != nil{
		return t, err
	}
	go cacheTemplate(templateName, t)
	return t, nil
}

//cacheTemplate: Adds template to cache
func cacheTemplate(key string, obj *template.Template){
	templateCache[key] = templateCacheObj{
		Expiry: time.Now().Unix() + config.ContentExpiry,
		Obj: obj,
	}
}

//retrivePage: Cache Page objects, otherwise attempt to compile
func retrivePage(path string) (Page, error){
	if _, ok := pageCache[path]; ok {
		if pageCache[path].CheckPageExpiry() {
			return pageCache[path].Obj, nil
		}
	}
	file, err := ioutil.ReadFile(config.WebRoot + "/" + path + ".md")
	if err != nil {
		return Page{}, err
	}
	content := blackfriday.MarkdownCommon(file)
	//Read tags from extended attribute
	tags, err := xattr.Getxattr(config.WebRoot + "/" + path + ".md", "tags")
	if err != nil {
		return Page{}, err
	}
	//Read title from extended attribute
	title, err := xattr.Getxattr(config.WebRoot + "/" + path + ".md", "title")
	if err != nil {
		return Page{}, err
	}
	page := Page{Title: string(title), Content: template.HTML(fmt.Sprintf("%s", content)), Tags: string(tags)}
	go cachePage(path, page)
	return page, nil
}

//cachePage: Adds page to cache
func cachePage(key string, obj Page){
	pageCache[key] = pageCacheObj{
		Expiry: time.Now().Unix() + config.ContentExpiry,
		Obj: obj,
	}
}

type Page struct {
	Title string
	Content template.HTML 
	Tags string
}

//Reverse proxy
func reverseProxyHandler(rp *httputil.ReverseProxy) func (w http.ResponseWriter, r *http.Request) {
	return func (w http.ResponseWriter, r *http.Request){
			rp.ServeHTTP(w, r)
		}
}


func markdownServer(w http.ResponseWriter, r *http.Request){
	path := r.URL.Path[1:]
	if path == ""{
		path = "index"
	}
	t, err := retriveTemplate("main.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	page, err := retrivePage(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w,page)
}

func (feedConf Feed) feedServer (w http.ResponseWriter, r *http.Request){
	var author Author
	if feedConf.Author != author {
		author = feedConf.Author
	} else {
		author = config.Author
	}
	feed := &feeds.Feed{
		Title:		feedConf.Title,
		Link: 		&feeds.Link{Href: config.Domain},
		Description:feedConf.Description,
		Author:		&feeds.Author{author.Name, author.Email},
		Created:	time.Now(),
	}
	entries, err := exploreDirectory(config.WebRoot + feedConf.Root, feedConf.Excludes)
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
			Author = author.Name
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
			Email = author.Email
		} else {
			Email = string(EmailRaw)
		}
		//Link
		Link := strings.TrimPrefix(entry, config.WebRoot + "/")
		Link = strings.TrimSuffix(Link, ".md")
		Link = config.Domain + "/" + Link
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

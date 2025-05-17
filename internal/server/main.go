package main

import (
	"html/template"
	"net/http"
)

type User struct {
	Name string
}

var templates = template.Must(template.ParseGlob("web/template/*.html"))

func main() {
	http.HandleFunc("/", indexHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	http.ListenAndServe(":8080", nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	user := User{
		Name: "User",
	}
	templates.ExecuteTemplate(w, "index.html", user)
}

package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

type FileInfo struct {
	Name  string
	Link  string
}

type Contents map[string][]FileInfo

var cwd string
var hostname string

func main() {
	var host string

	cwd, host = getArgs()
	hostname = getHostname()
	template, err := loadTemplate()
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Hostname is '%s'", hostname)
	log.Printf("Starting server on '%s'", host)

	router := gin.Default()
	router.SetHTMLTemplate(template)
	router.GET("/*path", getDocument)

	router.Run(host)
}

func getArgs() (string, string) {
	switch len(os.Args) {
	case 3:
		return os.Args[1], os.Args[2]
	case 2:
		return os.Args[1], "127.0.0.1:8080"
	default:
		return ".", "127.0.0.1:8080"
	}
}

func getHostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "Hostme"
	}

	return name
}

func loadTemplate() (*template.Template, error) {
	t := template.New("")
	for name, file := range Assets.Files {
		if file.IsDir() || !strings.HasSuffix(name, ".tmpl") {
			continue
		}
		h, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}
		t, err = t.New(name).Parse(string(h))
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}

func getMarkdown(path string) (string, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	if len(file) == 0 {
		return "", nil
	}

	renderer := html.NewRenderer(
		html.RendererOptions{Flags: html.CommonFlags | html.HrefTargetBlank},
	)

	markdownParser := parser.NewWithExtensions(parser.CommonExtensions | parser.AutoHeadingIDs)
	html := markdown.ToHTML(file, markdownParser, renderer)

	return string(html), nil
}

func getContents(hidden bool) (Contents, error) {
	contents := make(Contents)
	files, err := ioutil.ReadDir(cwd)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		if !hidden && strings.HasPrefix(file.Name(), ".") {
			continue
		}

		infos := make([]FileInfo, 0)
		markdowns, err := ioutil.ReadDir(filepath.Join(cwd, file.Name()))
		if err != nil {
			return nil, err
		}

		for _, md := range markdowns {
			if !hidden && strings.HasPrefix(md.Name(), ".") {
				continue
			}

			if filepath.Ext(md.Name()) == ".md" {
				infos = append(infos, FileInfo{
					Name: strings.TrimSuffix(md.Name(), ".md"),
					Link: filepath.Join(file.Name(), md.Name()),
				})
			}
		}

		if len(infos) > 0 {
			contents[file.Name()] = infos
		}
	}

	return contents, nil
}

func getDocument(c *gin.Context) {
	path := filepath.Join(cwd, c.Param("path"))
	_, hidden := c.GetQuery("hidden")
	md, err := getMarkdown(path)
	header := "Contents"

	var contents Contents

	if err != nil || len(md) == 0 {
		contents, err = getContents(hidden)
		if err != nil {
			c.Status(500)
			return
		}
	} else {
		header = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	c.HTML(http.StatusOK, "/templates/index.tmpl", gin.H{
		"title":    hostname,
		"header":   header,
		"contents": contents,
		"markdown": template.HTML(md),
	})
}

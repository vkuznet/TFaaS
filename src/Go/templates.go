package main

import (
	"bytes"
	"html/template"
	"path/filepath"
)

// consume list of templates and release their full path counterparts
func fileNames(tdir string, filenames ...string) []string {
	flist := []string{}
	for _, fname := range filenames {
		flist = append(flist, filepath.Join(tdir, fname))
	}
	return flist
}

// parse template with given data
func parseTmpl(tdir, tmpl string, data interface{}) string {
	buf := new(bytes.Buffer)
	filenames := fileNames(tdir, tmpl)
	funcMap := template.FuncMap{
		// The name "oddFunc" is what the function will be called in the template text.
		"oddFunc": func(i int) bool {
			if i%2 == 0 {
				return true
			}
			return false
		},
	}
	t := template.Must(template.New(tmpl).Funcs(funcMap).ParseFiles(filenames...))
	err := t.Execute(buf, data)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

// Templates structure
type Templates struct {
	header, footer, main string
}

// Header method for Templates structure
func (q Templates) Header(tdir string, tmplData map[string]interface{}) string {
	if q.header != "" {
		return q.header
	}
	q.header = parseTmpl(tdir, "header.tmpl", tmplData)
	return q.header
}

// Footer method for Templates structure
func (q Templates) Footer(tdir string, tmplData map[string]interface{}) string {
	if q.footer != "" {
		return q.footer
	}
	q.footer = parseTmpl(tdir, "footer.tmpl", tmplData)
	return q.footer
}

// Main method for Templates structure
func (q Templates) Main(tdir string, tmplData map[string]interface{}) string {
	if q.header != "" {
		return q.header
	}
	q.header = parseTmpl(tdir, "main.tmpl", tmplData)
	return q.header
}

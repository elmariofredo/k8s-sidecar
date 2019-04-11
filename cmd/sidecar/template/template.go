package template

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	log "github.com/golang/glog"
)

// Template wraps a text template and error, to make it easier to execute multiple templates and only check for errors
// once at the end (assuming one is only interested in the first error, which is usually the case).
type Template struct {
	tmpl *template.Template
	err  error
}

var funcs = template.FuncMap{
	"toUpper": strings.ToUpper,
	"toLower": strings.ToLower,
	"title":   strings.Title,
	// join is equal to strings.Join but inverts the argument order
	// for easier pipelining in templates.
	"join": func(sep string, s []string) string {
		return strings.Join(s, sep)
	},
	"reReplaceAll": func(pattern, repl, text string) string {
		re := regexp.MustCompile(pattern)
		return re.ReplaceAllString(text, repl)
	},
	"saveString": func(text string) string {
		text = strings.Replace(text, "\"", "", -1)
		return text
	},
	"timestemp": func() string {
		timestamp := strconv.FormatInt(time.Now().UTC().Unix(), 10)
		return timestamp
	},
	"indent": func(i int, s string) string {
		r, _ := regexp.Compile("(^|\n)")
		return r.ReplaceAllString(s, "\n"+strings.Repeat(" ", i))[1:]
	},
}

//LoadTemplateFile reads and parses all templates defined in the given file and constructs.Template.
func LoadTemplateFile(path string) (*Template, error) {
	log.V(1).Infof("Loading templates from %q", path)
	tmpl, err := template.New("").Option("missingkey=zero").Funcs(funcs).ParseFiles(path)
	if err != nil {
		return nil, err
	}
	return &Template{tmpl: tmpl}, nil
}

//LoadTemplateValue reads and parses all templates defined in the given file and constructs.Template.
func LoadTemplateValue(templateDef string) (*Template, error) {
	tmpl, err := template.New("").Option("missingkey=zero").Funcs(funcs).Parse(templateDef)
	if err != nil {
		return nil, err
	}
	return &Template{tmpl: tmpl}, nil
}

//Init base init template
func Init() *Template {
	tmpl := template.New("").Option("missingkey=zero").Funcs(funcs)
	return &Template{tmpl: tmpl}
}

// Execute parses the provided text (or returns it unchanged if not a Go template), associates it with the templates
// defined in t.tmpl (so they may be referenced and used) and applies the resulting template to the specified data
// object, returning the output as a string.
func (t *Template) Execute(text string, data interface{}) (string, error) {
	log.V(2).Infof("Executing template %q...", text)
	if !strings.Contains(text, "{{") {
		log.V(2).Infof("  returning unchanged.")
		return text, nil
	}

	if t.err != nil {
		return "", t.err
	}
	var tmpl *template.Template
	tmpl, t.err = t.tmpl.Clone()
	if t.err != nil {
		return "", t.err
	}
	tmpl, t.err = tmpl.New("").Option("missingkey=zero").Parse(text)
	if t.err != nil {
		log.V(2).Infof("  parse failed.")
		return "", t.err
	}
	var buf bytes.Buffer
	t.err = tmpl.Execute(&buf, data)
	ret := buf.String()
	log.V(2).Infof("  returning %q.", ret)
	return ret, t.err
}

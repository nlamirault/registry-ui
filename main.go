package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/jgsqware/registry-ui/auth"
	"github.com/spf13/viper"
)

const views = "views/"
const catalogTplt = `
Repositories [{{.Repositories | len}}]:
	{{range $key, $value := .Repositories}} ➜ {{$key}} [{{$value | len}}]:
		{{range $value}} ➜ {{if ne $key "-"}}{{$key}}/{{end}}{{.}}
		{{end}}
	{{end}}
`

var registryURI string

// {{range .Repositories}} ➜ {{.}}
// {{end}}
type catalog struct {
	Registry     string
	Repositories map[string][]string
}

type _catalog struct {
	Repositories []string `json:"repositories"`
}

func renderTemplate(w http.ResponseWriter, tmpl string, p interface{}) error {
	t, err := template.ParseFiles(views + tmpl + ".html")
	if err != nil {
		return fmt.Errorf("parsing view %s:%v", tmpl, err)
	}
	err = t.Execute(w, p)
	if err != nil {
		return fmt.Errorf("rendering view %s:%v", tmpl, err)
	}
	return nil
}

func loadPage(p string) interface{} {
	switch p {
	case "catalog":
		return GetCatalog()
	case "notfound":
		return "notfound"
	default:
		return nil
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	page := strings.TrimRight(r.URL.Path[len("/"):], "/")
	p := loadPage(page)
	if p == nil {
		http.Redirect(w, r, "/notfound", http.StatusNotFound)
		return
	}
	err := renderTemplate(w, page, p)
	if err != nil {
		log.Printf("%s handler: %v", page, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	viper.SetEnvPrefix("registryui")
	viper.SetDefault("port", 8080)
	viper.AutomaticEnv()

	registryURI = viper.GetString("hub_uri")
	if registryURI == "" {
		log.Fatalln("no registry uri provided")
	}

	http.DefaultClient.Transport = &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		DisableCompression: true,
	}

	var isCmd = flag.Bool("sout", false, "Display registry in stdout")
	flag.Parse()

	if *isCmd == true {
		c := GetCatalog()
		err := template.Must(template.New("catalog").Parse(catalogTplt)).Execute(os.Stdout, c)
		if err != nil {
			log.Fatalf("rendering : %v", err)
		}
		os.Exit(0)
	}
	s := fmt.Sprintf(":%d", viper.GetInt("port"))
	log.Printf("Starting Server on %s\n", s)
	http.HandleFunc("/", viewHandler)
	http.ListenAndServe(s, nil)

}

func GetCatalog() catalog {
	req, err := http.NewRequest("GET", "http://"+registryURI+"/v2/_catalog", nil)
	res, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Fatalf("retrieving catalog: %v", err)
	}

	if res.StatusCode == http.StatusUnauthorized {
		err := auth.Authenticate(res, req)

		if err != nil {
			log.Fatalf("authenticating: %v", err)
		}

		res, err = http.DefaultClient.Do(req)
		if err != nil {
			log.Fatalf("retrieving catalog: %v", err)
		}

	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("reading body: %v", err)
	}

	var d _catalog
	if err := json.Unmarshal(b, &d); err != nil {
		log.Fatalf("marshalling result: err")
	}

	var c catalog
	c.Repositories = make(map[string][]string)
	for _, repository := range d.Repositories {
		if strings.Contains(repository, "/") {
			r := strings.Split(repository, "/")
			c.Repositories[r[0]] = append(c.Repositories[r[0]], r[1])
		} else {
			c.Repositories["-"] = append(c.Repositories["-"], repository)
		}
	}
	return c
}

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
		{{range $value}} ➜ {{.Name}}
			➜ Tags:{{.Tags}}
		{{end}}
	{{end}}
`

var registryURI string

type image struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type catalog struct {
	Registry     string
	Repositories map[string][]image
}

type _catalog struct {
	Repositories []string `json:"repositories"`
}

type action interface {
	GetAction() string
}

func (_catalog) GetAction() string {
	return "_catalog"
}

func (image) GetAction() string {
	return "image"
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

func ToIndentJSON(v interface{}) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return nil, err
	}
	return b, nil
}

func doRequest(method string, uri string) []byte {
	req, err := http.NewRequest(method, uri, nil)
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

	return b
}

func GetCatalog() catalog {
	var d _catalog
	b := doRequest("GET", "http://"+registryURI+"/v2/_catalog")
	if err := json.Unmarshal(b, &d); err != nil {
		log.Fatalf("marshalling result: err")
	}

	var c catalog
	c.Registry = registryURI
	c.Repositories = make(map[string][]image)
	for _, repository := range d.Repositories {
		if strings.Contains(repository, "/") {
			r := strings.Split(repository, "/")
			c.Repositories[r[0]] = append(c.Repositories[r[0]], GetTags(repository))
		} else {
			c.Repositories["-"] = append(c.Repositories["-"], GetTags(repository))
		}
	}
	return c
}

func GetTags(imageName string) image {
	var i image
	b := doRequest("GET", "http://"+registryURI+"/v2/"+imageName+"/tags/list")

	if err := json.Unmarshal(b, &i); err != nil {
		log.Fatalf("marshalling result: err")
	}
	return i

}

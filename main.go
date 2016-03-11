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

	"github.com/gorilla/mux"
	"github.com/jgsqware/registry-ui/auth"
	"github.com/spf13/viper"
	"github.com/wemanity-belgium/hyperclair/clair"
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

var templates = template.Must(template.ParseFiles(views+"catalog.html", views+"footer.html", views+"header.html", views+"index.html", views+"report.html", views+"users.html"))

var registryURI string

type templateData struct {
	AccountMgmt      bool
	HyperclairReport bool
	Data             interface{}
}
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

func renderTemplate(w http.ResponseWriter, tmpl string, p interface{}) error {
	d := templateData{AccountMgmt: viper.GetBool("account_mgmt_enabled"), HyperclairReport: viper.GetBool("hyperclair_report_enabled"), Data: p}
	err := templates.ExecuteTemplate(w, tmpl+".html", d)
	if err != nil {
		return fmt.Errorf("rendering view %s:%v", tmpl, err)
	}
	return nil
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Index Handler")
	err := renderTemplate(w, "index", GetCatalog())
	if err != nil {
		log.Printf("index handler: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
func reportsHandler(w http.ResponseWriter, r *http.Request) {

}
func catalogHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Catalog Handler")
	err := renderTemplate(w, "catalog", GetCatalog())
	if err != nil {
		log.Printf("catalog handler: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func reportHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Report Handler")

	reg := r.FormValue("registry")
	image := r.FormValue("image")
	tag := r.FormValue("reference")
	if tag == "" {
		tag = "latest"
	}
	url := "http://hyperclair:9999/v1/" + image + "/analysis?realm=" + reg + "&reference=" + tag
	fmt.Println("url: " + url)
	var t clair.ImageAnalysis
	b := doRequest(http.MethodGet, url)

	if err := json.Unmarshal(b, &t); err != nil {
		log.Fatalf("marshalling result: ", err)
	}

	err := renderTemplate(w, "report", t)
	if err != nil {
		log.Printf("report handler: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		err := renderTemplate(w, "users", auth.Config)
		if err != nil {
			log.Fatalf("rendering users: %v", err)
		}
	case http.MethodPost:
		action := r.FormValue("method")
		switch action {
		case "delete":
			u := r.FormValue("username")
			err := auth.DeleteUser(u)
			if err != nil {
				log.Fatalf("deleting user '%v': %v", u, err)
			}
		case "add":
			u, p := r.FormValue("username"), r.FormValue("password")
			err := auth.AddUser(u, p)
			if err != nil {
				log.Fatalf("adding user '%v': %v", u, err)
			}
		}
		http.Redirect(w, r, "/users", http.StatusFound)
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

	if viper.GetBool("account_mgmt_enabled") {
		if viper.GetString("account_mgmt_config") == "" {
			log.Fatalln("account management enabled but no config file")
		}
		auth.ReadConfig(viper.GetString("account_mgmt_config"))
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

	router := mux.NewRouter()
	router.Path("/catalog").HandlerFunc(catalogHandler).Methods("GET")
	router.Path("/report").HandlerFunc(reportHandler).Methods("GET")
	router.Path("/").HandlerFunc(indexHandler).Methods("GET")
	router.Path("/users").HandlerFunc(usersHandler).Methods("GET", "POST")
	router.PathPrefix("/css").Handler(http.StripPrefix("/css/", http.FileServer(http.Dir("./views/static/css"))))
	router.PathPrefix("/styles").Handler(http.StripPrefix("/styles/", http.FileServer(http.Dir("./views/static/styles"))))
	http.Handle("/", router)
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

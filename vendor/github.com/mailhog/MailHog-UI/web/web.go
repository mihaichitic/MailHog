package web

import (
	"bytes"
	"errors"
	"html/template"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/pat"
	"github.com/mailhog/MailHog-UI/config"
)

var APIHost string
var WebPath string

type Web struct {
	config *config.Config
	asset  func(string) ([]byte, error)
}

func CreateWeb(cfg *config.Config, r http.Handler, asset func(string) ([]byte, error)) *Web {
	web := &Web{
		config: cfg,
		asset:  asset,
	}

	pat := r.(*pat.Router)

	WebPath = cfg.WebPath

	log.Printf("Serving under http://%s%s/", cfg.UIBindAddr, WebPath)

	pat.Path(WebPath + "/images/{file:.*}").Methods("GET").HandlerFunc(web.Static("assets/images/{{file}}"))
	pat.Path(WebPath + "/css/{file:.*}").Methods("GET").HandlerFunc(web.Static("assets/css/{{file}}"))
	pat.Path(WebPath + "/js/{file:.*}").Methods("GET").HandlerFunc(web.Static("assets/js/{{file}}"))
	pat.Path(WebPath + "/fonts/{file:.*}").Methods("GET").HandlerFunc(web.Static("assets/fonts/{{file}}"))
	pat.StrictSlash(true).Path(WebPath + "/").Methods("GET").HandlerFunc(web.Index())

	return web
}

func (web Web) Static(pattern string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		fp := strings.TrimSuffix(pattern, "{{file}}") + req.URL.Query().Get(":file")
		//if b, err := web.asset(fp); err == nil {
		if b, err := webAssetWrapper(web, fp); err == nil {
			ext := filepath.Ext(fp)

			w.Header().Set("Content-Type", mime.TypeByExtension(ext))
			w.WriteHeader(200)
			w.Write(b)
			return
		}
		log.Printf("[UI] File not found: %s", fp)
		w.WriteHeader(404)
	}
}

func (web Web) Index() func(http.ResponseWriter, *http.Request) {
	tmpl := template.New("index.html")
	tmpl.Delims("[:", ":]")

	//asset, err := web.asset("assets/templates/index.html")
	asset, err := webAssetWrapper(web, "assets/templates/index.html")
	if err != nil {
		log.Fatalf("[UI] Error loading index.html: %s", err)
	}

	//asset, err := readLocalAsset("assets/templates/index.html")
	//if err != nil {
	//	log.Printf("Fallback to builtin asset assets/templates/index.html")
	//	asset, err = web.asset("assets/templates/index.html")
	//	if err != nil {
	//		log.Fatalf("[UI] Error loading index.html: %s", err)
	//	}
	//}

	tmpl, err = tmpl.Parse(string(asset))
	if err != nil {
		log.Fatalf("[UI] Error parsing index.html: %s", err)
	}

	layout := template.New("layout.html")
	layout.Delims("[:", ":]")

	//asset, err = web.asset("assets/templates/layout.html")
	asset, err = webAssetWrapper(web, "assets/templates/layout.html")
	if err != nil {
		log.Fatalf("[UI] Error loading layout.html: %s", err)
	}

	//asset, err = readLocalAsset("assets/templates/layout.html")
	//if err != nil {
	//	log.Printf("Fallback to builtin asset assets/templates/layout.html")
	//	asset, err = web.asset("assets/templates/layout.html")
	//	if err != nil {
	//		log.Fatalf("[UI] Error loading layout.html: %s", err)
	//	}
	//}

	layout, err = layout.Parse(string(asset))
	if err != nil {
		log.Fatalf("[UI] Error parsing layout.html: %s", err)
	}

	return func(w http.ResponseWriter, req *http.Request) {
		data := map[string]interface{}{
			"config":  web.config,
			"Page":    "Browse",
			"APIHost": APIHost,
		}

		b := new(bytes.Buffer)
		err := tmpl.Execute(b, data)

		if err != nil {
			log.Printf("[UI] Error executing template: %s", err)
			w.WriteHeader(500)
			return
		}

		data["Content"] = template.HTML(b.String())

		b = new(bytes.Buffer)
		err = layout.Execute(b, data)

		if err != nil {
			log.Printf("[UI] Error executing template: %s", err)
			w.WriteHeader(500)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(200)
		w.Write(b.Bytes())
	}
}

func webAssetWrapper(web Web, assetPath string) ([]byte, error) {
	asset, err := readLocalAsset(assetPath)
	if err != nil {
		log.Printf("Fallback to builtin asset %s", assetPath)
		asset, err = web.asset(assetPath)
	}

	return asset, nil
}

// read asset from disk if exists
func readLocalAsset(assetPath string) ([]byte, error) {
	//currentPath, err := os.Getwd()
	//if err != nil {
	//	log.Fatalf("[UI] Error loading %s: %s", assetPath, err)
	//}

	ex, err := os.Executable()
	if err != nil {
		log.Fatalf("[UI] Error loading %s: %s", assetPath, err)
	}
	currentPath := filepath.Dir(ex)

	absoluteAssetPath := currentPath + "/" + assetPath;
	log.Printf("Reading local asset %s\n", absoluteAssetPath)
	if _, err := os.Stat(absoluteAssetPath); err != nil {
		return []byte{}, errors.New("Asset could not be found on disk")
	}

	asset, err := ioutil.ReadFile(absoluteAssetPath)
	if err != nil {
		log.Fatalf("[UI] Error loading %s: %s", assetPath, err)
	}

	return asset, nil
}

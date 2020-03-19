package main

import (
	"flag"
	"fmt"
	"os"

	gohttp "net/http"

	"github.com/gorilla/pat"
	"github.com/ian-kent/go-log/log"
	"github.com/mailhog/MailHog-Server/api"
	cfgapi "github.com/mailhog/MailHog-Server/config"
	"github.com/mailhog/MailHog-Server/smtp"
	"github.com/mailhog/MailHog-UI/assets"
	cfgui "github.com/mailhog/MailHog-UI/config"
	"github.com/mailhog/MailHog-UI/web"
	cfgcom "MailHog/config"
	"github.com/mailhog/http"
	"github.com/mailhog/mhsendmail/cmd"
	"golang.org/x/crypto/bcrypt"
)

var apiconf *cfgapi.Config
var uiconf *cfgui.Config
var comconf *cfgcom.Config
var exitCh chan int
var version string

func configure() {
	cfgcom.RegisterFlags()
	cfgapi.RegisterFlags()
	cfgui.RegisterFlags()
	flag.Parse()
	apiconf = cfgapi.Configure()
	uiconf = cfgui.Configure()
	comconf = cfgcom.Configure()

	apiconf.WebPath = comconf.WebPath
	uiconf.WebPath = comconf.WebPath
}

func main() {
	// to restore original asset from src/MailHog/vendor/github.com/mailhog/MailHog-UI/assets/assets.go, modify and probably having to deploy along with the binary
	// check in https://github.com/mailhog/MailHog-UI the original assets
	//err := assets.RestoreAsset("/go/src/MailHog/dist","assets/templates/index.html")
	//err := assets.RestoreAsset("/go/src/MailHog/dist","assets/js/controllers.js")
	//fmt.Printf("%#v\n\n", err)
	//return

	//we want to solve the following:
	//1. sorting   ->   100%; | orderBy:'Created':true in /go/src/MailHog/dist/assets/templates/index.html, we need to build it again in assets, else we need the file, but it's also hackish
	//2. load fast even with large attachments   ->   100%; in github.com/mailhog/data/message.go FromBytes use bytes.Buffer to concatenate strings
	//3. show/mark the email has attachment, either in listing or in preview, preferable both   ->   100%; added paperclip icon on listing and Attachment lines in preview; extract only filename with new methid in controllers.js parseAttachmentName
	//4. show all the recipients on To:, preferrable separated To. Cc. Bcc.   ->   100%; we add on To: all recipients; from them you can find in Cc: which were in fact Cc-ed; Bcc: could not be extracted separately, it's included in To:
	//5. also do I need to bring the entire message in frontend? in listing? with full body+attachments? maybe this helps the listing loading   ->   0%, not optimal but it works now with 2. solved
	//6. TODO there is too much processing in receiving a mail too, if an attachment of 6MB, it takes minutes to receive the email; it analyses every line in message and concatenating

	//build and tun with: go build -o build/MailHog main.go && ./build/MailHog

	if len(os.Args) > 1 && (os.Args[1] == "-version" || os.Args[1] == "--version") {
		fmt.Println("MailHog version: " + version)
		os.Exit(0)
	}

	if len(os.Args) > 1 && os.Args[1] == "sendmail" {
		args := os.Args
		os.Args = []string{args[0]}
		if len(args) > 2 {
			os.Args = append(os.Args, args[2:]...)
		}
		cmd.Go()
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "bcrypt" {
		var pw string
		if len(os.Args) > 2 {
			pw = os.Args[2]
		} else {
			// TODO: read from stdin
		}
		b, err := bcrypt.GenerateFromPassword([]byte(pw), 4)
		if err != nil {
			log.Fatalf("error bcrypting password: %s", err)
			os.Exit(1)
		}
		fmt.Println(string(b))
		os.Exit(0)
	}

	configure()

	if comconf.AuthFile != "" {
		http.AuthFile(comconf.AuthFile)
	}

	exitCh = make(chan int)
	if uiconf.UIBindAddr == apiconf.APIBindAddr {
		cb := func(r gohttp.Handler) {
			web.CreateWeb(uiconf, r.(*pat.Router), assets.Asset)
			api.CreateAPI(apiconf, r.(*pat.Router))
		}
		go http.Listen(uiconf.UIBindAddr, assets.Asset, exitCh, cb)
	} else {
		cb1 := func(r gohttp.Handler) {
			api.CreateAPI(apiconf, r.(*pat.Router))
		}
		cb2 := func(r gohttp.Handler) {
			web.CreateWeb(uiconf, r.(*pat.Router), assets.Asset)
		}
		go http.Listen(apiconf.APIBindAddr, assets.Asset, exitCh, cb1)
		go http.Listen(uiconf.UIBindAddr, assets.Asset, exitCh, cb2)
	}
	go smtp.Listen(apiconf, exitCh)

	for {
		select {
		case <-exitCh:
			log.Printf("Received exit signal")
			os.Exit(0)
		}
	}
}

/*

Add some random content to the end of this file, hopefully tricking GitHub
into recognising this as a Go repo instead of Makefile.

A gopher, ASCII art style - borrowed from
https://gist.github.com/belbomemo/b5e7dad10fa567a5fe8a

          ,_---~~~~~----._
   _,,_,*^____      _____``*g*\"*,
  / __/ /'     ^.  /      \ ^@q   f
 [  @f | @))    |  | @))   l  0 _/
  \`/   \~____ / __ \_____/    \
   |           _l__l_           I
   }          [______]           I
   ]            | | |            |
   ]             ~ ~             |
   |                            |
    |                           |

*/

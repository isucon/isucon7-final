package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/braintree/manners"
	"github.com/golang/gddo/httputil"
	"github.com/lestrrat/go-server-starter/listener"

	"portal"
)

var (
	addr = flag.String("listen", "localhost:3333", "`address` to listen to")
)

const (
	pathPrefixInternal = "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX/"
)

type httpError interface {
	HttpStatus() int
	error
}

type handler func(http.ResponseWriter, *http.Request) error

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	w.ResponseWriter.WriteHeader(status)
	w.status = status
}

func remoteAddr(req *http.Request) string {
	if addr := req.Header.Get("X-Forwarded-For"); len(addr) != 0 {
		return addr
	}
	return req.RemoteAddr
}

func (fn handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var rb httputil.ResponseBuffer
	rw := responseWriter{&rb, http.StatusOK}

	defer func() {
		if rv := recover(); rv != nil {
			var buf [4096]byte
			n := runtime.Stack(buf[:], true)
			log.Printf("panic: [%s %s] %+v", req.Method, req.URL.Path, rv)
			log.Printf("%s", string(buf[:n]))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}

		log.Printf("method:%s\tpath:%s\tstatus:%d\tremote:%s", req.Method, req.URL.RequestURI(), rw.status, remoteAddr(req))
	}()

	if portal.GetContestStatus() == portal.ContestStatusNotStarted &&
		!strings.HasPrefix(req.URL.Path, "/"+pathPrefixInternal) &&
		!strings.HasPrefix(req.URL.Path, "/login") &&
		!strings.HasPrefix(req.URL.Path, "/static/") {
		http.Error(w, "Qualifier has not started yet", http.StatusForbidden)
		return
	}

	err := fn(&rw, req)
	if err == nil {
		rb.Header().Set("X-Isu7QPortal-Version", portal.AppVersion)
		rb.WriteTo(w)
	} else {
		if he, ok := err.(httpError); ok {
			rw.status = he.HttpStatus()
			http.Error(w, he.Error(), he.HttpStatus())
			return
		}

		log.Printf("error: [%s %s] %s", req.Method, req.URL.Path, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func buildMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/", handler(portal.ServeIndex))
	mux.Handle("/favicon.ico", http.NotFoundHandler())
	mux.Handle("/login", handler(portal.ServeLogin))
	mux.Handle("/static/", handler(portal.ServeStatic))
	mux.Handle("/queue", handler(portal.ServeQueueJob))
	mux.Handle("/team", handler(portal.ServeUpdateTeam))

	mux.Handle("/"+pathPrefixInternal+"job/new", handler(portal.ServeNewJob))
	mux.Handle("/"+pathPrefixInternal+"job/result", handler(portal.ServePostResult))
	mux.Handle("/"+pathPrefixInternal+"debug/vars", handler(portal.ServeDebugExpvar))
	mux.Handle("/"+pathPrefixInternal+"debug/queue", handler(portal.ServeDebugQueue))
	mux.Handle("/"+pathPrefixInternal+"debug/result", handler(portal.ServeDebugResult))
	mux.Handle("/"+pathPrefixInternal+"debug/log", handler(portal.ServeDebugLog))
	mux.Handle("/"+pathPrefixInternal+"debug/queuejob", handler(portal.ServeDebugQueueJob))
	mux.Handle("/"+pathPrefixInternal+"debug/queueallteam", handler(portal.ServeDebugQueueAllTeam))
	mux.Handle("/"+pathPrefixInternal+"debug/leaderboard", handler(portal.ServeDebugLeaderboard))
	mux.Handle("/"+pathPrefixInternal+"admin", handler(portal.ServeAdminPage))
	mux.Handle("/"+pathPrefixInternal+"admin/server", handler(portal.ServeAdminServer))

	return mux
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	log.SetPrefix("[isucon7f-portal] ")

	flag.Parse()
	if *addr == "" {
		flag.Usage()
		log.Fatal("-listen required")
	}

	sigc := make(chan os.Signal)
	signal.Notify(sigc, syscall.SIGTERM)
	go func() {
		for {
			s := <-sigc
			if s == syscall.SIGTERM {
				log.Println("got SIGTERM; shutting down...")
				manners.Close()
			}
		}
	}()

	log.Print("initializing...")

	err := portal.InitState()
	if err != nil {
		log.Fatal(err)
	}

	err = portal.InitWeb()
	if err != nil {
		log.Fatal(err)
	}

	mux := buildMux()

	var l net.Listener
	ll, err := listener.ListenAll()
	if err != nil {
		log.Printf("go-server-starter: %s", err)
		log.Printf("fallback to standalone; server starting at %s", *addr)

		l, err = net.Listen("tcp", *addr)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Printf("running under server-starter; server starting at %s", ll[0].Addr())
		l = ll[0]
	}

	go func() {
		for {
			aborted, err := portal.CheckTimeoutJob()
			if err != nil {
				log.Println(err)
			}
			if 0 < aborted {
				log.Printf("%d jobs were aborted because of timeout", aborted)
			}
			time.Sleep(1 * time.Second)
		}
	}()

	err = manners.Serve(l, mux)
	if err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"net/http"

	"charm.land/log/v2"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.HandleFunc("/", defaultRoute)

	log.Info("Starting server", "port", 8000)
	if err := http.ListenAndServe(":8000", r); err != nil {
		log.Fatal("failed to start the server!")
	}
}

func defaultRoute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<body style="background:#1e1e1e;color:#d4d4d4"><pre>
		99 104 105 109 101 114 97

		██████╗██╗  ██╗██╗███╗   ███╗███████╗██████╗  █████╗
		██╔════╝██║  ██║██║████╗ ████║██╔════╝██╔══██╗██╔══██╗
		██║     ███████║██║██╔████╔██║█████╗  ██████╔╝███████║
		██║     ██╔══██║██║██║╚██╔╝██║██╔══╝  ██╔══██╗██╔══██║
		╚██████╗██║  ██║██║██║ ╚═╝ ██║███████╗██║  ██║██║  ██║
		╚═════╝╚═╝  ╚═╝╚═╝╚═╝     ╚═╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═╝
		
		Welcome to chimera API.
		do visit 
		<a href="https://github.com/imrishabk/chimera">github</a>
		for more info.
		</pre></body>
		`))
}

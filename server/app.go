package server

// This is a first-cut at making the core application in to an extensible
// package - it is likely that it will change (20201207/thisisaaronland)

import (
	"context"
	"flag"
	"fmt"
	"github.com/NYTimes/gziphandler"
	"github.com/aaronland/go-http-bootstrap"
	"github.com/aaronland/go-http-ping"
	"github.com/aaronland/go-http-server"
	"github.com/aaronland/go-http-tangramjs"
	"github.com/rs/cors"
	"github.com/sfomuseum/go-flags/flagset"
	"github.com/sfomuseum/go-flags/lookup"
	http_flags "github.com/whosonfirst/go-whosonfirst-spatial-http/flags"
	"github.com/whosonfirst/go-whosonfirst-spatial-http/http"
	"github.com/whosonfirst/go-whosonfirst-spatial-http/templates/html"
	"github.com/whosonfirst/go-whosonfirst-spatial-pip/api"
	"github.com/whosonfirst/go-whosonfirst-spatial/app"
	"github.com/whosonfirst/go-whosonfirst-spatial/flags"
	"html/template"
	"log"
	gohttp "net/http"
)

type HTTPServerApplication struct {
}

func NewHTTPServerApplication(ctx context.Context) (*HTTPServerApplication, error) {

	server_app := &HTTPServerApplication{}
	return server_app, nil
}

func (server_app *HTTPServerApplication) Run(ctx context.Context) error {

	fs, err := flags.CommonFlags()

	if err != nil {
		return fmt.Errorf("Failed to instantiate common flags, %v", err)
	}

	err = flags.AppendIndexingFlags(fs)

	if err != nil {
		return fmt.Errorf("Failed to append indexings flags, %v", err)
	}

	err = http_flags.AppendWWWFlags(fs)

	if err != nil {
		return fmt.Errorf("Failed to append www flags, %v", err)
	}

	flagset.Parse(fs)

	return server_app.RunWithFlagSet(ctx, fs)
}

func (server_app *HTTPServerApplication) RunWithFlagSet(ctx context.Context, fs *flag.FlagSet) error {

	err := flags.ValidateCommonFlags(fs)

	if err != nil {
		return fmt.Errorf("Failed to validate common flags, %v", err)
	}

	err = flags.ValidateIndexingFlags(fs)

	if err != nil {
		return fmt.Errorf("Failed to validate indexing flags, %v", err)
	}

	err = http_flags.ValidateWWWFlags(fs)

	if err != nil {
		return fmt.Errorf("Failed to validate www flags, %v", err)
	}

	enable_www, _ := lookup.BoolVar(fs, "enable-www")
	// enable_candidates, _ := lookup.BoolVar(fs, "enable-candidates")

	enable_geojson, _ := lookup.BoolVar(fs, "enable-geojson")

	nextzen_apikey, _ := lookup.StringVar(fs, "nextzen-apikey")
	nextzen_style_url, _ := lookup.StringVar(fs, "nextzen-style-url")
	nextzen_tile_url, _ := lookup.StringVar(fs, "nextzen-tile-url")

	initial_lat, _ := lookup.Float64Var(fs, "initial-latitude")
	initial_lon, _ := lookup.Float64Var(fs, "initial-longitude")
	initial_zoom, _ := lookup.IntVar(fs, "initial-zoom")

	server_uri, _ := lookup.StringVar(fs, "server-uri")

	data_endpoint, _ := lookup.StringVar(fs, "data-endpoint")

	spatial_app, err := app.NewSpatialApplicationWithFlagSet(ctx, fs)

	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to create new spatial application, because: %v", err))
	}

	logger := spatial_app.Logger

	paths := fs.Args()

	err = spatial_app.IndexPaths(ctx, paths...)

	if err != nil {
		return fmt.Errorf("Failed to index paths, because %s", err)
	}

	mux := gohttp.NewServeMux()

	ping_handler, err := ping.PingHandler()

	if err != nil {
		return fmt.Errorf("failed to create ping handler because %s", err)
	}

	mux.Handle("/health/ping", ping_handler)

	enable_cors := true
	enable_gzip := true

	cors_origins := []string{"*"}

	var cors_wrapper *cors.Cors

	if enable_cors {
		cors_wrapper = cors.New(cors.Options{
			AllowedOrigins: cors_origins,
		})
	}

	api_pip_opts := &api.PointInPolygonHandlerOptions{
		EnableGeoJSON: enable_geojson,
	}

	api_pip_handler, err := api.PointInPolygonHandler(spatial_app, api_pip_opts)

	if err != nil {
		return fmt.Errorf("failed to create point-in-polygon handler because %s", err)
	}

	if enable_cors {
		api_pip_handler = cors_wrapper.Handler(api_pip_handler)
	}

	if enable_gzip {
		api_pip_handler = gziphandler.GzipHandler(api_pip_handler)
	}

	log.Println("Register /api/point-in-polygon handler")
	mux.Handle("/api/point-in-polygon", api_pip_handler)

	/*
		if enable_candidates {

			logger.Debug("setting up candidates handler")

			candidates_handler, err := api.PointInPolygonCandidatesHandler(spatial_app)

			if err != nil {
				return fmt.Errorf("failed to create Spatial handler because %s", err)
			}

			if enable_cors {
				candidates_handler = cors_wrapper.Handler(candidates_handler)
			}

			mux.Handle("/api/point-in-polygon/candidates", candidates_handler)
		}
	*/

	if enable_www {

		t, err := template.ParseFS(html.FS, "*.html")

		if err != nil {
			return fmt.Errorf("Unable to parse templates, %v", err)
		}

		bootstrap_opts := bootstrap.DefaultBootstrapOptions()

		tangramjs_opts := tangramjs.DefaultTangramJSOptions()
		tangramjs_opts.Nextzen.APIKey = nextzen_apikey
		tangramjs_opts.Nextzen.StyleURL = nextzen_style_url
		tangramjs_opts.Nextzen.TileURL = nextzen_tile_url

		err = tangramjs.AppendAssetHandlers(mux)

		if err != nil {
			return fmt.Errorf("Failed to append tangram.js assets, %v", err)
		}

		err = bootstrap.AppendAssetHandlers(mux)

		if err != nil {
			return fmt.Errorf("Failed to append bootstrap assets, %v", err)
		}

		err = http.AppendStaticAssetHandlers(mux)

		if err != nil {
			return fmt.Errorf("Failed to append static assets, %v", err)
		}

		http_pip_opts := &http.PointInPolygonHandlerOptions{
			Templates:        t,
			InitialLatitude:  initial_lat,
			InitialLongitude: initial_lon,
			InitialZoom:      initial_zoom,
			DataEndpoint:     data_endpoint,
		}

		http_pip_handler, err := http.PointInPolygonHandler(spatial_app, http_pip_opts)

		if err != nil {
			return fmt.Errorf("failed to create (bundled) www handler because %s", err)
		}

		http_pip_handler = bootstrap.AppendResourcesHandler(http_pip_handler, bootstrap_opts)
		http_pip_handler = tangramjs.AppendResourcesHandler(http_pip_handler, tangramjs_opts)

		log.Println("Register /point-in-polygon handler")
		mux.Handle("/point-in-polygon", http_pip_handler)

		index_opts := &http.IndexHandlerOptions{
			Templates: t,
		}
		
		index_handler, err := http.IndexHandler(index_opts)

		if err != nil {
			return fmt.Errorf("Failed to create index handler, %v", err)
		}

		index_handler = bootstrap.AppendResourcesHandler(index_handler, bootstrap_opts)

		log.Println("Register / handler")
		mux.Handle("/", index_handler)
	}

	s, err := server.NewServer(ctx, server_uri)

	if err != nil {
		return fmt.Errorf("Failed to create new server for '%s', %v", server_uri, err)
	}

	log.Println(s.Address())
	logger.Info("Listening on %s", s.Address())

	err = s.ListenAndServe(ctx, mux)

	if err != nil {
		return fmt.Errorf("Failed to start server, %v", err)
	}

	return nil
}

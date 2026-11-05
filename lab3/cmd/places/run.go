package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"places/internal/locale"
	"places/internal/webui"
	"strings"
	"time"
)

const (
	defaultLanguage     = locale.Russian
	defaultRadiusMeters = 3000
	minRadiusMeters     = 10
	maxRadiusMeters     = 10000
	defaultPlacesLimit  = 5
	minPlacesLimit      = 1
	maxPlacesLimit      = 10
	defaultAddress      = "127.0.0.1:8080"
	headerTimeout       = 5 * time.Second
	idleTimeout         = 60 * time.Second
	shutdownTimeout     = 5 * time.Second
)

func run() (err error) {
	uiLanguage := argumentLanguage(os.Args[1:])
	tr := func(ru, en string) string { return locale.Text(uiLanguage, ru, en) }
	defer func() {
		if err != nil {
			err = fmt.Errorf("%s: %w", tr("Ошибка", "Error"), err)
		}
	}()
	flags := flag.NewFlagSet("places", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {
		fmt.Fprintln(os.Stdout, tr("Использование: places [параметры]", "Usage: places [options]"))
		flags.SetOutput(os.Stdout)
		flags.PrintDefaults()
		flags.SetOutput(io.Discard)
	}

	language := flags.String("lang", defaultLanguage, tr("язык интерфейса и API: ru или en", "interface and API language: ru or en"))
	radius := flags.Int("radius", defaultRadiusMeters, fmt.Sprintf(tr("радиус поиска, %d–%d метров", "search radius, %d-%d meters"), minRadiusMeters, maxRadiusMeters))
	limit := flags.Int("limit", defaultPlacesLimit, fmt.Sprintf(tr("количество мест, %d–%d", "number of places, %d-%d"), minPlacesLimit, maxPlacesLimit))
	address := flags.String("addr", defaultAddress, tr("адрес веб-сервера", "web server address"))
	if parseErr := flags.Parse(os.Args[1:]); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	if (*language != locale.Russian && *language != locale.English) || *radius < minRadiusMeters || *radius > maxRadiusMeters || *limit < minPlacesLimit || *limit > maxPlacesLimit {
		return errors.New(tr("некорректные параметры; см. -h", "invalid options; see -h"))
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	listener, listenErr := net.Listen("tcp", *address)
	if listenErr != nil {
		return listenErr
	}
	app := webui.New(ctx, webui.Options{Language: *language, Radius: *radius, Limit: *limit}, nil)
	server := &http.Server{Handler: app.Handler(), ReadHeaderTimeout: headerTimeout, IdleTimeout: idleTimeout}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	fmt.Printf(tr("Откройте в браузере: http://%s\n", "Open in your browser: http://%s\n"), listener.Addr())
	serveErr := server.Serve(listener)
	if errors.Is(serveErr, http.ErrServerClosed) {
		return nil
	}
	return serveErr
}

func argumentLanguage(args []string) string {
	language := defaultLanguage
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" || !strings.HasPrefix(arg, "-") {
			break
		}
		name := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")
		key, value, hasValue := strings.Cut(name, "=")
		switch key {
		case "lang", "radius", "limit", "addr":
			if !hasValue && i+1 < len(args) {
				i++
				value = args[i]
			}
			if key == "lang" {
				language = value
			}
		}
	}
	return language
}

package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"places/internal/locale"
	"places/internal/places"
	"strconv"
	"strings"
	"time"
)

const (
	defaultLanguage            = locale.Russian
	defaultRadiusMeters        = 3000
	minRadiusMeters            = 10
	maxRadiusMeters            = 10000
	defaultPlacesLimit         = 5
	minPlacesLimit             = 1
	maxPlacesLimit             = 10
	searchTimeout              = 20 * time.Second
	displayCoordinatePrecision = 5
	exploreTimeout             = 35 * time.Second
	freezingTemperatureCelsius = 0
)

func run() (err error) {
	// Choose help/error language before parsing, including `-lang en -h`.
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
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print(tr("Куда отправимся? Название города, улицы или места: ", "Where shall we go? Enter a city, street or place: "))
	if !scanner.Scan() {
		return scanner.Err()
	}
	service := places.New(*language, *radius, *limit)
	searchCtx, cancelSearch := context.WithTimeout(ctx, searchTimeout)
	locations, err := service.SearchLocations(searchCtx, scanner.Text()).Await(searchCtx)
	cancelSearch()
	if err != nil {
		return err
	}
	if len(locations) == 0 {
		fmt.Println(tr("Ничего не найдено. Попробуйте уточнить название и город.", "Nothing found. Try a more specific name and city."))
		return nil
	}
	for i, location := range locations {
		fmt.Printf("%d. %s (%.*f, %.*f)\n", i+1, location.Name, displayCoordinatePrecision, location.Latitude, displayCoordinatePrecision, location.Longitude)
	}
	var choice int
	for {
		fmt.Print(tr("Номер локации: ", "Location number: "))
		if !scanner.Scan() {
			return scanner.Err()
		}
		choice, err = strconv.Atoi(strings.TrimSpace(scanner.Text()))
		if err == nil && choice >= 1 && choice <= len(locations) {
			break
		}
		fmt.Printf(tr("Введите целое число от 1 до %d.\n", "Enter an integer from 1 to %d.\n"), len(locations))
	}
	exploreCtx, cancelExplore := context.WithTimeout(ctx, exploreTimeout)
	defer cancelExplore()
	result, err := service.Explore(exploreCtx, locations[choice-1]).Await(exploreCtx)
	if err != nil {
		return err
	}
	// Render only after the complete aggregate is available.
	var out strings.Builder
	fmt.Fprintf(&out, tr("\n%s\nПогода на %s (местное время): %.1f °C, ощущается как %.1f °C, ветер %.1f км/ч.\n", "\n%s\nWeather at %s (local time): %.1f °C, feels like %.1f °C, wind %.1f km/h.\n"), result.Location.Name, result.Weather.Time, result.Weather.Temperature, result.Weather.FeelsLike, result.Weather.Wind)
	if result.Weather.Temperature < freezingTemperatureCelsius {
		fmt.Fprintln(&out, tr("Режим пельменя в морозилке: одевайтесь теплее.", "Frozen dumpling mode: dress warmly."))
	}
	if len(result.Places) == 0 {
		fmt.Fprintln(&out, tr("Рядом нет статей с координатами. Попробуйте увеличить -radius или сменить -lang.", "No geotagged articles nearby. Try increasing -radius or changing -lang."))
	}
	for i, place := range result.Places {
		fmt.Fprintf(&out, tr("\n%d. %s — %.0f м\n%s\n%s\n", "\n%d. %s — %.0f m\n%s\n%s\n"), i+1, place.Title, place.Distance, place.Description, place.URL)
	}
	fmt.Fprintln(&out, tr("\nИсточники: Photon / © OpenStreetMap contributors; Open-Meteo (CC BY 4.0); Wikipedia (лицензия указана в статьях).", "\nSources: Photon / © OpenStreetMap contributors; Open-Meteo (CC BY 4.0); Wikipedia (see articles for licensing)."))
	fmt.Print(out.String())
	return nil
}

// argumentLanguage also supports Go flag syntax --lang=en and -lang=en.
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
		case "lang", "radius", "limit":
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

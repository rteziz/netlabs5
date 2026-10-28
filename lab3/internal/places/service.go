package places

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"places/internal/asynchttp"
	"places/internal/future"
	"places/internal/locale"
	"strconv"
	"strings"
	"time"
)

const (
	httpTimeout          = 15 * time.Second
	geocodeURL           = "https://photon.komoot.io/api/"
	weatherURL           = "https://api.open-meteo.com/v1/forecast"
	wikiURLFormat        = "https://%s.wikipedia.org/w/api.php"
	coordinatePrecision  = 6
	coordinateFloatBits  = 64
	locationLimit        = 8
	wikiFormatVersion    = "2"
	currentWeatherFields = "temperature_2m,apparent_temperature,wind_speed_10m"
	automaticTimezone    = "auto"
	wikiAPIPath          = "/w/api.php"
	apiEnabled           = "1"
	descriptionMaxChars  = 1200
)

// GeoJSON uses longitude before latitude.
const (
	longitudeIndex = iota
	latitudeIndex
	geoCoordinateCount
)

// Positions in All(weather, places); unrelated to completion order.
const (
	weatherResultIndex = iota
	placesResultIndex
)

type Location struct {
	Name                string
	Latitude, Longitude float64
}
type Weather struct {
	Time        string  `json:"time"`
	Temperature float64 `json:"temperature_2m"`
	FeelsLike   float64 `json:"apparent_temperature"`
	Wind        float64 `json:"wind_speed_10m"`
}
type Place struct {
	Title, Description, URL string
	Distance                float64
}
type Result struct {
	Location Location
	Weather  Weather
	Places   []Place
}
type Service struct {
	Language                        string
	Client                          *asynchttp.Client
	GeocodeURL, WeatherURL, WikiURL string
	Radius, Limit                   int
}

func New(language string, radius, limit int) *Service {
	return &Service{
		Language:   language,
		Client:     &asynchttp.Client{HTTP: &http.Client{Timeout: httpTimeout}},
		GeocodeURL: geocodeURL, WeatherURL: weatherURL,
		WikiURL: fmt.Sprintf(wikiURLFormat, language), Radius: radius, Limit: limit,
	}
}
func endpoint(base string, params url.Values) string { return base + "?" + params.Encode() }
func number(value float64) string {
	return strconv.FormatFloat(value, 'f', coordinatePrecision, coordinateFloatBits)
}

// SearchLocations implements step 2; it never awaits an intermediate result.
func (s *Service) SearchLocations(ctx context.Context, query string) *future.Future[[]Location] {
	type response struct {
		Features []struct {
			Geometry struct {
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
			Properties struct{ Name, Street, City, State, Country string } `json:"properties"`
		} `json:"features"`
	}
	if strings.TrimSpace(query) == "" {
		return future.Ready[[]Location](nil, errors.New(locale.Text(s.Language, "пустой поисковый запрос", "empty search query")))
	}
	params := url.Values{"q": {query}, "limit": {strconv.Itoa(locationLimit)}}
	// The public Photon server supports English; keep its default for Russian.
	if s.Language == locale.English {
		params.Set("lang", locale.English)
	}
	return future.Map(asynchttp.GetJSON[response](ctx, s.Client, endpoint(s.GeocodeURL, params)), func(data response) ([]Location, error) {
		locations := make([]Location, 0, len(data.Features))
		for _, feature := range data.Features {
			coords := feature.Geometry.Coordinates
			if len(coords) < geoCoordinateCount {
				continue
			}
			p := feature.Properties
			parts := []string{}
			seen := map[string]bool{}
			for _, part := range []string{p.Name, p.Street, p.City, p.State, p.Country} {
				if part != "" && !seen[part] {
					parts = append(parts, part)
					seen[part] = true
				}
			}
			locations = append(locations, Location{strings.Join(parts, ", "), coords[latitudeIndex], coords[longitudeIndex]})
		}
		return locations, nil
	})
}

type wikiResponse struct {
	Error *struct{ Code, Info string } `json:"error"`
	Query struct {
		Geosearch []struct {
			PageID   int     `json:"pageid"`
			Title    string  `json:"title"`
			Distance float64 `json:"dist"`
		} `json:"geosearch"`
		Pages []struct {
			Extract string `json:"extract"`
			Missing bool   `json:"missing"`
		} `json:"pages"`
	} `json:"query"`
}

func (s *Service) wiki(ctx context.Context, params url.Values) *future.Future[wikiResponse] {
	params.Set("action", "query")
	params.Set("format", "json")
	params.Set("formatversion", wikiFormatVersion)
	return future.Map(asynchttp.GetJSON[wikiResponse](ctx, s.Client, endpoint(s.WikiURL, params)), func(data wikiResponse) (wikiResponse, error) {
		if data.Error != nil {
			return data, fmt.Errorf("Wikipedia: %s: %s", data.Error.Code, data.Error.Info)
		}
		return data, nil
	})
}

// Explore implements steps 4–5: weather || (nearby -> All(descriptions)).
// Every intermediate value travels through callbacks, never through service state.
func (s *Service) Explore(ctx context.Context, location Location) *future.Future[Result] {
	type weatherResponse struct {
		Current *Weather `json:"current"`
	}
	weather := future.Map(asynchttp.GetJSON[weatherResponse](ctx, s.Client, endpoint(s.WeatherURL, url.Values{
		"latitude": {number(location.Latitude)}, "longitude": {number(location.Longitude)},
		"current": {currentWeatherFields}, "timezone": {automaticTimezone},
	})), func(data weatherResponse) (Result, error) {
		if data.Current == nil {
			return Result{}, errors.New(locale.Text(s.Language, "API погоды не вернул current", "weather API did not return current conditions"))
		}
		return Result{Weather: *data.Current}, nil
	})
	nearby := s.wiki(ctx, url.Values{"list": {"geosearch"}, "gscoord": {number(location.Latitude) + "|" + number(location.Longitude)}, "gsradius": {strconv.Itoa(s.Radius)}, "gslimit": {strconv.Itoa(s.Limit)}})
	descriptions := future.Then(nearby, func(data wikiResponse) *future.Future[[]Place] {
		requests := make([]*future.Future[Place], 0, len(data.Query.Geosearch))
		for _, item := range data.Query.Geosearch {
			place := Place{Title: item.Title, Distance: item.Distance, URL: strings.TrimSuffix(s.WikiURL, wikiAPIPath) + "/?curid=" + strconv.Itoa(item.PageID)}
			request := s.wiki(ctx, url.Values{"prop": {"extracts"}, "pageids": {strconv.Itoa(item.PageID)}, "exintro": {apiEnabled}, "explaintext": {apiEnabled}, "exchars": {strconv.Itoa(descriptionMaxChars)}})
			requests = append(requests, future.Map(request, func(description wikiResponse) (Place, error) {
				place.Description = locale.Text(s.Language, "Описание отсутствует. Даже Википедия иногда молчит.", "No description available. Even Wikipedia goes quiet sometimes.")
				if len(description.Query.Pages) > 0 && description.Query.Pages[0].Extract != "" {
					place.Description = description.Query.Pages[0].Extract
				}
				return place, nil
			}))
		}
		return future.All(requests...)
	})
	places := future.Map(descriptions, func(items []Place) (Result, error) { return Result{Places: items}, nil })
	return future.Map(future.All(weather, places), func(parts []Result) (Result, error) {
		return Result{Location: location, Weather: parts[weatherResultIndex].Weather, Places: parts[placesResultIndex].Places}, nil
	})
}

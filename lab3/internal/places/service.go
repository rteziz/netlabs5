package places

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"places/internal/asynchttp"
	"places/internal/future"
	"places/internal/locale"
	"strconv"
	"strings"
	"time"
)

const (
	httpTimeout           = 15 * time.Second
	geocodeURL            = "https://graphhopper.com/api/1/geocode"
	weatherURL            = "https://api.open-meteo.com/v1/forecast"
	tripURLFormat         = "https://api.opentripmap.com/0.1/%s/places"
	wikiURLFormat         = "https://%s.wikipedia.org/w/api.php"
	coordinatePrecision   = 6
	coordinateFloatBits   = 64
	locationLimit         = 8
	filteredLocationLimit = 10
	tripTimeout           = 12 * time.Second
	wikiFormatVersion     = "2"
	thumbnailSize         = 600
	currentWeatherFields  = "temperature_2m,apparent_temperature,wind_speed_10m,pressure_msl,relative_humidity_2m"
	wikiAPIPath           = "/w/api.php"
	apiEnabled            = "1"
	descriptionMaxChars   = 1200
)

const (
	weatherResultIndex = iota
	placesResultIndex
)

var ErrMissingAPIKey = errors.New("missing API key: configure GRAPHHOPPER_API_KEY")

type Location struct {
	Name                string
	OSMKey              string `json:"OSMKey"`
	OSMValue            string `json:"OSMValue"`
	Latitude, Longitude float64
}
type Weather struct {
	Time        string  `json:"time"`
	Temperature float64 `json:"temperature"`
	FeelsLike   float64 `json:"feels_like"`
	Pressure    float64 `json:"pressure"`
	Humidity    float64 `json:"humidity"`
	Description string  `json:"description"`
	Wind        float64 `json:"wind_speed"`
}
type Place struct {
	ImageURL, ImagePageURL  string
	Title, Description, URL string
	Distance                float64
}
type Result struct {
	Location     Location
	Weather      Weather
	Places       []Place
	PlacesSource string
}
type Service struct {
	Language                        string
	PlaceType                       string
	GraphHopperKey                  string
	OpenTripMapKey, TripURL         string
	Client                          *asynchttp.Client
	GeocodeURL, WeatherURL, WikiURL string
	Radius, Limit                   int
}

func New(language string, radius, limit int) *Service {
	return &Service{
		Language:       language,
		OpenTripMapKey: strings.TrimSpace(os.Getenv("OPENTRIPMAP_API_KEY")),
		TripURL:        fmt.Sprintf(tripURLFormat, language),
		GraphHopperKey: strings.TrimSpace(os.Getenv("GRAPHHOPPER_API_KEY")),
		Client:         asynchttp.New(httpTimeout),
		GeocodeURL:     geocodeURL, WeatherURL: weatherURL,
		WikiURL: fmt.Sprintf(wikiURLFormat, language), Radius: radius, Limit: limit,
	}
}
func endpoint(base string, params url.Values) string { return base + "?" + params.Encode() }
func number(value float64) string {
	return strconv.FormatFloat(value, 'f', coordinatePrecision, coordinateFloatBits)
}

func (s *Service) SearchLocations(ctx context.Context, query string) *future.Future[[]Location] {
	type response struct {
		Hits []struct {
			Point                                           *struct{ Lat, Lng float64 }
			Name, Street, Housenumber, City, State, Country string
			OSMKey                                          string `json:"osm_key"`
			OSMValue                                        string `json:"osm_value"`
		}
	}
	if s.GraphHopperKey == "" {
		return future.Ready[[]Location](nil, ErrMissingAPIKey)
	}
	if strings.TrimSpace(query) == "" {
		return future.Ready[[]Location](nil, errors.New(locale.Text(s.Language, "пустой поисковый запрос", "empty search query")))
	}
	// При фильтрации просим больше кандидатов: часть ответов будет отброшена по OSMValue.
	limit := locationLimit
	if s.PlaceType != "" {
		limit = filteredLocationLimit
	}
	params := url.Values{"q": {query}, "limit": {strconv.Itoa(limit)}, "locale": {s.Language}, "key": {s.GraphHopperKey}}
	request := asynchttp.GetJSON[response](ctx, s.Client, endpoint(s.GeocodeURL, params))
	return future.Map(request, func(data response) ([]Location, error) {
		locations := make([]Location, 0, len(data.Hits))
		for _, feature := range data.Hits {
			if s.PlaceType != "" && feature.OSMValue != s.PlaceType {
				continue
			}
			coords := feature.Point
			if coords == nil {
				continue
			}
			p := feature
			parts := []string{}
			seen := map[string]bool{}
			for _, part := range []string{p.Name, p.Street, p.Housenumber, p.City, p.State, p.Country} {
				if part != "" && !seen[part] {
					parts = append(parts, part)
					seen[part] = true
				}
			}
			locations = append(locations, Location{Name: strings.Join(parts, ", "), Latitude: coords.Lat, Longitude: coords.Lng, OSMKey: feature.OSMKey, OSMValue: feature.OSMValue})
		}
		return locations, nil
	})
}

type wikiResponse struct {
	Error *struct{ Code, Info string } `json:"error"`
	Query struct {
		Geosearch []struct {
			PageID   int `json:"pageid"`
			Title    string
			Distance float64 `json:"dist"`
		} `json:"geosearch"`
		Pages []struct {
			Extract   string                  `json:"extract"`
			PageImage string                  `json:"pageimage"`
			Thumbnail struct{ Source string } `json:"thumbnail"`
			Missing   bool                    `json:"missing"`
		} `json:"pages"`
	} `json:"query"`
}

func (s *Service) wiki(ctx context.Context, base string, params url.Values) *future.Future[wikiResponse] {
	params.Set("action", "query")
	params.Set("format", "json")
	params.Set("formatversion", wikiFormatVersion)
	return future.Map(asynchttp.GetJSON[wikiResponse](ctx, s.Client, endpoint(base, params)), func(data wikiResponse) (wikiResponse, error) {
		if data.Error != nil {
			return data, fmt.Errorf("Wikipedia: %s: %s", data.Error.Code, data.Error.Info)
		}
		return data, nil
	})
}

func (s *Service) Explore(ctx context.Context, location Location) *future.Future[Result] {
	type weatherResponse struct {
		Current *struct {
			Time        string
			Temperature float64 `json:"temperature_2m"`
			FeelsLike   float64 `json:"apparent_temperature"`
			Wind        float64 `json:"wind_speed_10m"`
			Pressure    float64 `json:"pressure_msl"`
			Humidity    float64 `json:"relative_humidity_2m"`
		} `json:"current"`
	}
	// Погода и поиск мест зависят только от координат и запускаются независимо.
	weatherParams := url.Values{
		"latitude": {number(location.Latitude)}, "longitude": {number(location.Longitude)},
		"current": {currentWeatherFields}, "timezone": {"auto"}, "wind_speed_unit": {"ms"},
	}
	weatherRequest := asynchttp.GetJSON[weatherResponse](ctx, s.Client, endpoint(s.WeatherURL, weatherParams))
	weather := future.Map(weatherRequest, func(data weatherResponse) (Result, error) {
		if data.Current == nil {
			return Result{}, errors.New(locale.Text(s.Language, "API погоды не вернул текущие условия", "weather API did not return current conditions"))
		}
		current := data.Current
		return Result{Weather: Weather{Time: current.Time, Temperature: current.Temperature, FeelsLike: current.FeelsLike, Wind: current.Wind, Pressure: current.Pressure, Humidity: current.Humidity}}, nil
	})
	nearby := s.nearby(ctx, location)
	return future.Map(future.All(weather, nearby), func(parts []Result) (Result, error) {
		places := parts[placesResultIndex]
		return Result{
			Location:     location,
			Weather:      parts[weatherResultIndex].Weather,
			Places:       namedPlaces(places.Places),
			PlacesSource: places.PlacesSource,
		}, nil
	})
}

func (s *Service) tripPlaces(ctx context.Context, location Location) *future.Future[[]Place] {
	type nearbyPlace struct {
		XID, Name string
		Distance  float64 `json:"dist"`
	}
	nearby := asynchttp.GetJSON[[]nearbyPlace](ctx, s.Client, endpoint(s.TripURL+"/radius", url.Values{
		"lat": {number(location.Latitude)}, "lon": {number(location.Longitude)},
		"radius": {strconv.Itoa(s.Radius)}, "limit": {strconv.Itoa(s.Limit)},
		"format": {"json"}, "apikey": {s.OpenTripMapKey},
	}))
	// Список xid нужен до запуска параллельных запросов карточек.
	return future.Then(nearby, func(items []nearbyPlace) *future.Future[[]Place] {
		requests := make([]*future.Future[Place], 0, len(items))
		for _, item := range items {
			place := Place{Title: item.Name, Distance: item.Distance, Description: locale.Text(s.Language, "Статья в Wikipedia отсутствует.", "No Wikipedia article available.")}
			type detail struct {
				Name, Wikipedia string
				Info            struct{ Descr string } `json:"info"`
				Extracts        struct{ Text string }  `json:"wikipedia_extracts"`
			}
			details := asynchttp.GetJSON[detail](ctx, s.Client, endpoint(s.TripURL+"/xid/"+url.PathEscape(item.XID), url.Values{"apikey": {s.OpenTripMapKey}}))
			card := future.Then(details, func(data detail) *future.Future[Place] {
				if strings.TrimSpace(data.Name) != "" {
					place.Title = data.Name
				}
				if text := strings.TrimSpace(data.Info.Descr); text != "" {
					place.Description = text
				} else if text := strings.TrimSpace(data.Extracts.Text); text != "" {
					place.Description = text
				}
				article, err := url.Parse(data.Wikipedia)
				if err != nil || article.Scheme != "https" || !strings.HasSuffix(article.Hostname(), ".wikipedia.org") || article.Port() != "" || article.User != nil || !strings.HasPrefix(article.Path, "/wiki/") {
					return future.Ready(place, nil)
				}
				title := strings.TrimPrefix(article.Path, "/wiki/")
				if title == "" {
					return future.Ready(place, nil)
				}
				place.URL = article.String()
				base := "https://" + article.Hostname() + wikiAPIPath
				if s.WikiURL != fmt.Sprintf(wikiURLFormat, s.Language) {
					base = s.WikiURL
				}
				request := s.wiki(ctx, base, url.Values{"prop": {"extracts|pageimages"}, "titles": {title}, "redirects": {apiEnabled}, "piprop": {"thumbnail|name"}, "pithumbsize": {strconv.Itoa(thumbnailSize)}, "pilicense": {"free"}, "exintro": {apiEnabled}, "explaintext": {apiEnabled}, "exchars": {strconv.Itoa(descriptionMaxChars)}})
				return future.Map(request, func(description wikiResponse) (Place, error) {
					if len(description.Query.Pages) > 0 {
						page := description.Query.Pages[0]
						if !page.Missing {
							if strings.TrimSpace(page.Extract) != "" {
								place.Description = page.Extract
							}
							place.ImageURL = page.Thumbnail.Source
							if page.PageImage != "" {
								place.ImagePageURL = "https://" + article.Hostname() + "/wiki/File:" + url.PathEscape(page.PageImage)
							}
						} else {
							place.URL = ""
						}
					}
					return place, nil
				})
			})
			requests = append(requests, keepPartialPlace(card, &place))
		}
		return future.All(requests...)
	})
}

func (s *Service) nearby(ctx context.Context, location Location) *future.Future[Result] {
	wiki := func(source string) *future.Future[Result] {
		return future.Map(s.wikiPlaces(ctx, location), func(items []Place) (Result, error) { return Result{Places: items, PlacesSource: source}, nil })
	}
	if s.PlaceType == "" {
		return wiki("Wikipedia")
	}
	if s.OpenTripMapKey == "" {
		return wiki("Wikipedia (OpenTripMap fallback)")
	}
	result := future.New[Result]()
	// Запасной поиск в Википедии использует общий ctx после истечения срока OpenTripMap.
	tripCtx, cancel := context.WithTimeout(ctx, tripTimeout)
	s.tripPlaces(tripCtx, location).OnComplete(func(items []Place, err error) {
		cancel()
		if ctx.Err() != nil {
			result.Resolve(Result{}, ctx.Err())
			return
		}
		if err == nil && len(items) > 0 {
			result.Resolve(Result{Places: items, PlacesSource: "OpenTripMap"}, nil)
			return
		}
		log.Print("[API] OpenTripMap unavailable or empty; falling back to Wikipedia")
		wiki("Wikipedia (OpenTripMap fallback)").OnComplete(result.Resolve)
	})
	return result
}

func (s *Service) wikiPlaces(ctx context.Context, location Location) *future.Future[[]Place] {
	nearby := s.wiki(ctx, s.WikiURL, url.Values{"list": {"geosearch"}, "gscoord": {number(location.Latitude) + "|" + number(location.Longitude)}, "gsradius": {strconv.Itoa(s.Radius)}, "gslimit": {strconv.Itoa(s.Limit)}})
	return future.Then(nearby, func(data wikiResponse) *future.Future[[]Place] {
		requests := make([]*future.Future[Place], 0, len(data.Query.Geosearch))
		for _, item := range data.Query.Geosearch {
			place := Place{Title: item.Title, Distance: item.Distance, URL: strings.TrimSuffix(s.WikiURL, wikiAPIPath) + "/?curid=" + strconv.Itoa(item.PageID), Description: locale.Text(s.Language, "Описание отсутствует.", "No description available.")}
			request := s.wiki(ctx, s.WikiURL, url.Values{"prop": {"extracts|pageimages"}, "pageids": {strconv.Itoa(item.PageID)}, "piprop": {"thumbnail|name"}, "pithumbsize": {strconv.Itoa(thumbnailSize)}, "pilicense": {"free"}, "exintro": {apiEnabled}, "explaintext": {apiEnabled}, "exchars": {strconv.Itoa(descriptionMaxChars)}})
			card := future.Map(request, func(data wikiResponse) (Place, error) {
				if len(data.Query.Pages) > 0 {
					page := data.Query.Pages[0]
					if strings.TrimSpace(page.Extract) != "" {
						place.Description = page.Extract
					}
					place.ImageURL = page.Thumbnail.Source
					if page.PageImage != "" {
						place.ImagePageURL = strings.TrimSuffix(s.WikiURL, wikiAPIPath) + "/wiki/File:" + url.PathEscape(page.PageImage)
					}
				}
				return place, nil
			})
			requests = append(requests, keepPartialPlace(card, &place))
		}
		return future.All(requests...)
	})
}

func keepPartialPlace(card *future.Future[Place], partial *Place) *future.Future[Place] {
	result := future.New[Place]()
	card.OnComplete(func(value Place, err error) {
		if err != nil {
			log.Printf("[API] incomplete place %q: %v", partial.Title, err)
			value = *partial
		}
		result.Resolve(value, nil)
	})
	return result
}

func namedPlaces(items []Place) []Place {
	result := make([]Place, 0, len(items))
	for _, item := range items {
		item.Title = strings.TrimSpace(item.Title)
		if item.Title != "" {
			result = append(result, item)
		}
	}
	return result
}

package places

var PlaceTypes = []string{
	"bus_stop", "platform", "station", "tram_stop", "subway_entrance",
	"residential", "pedestrian", "footway", "primary", "secondary", "tertiary",
	"museum", "attraction", "viewpoint", "artwork", "hotel", "hostel",
	"park", "garden", "square", "playground", "stadium", "sports_centre",
	"cafe", "restaurant", "fast_food", "bar", "pub", "school", "university",
	"hospital", "pharmacy", "library", "cinema", "theatre", "parking",
	"supermarket", "mall", "monument", "memorial", "castle", "church",
}

const maxPlaceTypeLength = 64

// Список выше содержит подсказки. Вручную можно ввести и другой корректный OSM-код.
func ValidPlaceType(value string) bool {
	if len(value) > maxPlaceTypeLength {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

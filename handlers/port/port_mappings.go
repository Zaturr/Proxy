package port

import "strings"

// PortMapping representa un mapeo de path a puerto
type PortMapping struct {
	PathPattern string
	MatchType   string // "contains" o "prefix"
	Port        string
}

// GetPortMappings devuelve todos los mapeos de puertos configurados
func GetPortMappings() []PortMapping {
	return []PortMapping{
		{PathPattern: "/auth", MatchType: "contains", Port: "8086"},
		{PathPattern: "/hi", MatchType: "contains", Port: "8101"},
		{PathPattern: "/jsonplaceholder", MatchType: "prefix", Port: "8080"},
		{PathPattern: "/hello", MatchType: "contains", Port: "8080"},
		{PathPattern: "/echo", MatchType: "contains", Port: "8080"},
		{PathPattern: "/callback", MatchType: "contains", Port: "8080"},
		{PathPattern: "/api", MatchType: "prefix", Port: "8081"},

		{PathPattern: "/check", MatchType: "contains", Port: "3500"},
		{PathPattern: "/anothrapi", MatchType: "contains", Port: "8082"},
		{PathPattern: "/ath", MatchType: "contains", Port: "8085"},
		{PathPattern: "/sycom", MatchType: "contains", Port: "8102"},

		{PathPattern: "/visor/hello", MatchType: "contains", Port: "5090"},
		{PathPattern: "/visor", MatchType: "prefix", Port: "5091"},

		{PathPattern: "/grafana", MatchType: "contains", Port: "3005"},
		{PathPattern: "/sample", MatchType: "prefix", Port: "8101"},
	}
}

// GetTargetPortByPath mapea un path a un puerto específico
func GetTargetPortByPath(path string, defaultPort string) string {
	mappings := GetPortMappings()

	for _, mapping := range mappings {
		if matchesPath(path, mapping) {
			return mapping.Port
		}
	}

	return defaultPort
}

// matchesPath verifica si un path coincide con un mapeo
func matchesPath(path string, mapping PortMapping) bool {
	switch mapping.MatchType {
	case "contains":
		return strings.Contains(path, mapping.PathPattern)
	case "prefix":
		return strings.HasPrefix(path, mapping.PathPattern)
	default:
		return false
	}
}

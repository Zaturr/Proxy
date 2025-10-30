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
		// Puerto 1231 - BANCRECER
		{PathPattern: "/bancrecer/simf/1.0.0/enviar", MatchType: "contains", Port: "1231"},
		{PathPattern: "/ddinme/v2/send/otp", MatchType: "contains", Port: "1231"},
		{PathPattern: "/debit/v2/operation", MatchType: "contains", Port: "1231"},
		{PathPattern: "/debit/v2/operations/date", MatchType: "contains", Port: "1231"},
		{PathPattern: "/bancrecer/simf/1.0.0/operaciones", MatchType: "contains", Port: "1231"},
		{PathPattern: "/async", MatchType: "contains", Port: "1231"},

		// Puerto 2021 - BANCRECER
		{PathPattern: "/bancrecer/simf/1.0.0/op", MatchType: "contains", Port: "2021"},

		// Puerto 8101 - MAIN
		{PathPattern: "/", MatchType: "prefix", Port: "8101"},

		// Puerto 3005 - GRAFANA
		{PathPattern: "/grafana", MatchType: "contains", Port: "3005"},

		// Puerto 5090 - VISOR
		{PathPattern: "/", MatchType: "prefix", Port: "5090"},

		// Puerto 5091 - VISOR_API
		{PathPattern: "/", MatchType: "prefix", Port: "5091"},

		// Puerto 8102 - SYCOM
		{PathPattern: "/sycom", MatchType: "contains", Port: "8102"},

		// Puerto 8085 - AUTH
		{PathPattern: "/", MatchType: "prefix", Port: "8085"},

		// Puerto 8082 - ANOTHERAPI
		{PathPattern: "/", MatchType: "prefix", Port: "8082"},

		// Puerto 3500 - CHECKOUT
		{PathPattern: "/", MatchType: "prefix", Port: "3500"},

		// 	{PathPattern: "/auth", MatchType: "contains", Port: "8086"},
		// 	{PathPattern: "/hi", MatchType: "contains", Port: "8101"},
		// 	{PathPattern: "/jsonplaceholder", MatchType: "prefix", Port: "8080"},
		// 	{PathPattern: "/hello", MatchType: "contains", Port: "8080"},
		// 	{PathPattern: "/echo", MatchType: "contains", Port: "8080"},
		// 	{PathPattern: "/callback", MatchType: "contains", Port: "8080"},
		// 	{PathPattern: "/api", MatchType: "prefix", Port: "8081"},
		// 	{PathPattern: "/check", MatchType: "contains", Port: "3500"},
		// 	{PathPattern: "/anothrapi", MatchType: "contains", Port: "8082"},
		// 	{PathPattern: "/ath", MatchType: "contains", Port: "8085"},
		// 	{PathPattern: "/visor/hello", MatchType: "contains", Port: "5090"},
		// 	{PathPattern: "/visor", MatchType: "prefix", Port: "5091"},
		// 	{PathPattern: "/sample", MatchType: "prefix", Port: "8101"},
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

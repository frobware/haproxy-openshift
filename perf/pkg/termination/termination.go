package termination

type TrafficType string

const (
	Edge        TrafficType = "edge"
	HTTP        TrafficType = "http"
	Passthrough TrafficType = "passthrough"
	Reencrypt   TrafficType = "reencrypt"
)

var AllTerminationTypes = [...]TrafficType{
	Edge,
	HTTP,
	Passthrough,
	Reencrypt,
}

func (t TrafficType) Scheme() string {
	switch t {
	case HTTP:
		return "http"
	default:
		return "https"
	}
}

func (t TrafficType) Port() int64 {
	switch t {
	case HTTP:
		return 8080
	default:
		return 8443
	}
}

func ParseTrafficType(s string) TrafficType {
	switch s {
	case "http":
		return HTTP
	case "edge":
		return Edge
	case "reencrypt":
		return Reencrypt
	case "passthrough":
		return Passthrough
	}
	panic("unknown taffic type" + s)
}

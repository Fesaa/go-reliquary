package reliquary

const (
	PCAP_FILTER string = "udp portrange 23301-23302"

	HEADER_LEN = 12
	TAIL_LEN   = 4

	HEADER_OVERHEAD = HEADER_LEN + TAIL_LEN
)

var (
	PORTS = []string{"23301", "23302"}
)

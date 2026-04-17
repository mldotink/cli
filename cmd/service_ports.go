package cmd

import (
	"fmt"

	ink "github.com/mldotink/sdk-go"
)

type endpointPort struct {
	name             string
	port             string
	protocol         string
	visibility       string
	internalEndpoint string
	publicEndpoint   string
}

func singlePublicHTTPPort(port int) []ink.ServicePortInput {
	return []ink.ServicePortInput{{
		Name:       "http",
		Port:       port,
		Protocol:   "http",
		Visibility: "public",
	}}
}

func inkServicePorts(ports []ink.ServicePort) []endpointPort {
	result := make([]endpointPort, 0, len(ports))
	for _, port := range ports {
		result = append(result, endpointPort{
			name:             port.Name,
			port:             port.Port,
			protocol:         port.Protocol,
			visibility:       port.Visibility,
			internalEndpoint: port.InternalEndpoint,
			publicEndpoint:   port.PublicEndpoint,
		})
	}
	return result
}

func preferredServiceEndpoint(ports []endpointPort, customDomain string) string {
	if customDomain != "" {
		return "https://" + customDomain
	}
	if endpoint, ok := firstPublicEndpointByProtocol(ports, "http"); ok {
		return endpoint
	}
	if endpoint, ok := firstPublicEndpointByProtocol(ports, "tcp"); ok {
		return endpoint
	}
	for _, port := range ports {
		if port.publicEndpoint != "" {
			return port.publicEndpoint
		}
	}
	return ""
}

func firstPublicEndpointByProtocol(ports []endpointPort, protocol string) (string, bool) {
	for _, port := range ports {
		if port.visibility != "public" || port.protocol != protocol {
			continue
		}
		if port.publicEndpoint != "" {
			return port.publicEndpoint, true
		}
	}
	return "", false
}

func renderPortSummary(port endpointPort) string {
	public := "—"
	if port.publicEndpoint != "" {
		public = accent.Render(port.publicEndpoint)
	}

	return fmt.Sprintf(
		"  %-10s %-12s public %s\n  %-10s %-12s internal %s",
		bold.Render(port.name),
		fmt.Sprintf("%s/%s:%s", port.visibility, port.protocol, port.port),
		public,
		"",
		"",
		dim.Render(port.internalEndpoint),
	)
}

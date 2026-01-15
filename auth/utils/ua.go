package utils

import (
	"strings"

	"github.com/mileusna/useragent"
)

func ParseDeviceInfo(ua string) string {
	client := useragent.Parse(ua)

	var parts []string

	// Устройство / платформа
	if client.Mobile {
		parts = append(parts, "Mobile")
		if client.Tablet {
			parts = append(parts, "Tablet")
		}
	} else if client.Desktop {
		parts = append(parts, "Desktop")
	}

	// ОС
	if client.OS != "" {
		osName := client.OS
		if client.OSVersion != "" {
			osName += " " + client.OSVersion
		}
		parts = append(parts, osName)
	}

	// Браузер
	if client.Name != "" {
		browser := client.Name
		if client.Version != "" {
			browser += " " + client.Version
		}
		parts = append(parts, browser)
	}

	if len(parts) == 0 {
		return "Unknown"
	}

	return strings.Join(parts, " / ")
}

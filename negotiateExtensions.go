package gows

import "strings"

func negotiateExtensions(offers []string, available []Extension) (accepted []Extension, response string) {

	var responses []string

	for _, offer := range offers {

		name, params := parseExtensionHeader(offer)

		for _, ext := range available {

			if name == ext.Name() {

				agreed, ok := ext.Negotiate(params)

				if ok == true {

					accepted = append(accepted, ext)
					responses = append(responses, formatExtensionHeader(name, agreed))

					// Each extension name appears at most once
					break

				}

			}

		}

	}

	if len(responses) > 0 {
		response = strings.Join(responses, ", ")
	}

	return accepted, response

}


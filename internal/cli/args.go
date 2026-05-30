package cli

import "strings"

type ParsedArgs struct {
	Options     map[string]string
	Positionals []string
	Help        bool
}

func ParseArgs(argv []string) ParsedArgs {
	parsed := ParsedArgs{
		Options: make(map[string]string),
	}

	for i := 0; i < len(argv); i++ {
		token := argv[i]

		if token == "--help" || token == "-h" {
			parsed.Help = true
			continue
		}

		if strings.HasPrefix(token, "--") {
			withoutPrefix := strings.TrimPrefix(token, "--")
			if withoutPrefix == "" {
				continue
			}

			if key, value, ok := strings.Cut(withoutPrefix, "="); ok {
				if key != "" {
					parsed.Options[key] = value
				}
				continue
			}

			if i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "--") {
				parsed.Options[withoutPrefix] = argv[i+1]
				i++
			}

			continue
		}

		parsed.Positionals = append(parsed.Positionals, token)
	}

	return parsed
}

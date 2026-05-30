package prdetail

import (
	"path/filepath"
	"regexp"
	"strings"
)

var likelyGeneratedPathPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(^|/)dist/`),
	regexp.MustCompile(`(^|/)build/`),
	regexp.MustCompile(`(^|/)coverage/`),
	regexp.MustCompile(`(^|/)vendor/`),
	regexp.MustCompile(`(^|/)generated/`),
	regexp.MustCompile(`(^|/)__snapshots__/`),
	regexp.MustCompile(`(?i)\.min\.[a-z0-9]+$`),
	regexp.MustCompile(`(?i)\.lock$`),
	regexp.MustCompile(`(?i)^pnpm-lock\.yaml$`),
	regexp.MustCompile(`(?i)^bun\.lockb$`),
	regexp.MustCompile(`(?i)^yarn\.lock$`),
	regexp.MustCompile(`(?i)^package-lock\.json$`),
	regexp.MustCompile(`^Cargo\.lock$`),
}

var likelyBinaryExtensions = map[string]struct{}{
	"png": {}, "jpg": {}, "jpeg": {}, "gif": {}, "webp": {}, "bmp": {}, "ico": {}, "svg": {},
	"pdf": {}, "zip": {}, "gz": {}, "tar": {}, "rar": {}, "7z": {}, "mp3": {}, "mp4": {},
	"mov": {}, "avi": {}, "wav": {}, "ogg": {}, "ttf": {}, "otf": {}, "woff": {}, "woff2": {},
	"eot": {}, "jar": {}, "exe": {}, "dll": {}, "so": {}, "dylib": {}, "class": {},
}

func isLikelyGeneratedFile(path string) bool {
	for _, pattern := range likelyGeneratedPathPatterns {
		if pattern.MatchString(path) {
			return true
		}
	}
	return false
}

func isLikelyBinaryFile(path string) bool {
	ext := filepath.Ext(path)
	if ext == "" || ext == "." {
		return false
	}
	_, ok := likelyBinaryExtensions[strings.ToLower(strings.TrimPrefix(ext, "."))]
	return ok
}

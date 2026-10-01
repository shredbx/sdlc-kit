package video

// Platform identifies the upstream video host a link belongs to. It is a
// dictionary-backed discriminator (never a raw string): the authoritative value
// set lives in the video.platform dictionary YAML. Adding a new host = a new
// Platform constant + a new resolver_*.go adapter behind the Resolver interface,
// with zero changes to the registry dispatch.
type Platform string

const (
	// PlatformYouTube covers youtube.com, m.youtube.com and youtu.be links.
	PlatformYouTube Platform = "youtube"
	// PlatformTikTok covers tiktok.com links.
	PlatformTikTok Platform = "tiktok"
	// PlatformInstagram covers instagram.com reel/post links.
	PlatformInstagram Platform = "instagram"
	// PlatformFacebook is reserved for facebook.com links (no resolver wired in
	// Phase 1; present so the dictionary is complete for the usage-spec).
	PlatformFacebook Platform = "facebook"
)

// ParsePlatform maps a raw string to a Platform, reporting whether the value is
// a known dictionary member. Unknown values map to ("", false).
func ParsePlatform(s string) (Platform, bool) {
	p := Platform(s)
	if p.Valid() {
		return p, true
	}
	return "", false
}

// Valid reports whether p is one of the known video.platform dictionary values.
func (p Platform) Valid() bool {
	switch p {
	case PlatformYouTube, PlatformTikTok, PlatformInstagram, PlatformFacebook:
		return true
	}
	return false
}

// String returns the raw dictionary code.
func (p Platform) String() string { return string(p) }

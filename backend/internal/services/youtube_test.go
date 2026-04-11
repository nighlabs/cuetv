package services

import "testing"

func TestExtractYouTubeVideoID(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		// Valid URLs
		{"watch format", "https://www.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"watch with extra params", "https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=30", "dQw4w9WgXcQ", false},
		{"short URL", "https://youtu.be/dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"short URL with params", "https://youtu.be/dQw4w9WgXcQ?t=30", "dQw4w9WgXcQ", false},
		{"embed format", "https://www.youtube.com/embed/dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"shorts format", "https://www.youtube.com/shorts/dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"no www", "https://youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"mobile", "https://m.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"http scheme", "http://www.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"with dashes and underscores", "https://www.youtube.com/watch?v=abc-_1AbCd3", "abc-_1AbCd3", false},

		// Invalid URLs
		{"empty", "", "", true},
		{"not youtube", "https://vimeo.com/12345", "", true},
		{"no video ID", "https://www.youtube.com/watch", "", true},
		{"invalid video ID", "https://www.youtube.com/watch?v=short", "", true},
		{"random path", "https://www.youtube.com/channel/UCxyz", "", true},
		{"bare video ID", "dQw4w9WgXcQ", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractYouTubeVideoID(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractYouTubeVideoID(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractYouTubeVideoID(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

package config

import "testing"

func TestEnderecoDosUploads(t *testing.T) {
	casos := []struct{ front, base, quer string }{
		{"https://evve-delta.vercel.app", "", "https://evve-delta.vercel.app/api/v1/uploads"},
		{"https://evve-delta.vercel.app/", "http://localhost:8080/uploads", "https://evve-delta.vercel.app/api/v1/uploads"},
		{"https://evve-delta.vercel.app", "https://cdn.evve.app", "https://cdn.evve.app"},
		{"http://localhost:5173", "http://localhost:8080/uploads", "http://localhost:8080/uploads"},
	}
	for _, c := range casos {
		t.Setenv("FRONTEND_URL", c.front)
		t.Setenv("UPLOADS_BASE_URL", c.base)
		if got := Carregar().UploadsBaseURL; got != c.quer {
			t.Errorf("front=%s base=%q: %s, esperado %s", c.front, c.base, got, c.quer)
		}
	}
}

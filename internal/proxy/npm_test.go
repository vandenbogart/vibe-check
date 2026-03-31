package proxy

import (
	"testing"
)

func TestParseNPMTarballPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantPkg string
		wantVer string
		wantOK  bool
	}{
		{
			name:    "unscoped simple",
			path:    "/express/-/express-4.21.0.tgz",
			wantPkg: "express",
			wantVer: "4.21.0",
			wantOK:  true,
		},
		{
			name:    "unscoped lodash",
			path:    "/lodash/-/lodash-4.17.21.tgz",
			wantPkg: "lodash",
			wantVer: "4.17.21",
			wantOK:  true,
		},
		{
			name:    "scoped @types/node",
			path:    "/@types/node/-/node-22.0.0.tgz",
			wantPkg: "@types/node",
			wantVer: "22.0.0",
			wantOK:  true,
		},
		{
			name:    "scoped @babel/core",
			path:    "/@babel/core/-/core-7.24.0.tgz",
			wantPkg: "@babel/core",
			wantVer: "7.24.0",
			wantOK:  true,
		},
		{
			name:   "metadata request unscoped",
			path:   "/express",
			wantOK: false,
		},
		{
			name:   "metadata request scoped",
			path:   "/@types/node",
			wantOK: false,
		},
		{
			name:   "wrong extension",
			path:   "/express/-/express-4.21.0.json",
			wantOK: false,
		},
		{
			name:   "root path",
			path:   "/",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg, ver, ok := ParseNPMTarballPath(tt.path)
			if ok != tt.wantOK {
				t.Errorf("ParseNPMTarballPath(%q) ok = %v, want %v", tt.path, ok, tt.wantOK)
			}
			if pkg != tt.wantPkg {
				t.Errorf("ParseNPMTarballPath(%q) pkg = %q, want %q", tt.path, pkg, tt.wantPkg)
			}
			if ver != tt.wantVer {
				t.Errorf("ParseNPMTarballPath(%q) ver = %q, want %q", tt.path, ver, tt.wantVer)
			}
		})
	}
}

func TestIsNPMTarballPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "unscoped tarball",
			path: "/express/-/express-4.21.0.tgz",
			want: true,
		},
		{
			name: "scoped tarball",
			path: "/@types/node/-/node-22.0.0.tgz",
			want: true,
		},
		{
			name: "metadata request",
			path: "/express",
			want: false,
		},
		{
			name: "wrong extension",
			path: "/express/-/express-4.21.0.json",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNPMTarballPath(tt.path)
			if got != tt.want {
				t.Errorf("IsNPMTarballPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

package proxy

import (
	"testing"
)

func TestParsePyPIFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantPkg  string
		wantVer  string
		wantOK   bool
	}{
		{
			name:     "sdist tar.gz lowercase",
			filename: "flask-3.1.0.tar.gz",
			wantPkg:  "flask",
			wantVer:  "3.1.0",
			wantOK:   true,
		},
		{
			name:     "sdist tar.gz mixed case",
			filename: "Flask-3.1.0.tar.gz",
			wantPkg:  "flask",
			wantVer:  "3.1.0",
			wantOK:   true,
		},
		{
			name:     "sdist tar.gz multi-hyphen name",
			filename: "my-cool-package-1.2.3.tar.gz",
			wantPkg:  "my-cool-package",
			wantVer:  "1.2.3",
			wantOK:   true,
		},
		{
			name:     "sdist zip",
			filename: "numpy-2.0.0.zip",
			wantPkg:  "numpy",
			wantVer:  "2.0.0",
			wantOK:   true,
		},
		{
			name:     "wheel simple",
			filename: "flask-3.1.0-py3-none-any.whl",
			wantPkg:  "flask",
			wantVer:  "3.1.0",
			wantOK:   true,
		},
		{
			name:     "wheel mixed case",
			filename: "Flask-3.1.0-py3-none-any.whl",
			wantPkg:  "flask",
			wantVer:  "3.1.0",
			wantOK:   true,
		},
		{
			name:     "wheel underscore name",
			filename: "my_cool_package-1.2.3-py3-none-any.whl",
			wantPkg:  "my-cool-package",
			wantVer:  "1.2.3",
			wantOK:   true,
		},
		{
			name:     "wheel platform-specific",
			filename: "numpy-2.0.0-cp312-cp312-linux_x86_64.whl",
			wantPkg:  "numpy",
			wantVer:  "2.0.0",
			wantOK:   true,
		},
		{
			name:     "empty string",
			filename: "",
			wantPkg:  "",
			wantVer:  "",
			wantOK:   false,
		},
		{
			name:     "not a package",
			filename: "not-a-package",
			wantPkg:  "",
			wantVer:  "",
			wantOK:   false,
		},
		{
			name:     "filename with URL fragment",
			filename: "flask-3.1.0.tar.gz#sha256=abcdef123456",
			wantPkg:  "flask",
			wantVer:  "3.1.0",
			wantOK:   true,
		},
		{
			name:     "wheel metadata PEP 658",
			filename: "flask-3.1.0-py3-none-any.whl.metadata",
			wantPkg:  "flask",
			wantVer:  "3.1.0",
			wantOK:   true,
		},
		{
			name:     "sdist metadata PEP 658",
			filename: "flask-3.1.0.tar.gz.metadata",
			wantPkg:  "flask",
			wantVer:  "3.1.0",
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg, ver, ok := ParsePyPIFilename(tt.filename)
			if ok != tt.wantOK {
				t.Errorf("ParsePyPIFilename(%q) ok = %v, want %v", tt.filename, ok, tt.wantOK)
			}
			if pkg != tt.wantPkg {
				t.Errorf("ParsePyPIFilename(%q) pkg = %q, want %q", tt.filename, pkg, tt.wantPkg)
			}
			if ver != tt.wantVer {
				t.Errorf("ParsePyPIFilename(%q) ver = %q, want %q", tt.filename, ver, tt.wantVer)
			}
		})
	}
}

func TestIsPyPIDownloadPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "packages path with hash",
			path: "/packages/xx/yy/hash/flask-3.1.0.tar.gz",
			want: true,
		},
		{
			name: "packages path legacy layout",
			path: "/packages/source/f/flask/Flask-3.1.0.tar.gz",
			want: true,
		},
		{
			name: "simple path",
			path: "/simple/flask/",
			want: false,
		},
		{
			name: "simple root",
			path: "/simple/",
			want: false,
		},
		{
			name: "root path",
			path: "/",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPyPIDownloadPath(tt.path)
			if got != tt.want {
				t.Errorf("IsPyPIDownloadPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRewritePyPISimpleHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single link with hash",
			input:    `<a href="https://files.pythonhosted.org/packages/xx/yy/hash/flask-3.1.0.tar.gz#sha256=abc">flask-3.1.0.tar.gz</a>`,
			expected: `<a href="/packages/xx/yy/hash/flask-3.1.0.tar.gz#sha256=abc">flask-3.1.0.tar.gz</a>`,
		},
		{
			name:     "multiple links",
			input:    `<a href="https://files.pythonhosted.org/packages/a/b/c/foo-1.0.tar.gz#sha256=x">foo</a>\n<a href="https://files.pythonhosted.org/packages/d/e/f/foo-1.0-py3-none-any.whl#sha256=y">foo</a>`,
			expected: `<a href="/packages/a/b/c/foo-1.0.tar.gz#sha256=x">foo</a>\n<a href="/packages/d/e/f/foo-1.0-py3-none-any.whl#sha256=y">foo</a>`,
		},
		{
			name:     "no matching URLs",
			input:    `<a href="/simple/flask/">flask</a>`,
			expected: `<a href="/simple/flask/">flask</a>`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RewritePyPISimpleHTML(tt.input)
			if got != tt.expected {
				t.Errorf("RewritePyPISimpleHTML()\ngot:  %q\nwant: %q", got, tt.expected)
			}
		})
	}
}

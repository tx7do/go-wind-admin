package converter

import (
	"strings"
	"testing"

	"github.com/jinzhu/inflection"

	resourceV1 "go-wind-admin/api/gen/go/resource/service/v1"
)

func permBaseFromPath(p string) string {
	path := strings.TrimSpace(p)
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}
	base := strings.ReplaceAll(path, "/", ":")
	return inflection.Singular(base)
}

func TestMenuPermissionConverter_ConvertCode(t *testing.T) {
	c := NewMenuPermissionConverter()

	tests := []struct {
		name     string
		fullPath string
		title    string
		typ      resourceV1.Menu_Type
		want     string
	}{
		{
			name:     "catalog dir",
			fullPath: "/users",
			title:    "",
			typ:      resourceV1.Menu_CATALOG,
			want:     permBaseFromPath("/users") + ":dir",
		},
		{
			name:     "menu view",
			fullPath: "/orders/",
			title:    "",
			typ:      resourceV1.Menu_MENU,
			want:     permBaseFromPath("/orders/") + ":view",
		},
		{
			name:     "embedded view",
			fullPath: "/foo",
			title:    "",
			typ:      resourceV1.Menu_EMBEDDED,
			want:     permBaseFromPath("/foo") + ":view",
		},
		{
			name:     "link jump",
			fullPath: "/admin/settings",
			title:    "",
			typ:      resourceV1.Menu_LINK,
			want:     "setting:jump",
		},
		{
			name:     "button default act",
			fullPath: "admin/button",
			title:    "",
			typ:      resourceV1.Menu_BUTTON,
			want:     "button:act",
		},
		{
			name:     "button create (中文)",
			fullPath: "/users",
			title:    "新增",
			typ:      resourceV1.Menu_BUTTON,
			want:     permBaseFromPath("/users") + ":create",
		},
		{
			name:     "button export (中文)",
			fullPath: "/reports",
			title:    "导出",
			typ:      resourceV1.Menu_BUTTON,
			want:     "report:export",
		},
		{
			name:     "complex keep",
			fullPath: "/admin/inner",
			title:    "",
			typ:      resourceV1.Menu_MENU,
			want:     "inner:view",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.ConvertCode(tt.fullPath, tt.title, tt.typ)
			if got != tt.want {
				t.Fatalf("ConvertCode(%q, %q, %v) = %q, want %q", tt.fullPath, tt.title, tt.typ, got, tt.want)
			}
		})
	}
}

func TestMenuPermissionConverter_typeToAction(t *testing.T) {
	c := NewMenuPermissionConverter()

	tests := []struct {
		name  string
		typ   resourceV1.Menu_Type
		title string
		want  string
	}{
		{name: "catalog -> dir", typ: resourceV1.Menu_CATALOG, title: "", want: "dir"},
		{name: "menu -> access", typ: resourceV1.Menu_MENU, title: "", want: "view"},
		{name: "embedded -> view", typ: resourceV1.Menu_EMBEDDED, title: "", want: "view"},
		{name: "link -> jump", typ: resourceV1.Menu_LINK, title: "", want: "jump"},
		{name: "unknown type -> empty", typ: resourceV1.Menu_Type(999), title: "", want: ""},

		// Button cases: 不同 title 映射到不同 action
		{name: "button empty -> act", typ: resourceV1.Menu_BUTTON, title: "", want: "act"},
		{name: "button trim space -> act", typ: resourceV1.Menu_BUTTON, title: "   ", want: "act"},
		{name: "button add Chinese -> add", typ: resourceV1.Menu_BUTTON, title: "新增", want: "create"},
		{name: "button add English -> add", typ: resourceV1.Menu_BUTTON, title: "Add User", want: "create"},
		{name: "button edit -> edit", typ: resourceV1.Menu_BUTTON, title: "编辑", want: "edit"},
		{name: "button save -> edit", typ: resourceV1.Menu_BUTTON, title: "Save", want: "edit"},
		{name: "button delete -> delete", typ: resourceV1.Menu_BUTTON, title: "删除", want: "delete"},
		{name: "button import -> import", typ: resourceV1.Menu_BUTTON, title: "导入", want: "import"},
		{name: "button export -> export", typ: resourceV1.Menu_BUTTON, title: "导出", want: "export"},
		{name: "button mixed -> add (prefix match)", typ: resourceV1.Menu_BUTTON, title: "Add-to-list", want: "create"},
		{name: "button substring fallback -> export", typ: resourceV1.Menu_BUTTON, title: "一键导出为Excel", want: "export"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.typeToAction(tt.title, tt.typ)
			if got != tt.want {
				t.Fatalf("typeToAction(%q, %v) = %q, want %q", tt.title, tt.typ, got, tt.want)
			}
		})
	}
}

func TestMenuPermissionConverte1(t *testing.T) {
	c := NewMenuPermissionConverter()
	t.Log(c.ConvertCode("/opa/users/:id", "", resourceV1.Menu_MENU))
}

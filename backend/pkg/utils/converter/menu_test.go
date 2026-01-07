package converter

import (
	"strings"
	"testing"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"github.com/jinzhu/inflection"
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
		typ      permissionV1.Menu_Type
		want     string
	}{
		{
			name:     "catalog dir",
			fullPath: "/users",
			title:    "",
			typ:      permissionV1.Menu_CATALOG,
			want:     permBaseFromPath("/users") + ":dir",
		},
		{
			name:     "menu access",
			fullPath: "/orders/",
			title:    "",
			typ:      permissionV1.Menu_MENU,
			want:     permBaseFromPath("/orders/") + ":access",
		},
		{
			name:     "embedded view",
			fullPath: "/foo",
			title:    "",
			typ:      permissionV1.Menu_EMBEDDED,
			want:     permBaseFromPath("/foo") + ":view",
		},
		{
			name:     "link jump",
			fullPath: "/admin/settings",
			title:    "",
			typ:      permissionV1.Menu_LINK,
			want:     permBaseFromPath("/admin/settings") + ":jump",
		},
		{
			name:     "button default act",
			fullPath: "admin/button",
			title:    "",
			typ:      permissionV1.Menu_BUTTON,
			want:     permBaseFromPath("admin/button") + ":act",
		},
		{
			name:     "button add (中文)",
			fullPath: "/users",
			title:    "新增",
			typ:      permissionV1.Menu_BUTTON,
			want:     permBaseFromPath("/users") + ":add",
		},
		{
			name:     "button export (中文)",
			fullPath: "/reports",
			title:    "导出",
			typ:      permissionV1.Menu_BUTTON,
			want:     permBaseFromPath("/reports") + ":export",
		},
		{
			name:     "complex keep v2",
			fullPath: "/admin/v2/inner",
			title:    "",
			typ:      permissionV1.Menu_MENU,
			want:     permBaseFromPath("/admin/v2/inner") + ":access",
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
		typ   permissionV1.Menu_Type
		title string
		want  string
	}{
		{name: "catalog -> dir", typ: permissionV1.Menu_CATALOG, title: "", want: "dir"},
		{name: "menu -> access", typ: permissionV1.Menu_MENU, title: "", want: "access"},
		{name: "embedded -> view", typ: permissionV1.Menu_EMBEDDED, title: "", want: "view"},
		{name: "link -> jump", typ: permissionV1.Menu_LINK, title: "", want: "jump"},
		{name: "unknown type -> empty", typ: permissionV1.Menu_Type(999), title: "", want: ""},

		// Button cases: 不同 title 映射到不同 action
		{name: "button empty -> act", typ: permissionV1.Menu_BUTTON, title: "", want: "act"},
		{name: "button trim space -> act", typ: permissionV1.Menu_BUTTON, title: "   ", want: "act"},
		{name: "button add chinese -> add", typ: permissionV1.Menu_BUTTON, title: "新增", want: "add"},
		{name: "button add english -> add", typ: permissionV1.Menu_BUTTON, title: "Add User", want: "add"},
		{name: "button edit -> edit", typ: permissionV1.Menu_BUTTON, title: "编辑", want: "edit"},
		{name: "button save -> edit", typ: permissionV1.Menu_BUTTON, title: "Save", want: "edit"},
		{name: "button delete -> delete", typ: permissionV1.Menu_BUTTON, title: "删除", want: "delete"},
		{name: "button import -> import", typ: permissionV1.Menu_BUTTON, title: "导入", want: "import"},
		{name: "button export -> export", typ: permissionV1.Menu_BUTTON, title: "导出", want: "export"},
		{name: "button mixed -> add (prefix match)", typ: permissionV1.Menu_BUTTON, title: "Add-to-list", want: "add"},
		{name: "button substring fallback -> export", typ: permissionV1.Menu_BUTTON, title: "一键导出为Excel", want: "export"},
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

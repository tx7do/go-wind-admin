package converter

import (
	"testing"
)

func TestApiPermissionConverter_ConvertByPath(t *testing.T) {
	c := NewApiPermissionConverter()

	cases := []struct {
		name   string
		method string
		path   string
		want   string
	}{
		{"get list users", "GET", "/v1/users", "user:view"},
		{"get single user", "GET", "/v1/users/{id}", "user:view"},
		{"create user", "POST", "/v1/users", "user:create"},
		{"update user", "PUT", "/v1/users/{id}", "user:edit"},
		{"delete user", "DELETE", "/v1/users/{id}", "user:delete"},
		{"nested admin settings", "GET", "/api/v1/admin/settings", "admin:view"},
		{"hyphen group", "GET", "/v1/user-groups", "user-group:view"},
		{"get task by typeNames", "GET", "/admin/v1/tasks:type-names", "task:view"},
		{"walk route", "GET", "/admin/v1/apis/walk-route", "api:view"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := c.ConvertCodeByPath(tc.method, tc.path)
			if got != tc.want {
				t.Fatalf("ConvertCodeByPath(%q, %q) = %q, want %q", tc.method, tc.path, got, tc.want)
			}
		})
	}
}

func TestApiPermissionConverter_ConvertByOperationID(t *testing.T) {
	c := NewApiPermissionConverter()

	cases := []struct {
		name string
		op   string
		want string
	}{
		{"rpc with service and name", "TaskService_ListTaskTypeName", "task:task-type-name:list"},
		{"rpc without Service suffix", "Task_ListTaskTypeName", "task:task-type-name:list"},
		{"get walk route data", "ApiService_GetWalkRouteData", "api:walk-route-data:get"},
		{"rpc without name", "Task_List", "task:list"},
		{"invalid rpc format", "GetStatus", ""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := c.ConvertCodeByOperationID(tc.op)
			if got != tc.want {
				t.Fatalf("ConvertCodeByOperationID(%q) = %q, want %q", tc.op, got, tc.want)
			}
		})
	}
}

func TestStripVersionPrefix(t *testing.T) {
	c := NewApiPermissionConverter()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"root slash", "/", ""},
		{"no leading slash v1/users", "v1/users", "users"},
		{"v1 users", "/v1/users", "users"},
		{"api v1 admin", "/api/v1/admin/settings", "admin/settings"},
		{"v2 only", "/v2", ""},
		{"v10 without slash", "v10", ""},
		{"api v10", "/api/v10", ""},
		{"double slash after version", "/api/v1//admin", "admin"},
		{"double slash after version", "/api/v1.9/admin", "admin"},
		{"no version path", "/foo/bar", "foo/bar"},
		{"no version path", "/admin/v1/tasks:type-names", "tasks:type-names"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := c.stripVersionPrefix(tc.in)
			if got != tc.want {
				t.Fatalf("stripVersionPrefix(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestRemovePathParams(t *testing.T) {
	c := NewApiPermissionConverter()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"root slash", "/", ""},
		{"only param", "/{id}", ""},
		{"only param no slash", "{id}", ""},
		{"param with spaces", "/{ id }/", ""},
		{"simple resource", "/users", "users"},
		{"resource with param", "/users/{id}", "users"},
		{"nested resources", "/users/{id}/posts/{postId}", "users/posts"},
		{"trailing slash", "/users/{id}/posts/", "users/posts"},
		{"leading param", "/{id}/users", "users"},
		{"all params", "/{id}/{pid}", ""},
		{"double slashes", "/users//{id}//posts", "users/posts"},
		{"colon segment", "/tasks:type-names/{id}", "tasks:type-names"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := c.removePathParams(tc.in)
			if got != tc.want {
				t.Fatalf("removePathParams(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSingularizeSegments(t *testing.T) {
	c := NewApiPermissionConverter()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"already singular", "user:profile", "user:profile"},
		{"simple plural", "tasks", "task"},
		{"colon segments", "tasks:type-names", "task:type-name"},
		{"hyphen segments", "tasks-users", "task-user"},
		{"underline segments", "tasks_users", "task_user"},
		{"slash and underscore", "tasks/type_names", "task/type_name"},
		{"mixed separators", "tasks::names--items", "task::name--item"},
		{"irregular plural", "statuses:passes", "status:pass"},
		{"multiple separators preserved", "tasks--and__more::names", "task--and__more::name"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := c.singularizeSegments(tc.in)
			if got != tc.want {
				t.Fatalf("singularizeSegments(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestMethodToAction(t *testing.T) {
	c := NewApiPermissionConverter()

	cases := []struct {
		name   string
		method string
		path   string
		want   string
	}{
		{"GET -> view", "GET", "/v1/users", "view"},
		{"get lowercase -> view", "get", "/v1/users", "view"},
		{"POST -> create", "POST", "/v1/users", "create"},
		{"mixed case POST -> create", "PoSt", "/v1/users", "create"},
		{"PUT -> edit", "PUT", "/v1/users/1", "edit"},
		{"PATCH -> edit", "PATCH", "/v1/users/1", "edit"},
		{"DELETE -> delete", "DELETE", "/v1/users/1", "delete"},
		{"unknown method -> lowercase", "OPTIONS", "/v1/users", "options"},
		{"path ends with /list overrides method", "POST", "/v1/users/list", "view"},
		{"path ends with /list and GET", "GET", "/v1/users/list", "view"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := c.methodToAction(tc.method, tc.path)
			if got != tc.want {
				t.Fatalf("methodToAction(%q, %q) = %q, want %q", tc.method, tc.path, got, tc.want)
			}
		})
	}
}

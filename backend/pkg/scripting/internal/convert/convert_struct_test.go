package convert

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestToLuaValue_Struct(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	type Address struct {
		Street string
		City   string
		Zip    int
	}

	type Person struct {
		Name    string
		Age     int
		Active  bool
		Address Address
	}

	person := Person{
		Name:   "John Doe",
		Age:    30,
		Active: true,
		Address: Address{
			Street: "123 Main St",
			City:   "NYC",
			Zip:    10001,
		},
	}

	// ConvertCode to Lua
	luaValue := ToLuaValue(L, person)

	// Should be a table
	table, ok := luaValue.(*lua.LTable)
	if !ok {
		t.Fatalf("Expected table, got %T", luaValue)
	}

	// Check fields
	name := table.RawGetString("Name")
	if name.Type() != lua.LTString {
		t.Errorf("Expected Name to be string, got %v", name.Type())
	}
	if lua.LVAsString(name) != "John Doe" {
		t.Errorf("Expected Name to be 'John Doe', got %v", lua.LVAsString(name))
	}

	age := table.RawGetString("Age")
	if age.Type() != lua.LTNumber {
		t.Errorf("Expected Age to be number, got %v", age.Type())
	}
	if int(lua.LVAsNumber(age)) != 30 {
		t.Errorf("Expected Age to be 30, got %v", int(lua.LVAsNumber(age)))
	}

	active := table.RawGetString("Active")
	if active.Type() != lua.LTBool {
		t.Errorf("Expected Active to be bool, got %v", active.Type())
	}
	if !lua.LVAsBool(active) {
		t.Error("Expected Active to be true")
	}

	// Check nested struct
	address := table.RawGetString("Address")
	if address.Type() != lua.LTTable {
		t.Errorf("Expected Address to be table, got %v", address.Type())
	}

	addrTable := address.(*lua.LTable)
	city := addrTable.RawGetString("City")
	if lua.LVAsString(city) != "NYC" {
		t.Errorf("Expected City to be 'NYC', got %v", lua.LVAsString(city))
	}

	t.Log("✓ Struct conversion test passed")
}

func TestToLuaValue_StructPointer(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	type Config struct {
		Host string
		Port int
	}

	config := &Config{
		Host: "localhost",
		Port: 8080,
	}

	// ConvertCode pointer to Lua
	luaValue := ToLuaValue(L, config)

	// Should be a table (pointer dereferenced)
	table, ok := luaValue.(*lua.LTable)
	if !ok {
		t.Fatalf("Expected table, got %T", luaValue)
	}

	host := table.RawGetString("Host")
	if lua.LVAsString(host) != "localhost" {
		t.Errorf("Expected Host to be 'localhost', got %v", lua.LVAsString(host))
	}

	port := table.RawGetString("Port")
	if int(lua.LVAsNumber(port)) != 8080 {
		t.Errorf("Expected Port to be 8080, got %v", int(lua.LVAsNumber(port)))
	}

	t.Log("✓ Struct pointer conversion test passed")
}

func TestToLuaValue_StructWithSlice(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	type User struct {
		Name  string
		Roles []string
	}

	user := User{
		Name:  "Alice",
		Roles: []string{"admin", "user", "moderator"},
	}

	luaValue := ToLuaValue(L, user)

	table, ok := luaValue.(*lua.LTable)
	if !ok {
		t.Fatalf("Expected table, got %T", luaValue)
	}

	roles := table.RawGetString("Roles")
	if roles.Type() != lua.LTTable {
		t.Errorf("Expected Roles to be table, got %v", roles.Type())
	}

	rolesTable := roles.(*lua.LTable)

	// Check first role (Lua is 1-indexed)
	role1 := rolesTable.RawGetInt(1)
	if lua.LVAsString(role1) != "admin" {
		t.Errorf("Expected first role to be 'admin', got %v", lua.LVAsString(role1))
	}

	// Check array length
	if rolesTable.Len() != 3 {
		t.Errorf("Expected 3 roles, got %d", rolesTable.Len())
	}

	t.Log("✓ Struct with slice conversion test passed")
}

func TestToLuaValue_StructWithMap(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	type Config struct {
		Name     string
		Metadata map[string]string
	}

	config := Config{
		Name: "MyApp",
		Metadata: map[string]string{
			"version":     "1.0.0",
			"environment": "production",
		},
	}

	luaValue := ToLuaValue(L, config)

	table, ok := luaValue.(*lua.LTable)
	if !ok {
		t.Fatalf("Expected table, got %T", luaValue)
	}

	metadata := table.RawGetString("Metadata")
	if metadata.Type() != lua.LTTable {
		t.Errorf("Expected Metadata to be table, got %v", metadata.Type())
	}

	metaTable := metadata.(*lua.LTable)
	version := metaTable.RawGetString("version")
	if lua.LVAsString(version) != "1.0.0" {
		t.Errorf("Expected version to be '1.0.0', got %v", lua.LVAsString(version))
	}

	t.Log("✓ Struct with map conversion test passed")
}

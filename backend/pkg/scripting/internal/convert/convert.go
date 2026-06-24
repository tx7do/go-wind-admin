package convert

import (
	"fmt"
	"reflect"

	lua "github.com/yuin/gopher-lua"
)

// ToLuaValue converts a Go value to a Lua value.
// This is the primary function for Go → Lua conversion.
// Supports primitives, maps, slices, and nested structures.
//
// Inspired by dragon-mud's getLValue and ValueFor pattern.
func ToLuaValue(L *lua.LState, val any) lua.LValue {
	if val == nil {
		return lua.LNil
	}

	// Handle if already a Lua value
	if lv, ok := val.(lua.LValue); ok {
		return lv
	}

	switch v := val.(type) {
	case bool:
		return lua.LBool(v)
	case string:
		return lua.LString(v)
	case int:
		return lua.LNumber(v)
	case int8:
		return lua.LNumber(v)
	case int16:
		return lua.LNumber(v)
	case int32:
		return lua.LNumber(v)
	case int64:
		return lua.LNumber(v)
	case uint:
		return lua.LNumber(v)
	case uint8:
		return lua.LNumber(v)
	case uint16:
		return lua.LNumber(v)
	case uint32:
		return lua.LNumber(v)
	case uint64:
		return lua.LNumber(v)
	case float32:
		return lua.LNumber(v)
	case float64:
		return lua.LNumber(v)
	case map[string]any:
		return tableFromMap(L, v)
	case []any:
		return tableFromSlice(L, v)
	default:
		// Use reflection for complex types (structs, other maps, other slices)
		return reflectToLua(L, val)
	}
}

// ToGoValue converts a Lua value to the best associated Go type.
// This is the primary function for Lua → Go conversion.
//
// Based on dragon-mud's AsRaw pattern:
// - LTString → string
// - LTBool → bool
// - LTNil → nil
// - LTNumber → float64
// - LTTable → map[string]any or []any (based on structure)
func ToGoValue(val lua.LValue) any {
	if val == nil || val == lua.LNil {
		return nil
	}

	switch v := val.(type) {
	case lua.LBool:
		return bool(v)
	case lua.LNumber:
		return float64(v)
	case lua.LString:
		return string(v)
	case *lua.LTable:
		return tableToGo(v)
	default:
		return nil
	}
}

// tableFromMap creates a Lua table from a Go map.
// Recursively handles nested maps and slices.
func tableFromMap(L *lua.LState, m map[string]any) *lua.LTable {
	table := L.NewTable()
	for key, value := range m {
		table.RawSetString(key, ToLuaValue(L, value))
	}
	return table
}

// tableFromSlice creates a Lua table from a Go slice.
// Lua tables use 1-based indexing, so index 0 becomes 1.
func tableFromSlice(L *lua.LState, s []any) *lua.LTable {
	table := L.NewTable()
	for i, value := range s {
		table.RawSetInt(i+1, ToLuaValue(L, value))
	}
	return table
}

// tableToGo converts a Lua table to either a Go slice or map.
// Uses MaxN() to determine if the table is array-like or map-like.
//
// Based on dragon-mud's AsSliceInterface and AsMapStringInterface pattern:
// - If table has numeric indices starting from 1 (MaxN > 0), treat as slice
// - Otherwise, treat as map
func tableToGo(t *lua.LTable) any {
	maxN := t.MaxN()
	if maxN > 0 {
		// Array-like table (has numeric indices)
		arr := make([]any, 0, maxN)
		for i := 1; i <= maxN; i++ {
			val := t.RawGetInt(i)
			arr = append(arr, ToGoValue(val))
		}
		return arr
	}

	// Map-like table (has string keys or is empty)
	m := make(map[string]any)
	t.ForEach(func(k, val lua.LValue) {
		if key, ok := k.(lua.LString); ok {
			m[string(key)] = ToGoValue(val)
		}
	})
	return m
}

// ToString converts a Lua value to a Go string.
// Uses Lua's built-in string conversion rules.
func ToString(val lua.LValue) string {
	return lua.LVAsString(val)
}

// ToNumber converts a Lua value to a Go float64.
// Uses Lua's built-in number conversion rules.
func ToNumber(val lua.LValue) float64 {
	return float64(lua.LVAsNumber(val))
}

// ToBool converts a Lua value to a Go bool.
// Uses Lua's boolean semantics (only nil and false are false).
func ToBool(val lua.LValue) bool {
	return lua.LVAsBool(val)
}

// reflectToLua uses reflection to convert complex Go types to Lua values.
// Handles: structs, pointers, maps, slices, and other types.
func reflectToLua(L *lua.LState, val any) lua.LValue {
	if val == nil {
		return lua.LNil
	}

	v := reflect.ValueOf(val)

	// Dereference pointers
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return lua.LNil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Bool:
		return lua.LBool(v.Bool())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return lua.LNumber(v.Int())

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return lua.LNumber(v.Uint())

	case reflect.Float32, reflect.Float64:
		return lua.LNumber(v.Float())

	case reflect.String:
		return lua.LString(v.String())

	case reflect.Slice, reflect.Array:
		return reflectSliceToLua(L, v)

	case reflect.Map:
		return reflectMapToLua(L, v)

	case reflect.Struct:
		return reflectStructToLua(L, v)

	case reflect.Interface:
		// Get the actual value inside the interface
		if v.IsNil() {
			return lua.LNil
		}
		return reflectToLua(L, v.Interface())

	default:
		// For unsupported types, convert to string
		return lua.LString(fmt.Sprintf("%v", val))
	}
}

// reflectStructToLua converts a Go struct to a Lua table.
// Exported fields become table keys.
func reflectStructToLua(L *lua.LState, v reflect.Value) *lua.LTable {
	table := L.NewTable()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Use field name as key
		key := field.Name

		// ConvertCode field value to Lua
		luaValue := reflectToLua(L, fieldValue.Interface())
		table.RawSetString(key, luaValue)
	}

	return table
}

// reflectMapToLua converts a Go map to a Lua table.
func reflectMapToLua(L *lua.LState, v reflect.Value) *lua.LTable {
	table := L.NewTable()

	iter := v.MapRange()
	for iter.Next() {
		key := iter.Key()
		val := iter.Value()

		// ConvertCode key to string (Lua table keys)
		var keyStr string
		switch key.Kind() {
		case reflect.String:
			keyStr = key.String()
		default:
			keyStr = fmt.Sprintf("%v", key.Interface())
		}

		// ConvertCode value to Lua
		luaValue := reflectToLua(L, val.Interface())
		table.RawSetString(keyStr, luaValue)
	}

	return table
}

// reflectSliceToLua converts a Go slice/array to a Lua table (array).
func reflectSliceToLua(L *lua.LState, v reflect.Value) *lua.LTable {
	table := L.NewTable()

	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		luaValue := reflectToLua(L, elem.Interface())
		table.RawSetInt(i+1, luaValue) // Lua uses 1-based indexing
	}

	return table
}

// Filename: luaYaegi.go
package main

/*
#cgo CFLAGS: -I./deps/LuaJIT/src
#cgo LDFLAGS: -L. -lwrapper

#include <stdlib.h>
#include <lua.h>
#include <lauxlib.h>

// Declare our C wrapper functions so Go can see them.
void my_lua_getglobal(lua_State *L, const char *name);
int my_lua_isfunction(lua_State *L, int n);
void my_lua_newtable(lua_State *L);
void my_lua_call (lua_State *L, int nargs, int nresults);
int my_lua_pcall(lua_State *L, int nargs, int nresults, int errfunc);
int my_lua_gettop(lua_State *L);
int my_lua_type(lua_State *L, int idx);
int my_lua_toboolean (lua_State *L, int index);
double my_lua_tonumber (lua_State *L, int index);
void my_lua_pushvalue (lua_State *L, int index);
void my_lua_pop(lua_State *L, int n);
void my_lua_pushcfunction(lua_State *L, lua_CFunction f);
void my_lua_pushstring(lua_State *L, const char *s);
void my_lua_settable(lua_State *L, int idx);
int my_luaL_newmetatable (lua_State *L, const char *tname);
void my_lua_pushnil (lua_State *L);
void my_lua_pushnumber (lua_State *L, lua_Number n);
void my_lua_pushboolean (lua_State *L, int b);
void my_lua_setfield (lua_State *L, int index, const char *k);
void *my_lua_newuserdata (lua_State *L, size_t size);
void my_luaL_getmetatable (lua_State *L, const char *tname);
int my_lua_setmetatable (lua_State *L, int index);
const char* my_lua_tostring(lua_State *L, int n);
void myprint(char* s);

// Forward declaration for the C-callable wrapper function.
int yaegiExec(lua_State *L);
int go_yaegi_shutdown(lua_State *L);
int newInterpreter(lua_State *L);
int execGoFunc(lua_State *L);

*/
import "C"
import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"unsafe"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

type FuncRegistry struct {
	funcs map[string]reflect.Value
}

func NewRegistry() *FuncRegistry {
	return &FuncRegistry{funcs: make(map[string]reflect.Value)}
}

func (r *FuncRegistry) Register(name string, fn any) {
	r.funcs[name] = reflect.ValueOf(fn)
}

func (r *FuncRegistry) Call(name string, args ...any) ([]any, error) {
	fn, ok := r.funcs[name]
	if !ok {
		return nil, fmt.Errorf("function %q not found", name)
	}

	t := fn.Type()
	numIn := t.NumIn()
	isVariadic := t.IsVariadic()

	// Argument count check
	if !isVariadic && len(args) != numIn {
		return nil, fmt.Errorf("wrong number of arguments for %q: got %d, need %d",
			name, len(args), numIn)
	}
	if isVariadic && len(args) < numIn-1 {
		return nil, fmt.Errorf("not enough arguments for variadic %q", name)
	}

	// Prepare argument list
	in := []reflect.Value{}
	for i, arg := range args {
		var expected reflect.Type
		if isVariadic && i >= numIn-1 {
			expected = t.In(numIn - 1).Elem() // element type of variadic parameter
		} else {
			expected = t.In(i)
		}
		val := reflect.ValueOf(arg)
		if !val.Type().AssignableTo(expected) {
			return nil, fmt.Errorf("wrong type for arg %d of %q: got %v, need %v",
				i, name, val.Type(), expected)
		}
		in = append(in, val)
	}

	out := fn.Call(in)

	results := make([]any, len(out))
	for i, v := range out {
		results[i] = v.Interface()
	}

	return results, nil
}

type interpreter struct {
	goInterpreter *interp.Interpreter
	luaMutex      sync.Mutex
	GlobalCancel  context.CancelFunc
	hostLuaState  *C.lua_State
	funcRegistry  *FuncRegistry
}

var interpreters = make(map[uintptr]*interpreter)

//export newInterpreter
func newInterpreter(L *C.lua_State) C.int {
	if old := interpreters[uintptr(unsafe.Pointer(L))]; old != nil {
		if old.GlobalCancel != nil {
			old.GlobalCancel()
		}
		delete(interpreters, uintptr(unsafe.Pointer(L)))
	}

	myInterp := &interpreter{hostLuaState: L, funcRegistry: NewRegistry()}
	interpreters[uintptr(unsafe.Pointer(L))] = myInterp
	ctx, cancel := context.WithCancel(context.Background())
	myInterp.GlobalCancel = cancel
	myInterp.goInterpreter = interp.New(interp.Options{})
	myInterp.goInterpreter.Use(stdlib.Symbols)

	bridgePkg := map[string]reflect.Value{
		"CallLua":      reflect.ValueOf(myInterp.CallLua),
		"FuncRegistry": reflect.ValueOf(myInterp.funcRegistry),
	}

	myInterp.goInterpreter.Use(interp.Exports{"bridge/bridge": bridgePkg})
	myInterp.goInterpreter.Use(interp.Exports{"GlobalCtx/": {
		"outCtx": reflect.ValueOf(ctx),
	},
	})

	fmt.Println("[Go] Yaegi interpreter initialized for Lua.")
	return 0
}

func luaError(L *C.lua_State, errorMsg string) {
	errTxt := C.CString("error")
	defer C.free(unsafe.Pointer(errTxt))
	cErrorMsg := C.CString(errorMsg)
	defer C.free(unsafe.Pointer(cErrorMsg))
	C.my_lua_getglobal(L, errTxt)
	C.my_lua_pushstring(L, cErrorMsg)
	C.my_lua_call(L, 1, 1)
	C.my_lua_pop(L, 1)
}

//export yaegiExec
func yaegiExec(L *C.lua_State) C.int {
	myInterp := interpreters[uintptr(unsafe.Pointer(L))]
	if myInterp == nil {
		return 0
	}
	goCode := C.GoString(C.my_lua_tostring(L, 1))
	fmt.Printf("[Go] Executing go code:\n---\n%s\n---\n", goCode)
	_, err := myInterp.goInterpreter.Eval(goCode)
	if err != nil {
		luaError(L, fmt.Sprintf("[Go Error] Yaegi execution failed: %v\n", err))
	}
	return 0
}

//export go_yaegi_shutdown
func go_yaegi_shutdown(L *C.lua_State) C.int {
	myInterp := interpreters[uintptr(unsafe.Pointer(L))]
	if myInterp == nil {
		return 0
	}
	myInterp.GlobalCancel()
	fmt.Println("Canceling")
	delete(interpreters, uintptr(unsafe.Pointer(L)))
	return 0
}

func push(L *C.lua_State, name string, fn unsafe.Pointer) {
	cs := C.CString(name)
	C.my_lua_pushstring(L, cs)
	C.free(unsafe.Pointer(cs))
	C.my_lua_pushcfunction(L, (C.lua_CFunction)(fn))
	C.my_lua_settable(L, -3)
}

func luaValueToGo(L *C.lua_State, idx C.int) any {
	switch C.my_lua_type(L, idx) {
	case C.LUA_TNIL:
		return reflect.Zero(reflect.TypeOf((*interface{})(nil)).Elem())
	case C.LUA_TBOOLEAN:
		return C.my_lua_toboolean(L, idx) != 0
	case C.LUA_TNUMBER:
		return float64(C.my_lua_tonumber(L, idx))
	case C.LUA_TSTRING:
		return C.GoString(C.my_lua_tostring(L, idx))
	case C.LUA_TTABLE, C.LUA_TFUNCTION, C.LUA_TUSERDATA, C.LUA_TTHREAD:
		// Use Lua's tostring() equivalent
		C.my_lua_getglobal(L, C.CString("tostring"))
		C.my_lua_pushvalue(L, idx)
		C.my_lua_call(L, 1, 1)
		s := C.GoString(C.my_lua_tostring(L, -1))
		C.my_lua_pop(L, 1)
		return s
	default:
		return fmt.Sprintf("<unsupported type %d>", C.my_lua_type(L, idx))
	}
}

func goValueToLua(L *C.lua_State, v any) {
	switch val := v.(type) {
	case nil:
		C.my_lua_pushnil(L)

	case bool:
		var b C.int
		if val {
			b = 1
		}
		C.my_lua_pushboolean(L, b)

	// Signed integers
	case int, int8, int16, int32, int64:
		C.my_lua_pushnumber(L, C.double(reflect.ValueOf(val).Int()))

	// Unsigned integers
	case uint, uint8, uint16, uint32, uint64:
		C.my_lua_pushnumber(L, C.double(reflect.ValueOf(val).Uint()))

	case float32:
		C.my_lua_pushnumber(L, C.double(val))
	case float64:
		C.my_lua_pushnumber(L, C.double(val))

	case string:
		cstr := C.CString(val)
		C.my_lua_pushstring(L, cstr)
		C.free(unsafe.Pointer(cstr))

	default:
		s := fmt.Sprintf("%v", val)
		cstr := C.CString(s)
		C.my_lua_pushstring(L, cstr)
		C.free(unsafe.Pointer(cstr))
	}
}

//export execGoFunc
func execGoFunc(L *C.lua_State) C.int {
	myInterp := interpreters[uintptr(unsafe.Pointer(L))]
	if myInterp == nil {
		return 0
	}
	funcName := C.GoString(C.my_lua_tostring(L, 1))
	arg_n := int(C.my_lua_gettop(L)) - 1
	args := make([]any, 0, arg_n)
	for i := 2; i < arg_n+2; i++ {
		arg := luaValueToGo(L, C.int(i))
		args = append(args, arg)
	}
	results, err := myInterp.funcRegistry.Call(funcName, args...)
	if err != nil {
		luaError(L, err.Error())
		return 0
	}

	for _, r := range results {
		goValueToLua(L, r)
	}

	return C.int(len(results))
}

//export luaopen_luaYaegi
func luaopen_luaYaegi(L *C.lua_State) C.int {
	newInterpreter(L)

	mtName := C.CString("MyResourceMeta")
	keyGC := C.CString("__gc")
	C.my_luaL_newmetatable(L, mtName)
	C.my_lua_pushcfunction(L, (C.lua_CFunction)(unsafe.Pointer(C.go_yaegi_shutdown)))
	C.my_lua_setfield(L, -2, keyGC)
	C.my_lua_pop(L, 1)

	C.my_lua_newtable(L)
	push(L, "exec", unsafe.Pointer(C.yaegiExec))
	push(L, "shutdown", unsafe.Pointer(C.go_yaegi_shutdown))
	push(L, "new", unsafe.Pointer(C.newInterpreter))
	push(L, "execGoFunc", unsafe.Pointer(C.execGoFunc))

	C.my_lua_newuserdata(L, C.size_t(1))
	C.my_luaL_getmetatable(L, mtName)
	C.my_lua_setmetatable(L, -2)

	fld := C.CString("__gc_proxy")
	C.my_lua_setfield(L, -2, fld)

	C.free(unsafe.Pointer(mtName))
	C.free(unsafe.Pointer(keyGC))
	C.free(unsafe.Pointer(fld))

	return 1 // return module table
}

func (self *interpreter) CallLua(functionName string, args ...interface{}) (interface{}, error) {
	self.luaMutex.Lock()
	defer self.luaMutex.Unlock()

	cFuncName := C.CString(functionName)
	defer C.free(unsafe.Pointer(cFuncName))

	// push function
	C.my_lua_getglobal(self.hostLuaState, cFuncName)
	if C.my_lua_isfunction(self.hostLuaState, -1) == 0 {
		C.my_lua_pop(self.hostLuaState, 1)
		return nil, fmt.Errorf("lua function %q not found or is not a function", functionName)
	}

	// push arguments
	for _, a := range args {
		goValueToLua(self.hostLuaState, a)
	}
	argCount := len(args)

	// call function (multret)
	if C.my_lua_pcall(self.hostLuaState, C.int(argCount), C.LUA_MULTRET, 0) != C.LUA_OK {
		errStr := C.GoString(C.my_lua_tostring(self.hostLuaState, -1))
		C.my_lua_pop(self.hostLuaState, 1)
		return nil, fmt.Errorf("error calling lua function %q: %s", functionName, errStr)
	}

	// collect results
	nRet := int(C.lua_gettop(self.hostLuaState))
	if nRet == 0 {
		return nil, nil
	}

	if nRet == 1 {
		res := luaValueToGo(self.hostLuaState, -1)
		C.my_lua_pop(self.hostLuaState, 1)
		return res, nil
	}

	// multiple results
	results := make([]any, nRet)
	for i := nRet; i >= 1; i-- { // preserve original order
		results[i-1] = luaValueToGo(self.hostLuaState, C.int(-1))
		C.my_lua_pop(self.hostLuaState, 1)
	}
	return results, nil
}

func main() {}

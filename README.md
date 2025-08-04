# luaYaegi

`luaYaegi` is a shared library that embeds the [Yaegi](https://github.com/traefik/yaegi) Go interpreter inside LuaJIT.  
It lets Lua scripts compile, execute, and interact with native Go code at run-time.

---

## Capabilities

* **Execute arbitrary Go code** from Lua with `yaegi.exec`.
* **Register Go functions** (via `bridge.FuncRegistry.Register`) and call them from Lua with `yaegi.execGoFunc`.
* **Call back into Lua** from Go using the `CallLua` helper.
* **Cancel execution** and release resources with `yaegi.shutdown`.
* Automatic Lua ↔ Go value conversion for `nil`, booleans, numbers, and strings.

---

## Quick start

```lua
local yaegi = require "luaYaegi"

-- Run plain Go code
yaegi.exec([[
import "fmt"
fmt.Println("from Go:", 2+3)
]])

-- Register a Go function and call it
yaegi.exec([[
package main
import "bridge/bridge"
func init() {
    bridge.FuncRegistry.Register("Add", func(a, b int) int { return a + b })
}
]])
print("Add returned", yaegi.execGoFunc("Add", 7, 5))  -- 12

-- Shut down interpreter
yaegi.shutdown()
```

---

## Build targets (Makefile defaults)

| Target OS | Output file      | Toolchain / flags                                                 |
|-----------|------------------|-------------------------------------------------------------------|
| Linux     | `luaYaegi.so`    | `gcc` + `go build -buildmode=c-shared`                            |
| Windows   | `luaYaegi.dll`   | `x86_64-w64-mingw32-gcc` via mingw-w64                            |
| macOS     | `luaYaegi.dylib` | `clang` with `MACOSX_DEPLOYMENT_TARGET=10.15`                     |

The Makefile:

1. Builds LuaJIT in `deps/LuaJIT/`.  
2. Compiles `wrapper.c` and archives it into `libwrapper.a`.  
3. Builds the shared object for each platform.  
4. Strips the Linux binary (`strip --strip-unneeded`).

### Commands

```bash
make            # build linux, windows, darwin (runs only valid targets on host)
make clean      # remove artefacts and clean LuaJIT sub-tree
```

Cross-compiling for Windows requires the mingw-w64 toolchain.

---

## Manual build (Linux example)

```bash
sudo apt install build-essential x86_64-w64-mingw32-gcc clang
git clone --recursive https://github.com/Edru2/luaYaegi
cd luaYaegi
make linux
cp luaYaegi.so /path/to/your/lua/modules/
```

---

## Lua API

| Function | Description |
|----------|-------------|
| `yaegi.exec(code)` | Compile and run Go source string `code`. Errors propagate to Lua. |
| `yaegi.execGoFunc(name, …args)` | Call Go function `name` (registered through `FuncRegistry`). Returns all Go results. |
| `yaegi.new()` | Create a new interpreter (normally not required—`require` calls it). |
| `yaegi.shutdown()` | Cancel the interpreter’s context and free resources. |

### Registering Go functions

```go
import "bridge/bridge"

bridge.FuncRegistry.Register("Add", func(a, b int) int { return a + b })
```

Call from Lua:

```lua
local r = yaegi.execGoFunc("Add", 3, 4)  -- 7
```

---

## Error handling

* Compilation or run-time errors in Go trigger `error()` in Lua.  
* Wrap untrusted calls with `pcall`.

---

## License

MIT. See [LICENSE](./LICENSE).

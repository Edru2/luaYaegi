// Filename: wrapper.c
#include <lua.h> 
#include <lauxlib.h>
#include <stdio.h>

void my_lua_getglobal(lua_State *L, const char *name) {
    lua_getglobal(L, name);
}

int my_lua_isfunction(lua_State *L, int n) {
    return lua_isfunction(L, n);
}

void my_lua_newtable(lua_State *L) {
    lua_newtable(L);
}

int my_lua_pcall(lua_State *L, int nargs, int nresults, int errfunc) {
    return lua_pcall(L, nargs, nresults, errfunc);
}

void my_lua_call (lua_State *L, int nargs, int nresults) {
    lua_call (L, nargs, nresults);
}

void my_lua_pop(lua_State *L, int n) {
    lua_pop(L, n);
}

void my_lua_pushcfunction(lua_State *L, lua_CFunction f) {
    lua_pushcfunction(L, f);
}

void my_lua_pushstring(lua_State *L, const char *s) {
    lua_pushstring(L, s);
}

void my_lua_settable(lua_State *L, int idx) {
    lua_settable(L, -3);
}

int my_lua_gettop(lua_State *L) {
    return lua_gettop(L);
}

void my_lua_pushvalue (lua_State *L, int index) {
    lua_pushvalue (L, index);
}

void my_lua_pushnumber (lua_State *L, lua_Number n) {
    lua_pushnumber (L, n);
}

void my_lua_pushboolean (lua_State *L, int b) {
    lua_pushboolean (L, b);
}

void my_lua_pushnil (lua_State *L){
    lua_pushnil (L);
}

int my_lua_type(lua_State *L, int idx) {
    return lua_type(L, idx);
}

int my_lua_toboolean (lua_State *L, int index) {
    return lua_toboolean (L, index);
}

lua_Number my_lua_tonumber (lua_State *L, int index) {
    return lua_tonumber (L, index);
}

int my_luaL_newmetatable (lua_State *L, const char *tname) {
    return luaL_newmetatable (L, tname);
}

void my_lua_setfield (lua_State *L, int index, const char *k) {
    return lua_setfield (L, index, k);
}

void *my_lua_newuserdata (lua_State *L, size_t size) {
    return lua_newuserdata (L, size);
}

void my_luaL_getmetatable (lua_State *L, const char *tname) {
    return luaL_getmetatable (L, tname);
}

int my_lua_setmetatable (lua_State *L, int index) {
    return lua_setmetatable (L, index);
}

const char* my_lua_tostring(lua_State *L, int n) {
    return lua_tostring(L, n);
}

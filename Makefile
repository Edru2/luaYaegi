# Default settings
TARGET_LINUX = luaYaegi.so
TARGET_WINDOWS = luaYaegi.dll
TARGET_DARWIN = luaYaegi.dylib
GO_BUILD_MODE = -buildmode=c-shared
GO_SOURCE = luaYaegi.go
LUA_FOLDER = ./deps/LuaJIT/

ifeq ($(OS),Windows_NT)
	WIN_CC = x86_64-w64-mingw32-gcc
	WIN_AR = x86_64-w64-mingw32-ar
	WIN_MK = mingw32-make -C $(LUA_FOLDER) src
else
	WIN_CC = x86_64-w64-mingw32-gcc
	WIN_AR = x86_64-w64-mingw32-ar
	WIN_MK = make -C $(LUA_FOLDER) \
		HOST_CC="gcc" \
		CROSS=x86_64-w64-mingw32- \
		TARGET_SYS=Windows
endif

all: linux windows darwin

linux:
	@UNAME_S=$$(uname -s); \
	if [ "$$UNAME_S" = "Linux" ]; then \
	echo "Building for Linux..."; \
	make -C $(LUA_FOLDER); \
	gcc -c -I$(LUA_FOLDER)src -o wrapper.o wrapper.c; \
	ar rcs libwrapper.a wrapper.o; \
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build $(GO_BUILD_MODE) -o $(TARGET_LINUX) $(GO_SOURCE); \
	strip --strip-unneeded $(TARGET_LINUX); \
	else \
		echo "Not building for Linux as the operating system is not Linux."; \
	fi

windows:
	@echo "Building for Windows..."
	@$(WIN_MK) 
	@$(WIN_CC) -c -I$(LUA_FOLDER)src -o wrapper.o wrapper.c
	@$(WIN_AR) rcs libwrapper.a wrapper.o
	@ln -s $(LUA_FOLDER)src/libluajit-5.1.dll.a liblua.a
	@CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=$(WIN_CC) CGO_CFLAGS="-I$(LUA_FOLDER)src" CGO_LDFLAGS="-shared -O2 -L. -llua" go build $(GO_BUILD_MODE) -o $(TARGET_WINDOWS)

darwin:
	@UNAME_S=$$(uname -s); \
	if [ "$$UNAME_S" = "Darwin" ]; then \
		echo "Building for Darwin (macOS)..."; \
		export MACOSX_DEPLOYMENT_TARGET=10.15; \
		make -C $(LUA_FOLDER); \
		ln -s $(LUA_FOLDER)src/libluajit.a liblua.a; \
		clang -c -I$(LUA_FOLDER)src -o wrapper.o wrapper.c; \
		ar rcs libwrapper.a wrapper.o; \
		CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 CGO_LDFLAGS="-shared -O2 -L. -llua" go build $(GO_BUILD_MODE) -o $(TARGET_DARWIN) $(GO_SOURCE); \
	else \
		echo "Not building for Darwin (macOS) as the operating system is not Darwin."; \
	fi

clean:
	@echo "Cleaning up..."
	@rm -f $(TARGET_LINUX) $(TARGET_WINDOWS) $(TARGET_DARWIN)
ifeq ($(OS),Windows_NT)
	@mingw32-make -C $(LUA_FOLDER) clean
	@mingw32-make -C $(LUA_FOLDER)src clean
else
	@make -C $(LUA_FOLDER) clean
	@make -C $(LUA_FOLDER)src clean
endif



package lua

import (
	"net/http"

	luastrings "github.com/chai2010/glua-strings"
	gluahttp "github.com/cjoudrey/gluahttp"
	gluaurl "github.com/cjoudrey/gluaurl"
	gluahttpscrape "github.com/felipejfc/gluahttpscrape"
	gluatemplate "github.com/kohkimakimoto/gluatemplate"
	gluare "github.com/yuin/gluare"
	lua "github.com/yuin/gopher-lua"
)

var luaHTTPClient = &http.Client{}

func preloadModules(L *lua.LState) {
	luastrings.Preload(L)
	L.PreloadModule("template", gluatemplate.Loader)
	L.PreloadModule("url", gluaurl.Loader)
	L.PreloadModule("re", gluare.Loader)
	L.PreloadModule("http", luaHTTPModuleLoader(luaHTTPClient))
	L.PreloadModule("scrape", gluahttpscrape.NewHttpScrapeModule().Loader)
}

func luaHTTPModuleLoader(client *http.Client) lua.LGFunction {
	httpModule := gluahttp.NewHttpModule(client)

	return func(L *lua.LState) int {
		httpModule.Loader(L)

		// gluahttp's batch helper runs requests concurrently against one LState.
		// Keep the single-request API and hide the unsafe batch entrypoint.
		if mod, ok := L.Get(-1).(*lua.LTable); ok {
			mod.RawSetString("request_batch", lua.LNil)
		}

		return 1
	}
}

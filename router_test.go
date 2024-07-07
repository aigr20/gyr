package gyr_test

import (
	"bytes"
	"encoding/json"
	"gyr"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func sendRequest(router *gyr.Router, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func createPayload(object any) *bytes.Reader {
	marshaledPayload, err := json.Marshal(object)
	if err != nil {
		return nil
	}
	return bytes.NewReader(marshaledPayload)
}

func defaultTestRouter() *gyr.Router {
	router := gyr.DefaultRouter()
	router.Path("/test").Get(func(ctx *gyr.Context) {
		ctx.Response.Text("Routed")
	})

	return router
}

func TestRoutingSucceeds(t *testing.T) {
	router := defaultTestRouter()
	request, _ := http.NewRequest(http.MethodGet, "/test", nil)
	response := sendRequest(router, request)

	t.Run("Status code", func(t *testing.T) {
		if response.Result().StatusCode != http.StatusOK {
			t.FailNow()
		}
	})

	t.Run("Response content", func(t *testing.T) {
		if response.Body.String() != "Routed" {
			t.FailNow()
		}
	})
}

func TestRoutingMethodNotAllowed(t *testing.T) {
	router := defaultTestRouter()
	request, _ := http.NewRequest(http.MethodPost, "/test", nil)
	response := sendRequest(router, request)

	t.Run("Status code", func(t *testing.T) {
		if response.Result().StatusCode != http.StatusMethodNotAllowed {
			t.FailNow()
		}
	})

	t.Run("Response content", func(t *testing.T) {
		expected := "405 - Method Not Allowed"
		if response.Body.String() != expected {
			t.Logf("Expected \"%s\". Received \"%s\"\n", expected, response.Body.String())
			t.FailNow()
		}
	})
}

func TestRoutingNotFound(t *testing.T) {
	router := defaultTestRouter()
	request, _ := http.NewRequest(http.MethodGet, "/no-route-here", nil)
	response := sendRequest(router, request)

	t.Run("Status code", func(t *testing.T) {
		if response.Result().StatusCode != http.StatusNotFound {
			t.FailNow()
		}
	})

	t.Run("Response content", func(t *testing.T) {
		expected := "404 - Not Found"
		if response.Body.String() != expected {
			t.Logf("Expected \"%s\". Received \"%s\"\n", expected, response.Body.String())
			t.FailNow()
		}
	})
}

func TestGlobalMiddleware(t *testing.T) {
	router := defaultTestRouter()
	x := 0
	router.Middleware(func(ctx *gyr.Context) {
		x += 1
	})
	request, _ := http.NewRequest(http.MethodGet, "/test", nil)
	sendRequest(router, request)
	if x != 1 {
		t.Logf("Expected %v. Received %v\n", 1, x)
		t.FailNow()
	}
}

func TestRouteMiddleware(t *testing.T) {
	router := defaultTestRouter()
	x := 0
	router.Path("/middleware-path").Get(func(ctx *gyr.Context) {
		ctx.Response.Text(strconv.Itoa(x))
	}).Middleware(func(ctx *gyr.Context) {
		x += 1
	})

	request, _ := http.NewRequest(http.MethodGet, "/middleware-path", nil)
	sendRequest(router, request)
	if x != 1 {
		t.Logf("Expected %v. Received %v\n", 1, x)
		t.FailNow()
	}
}

func TestFindRootRoute(t *testing.T) {
	router := defaultTestRouter()
	expected := router.Path("/").Get(func(ctx *gyr.Context) {
		ctx.Response.Text("I'm the root")
	})
	found := router.FindRoute("/")

	if expected != found {
		t.FailNow()
	}
}

func TestFindRouteWithVariable(t *testing.T) {
	router := defaultTestRouter()
	expected := router.Path("/with-var/:v")
	found := router.FindRoute("/with-var/27")

	if expected != found {
		t.FailNow()
	}
}

func TestFindRouteWithDashValueInVariable(t *testing.T) {
	router := defaultTestRouter()
	expected := router.Path("/with-var/:v")
	found := router.FindRoute("/with-var/test-test")

	if expected != found {
		t.FailNow()
	}
}

func TestFindRouteDoesNotFindPartialMatch(t *testing.T) {
	router := defaultTestRouter()
	found := router.FindRoute("/test/test")

	if found != nil {
		t.FailNow()
	}
}

func TestRouteWithIntPathVariable(t *testing.T) {
	router := defaultTestRouter()
	router.Path("/with-var/:v").Get(func(ctx *gyr.Context) {
		v := ctx.IntVariable("v")
		ctx.Response.Text(strconv.Itoa(v))
	})

	request, _ := http.NewRequest(http.MethodGet, "/with-var/10", nil)
	response := sendRequest(router, request)
	if response.Body.String() != "10" {
		t.Logf("Expected %v. Received %s\n", 10, response.Body.String())
		t.FailNow()
	}
}

func TestRouteWithStringPathVariable(t *testing.T) {
	router := defaultTestRouter()
	router.Path("/with-var/:v").Get(func(ctx *gyr.Context) {
		v := ctx.StringVariable("v")
		ctx.Response.Text(v)
	})

	request, _ := http.NewRequest(http.MethodGet, "/with-var/10-re-nm", nil)
	response := sendRequest(router, request)
	if response.Body.String() != "10-re-nm" {
		t.Logf("Expected %s. Received %s\n", "10-re-nm", response.Body.String())
		t.FailNow()
	}
}

func TestRouteWithFloatPathVariable(t *testing.T) {
	router := defaultTestRouter()
	router.Path("/with-var/:v").Get(func(ctx *gyr.Context) {
		v := ctx.FloatVariable("v")
		ctx.Response.Text(strconv.FormatFloat(v, 'f', -1, 64))
	})

	request, _ := http.NewRequest(http.MethodGet, "/with-var/10.3", nil)
	response := sendRequest(router, request)
	if response.Body.String() != "10.3" {
		t.Logf("Expected %v. Received %s\n", 10.3, response.Body.String())
		t.FailNow()
	}
}

func TestRouteWithBoolPathVariable(t *testing.T) {
	router := defaultTestRouter()
	router.Path("/with-var/:v").Get(func(ctx *gyr.Context) {
		v := ctx.BoolVariable("v")
		ctx.Response.Text(strconv.FormatBool(v))
	})

	request, _ := http.NewRequest(http.MethodGet, "/with-var/false", nil)
	response := sendRequest(router, request)
	if response.Body.String() != "false" {
		t.Logf("Expected %v. Received %s\n", false, response.Body.String())
		t.FailNow()
	}
}

type point struct {
	X int `json:"x" xml:"x"`
	Y int `json:"y" xml:"y"`
}

func TestSendJson(t *testing.T) {
	router := defaultTestRouter()
	router.Path("/json").Post(func(ctx *gyr.Context) {
		var p point
		err := ctx.ReadBody(&p)
		if err != nil {
			ctx.Response.Error("Failed reading JSON", http.StatusInternalServerError)
			return
		}

		p.X += 1
		p.Y += 2
		ctx.Response.Json(p)
	})

	payload := createPayload(point{X: 0, Y: 0})
	request, _ := http.NewRequest(http.MethodPost, "/json", payload)
	request.Header.Set("Content-Type", "application/json")
	response := sendRequest(router, request)

	var rp point
	err := json.NewDecoder(response.Body).Decode(&rp)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	t.Run("Content-Type", func(t *testing.T) {
		expected := "application/json"
		received := response.Result().Header.Get("Content-Type")
		if received != expected {
			t.Logf("Expected %s. Received %s\n", expected, received)
			t.FailNow()
		}
	})

	t.Run("Response content", func(t *testing.T) {
		expected := point{X: 1, Y: 2}
		if rp != expected {
			t.Logf("Expected %+v. Received %+v\n.", expected, rp)
			t.FailNow()
		}
	})
}

func TestReceiveJson(t *testing.T) {
	router := defaultTestRouter()
	router.Path("/json").Post(func(ctx *gyr.Context) {
		var p point
		err := ctx.ReadBody(&p)
		if err != nil {
			ctx.Response.Error("Failed reading JSON", http.StatusInternalServerError)
			return
		}

		ctx.Response.Text("Success!")
	})

	payload := createPayload(point{X: 1, Y: 3})
	request, _ := http.NewRequest(http.MethodPost, "/json", payload)
	request.Header.Set("Content-Type", "application/json")
	response := sendRequest(router, request)

	if response.Result().StatusCode != http.StatusOK {
		t.FailNow()
	}
}

func TestResponseStatusCode(t *testing.T) {
	router := defaultTestRouter()
	router.Path("/code").Get(func(ctx *gyr.Context) {
		ctx.Response.Status(http.StatusCreated).Json(point{})
	})

	request, _ := http.NewRequest(http.MethodGet, "/code", nil)
	response := sendRequest(router, request)

	t.Run("Status Code", func(t *testing.T) {
		if response.Result().StatusCode != http.StatusCreated {
			t.Logf("Expected %v. Received %v\n", http.StatusCreated, response.Result().StatusCode)
			t.FailNow()
		}
	})

	t.Run("Content-Type Header", func(t *testing.T) {
		received := response.Result().Header.Get("Content-Type")
		if received != "application/json" {
			t.Logf("Expected %s. Received %s\n", "application/json", received)
			t.FailNow()
		}
	})
}

package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestYear_Bdgr_Metodyka_Get_Formularze(t *testing.T) {
	app := setupApplication("db/")
	defer app.DBManager.Disconnect()

	router := app.Routes()
	req := httptest.NewRequest("GET", "/app/2025/bdgr/metodyka/formularze/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	t.Logf("response status: %d", w.Code)
	t.Logf("response body: %s", w.Body.String())

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestYear_Bdgr_Metodyka_Get_NoRedirect(t *testing.T) {
	app := setupApplication("db/")
	defer app.DBManager.Disconnect()

	router := app.Routes()

	req := httptest.NewRequest("GET", "/app/2025/bdgr/metodyka/formularze", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Fail if redirect detected
	if w.Code == http.StatusMovedPermanently || w.Code == http.StatusFound {
		t.Fatalf("Unexpected redirect: status %d, Location: %s",
			w.Code, w.Header().Get("Location"))
	}

	// Expect successful response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}
}

func TestLogin_Post(t *testing.T) {
	app := setupApplication("db/")
	defer app.DBManager.Disconnect()

	form := url.Values{}
	form.Add("email", "Szymon.Piechutowski@ierigz.waw.pl")
	form.Add("password", "Password2")
	
	str := strings.NewReader(form.Encode())

	req := httptest.NewRequest(http.MethodPost, "/login", str)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()

	app.LoginPost(rr, req)

	t.Logf("Status: %d", rr.Code)
	t.Logf("Body: %s", rr.Body.String())
}
package main

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// код писать тут

type User1 struct {
	Id     int    `xml:"id"`
	Name   string `xml:"name"`
	Age    int    `xml:"age"`
	About  string `xml:"about"`
	Gender string `xml:"gender"`
	FName  string `xml:"first_name"`
	LName  string `xml:"last_name"`
}

func TestAccessToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	checkErrorValue(t,
		"Bad AccessToken",
		SearchClient{AccessToken: "12346", URL: ts.URL},
		SearchRequest{Limit: 1, Offset: 0, Query: "", OrderField: "", OrderBy: 0},
	)
}

func TestTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	checkErrorValue(t,
		"timeout for",
		SearchClient{AccessToken: "Timeout", URL: ts.URL},
		SearchRequest{Limit: 1, Offset: 0, Query: "", OrderField: "", OrderBy: 0},
	)
}

func TestUnknownError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	checkErrorValue(t,
		"unknown error",
		SearchClient{AccessToken: "12346", URL: "notaserver"},
		SearchRequest{Limit: 1, Offset: 0, Query: "", OrderField: "", OrderBy: 0},
	)
}

func TestMinLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	checkErrorValue(
		t,
		"limit must be > 0",
		SearchClient{AccessToken: "12345", URL: ts.URL},
		SearchRequest{Limit: -1, Offset: 0, Query: "", OrderField: "", OrderBy: 0},
	)
}

func TestMinOffset(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	checkErrorValue(
		t,
		"offset must be > 0",
		SearchClient{AccessToken: "12345", URL: ts.URL},
		SearchRequest{Limit: 1, Offset: -1, Query: "", OrderField: "", OrderBy: 0},
	)
}

func TestBadOrderField(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	checkErrorValue(
		t,
		"OrderFeld HAHAHA invalid",
		SearchClient{AccessToken: "12345", URL: ts.URL},
		SearchRequest{Limit: 1, Offset: 0, Query: "", OrderField: "HAHAHA", OrderBy: 0},
	)
}

func TestUnknownBadRequestError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	checkErrorValue(
		t,
		"unknown bad request error",
		SearchClient{AccessToken: "unknownBRError", URL: ts.URL},
		SearchRequest{Limit: 1, Offset: 0, Query: "", OrderField: "", OrderBy: 0},
	)
}

func TestCantUnpack(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	checkErrorValue(
		t,
		"cant unpack result json",
		SearchClient{AccessToken: "CantUnpack", URL: ts.URL},
		SearchRequest{Limit: 1, Offset: 0, Query: "", OrderField: "", OrderBy: 0},
	)
}

func TestCantUnpackError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	checkErrorValue(
		t,
		"cant unpack error json",
		SearchClient{AccessToken: "CantUnpackError", URL: ts.URL},
		SearchRequest{Limit: 1, Offset: 0, Query: "", OrderField: "", OrderBy: 0},
	)
}

func TestFatalError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	checkErrorValue(
		t,
		"SearchServer fatal error",
		SearchClient{AccessToken: "FatalError", URL: ts.URL},
		SearchRequest{Limit: 1, Offset: 0, Query: "", OrderField: "", OrderBy: 0},
	)
}

func TestMaxLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	searchClient := SearchClient{AccessToken: "12345", URL: ts.URL}
	searchRequest := SearchRequest{Limit: 30, Offset: 0, Query: "", OrderField: "", OrderBy: 0}
	result, err := searchClient.FindUsers(searchRequest)
	if err != nil {
		t.Errorf("Unexpected error %v", err.Error())
	}
	if len(result.Users) != 25 {
		t.Errorf("Expect 25 users, got %v", len(result.Users))
	}

}
func TestNoNextPage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	searchClient := SearchClient{AccessToken: "12345", URL: ts.URL}
	searchRequest := SearchRequest{Limit: 1, Offset: 34, Query: "", OrderField: "", OrderBy: 0}
	result, err := searchClient.FindUsers(searchRequest)
	if err != nil {
		t.Errorf("Unexpected error %v", err.Error())
	}
	if len(result.Users) != 1 {
		t.Errorf("Where is unexpected next page, expect 1 result, got %v", len(result.Users))
	}

}

func checkErrorValue(t *testing.T, errValue string, searchClient SearchClient, searchRequest SearchRequest) {
	_, err := searchClient.FindUsers(searchRequest)

	if err == nil {
		t.Errorf("Expect error \"%v...\", got nil.", errValue)
	}

	if ok := strings.HasPrefix(err.Error(), errValue); !ok {
		t.Errorf("Expect error \"%v...\", got %v.", errValue, err.Error())
	}
}

func SearchServer(w http.ResponseWriter, r *http.Request) {

	limit, offset, query, orderField, orderBy, accessToken := getParams(r)

	switch accessToken {
	case "unknownBRError":
		retError(w, SearchErrorResponse{"Other error"}, http.StatusBadRequest)
		return
	case "CantUnpackError":
		w.WriteHeader(http.StatusBadRequest)
		_, errr := w.Write([]byte("]"))
		if errr != nil {
			retError(w, SearchErrorResponse{errr.Error()}, http.StatusInternalServerError)
			return
		}
		return
	case "CantUnpack":
		_, errr := w.Write([]byte("]"))
		if errr != nil {
			retError(w, SearchErrorResponse{errr.Error()}, http.StatusInternalServerError)
			return
		}
		return
	case "FatalError":
		w.WriteHeader(http.StatusInternalServerError)
		return
	case "Timeout":
		time.Sleep(2 * time.Second)
		return
	}

	if accessToken != "12345" {
		retError(w, SearchErrorResponse{""}, http.StatusUnauthorized)
		return
	}

	xmlFile, err := os.Open("dataset.xml")
	if err != nil {
		retError(w, SearchErrorResponse{err.Error()}, http.StatusInternalServerError)
		return
	}
	defer xmlFile.Close()
	decoder := xml.NewDecoder(xmlFile)

	var users []User
	errr := fillUsers(&users, decoder, query)
	if (errr != SearchErrorResponse{"nil"}) {
		retError(w, errr, http.StatusInternalServerError)
		return
	}
	errr = orderUsers(&users, orderField, orderBy)
	if (errr != SearchErrorResponse{"nil"}) {
		retError(w, errr, http.StatusBadRequest)
		return
	}
	if offset > len(users) {
		offset = len(users)
	}
	if limit+offset > len(users) {
		limit = len(users) - offset
	}

	jsonUsers, err := json.Marshal(users[offset : offset+limit])
	if err != nil {
		retError(w, SearchErrorResponse{err.Error()}, http.StatusInternalServerError)
		return
	}
	_, err = w.Write(jsonUsers)
	if err != nil {
		retError(w, SearchErrorResponse{err.Error()}, http.StatusInternalServerError)
		return
	}
}

func retError(w http.ResponseWriter, errSt SearchErrorResponse, status int) {
	errJson, err := json.Marshal(errSt)
	if err != nil {
		retError(w, SearchErrorResponse{err.Error()}, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	_, err = w.Write(errJson)
	if err != nil {
		retError(w, SearchErrorResponse{err.Error()}, http.StatusInternalServerError)
		return
	}
}

func getParams(r *http.Request) (limit int, offset int, query string, orderField string, orderBy int, accessToken string) {
	limit, err := strconv.Atoi(r.FormValue("limit"))
	if err != nil {
		limit = 0
	}
	if limit < 0 {
		limit = 0
	}
	offset, err = strconv.Atoi(r.FormValue("offset"))
	if err != nil {
		offset = 0
	}
	if offset < 0 {
		offset = 0
	}
	query = r.FormValue("query")
	orderField = r.FormValue("order_field")
	orderBy, err = strconv.Atoi(r.FormValue("order_by"))
	if err != nil {
		orderBy = 0
	}

	accessToken = r.Header.Get("AccessToken")
	return
}

func orderUsers(users *[]User, orderField string, orderBy int) SearchErrorResponse {
	if orderField == "" {
		orderField = "Name"
	}

	if orderField != "Id" && orderField != "Name" && orderField != "Age" {
		return SearchErrorResponse{"ErrorBadOrderField"}
	}

	if orderBy != 0 {
		var lessFunc func(i, j int) bool
		switch orderField {
		case "Id":
			lessFunc = func(i, j int) bool {
				if orderBy < 0 {
					return (*users)[i].Id > (*users)[j].Id
				}
				return (*users)[i].Id < (*users)[j].Id
			}
		case "Age":
			lessFunc = func(i, j int) bool {
				if orderBy < 0 {
					return (*users)[i].Age > (*users)[j].Age
				}
				return (*users)[i].Age < (*users)[j].Age
			}
		case "Name":
			lessFunc = func(i, j int) bool {
				if orderBy < 0 {
					return (*users)[i].Name > (*users)[j].Name
				}
				return (*users)[i].Name < (*users)[j].Name
			}
		}
		sort.Slice(*users, lessFunc)
	}

	return SearchErrorResponse{"nil"}
}

func fillUsers(users *[]User, decoder *xml.Decoder, query string) SearchErrorResponse {
	var user = User1{}

	for {
		tok, tokenErr := decoder.Token()
		if tokenErr == io.EOF {
			break
		} else if tokenErr != nil {
			return SearchErrorResponse{tokenErr.Error()}
		}
		if tok == nil {
			break
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			if tok.Name.Local == "row" {
				if err := decoder.DecodeElement(&user, &tok); err != nil {
					return SearchErrorResponse{err.Error()}
				}
				user.Name = user.FName + " " + user.LName

				if query != "" {
					NameAbout := user.Name + user.About
					if !strings.Contains(NameAbout, query) {
						continue
					}
				}

				*users = append(*users, User{Id: user.Id, Name: user.Name, Age: user.Age, About: user.About, Gender: user.Gender})
			}
		}
	}
	return SearchErrorResponse{"nil"}
}

package sourcegraph

import (
	"net/http"
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/go-sourcegraph/router"
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/person"
	"sourcegraph.com/sourcegraph/srclib/repo"
)

func TestDefsService_Get(t *testing.T) {
	setup()
	defer teardown()

	want := &Def{Def: graph.Def{SID: 1}}

	var called bool
	mux.HandleFunc(urlPath(t, router.Def, map[string]string{"RepoURI": "r.com/x", "UnitType": "t", "Unit": "u", "Path": "p"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")
		testFormValues(t, r, values{"Doc": "true"})

		writeJSON(w, want)
	})

	repo_, _, err := client.Defs.Get(DefSpec{Repo: "r.com/x", UnitType: "t", Unit: "u", Path: "p"}, &DefGetOptions{Doc: true})
	if err != nil {
		t.Errorf("Defs.Get returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(repo_, want) {
		t.Errorf("Defs.Get returned %+v, want %+v", repo_, want)
	}
}

func TestDefsService_List(t *testing.T) {
	setup()
	defer teardown()

	want := []*Def{{Def: graph.Def{SID: 1}}}

	var called bool
	mux.HandleFunc(urlPath(t, router.Defs, nil), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")
		testFormValues(t, r, values{
			"RepositoryURI": "r1",
			"Sort":          "name",
			"UnitTypes":     "a,b",
			"Direction":     "asc",
			"Kinds":         "a,b",
			"Exported":      "true",
			"Doc":           "true",
			"PerPage":       "1",
			"Page":          "2",
		})

		writeJSON(w, want)
	})

	defs, _, err := client.Defs.List(&DefListOptions{
		RepositoryURI: "r1",
		Sort:          "name",
		UnitTypes:     []string{"a", "b"},
		Direction:     "asc",
		Kinds:         []string{"a", "b"},
		Exported:      true,
		Doc:           true,
		ListOptions:   ListOptions{PerPage: 1, Page: 2},
	})
	if err != nil {
		t.Errorf("Defs.List returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(defs, want) {
		t.Errorf("Defs.List returned %+v, want %+v", defs, want)
	}
}

func TestDefsService_ListExamples(t *testing.T) {
	setup()
	defer teardown()

	want := []*Example{{Ref: graph.Ref{File: "f"}}}

	var called bool
	mux.HandleFunc(urlPath(t, router.DefExamples, map[string]string{"RepoURI": "r.com/x", "UnitType": "t", "Unit": "u", "Path": "p"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	refs, _, err := client.Defs.ListExamples(DefSpec{Repo: "r.com/x", UnitType: "t", Unit: "u", Path: "p"}, nil)
	if err != nil {
		t.Errorf("Defs.ListExamples returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(refs, want) {
		t.Errorf("Defs.ListExamples returned %+v, want %+v", refs, want)
	}
}

func TestDefsService_ListAuthors(t *testing.T) {
	setup()
	defer teardown()

	want := []*AugmentedDefAuthor{{User: &person.User{Login: "b"}}}

	var called bool
	mux.HandleFunc(urlPath(t, router.DefAuthors, map[string]string{"RepoURI": "r.com/x", "UnitType": "t", "Unit": "u", "Path": "p"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	authors, _, err := client.Defs.ListAuthors(DefSpec{Repo: "r.com/x", UnitType: "t", Unit: "u", Path: "p"}, nil)
	if err != nil {
		t.Errorf("Defs.ListAuthors returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(authors, want) {
		t.Errorf("Defs.ListAuthors returned %+v, want %+v", authors, want)
	}
}

func TestDefsService_ListClients(t *testing.T) {
	setup()
	defer teardown()

	want := []*AugmentedDefClient{{User: &person.User{Login: "b"}}}

	var called bool
	mux.HandleFunc(urlPath(t, router.DefClients, map[string]string{"RepoURI": "r.com/x", "UnitType": "t", "Unit": "u", "Path": "p"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	clients, _, err := client.Defs.ListClients(DefSpec{Repo: "r.com/x", UnitType: "t", Unit: "u", Path: "p"}, nil)
	if err != nil {
		t.Errorf("Defs.ListClients returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(clients, want) {
		t.Errorf("Defs.ListClients returned %+v, want %+v", clients, want)
	}
}

func TestDefsService_ListDependents(t *testing.T) {
	setup()
	defer teardown()

	want := []*AugmentedDefDependent{{Repo: &repo.Repository{URI: "r2"}}}

	var called bool
	mux.HandleFunc(urlPath(t, router.DefDependents, map[string]string{"RepoURI": "r.com/x", "UnitType": "t", "Unit": "u", "Path": "p"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	dependents, _, err := client.Defs.ListDependents(DefSpec{Repo: "r.com/x", UnitType: "t", Unit: "u", Path: "p"}, nil)
	if err != nil {
		t.Errorf("Defs.ListDependents returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(dependents, want) {
		t.Errorf("Defs.ListDependents returned %+v, want %+v", dependents, want)
	}
}

func TestDefsService_ListVersions(t *testing.T) {
	setup()
	defer teardown()

	want := []*Def{{Def: graph.Def{SID: 1}}}

	var called bool
	mux.HandleFunc(urlPath(t, router.DefVersions, map[string]string{"RepoURI": "r.com/x", "UnitType": "t", "Unit": "u", "Path": "p"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	versions, _, err := client.Defs.ListVersions(DefSpec{Repo: "r.com/x", UnitType: "t", Unit: "u", Path: "p"}, nil)
	if err != nil {
		t.Errorf("Defs.ListVersions returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(versions, want) {
		t.Errorf("Defs.ListVersions returned %+v, want %+v", versions, want)
	}
}

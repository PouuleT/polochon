package token_test

import (
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/odwrtw/polochon/token"
)

var configFile = strings.NewReader(`
- role: guest
  allowed:
    - TokenGetAllowed
    - MoviesListIDs
    - ShowsListSlugs
  allownotoken: true
  token:
  - name: guest1
    value: guest1token
  - name: guest2
    value: guest2token

- role: user
  include:
    - guest
  allowed:
    - TorrentsAdd
  token:
  - name: user1
    value: user1token

- role: admin
  include:
    - user
  allowed:
    - DeleteBySlugs
  token:
  - name: admin1
    value: admin1token
`)

func createExpectedManager() *token.Manager {
	rGuest := &token.Role{
		Name:    "guest",
		Allowed: []string{"TokenGetAllowed", "MoviesListIDs", "ShowsListSlugs"},
		Include: []*token.Role{},
	}

	rUser := &token.Role{
		Name:    "user",
		Allowed: []string{"TorrentsAdd"},
		Include: []*token.Role{rGuest},
	}

	rAdmin := &token.Role{
		Name:    "admin",
		Allowed: []string{"DeleteBySlugs"},
		Include: []*token.Role{rUser},
	}

	return &token.Manager{
		Roles: []*token.Role{rGuest, rUser, rAdmin},
		Tokens: []*token.Token{
			{
				Role:  rGuest,
				Name:  "guest1",
				Value: "guest1token",
			},
			{
				Role:  rGuest,
				Name:  "guest2",
				Value: "guest2token",
			},
			{
				Role:  rUser,
				Name:  "user1",
				Value: "user1token",
			},
			{
				Role:  rAdmin,
				Name:  "admin1",
				Value: "admin1token",
			},
		},
		NoTokenRole: rGuest,
	}

}

var invalidMock = []struct {
	File     io.Reader
	Expected string
}{
	{
		File: strings.NewReader(`
- role: doubleRole
  allowed:
    - TokenGetAllowed
    - MoviesListIDs
    - ShowsListSlugs
  token:
  - name: guest2
    value: guest2token
- role: doubleRole
  allowed:
    - TokenGetAllowed
    - MoviesListIDs
    - ShowsListSlugs
  token:
  - name: guest1
    value: guest1token
    `),
		Expected: "Invalid yml, role: \"doubleRole\" already exists",
	},
	{
		File: strings.NewReader(`
- role: guest
  allowed:
    - TokenGetAllowed
    - MoviesListIDs
    - ShowsListSlugs
  token:
  - name: guest1
    value: guest1token
  - name: guest1
    value: guest2token
    `),
		Expected: "Invalid yml, token name: \"guest1\" already exists",
	},
	{
		File: strings.NewReader(`
- role: guest
  allowed:
    - TokenGetAllowed
    - MoviesListIDs
    - ShowsListSlugs
  token:
  - name: guest1
    value: guest1token
  - name: guest2
    value: guest1token
    `),
		Expected: "Invalid yml, token value: \"guest1token\" already exists",
	},
	{
		File: strings.NewReader(`
- role: guest
  allowed:
    - TokenGetAllowed
    - MoviesListIDs
    - ShowsListSlugs
  include:
    - user
  token:
  - name: guest2
    value: guest2token
    `),
		Expected: "Invalid yml, role \"user\" included but not defined",
	},
	{
		File: strings.NewReader(`
- role: role1
  allownotoken: true
  allowed:
    - TokenGetAllowed
    - MoviesListIDs
    - ShowsListSlugs
  token:
  - name: guest2
    value: guest2token
- role: role2
  allownotoken: true
  allowed:
    - TokenGetAllowed
    - MoviesListIDs
    - ShowsListSlugs
  token:
  - name: guest1
    value: guest1token
    `),
		Expected: "No token role already declared, you can't use \"role2\"",
	},
}

func TestLoadValidConfig(t *testing.T) {
	manager, err := token.LoadFromYaml(configFile)

	if err != nil {
		t.Fatal(err)
	}
	if manager == nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(manager, createExpectedManager()) {
		t.Error("Config file not correctly interpreted")
	}
}

func TestInvalidConfig(t *testing.T) {
	for _, cfg := range invalidMock {
		manager, err := token.LoadFromYaml(cfg.File)
		if manager != nil {
			t.Error("Unexpected manager")
		}
		if err.Error() != cfg.Expected {
			t.Error("Expected:", cfg.Expected, "Got:", err.Error())
		}
	}
}
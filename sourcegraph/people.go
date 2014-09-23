package sourcegraph

import (
	"strconv"
	"strings"

	"github.com/sourcegraph/go-nnz/nnz"

	"sourcegraph.com/sourcegraph/go-sourcegraph/router"
	"sourcegraph.com/sourcegraph/srclib/db_common"
	"sourcegraph.com/sourcegraph/srclib/person"
)

// PeopleService communicates with the people-related endpoints in the
// Sourcegraph API.
type PeopleService interface {
	// Get fetches a person.
	// TODO(samer): add *Type pattern for options to docs.
	Get(person PersonSpec, opt *PersonGetOptions) (*Person, Response, error)

	// GetSettings fetches a person's configuration settings. If err is nil, then the returned PersonSettings must be
	// non-nill
	GetSettings(person PersonSpec) (*PersonSettings, Response, error)

	// UpdateSettings updates an person's configuration settings.
	UpdateSettings(person PersonSpec, settings PersonSettings) (Response, error)

	// ListEmails returns a list of a person's email addresses.
	ListEmails(person PersonSpec) ([]*EmailAddr, Response, error)

	// GetOrCreateFromGitHub creates a new person based a GitHub user.
	GetOrCreateFromGitHub(user GitHubUserSpec, opt *PersonGetOptions) (*Person, Response, error)

	// RefreshProfile updates the person's profile information from external
	// sources, such as GitHub.
	//
	// This operation is performed asynchronously on the server side (after
	// receiving the request) and the API currently has no way of notifying
	// callers when the operation completes.
	RefreshProfile(personSpec PersonSpec) (Response, error)

	// ComputeStats recomputes statistics about the person.
	//
	// This operation is performed asynchronously on the server side (after
	// receiving the request) and the API currently has no way of notifying
	// callers when the operation completes.
	ComputeStats(personSpec PersonSpec) (Response, error)

	// List people.
	List(opt *PersonListOptions) ([]*person.User, Response, error)

	// ListAuthors lists people who authored code that person uses.
	ListAuthors(person PersonSpec, opt *PersonListAuthorsOptions) ([]*AugmentedPersonUsageByClient, Response, error)

	// ListClients lists people who use code that person authored.
	ListClients(person PersonSpec, opt *PersonListClientsOptions) ([]*AugmentedPersonUsageOfAuthor, Response, error)

	// ListOrgs lists organizations that a person is a member of.
	ListOrgs(member PersonSpec, opt *PersonListOrgsOptions) ([]*Org, Response, error)
}

// peopleService implements PeopleService.
type peopleService struct {
	client *Client
}

var _ PeopleService = &peopleService{}

// PersonSpec specifies a person. At least one of Email, Login, and UID must be
// nonempty.
type PersonSpec struct {
	Email string
	Login string
	UID   int
}

// PathComponent returns the URL path component that specifies the person.
func (s *PersonSpec) PathComponent() string {
	if s.Email != "" {
		return s.Email
	}
	if s.Login != "" {
		return s.Login
	}
	if s.UID > 0 {
		return "$" + strconv.Itoa(s.UID)
	}
	panic("empty PersonSpec")
}

func (s *PersonSpec) RouteVars() map[string]string {
	return map[string]string{"PersonSpec": s.PathComponent()}
}

type Person struct {
	*person.User

	Stat person.Stats `json:",omitempty"`
}

// ParsePersonSpec parses a string generated by (*PersonSpec).String() and
// returns the equivalent PersonSpec struct.
func ParsePersonSpec(pathComponent string) (PersonSpec, error) {
	if strings.HasPrefix(pathComponent, "$") {
		uid, err := strconv.Atoi(pathComponent[1:])
		return PersonSpec{UID: uid}, err
	}
	if strings.Contains(pathComponent, "@") {
		return PersonSpec{Email: pathComponent}, nil
	}
	return PersonSpec{Login: pathComponent}, nil
}

type PersonGetOptions struct {
	// Stats is whether to include statistics about the person in the response.
	Stats bool `url:",omitempty"`
}

func (s *peopleService) Get(person_ PersonSpec, opt *PersonGetOptions) (*Person, Response, error) {
	url, err := s.client.url(router.Person, person_.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var person__ *Person
	resp, err := s.client.Do(req, &person__)
	if err != nil {
		return nil, resp, err
	}

	return person__, resp, nil
}

// EmailAddr is an email address associated with a person.
type EmailAddr struct {
	Email string // the email address (case-insensitively compared in the DB and API)

	Verified bool // whether this email address has been verified

	Primary bool // indicates this is the user's primary email (only 1 email can be primary per user)

	Guessed bool // whether Sourcegraph inferred via public data that this is an email for the user

	Blacklisted bool // indicates that this email should not be associated with the user (even if guessed in the future)
}

func (s *peopleService) ListEmails(person PersonSpec) ([]*EmailAddr, Response, error) {
	url, err := s.client.url(router.PersonEmails, person.RouteVars(), nil)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var emails []*EmailAddr
	resp, err := s.client.Do(req, &emails)
	if err != nil {
		return nil, resp, err
	}

	return emails, resp, nil
}

// PersonSettings describes a user's configuration settings.
type PersonSettings struct {
	// RequestedUpgradeAt is the date on which a user requested an upgrade
	RequestedUpgradeAt db_common.NullTime `json:",omitempty"`

	PlanSettings `json:",omitempty"`
	BuildEmails  *bool `json:",omitempty"`

	PullRequestSrcbotNotification *bool `json:",omitempty"`
}

// PlanSettings describes the pricing plan that the person or org has selected.
type PlanSettings struct {
	PlanID *string `json:",omitempty"`
}

func (s *peopleService) GetSettings(person PersonSpec) (*PersonSettings, Response, error) {
	url, err := s.client.url(router.PersonSettings, person.RouteVars(), nil)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var settings *PersonSettings
	resp, err := s.client.Do(req, &settings)
	if err != nil {
		return nil, resp, err
	}

	return settings, resp, nil
}

func (s *peopleService) UpdateSettings(person PersonSpec, settings PersonSettings) (Response, error) {
	url, err := s.client.url(router.PersonSettingsUpdate, person.RouteVars(), nil)
	if err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("PUT", url.String(), settings)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req, nil)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// GitHubUserSpec specifies a GitHub user, either by GitHub login or GitHub user
// ID.
type GitHubUserSpec struct {
	Login string
	ID    int
}

func (s GitHubUserSpec) RouteVars() map[string]string {
	if s.ID != 0 {
		panic("GitHubUserSpec ID not supported via HTTP API")
	} else if s.Login != "" {
		return map[string]string{"GitHubUserSpec": s.Login}
	}
	panic("empty GitHubUserSpec")
}

func (s *peopleService) GetOrCreateFromGitHub(user GitHubUserSpec, opt *PersonGetOptions) (*Person, Response, error) {
	url, err := s.client.url(router.PersonFromGitHub, user.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var person__ *Person
	resp, err := s.client.Do(req, &person__)
	if err != nil {
		return nil, resp, err
	}

	return person__, resp, nil
}

func (s *peopleService) RefreshProfile(person_ PersonSpec) (Response, error) {
	url, err := s.client.url(router.PersonRefreshProfile, person_.RouteVars(), nil)
	if err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("PUT", url.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req, nil)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (s *peopleService) ComputeStats(person_ PersonSpec) (Response, error) {
	url, err := s.client.url(router.PersonComputeStats, person_.RouteVars(), nil)
	if err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest("PUT", url.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req, nil)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// PersonListOptions specifies options for the PeopleService.List method.
type PersonListOptions struct {
	// NameOrLogin filters the results to only those with matching logins or
	// names.
	NameOrLogin string `url:",omitempty" json:",omitempty"`

	Sort      string `url:",omitempty" json:",omitempty"`
	Direction string `url:",omitempty" json:",omitempty"`

	ListOptions
}

func (s *peopleService) List(opt *PersonListOptions) ([]*person.User, Response, error) {
	url, err := s.client.url(router.People, nil, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var people []*person.User
	resp, err := s.client.Do(req, &people)
	if err != nil {
		return nil, resp, err
	}

	return people, resp, nil
}

type PersonUsageByClient struct {
	AuthorUID   nnz.Int    `db:"author_uid"`
	AuthorEmail nnz.String `db:"author_email"`
	RefCount    int        `db:"ref_count"`
}

type AugmentedPersonUsageByClient struct {
	Author *person.User
	*PersonUsageByClient
}

// PersonListAuthorsOptions specifies options for the PeopleService.ListAuthors
// method.
type PersonListAuthorsOptions PersonListOptions

func (s *peopleService) ListAuthors(person PersonSpec, opt *PersonListAuthorsOptions) ([]*AugmentedPersonUsageByClient, Response, error) {
	url, err := s.client.url(router.PersonAuthors, person.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var people []*AugmentedPersonUsageByClient
	resp, err := s.client.Do(req, &people)
	if err != nil {
		return nil, resp, err
	}

	return people, resp, nil
}

type PersonUsageOfAuthor struct {
	ClientUID   nnz.Int    `db:"client_uid"`
	ClientEmail nnz.String `db:"client_email"`
	RefCount    int        `db:"ref_count"`
}

type AugmentedPersonUsageOfAuthor struct {
	Client *person.User
	*PersonUsageOfAuthor
}

// PersonListClientsOptions specifies options for the PeopleService.ListClients
// method.
type PersonListClientsOptions PersonListOptions

func (s *peopleService) ListClients(person PersonSpec, opt *PersonListClientsOptions) ([]*AugmentedPersonUsageOfAuthor, Response, error) {
	url, err := s.client.url(router.PersonClients, person.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var people []*AugmentedPersonUsageOfAuthor
	resp, err := s.client.Do(req, &people)
	if err != nil {
		return nil, resp, err
	}

	return people, resp, nil
}

type PersonListOrgsOptions struct {
	ListOptions
}

func (s *peopleService) ListOrgs(member PersonSpec, opt *PersonListOrgsOptions) ([]*Org, Response, error) {
	url, err := s.client.url(router.PersonOrgs, member.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var orgs []*Org
	resp, err := s.client.Do(req, &orgs)
	if err != nil {
		return nil, resp, err
	}

	return orgs, resp, nil
}

type MockPeopleService struct {
	Get_                   func(person PersonSpec, opt *PersonGetOptions) (*Person, Response, error)
	ListEmails_            func(person PersonSpec) ([]*EmailAddr, Response, error)
	GetSettings_           func(person PersonSpec) (*PersonSettings, Response, error)
	UpdateSettings_        func(person PersonSpec, settings PersonSettings) (Response, error)
	GetOrCreateFromGitHub_ func(user GitHubUserSpec, opt *PersonGetOptions) (*Person, Response, error)
	RefreshProfile_        func(personSpec PersonSpec) (Response, error)
	ComputeStats_          func(personSpec PersonSpec) (Response, error)
	List_                  func(opt *PersonListOptions) ([]*person.User, Response, error)
	ListAuthors_           func(person PersonSpec, opt *PersonListAuthorsOptions) ([]*AugmentedPersonUsageByClient, Response, error)
	ListClients_           func(person PersonSpec, opt *PersonListClientsOptions) ([]*AugmentedPersonUsageOfAuthor, Response, error)
	ListOrgs_              func(member PersonSpec, opt *PersonListOrgsOptions) ([]*Org, Response, error)
}

var _ PeopleService = MockPeopleService{}

func (s MockPeopleService) Get(person PersonSpec, opt *PersonGetOptions) (*Person, Response, error) {
	if s.Get_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.Get_(person, opt)
}

func (s MockPeopleService) ListEmails(person PersonSpec) ([]*EmailAddr, Response, error) {
	if s.ListEmails_ == nil {
		return nil, nil, nil
	}
	return s.ListEmails_(person)
}

func (s MockPeopleService) GetSettings(person PersonSpec) (*PersonSettings, Response, error) {
	if s.GetSettings_ == nil {
		return nil, nil, nil
	}
	return s.GetSettings_(person)
}

func (s MockPeopleService) UpdateSettings(person PersonSpec, settings PersonSettings) (Response, error) {
	if s.UpdateSettings_ == nil {
		return nil, nil
	}
	return s.UpdateSettings_(person, settings)
}

func (s MockPeopleService) GetOrCreateFromGitHub(user GitHubUserSpec, opt *PersonGetOptions) (*Person, Response, error) {
	if s.GetOrCreateFromGitHub_ == nil {
		return nil, nil, nil
	}
	return s.GetOrCreateFromGitHub_(user, opt)
}

func (s MockPeopleService) RefreshProfile(personSpec PersonSpec) (Response, error) {
	if s.RefreshProfile_ == nil {
		return nil, nil
	}
	return s.RefreshProfile_(personSpec)
}

func (s MockPeopleService) ComputeStats(personSpec PersonSpec) (Response, error) {
	if s.ComputeStats_ == nil {
		return nil, nil
	}
	return s.ComputeStats_(personSpec)
}

func (s MockPeopleService) List(opt *PersonListOptions) ([]*person.User, Response, error) {
	if s.List_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.List_(opt)
}

func (s MockPeopleService) ListAuthors(person PersonSpec, opt *PersonListAuthorsOptions) ([]*AugmentedPersonUsageByClient, Response, error) {
	if s.ListAuthors_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListAuthors_(person, opt)
}

func (s MockPeopleService) ListClients(person PersonSpec, opt *PersonListClientsOptions) ([]*AugmentedPersonUsageOfAuthor, Response, error) {
	if s.ListClients_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListClients_(person, opt)
}

func (s MockPeopleService) ListOrgs(member PersonSpec, opt *PersonListOrgsOptions) ([]*Org, Response, error) {
	if s.ListOrgs_ == nil {
		return nil, nil, nil
	}
	return s.ListOrgs_(member, opt)
}

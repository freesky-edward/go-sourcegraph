package sourcegraph

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/sourcegraph/go-nnz/nnz"
	"sourcegraph.com/sourcegraph/srclib/person"
)

// RID is the numeric primary key for a repository.
type RID int

// Repo is a code repository returned by the Sourcegraph API.
type Repo struct {
	// RID is the numeric primary key for a repository.
	RID RID

	// URI is a normalized identifier for this repository based on its primary
	// clone URL. E.g., "github.com/user/repo".
	URI URI

	// Name is the base name (the final path component) of the repository,
	// typically the name of the directory that the repository would be cloned
	// into. (For example, for git://example.com/foo.git, the name is "foo".)
	Name string

	// OwnerUserID is the account that owns this repository.
	OwnerUserID person.UID `db:"owner_user_id"`

	// OwnerGitHubUserID is the GitHub user ID of this repository's owner, if this
	// is a GitHub repository.
	OwnerGitHubUserID nnz.Int `db:"owner_github_user_id" json:",omitempty"`

	// Description is a brief description of the repository.
	Description string `json:",omitempty"`

	// VCS is the short name of the VCS system that this repository uses: "git"
	// or "hg".
	VCS string `db:"vcs"`

	// CloneURL is the URL used to clone the repository from its original host.
	CloneURL string `db:"clone_url"`

	// If not empty, then CloneURL redirects to ActualCloneURL
	ActualCloneURL nnz.String `db:"actual_clone_url"`

	// HomepageURL is the URL to the repository's homepage, if any.
	HomepageURL nnz.String `db:"homepage_url"`

	// DefaultBranch is the default VCS branch used (typically "master" for git
	// repositories and "default" for hg repositories).
	DefaultBranch string `db:"default_branch"`

	// Language is the primary programming language used in this repository.
	Language string

	// GitHubStars is the number of stargazers this repository has on GitHub (or
	// 0 if it is not a GitHub repository).
	GitHubStars int `db:"github_stars"`

	// GitHubID is the GitHub ID of this repository. If a GitHub repository is
	// renamed, the ID remains the same and should be used to resolve across the
	// name change.
	GitHubID nnz.Int `db:"github_id" json:",omitempty"`

	// Disabled is whether this repo should not be downloaded and processed by the worker.
	Disabled bool `json:",omitempty"`

	// Deprecated repositories are labeled as such and hidden from global search results.
	Deprecated bool

	// Fork is whether this repository is a fork.
	Fork bool

	// Mirror is whether this repository is a mirror.
	Mirror bool

	// Private is whether this repository is private.
	Private bool

	// Stat holds repository statistics. It's only filled in if Repository{Get,List}Options has Stats == true.
	Stat RepoStats `db:"-" json:",omitempty"`
}

// IsGitHubRepository returns true iff this repository is hosted on GitHub.
func (r *Repo) IsGitHubRepository() bool {
	cloneURLStr := r.GetActualCloneURL()
	if cloneURLStr == "" {
		return strings.HasPrefix(strings.ToLower(string(r.URI)), "github.com/")
	}

	cloneURL, err := url.Parse(cloneURLStr)
	if err != nil {
		return false
	}

	return strings.ToLower(cloneURL.Host) == "github.com"
}

// Returns the most direct URL used to clone the repository, following any redirects
func (r *Repo) GetActualCloneURL() string {
	if r.ActualCloneURL == "" {
		return r.CloneURL
	}
	return string(r.ActualCloneURL)
}

const (
	Git = "git"
	Hg  = "hg"
)

func MapByURI(repos []*Repo) map[URI]*Repo {
	repoMap := make(map[URI]*Repo, len(repos))
	for _, repo := range repos {
		repoMap[URI(repo.URI)] = repo
	}
	return repoMap
}

type Repositories []*Repo

func (rs Repositories) URIs() (uris []URI) {
	uris = make([]URI, len(rs))
	for i, r := range rs {
		uris[i] = r.URI
	}
	return
}

// URI identifies a repository.
type URI string

// IsGitHubRepository returns true iff this repository is hosted on GitHub.
func (u URI) IsGitHubRepository() bool {
	return strings.HasPrefix(strings.ToLower(string(u)), "github.com/")
}

// IsGoogleCodeRepository returns true iff this repository is hosted on Google
// Code (code.google.com).
func (u URI) IsGoogleCodeRepository() bool {
	return strings.HasPrefix(strings.ToLower(string(u)), "code.google.com/p/")
}

// Scan implements database/sql.Scanner.
func (u *URI) Scan(v interface{}) error {
	if v == nil {
		*u = ""
		return nil
	} else if data, ok := v.([]byte); ok {
		*u = URI(data)
		return nil
	}
	return fmt.Errorf("%T.Scan failed: %v", u, v)
}

// Value implements database/sql/driver.Valuer
func (u URI) Value() (driver.Value, error) {
	return string(u), nil
}

// URIs is a wrapper type for a slice of URIs.
type URIs []URI

// Strings returns the URIs as strings.
func (us URIs) Strings() []string {
	s := make([]string, len(us))
	for i, u := range us {
		s[i] = string(u)
	}
	return s
}

// RepoSpec returns the RepoSpec that specifies r.
func (r *Repo) RepoSpec() RepoSpec {
	return RepoSpec{URI: string(r.URI), RID: int(r.RID)}
}

// ErrRenamed is an error type that indicates that a repository was renamed from
// OldURI to NewURI.
type ErrRenamed struct {
	// OldURI is the previous repository URI.
	OldURI URI

	// NewURI is the new URI that the repository was renamed to.
	NewURI URI
}

func (e ErrRenamed) Error() string {
	return fmt.Sprintf("repository URI %q was renamed to %q; use the new name", e.OldURI, e.NewURI)
}

// ErrNotExist is an error definitively indicating that no such repository
// exists.
var ErrNotExist = errors.New("repository does not exist on external host")

// ErrForbidden is an error indicating that the repository can no longer be
// accessed due to server's refusal to serve it (possibly DMCA takedowns on
// github etc)
var ErrForbidden = errors.New("repository is unavailable")

// ErrNotPersisted is an error indicating that no such repository is persisted
// locally. The repository might exist on a remote host, but it must be
// explicitly added (it will not be implicitly added via a Get call).
var ErrNotPersisted = errors.New("repository is not persisted locally, but it might exist remotely (explicitly add it to check)")

// ErrNotPersisted is an error indicating that repository cannot be created
// without an explicit clone URL, because it has a non-standard URI. It implies
// ErrNotPersisted.
var ErrNonStandardURI = errors.New("cannot infer repository clone URL because repository host is not standard; try adding it explicitly")

type ErrRedirect struct {
	RedirectURI URI
}

func (e ErrRedirect) Error() string {
	return fmt.Sprintf("the repository requested exists at another URI (%s)", e.RedirectURI)
}

var errRedirectMsgPattern = regexp.MustCompile(`the repository requested exists at another URI \(([^\(\)]*)\)`)

func ErrRedirectFromString(msg string) *ErrRedirect {
	if match := errRedirectMsgPattern.FindStringSubmatch(msg); len(match) == 2 {
		return &ErrRedirect{URI(match[1])}
	}
	return nil
}

// IsNotPresent returns whether err is one of ErrNotExist, ErrNotPersisted, or
// ErrRedirected.
func IsNotPresent(err error) bool {
	return err == ErrNotExist || err == ErrNotPersisted
}

func IsForbidden(err error) bool {
	return err == ErrForbidden
}

// ErrNoScheme is an error indicating that a clone URL contained no scheme
// component (e.g., "http://").
var ErrNoScheme = errors.New("clone URL has no scheme")

// ExternalHostTimeout is the timeout for HTTP requests to external repository
// hosts.
var ExternalHostTimeout = time.Second * 7

// StatType is the name of a repository statistic (see below for a listing).
type RepoStatType string

// Stats holds statistics for a repository.
type RepoStats map[RepoStatType]int

const (
	// StatXRefs is the number of external references to any def defined in a
	// repository (i.e., references from other repositories). It is only
	// computed per-repository (and not per-repository-commit) because it is
	// not easy to determine which specific commit a ref references.
	RepoStatXRefs = "xrefs"

	// StatAuthors is the number of resolved people who contributed code to any
	// def defined in a repository (i.e., references from other
	// repositories). It is only computed per-repository-commit.
	RepoStatAuthors = "authors"

	// StatClients is the number of resolved people who have committed refs that
	// reference a def defined in the repository. It is only computed
	// per-repository (and not per-repository-commit) because it is not easy to
	// determine which specific commit a ref references.
	RepoStatClients = "clients"

	// StatDependencies is the number of repositories that the repository
	// depends on. It is only computed per-repository-commit.
	RepoStatDependencies = "dependencies"

	// StatDependents is the number of repositories containing refs to a def
	// defined in the repository. It is only computed per-repository (and not
	// per-repository-commit) because it is not easy to determine which specific
	// commit a ref references.
	RepoStatDependents = "dependents"

	// StatDefs is the number of defs defined in a repository commit. It
	// is only computed per-repository-commit (or else it would count 1 def
	// for each revision of the repository that we have processed).
	RepoStatDefs = "defs"

	// StatExportedDefs is the number of exported defs defined in a
	// repository commit. It is only computed per-repository-commit (or else it
	// would count 1 def for each revision of the repository that we have
	// processed).
	RepoStatExportedDefs = "exported-defs"
)

var RepoStatTypes = map[RepoStatType]struct{}{RepoStatXRefs: struct{}{}, RepoStatAuthors: struct{}{}, RepoStatClients: struct{}{}, RepoStatDependencies: struct{}{}, RepoStatDependents: struct{}{}, RepoStatDefs: struct{}{}, RepoStatExportedDefs: struct{}{}}

// Value implements database/sql/driver.Valuer.
func (x RepoStatType) Value() (driver.Value, error) {
	return string(x), nil
}

// Scan implements database/sql.Scanner.
func (x *RepoStatType) Scan(v interface{}) error {
	if data, ok := v.([]byte); ok {
		*x = RepoStatType(data)
		return nil
	}
	return fmt.Errorf("%T.Scan failed: %v", x, v)
}
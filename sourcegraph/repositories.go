package sourcegraph

import (
	"errors"
	"fmt"
	"text/template"

	"github.com/sourcegraph/go-vcs/vcs"
	"github.com/sourcegraph/vcsstore/vcsclient"
	"github.com/sqs/go-github/github"

	"strconv"
	"strings"

	"sourcegraph.com/sourcegraph/go-sourcegraph/router"
	"sourcegraph.com/sourcegraph/srclib/authorship"
	"sourcegraph.com/sourcegraph/srclib/person"
	"sourcegraph.com/sourcegraph/srclib/repo"
)

// RepositoriesService communicates with the repository-related endpoints in the
// Sourcegraph API.
type RepositoriesService interface {
	// Get fetches a repository.
	Get(repo RepositorySpec, opt *RepositoryGetOptions) (*Repository, Response, error)

	// GetOrCreate fetches a repository using Get. If no such repository exists
	// with the URI, and the URI refers to a recognized repository host (such as
	// github.com), the repository's information is fetched from the external
	// host and the repository is created.
	GetOrCreate(repo RepositorySpec, opt *RepositoryGetOptions) (*Repository, Response, error)

	// GetSettings fetches a repository's configuration settings.
	GetSettings(repo RepositorySpec) (*RepositorySettings, Response, error)

	// UpdateSettings updates a repository's configuration settings.
	UpdateSettings(repo RepositorySpec, settings RepositorySettings) (Response, error)

	// RefreshProfile updates the repository metadata for a repository, fetching
	// it from an external host if the host is recognized (such as GitHub).
	//
	// This operation is performed asynchronously on the server side (after
	// receiving the request) and the API currently has no way of notifying
	// callers when the operation completes.
	RefreshProfile(repo RepositorySpec) (Response, error)

	// RefreshVCSData updates the repository VCS (git/hg) data, fetching all new
	// commits, branches, tags, and blobs.
	//
	// This operation is performed asynchronously on the server side (after
	// receiving the request) and the API currently has no way of notifying
	// callers when the operation completes.
	RefreshVCSData(repo RepositorySpec) (Response, error)

	// ComputeStats updates the statistics about a repository.
	//
	// This operation is performed asynchronously on the server side (after
	// receiving the request) and the API currently has no way of notifying
	// callers when the operation completes.
	ComputeStats(repo RepositorySpec) (Response, error)

	// Create adds the repository at cloneURL, filling in all information about
	// the repository that can be inferred from the URL (or, for GitHub
	// repositories, fetched from the GitHub API). If a repository with the
	// specified clone URL, or the same URI, already exists, it is returned.
	Create(newRepoSpec NewRepositorySpec) (*repo.Repository, Response, error)

	// GetReadme fetches the formatted README file for a repository.
	GetReadme(repo RepositorySpec) (*vcsclient.TreeEntry, Response, error)

	// List repositories.
	List(opt *RepositoryListOptions) ([]*Repository, Response, error)

	// List commits.
	ListCommits(repo RepositorySpec2, opt *RepositoryListCommitsOptions) ([]*Commit, Response, error)

	// GetCommit gets a commit.
	GetCommit(rev RepoRevSpec, opt *RepositoryGetCommitOptions) (*Commit, Response, error)

	// CompareCommits compares two commits. The head commit is specified in opt.
	CompareCommits(base RepoRevSpec, opt *RepositoryCompareCommitsOptions) (*CommitsComparison, Response, error)

	// ListBranches lists a repository's branches.
	ListBranches(repo RepositorySpec2, opt *RepositoryListBranchesOptions) ([]*vcs.Branch, Response, error)

	// ListTags lists a repository's tags.
	ListTags(repo RepositorySpec2, opt *RepositoryListTagsOptions) ([]*vcs.Tag, Response, error)

	// ListBadges lists the available badges for repo.
	ListBadges(repo RepositorySpec) ([]*Badge, Response, error)

	// ListCounters lists the available counters for repo.
	ListCounters(repo RepositorySpec) ([]*Counter, Response, error)

	// ListAuthors lists people who have contributed (i.e., committed) code to
	// repo.
	ListAuthors(repo RepositorySpec, opt *RepositoryListAuthorsOptions) ([]*AugmentedRepoAuthor, Response, error)

	// ListClients lists people who reference defs defined in repo.
	ListClients(repo RepositorySpec, opt *RepositoryListClientsOptions) ([]*AugmentedRepoClient, Response, error)

	// ListDependents lists repositories that contain defs referenced by
	// repo.
	ListDependencies(repo RepositorySpec, opt *RepositoryListDependenciesOptions) ([]*AugmentedRepoDependency, Response, error)

	// ListDependents lists repositories that reference defs defined in repo.
	ListDependents(repo RepositorySpec, opt *RepositoryListDependentsOptions) ([]*AugmentedRepoDependent, Response, error)

	// ListByContributor lists repositories that person has contributed (i.e.,
	// committed) code to.
	ListByContributor(person PersonSpec, opt *RepositoryListByContributorOptions) ([]*AugmentedRepoContribution, Response, error)

	// ListByClient lists repositories that contain defs referenced by
	// person.
	ListByClient(person PersonSpec, opt *RepositoryListByClientOptions) ([]*AugmentedRepoUsageByClient, Response, error)

	// ListByRefdAuthor lists repositories that reference code authored by
	// person.
	ListByRefdAuthor(person PersonSpec, opt *RepositoryListByRefdAuthorOptions) ([]*AugmentedRepoUsageOfAuthor, Response, error)
}

// repositoriesService implements RepositoriesService.
type repositoriesService struct {
	client *Client
}

var _ RepositoriesService = &repositoriesService{}

// RepositorySpec specifies a repository.
type RepositorySpec struct {
	URI string

	// CommitID specifies which revision of the repository's VCS data to fetch
	// data pertaining to. If CommitID is empty, the repository's default branch
	// will be used. If CommitID is non-empty, it will be resolved as a VCS
	// revision (so it can be a commit ID, branch, tag, etc.), and data
	// pertaining to that revision will be returned.
	CommitID string
}

func (s RepositorySpec) RouteVars() map[string]string {
	m := map[string]string{"RepoURI": s.URI}
	if s.CommitID != "" {
		m["Rev"] = s.CommitID
	}
	return m
}

// RepositorySpec2 specifies a repository. We are transitioning AWAY
// from using RepositorySpec because it actually specifies a repo at a
// certain commit, not just a repo. We'll rename it to RepoRevSpec
// and rename RepositorySpec2 to RepoSpec.
type RepositorySpec2 struct {
	URI string
	RID int
}

// PathComponent returns the URL path component that specifies the person.
func (s *RepositorySpec2) PathComponent() string {
	if s.URI != "" {
		return s.URI
	}
	if s.RID > 0 {
		return "R$" + strconv.Itoa(s.RID)
	}
	panic("empty RepositorySpec2")
}

func (s RepositorySpec2) RouteVars() map[string]string {
	if s.RID != 0 {
		panic("RID not supported yet")
	}
	return map[string]string{"RepoURI": s.URI}
}

// ParseRepositorySpec2 parses a string generated by
// (*RepositorySpec2).PathComponent() and returns the equivalent
// RepositorySpec2 struct.
func ParseRepositorySpec2(pathComponent string) (RepositorySpec2, error) {
	if pathComponent == "" {
		return RepositorySpec2{}, errors.New("empty repository spec")
	}
	if strings.HasPrefix(pathComponent, "R$") {
		rid, err := strconv.Atoi(pathComponent[2:])
		return RepositorySpec2{RID: rid}, err
	}
	return RepositorySpec2{URI: pathComponent}, nil
}

// UnmarshalRepositorySpec2 marshals a map containing route variables
// generated by (*RepositorySpec2).RouteVars() and returns the
// equivalent RepositorySpec2 struct.
func UnmarshalRepositorySpec2(routeVars map[string]string) (RepositorySpec2, error) {
	return ParseRepositorySpec2(routeVars["RepoURI"])
}

// RepoRevSpec specifies a repository at a specific commit (or
// revision specifier, such as a branch, which is resolved on the
// server side to a specific commit).
type RepoRevSpec struct {
	RepositorySpec2
	Rev string
}

func (s RepoRevSpec) RouteVars() map[string]string {
	m := s.RepositorySpec2.RouteVars()
	if s.Rev != "" {
		m["Rev"] = s.Rev
	}
	return m
}

// UnmarshalRepoRevSpec marshals a map containing route variables
// generated by (*RepoRevSpec).RouteVars() and returns the equivalent
// RepoRevSpec struct.
func UnmarshalRepoRevSpec(routeVars map[string]string) (RepoRevSpec, error) {
	repoSpec, err := UnmarshalRepositorySpec2(routeVars)
	if err != nil {
		return RepoRevSpec{}, err
	}
	return RepoRevSpec{RepositorySpec2: repoSpec, Rev: routeVars["Rev"]}, nil
}

// Repository is a code repository returned by the Sourcegraph API.
type Repository struct {
	*repo.Repository

	// CommitID is the commit which the Stats, Unsupported, and
	// NoticeTitle/NoticeBody apply to. If the Repository was fetched with an
	// empty (default) or non-commit-ID rev (such as a branch name), CommitID
	// contains the resolved commit ID for that revision specifier.
	//
	// If CommitID is empty, it means that either the revision in
	// RepositorySpec.CommitID could not be resolved to a commit ID, or the
	// repository's VCS has not been cloned to Sourcegraph yet.
	//
	// This field is only populated in the Get method's results, and even then
	// only if the RepositoryGetOptions.ResolveRevision or
	// RepositoryGetOptions.Build field is true.
	CommitID string

	// NoVCSData is true if the repository has NOT been cloned and no local copy
	// resides on the Sourcegraph servers. If it has been cloned successfully,
	// HasVCSData is false.
	//
	// If the caller specified no options to Get that require fetching VCS data
	// (e.g., ResolveRevision), NoVCSData will be false (its value is unknowable
	// without fetching VCS data).
	NoVCSData bool `json:",omitempty"`

	// Stat is a map of the statistics for the repository. It is only populated
	// if the options Stats field is true.
	Stat repo.Stats `json:",omitempty"`

	// Unsupported is whether Sourcegraph doesn't support this repository.
	Unsupported bool `json:",omitempty"`

	NoticeTitle, NoticeBody string `json:",omitempty"`
}

// Spec returns the RepositorySpec that specifies r.
func (r *Repository) Spec() RepositorySpec {
	return RepositorySpec{
		URI:      string(r.Repository.URI),
		CommitID: r.CommitID,
	}
}

type RepositoryGetOptions struct {
	// Stats is whether to include statistics about the repository in the
	// response.
	Stats bool `url:",omitempty"`

	// ResolveRevision is whether to include the resolved VCS revision in the
	// CommitID field in the response.
	ResolveRevision bool `url:",omitempty"`
}

func (s *repositoriesService) Get(repo RepositorySpec, opt *RepositoryGetOptions) (*Repository, Response, error) {
	url, err := s.client.url(router.Repository, repo.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var repo_ *Repository
	resp, err := s.client.Do(req, &repo_)
	if err != nil {
		return nil, resp, err
	}

	return repo_, resp, nil
}

func (s *repositoriesService) GetOrCreate(repo_ RepositorySpec, opt *RepositoryGetOptions) (*Repository, Response, error) {
	url, err := s.client.url(router.RepositoriesGetOrCreate, repo_.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("PUT", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var repo__ *Repository
	resp, err := s.client.Do(req, &repo__)
	if err != nil {
		return nil, resp, err
	}

	return repo__, resp, nil
}

// RepositorySettings describes a repository's configuration settings.
type RepositorySettings struct {
	Enabled *bool `json:",omitempty"`
}

func (s *repositoriesService) GetSettings(repo RepositorySpec) (*RepositorySettings, Response, error) {
	url, err := s.client.url(router.RepositorySettings, repo.RouteVars(), nil)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var settings *RepositorySettings
	resp, err := s.client.Do(req, &settings)
	if err != nil {
		return nil, resp, err
	}

	return settings, resp, nil
}

func (s *repositoriesService) UpdateSettings(repo RepositorySpec, settings RepositorySettings) (Response, error) {
	url, err := s.client.url(router.RepositorySettingsUpdate, repo.RouteVars(), nil)
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

func (s *repositoriesService) RefreshProfile(repo RepositorySpec) (Response, error) {
	url, err := s.client.url(router.RepositoryRefreshProfile, repo.RouteVars(), nil)
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

func (s *repositoriesService) RefreshVCSData(repo RepositorySpec) (Response, error) {
	url, err := s.client.url(router.RepositoryRefreshVCSData, repo.RouteVars(), nil)
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

func (s *repositoriesService) ComputeStats(repo RepositorySpec) (Response, error) {
	url, err := s.client.url(router.RepositoryComputeStats, repo.RouteVars(), nil)
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

type NewRepositorySpec struct {
	Type        repo.VCS
	CloneURLStr string `json:"CloneURL"`
}

func (s *repositoriesService) Create(newRepoSpec NewRepositorySpec) (*repo.Repository, Response, error) {
	url, err := s.client.url(router.RepositoriesCreate, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("POST", url.String(), newRepoSpec)
	if err != nil {
		return nil, nil, err
	}

	var repo_ *repo.Repository
	resp, err := s.client.Do(req, &repo_)
	if err != nil {
		return nil, resp, err
	}

	return repo_, resp, nil
}

func (s *repositoriesService) GetReadme(repo RepositorySpec) (*vcsclient.TreeEntry, Response, error) {
	url, err := s.client.url(router.RepositoryReadme, repo.RouteVars(), nil)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var readme *vcsclient.TreeEntry
	resp, err := s.client.Do(req, &readme)
	if err != nil {
		return nil, resp, err
	}

	return readme, resp, nil
}

type RepositoryListOptions struct {
	Name string `url:",omitempty" json:",omitempty"`

	// Specifies a search query for repositories. If specified, then the Sort and Direction options are ignored
	Query string `url:",omitempty" json:",omitempty"`

	URIs []string `url:",comma,omitempty" json:",omitempty"`

	BuiltOnly bool `url:",omitempty" json:",omitempty"`

	Sort      string `url:",omitempty" json:",omitempty"`
	Direction string `url:",omitempty" json:",omitempty"`

	NoFork bool `url:",omitempty" json:",omitempty"`

	Owner string `url:",omitempty" json:",omitempty"`

	ListOptions
}

func (s *repositoriesService) List(opt *RepositoryListOptions) ([]*Repository, Response, error) {
	url, err := s.client.url(router.Repositories, nil, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var repos []*Repository
	resp, err := s.client.Do(req, &repos)
	if err != nil {
		return nil, resp, err
	}

	return repos, resp, nil
}

type Commit struct {
	*vcs.Commit
}

type RepositoryListCommitsOptions struct {
	Head string `url:",omitempty" json:",omitempty"`
	ListOptions
}

func (s *repositoriesService) ListCommits(repo RepositorySpec2, opt *RepositoryListCommitsOptions) ([]*Commit, Response, error) {
	url, err := s.client.url(router.RepoCommits, repo.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var commits []*Commit
	resp, err := s.client.Do(req, &commits)
	if err != nil {
		return nil, resp, err
	}

	return commits, resp, nil
}

type RepositoryGetCommitOptions struct {
}

func (s *repositoriesService) GetCommit(rev RepoRevSpec, opt *RepositoryGetCommitOptions) (*Commit, Response, error) {
	url, err := s.client.url(router.RepoCommit, rev.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var commit *Commit
	resp, err := s.client.Do(req, &commit)
	if err != nil {
		return nil, resp, err
	}

	return commit, resp, nil
}

type CommitsComparison struct {
	Head, Base *Commit

	DefsAdded   []*Def
	DefsChanged []*Def
	DefsRemoved []*Def

	*github.CommitsComparison

	// TODO(x): add affected dependencies, dependents, users, authors
	// TODO(x): add new/fixed warnings
	// TODO(x): add file diffs
}

type RepositoryCompareCommitsOptions struct {
	HeadRev string `url:",omitempty" json:",omitempty"`
}

func (s *repositoriesService) CompareCommits(base RepoRevSpec, opt *RepositoryCompareCommitsOptions) (*CommitsComparison, Response, error) {
	url, err := s.client.url(router.RepoCompareCommits, base.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var cmp *CommitsComparison
	resp, err := s.client.Do(req, &cmp)
	if err != nil {
		return nil, resp, err
	}

	return cmp, resp, nil
}

type RepositoryListBranchesOptions struct {
	ListOptions
}

func (s *repositoriesService) ListBranches(repo RepositorySpec2, opt *RepositoryListBranchesOptions) ([]*vcs.Branch, Response, error) {
	url, err := s.client.url(router.RepoBranches, repo.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var branches []*vcs.Branch
	resp, err := s.client.Do(req, &branches)
	if err != nil {
		return nil, resp, err
	}

	return branches, resp, nil
}

type RepositoryListTagsOptions struct {
	ListOptions
}

func (s *repositoriesService) ListTags(repo RepositorySpec2, opt *RepositoryListTagsOptions) ([]*vcs.Tag, Response, error) {
	url, err := s.client.url(router.RepoTags, repo.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var tags []*vcs.Tag
	resp, err := s.client.Do(req, &tags)
	if err != nil {
		return nil, resp, err
	}

	return tags, resp, nil
}

type Badge struct {
	Name              string
	Description       string
	ImageURL          string
	UncountedImageURL string
	Markdown          string
}

func (b *Badge) HTML() string {
	return fmt.Sprintf(`<img src="%s" alt="%s">`, template.HTMLEscapeString(b.ImageURL), template.HTMLEscapeString(b.Name))
}

func (s *repositoriesService) ListBadges(repo RepositorySpec) ([]*Badge, Response, error) {
	url, err := s.client.url(router.RepositoryBadges, repo.RouteVars(), nil)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var badges []*Badge
	resp, err := s.client.Do(req, &badges)
	if err != nil {
		return nil, resp, err
	}

	return badges, resp, nil
}

type Counter struct {
	Name              string
	Description       string
	ImageURL          string
	UncountedImageURL string
	Markdown          string
}

func (c *Counter) HTML() string {
	return fmt.Sprintf(`<img src="%s" alt="%s">`, template.HTMLEscapeString(c.ImageURL), template.HTMLEscapeString(c.Name))
}

func (s *repositoriesService) ListCounters(repo RepositorySpec) ([]*Counter, Response, error) {
	url, err := s.client.url(router.RepositoryCounters, repo.RouteVars(), nil)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var counters []*Counter
	resp, err := s.client.Do(req, &counters)
	if err != nil {
		return nil, resp, err
	}

	return counters, resp, nil
}

// AugmentedRepoAuthor is a rel.RepoAuthor with the full person.User and
// graph.Def structs embedded.
type AugmentedRepoAuthor struct {
	User *person.User
	*authorship.RepoAuthor
}

type RepositoryListAuthorsOptions struct {
	ListOptions
}

func (s *repositoriesService) ListAuthors(repo RepositorySpec, opt *RepositoryListAuthorsOptions) ([]*AugmentedRepoAuthor, Response, error) {
	url, err := s.client.url(router.RepositoryAuthors, repo.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var authors []*AugmentedRepoAuthor
	resp, err := s.client.Do(req, &authors)
	if err != nil {
		return nil, resp, err
	}

	return authors, resp, nil
}

// AugmentedRepoClient is a rel.RepoClient with the full person.User and
// graph.Def structs embedded.
type AugmentedRepoClient struct {
	User *person.User
	*authorship.RepoClient
}

type RepositoryListClientsOptions struct {
	ListOptions
}

func (s *repositoriesService) ListClients(repo RepositorySpec, opt *RepositoryListClientsOptions) ([]*AugmentedRepoClient, Response, error) {
	url, err := s.client.url(router.RepositoryClients, repo.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var clients []*AugmentedRepoClient
	resp, err := s.client.Do(req, &clients)
	if err != nil {
		return nil, resp, err
	}

	return clients, resp, nil
}

type RepoDependency struct {
	ToRepo repo.URI `db:"to_repo"`
}

type AugmentedRepoDependency struct {
	Repo *repo.Repository
	*RepoDependency
}

type RepositoryListDependenciesOptions struct {
	ListOptions
}

func (s *repositoriesService) ListDependencies(repo RepositorySpec, opt *RepositoryListDependenciesOptions) ([]*AugmentedRepoDependency, Response, error) {
	url, err := s.client.url(router.RepositoryDependencies, repo.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var dependencies []*AugmentedRepoDependency
	resp, err := s.client.Do(req, &dependencies)
	if err != nil {
		return nil, resp, err
	}

	return dependencies, resp, nil
}

type RepoDependent struct {
	FromRepo repo.URI `db:"from_repo"`
}

type AugmentedRepoDependent struct {
	Repo *repo.Repository
	*RepoDependent
}

type RepositoryListDependentsOptions struct{ ListOptions }

func (s *repositoriesService) ListDependents(repo RepositorySpec, opt *RepositoryListDependentsOptions) ([]*AugmentedRepoDependent, Response, error) {
	url, err := s.client.url(router.RepositoryDependents, repo.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var dependents []*AugmentedRepoDependent
	resp, err := s.client.Do(req, &dependents)
	if err != nil {
		return nil, resp, err
	}

	return dependents, resp, nil
}

type AugmentedRepoContribution struct {
	Repo *repo.Repository
	*authorship.RepoContribution
}

type RepositoryListByContributorOptions struct {
	NoFork bool
	ListOptions
}

func (s *repositoriesService) ListByContributor(person PersonSpec, opt *RepositoryListByContributorOptions) ([]*AugmentedRepoContribution, Response, error) {
	url, err := s.client.url(router.PersonRepositoryContributions, person.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var repos []*AugmentedRepoContribution
	resp, err := s.client.Do(req, &repos)
	if err != nil {
		return nil, resp, err
	}

	return repos, resp, nil
}

// AugmentedRepoUsageByClient is a authorship.RepoUsageByClient with the full repo.Repository
// struct embedded.
type AugmentedRepoUsageByClient struct {
	DefRepo                       *repo.Repository
	*authorship.RepoUsageByClient `json:"RepoUsageByClient"`
}

type RepositoryListByClientOptions struct {
	ListOptions
}

func (s *repositoriesService) ListByClient(person PersonSpec, opt *RepositoryListByClientOptions) ([]*AugmentedRepoUsageByClient, Response, error) {
	url, err := s.client.url(router.PersonRepositoryDependencies, person.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var repos []*AugmentedRepoUsageByClient
	resp, err := s.client.Do(req, &repos)
	if err != nil {
		return nil, resp, err
	}

	return repos, resp, nil
}

// AugmentedRepoUsageOfAuthor is a authorship.RepoUsageOfAuthor with the full
// repo.Repository struct embedded.
type AugmentedRepoUsageOfAuthor struct {
	Repo                          *repo.Repository
	*authorship.RepoUsageOfAuthor `json:"RepoUsageOfAuthor"`
}

type RepositoryListByRefdAuthorOptions struct {
	ListOptions
}

func (s *repositoriesService) ListByRefdAuthor(person PersonSpec, opt *RepositoryListByRefdAuthorOptions) ([]*AugmentedRepoUsageOfAuthor, Response, error) {
	url, err := s.client.url(router.PersonRepositoryDependents, person.RouteVars(), opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var repos []*AugmentedRepoUsageOfAuthor
	resp, err := s.client.Do(req, &repos)
	if err != nil {
		return nil, resp, err
	}

	return repos, resp, nil
}

type MockRepositoriesService struct {
	Get_               func(spec RepositorySpec, opt *RepositoryGetOptions) (*Repository, Response, error)
	GetOrCreate_       func(repo RepositorySpec, opt *RepositoryGetOptions) (*Repository, Response, error)
	GetSettings_       func(repo RepositorySpec) (*RepositorySettings, Response, error)
	UpdateSettings_    func(repo RepositorySpec, settings RepositorySettings) (Response, error)
	RefreshProfile_    func(repo RepositorySpec) (Response, error)
	RefreshVCSData_    func(repo RepositorySpec) (Response, error)
	ComputeStats_      func(repo RepositorySpec) (Response, error)
	Create_            func(newRepoSpec NewRepositorySpec) (*repo.Repository, Response, error)
	GetReadme_         func(repo RepositorySpec) (*vcsclient.TreeEntry, Response, error)
	List_              func(opt *RepositoryListOptions) ([]*Repository, Response, error)
	ListCommits_       func(repo RepositorySpec2, opt *RepositoryListCommitsOptions) ([]*Commit, Response, error)
	GetCommit_         func(rev RepoRevSpec, opt *RepositoryGetCommitOptions) (*Commit, Response, error)
	CompareCommits_    func(base RepoRevSpec, opt *RepositoryCompareCommitsOptions) (*CommitsComparison, Response, error)
	ListBranches_      func(repo RepositorySpec2, opt *RepositoryListBranchesOptions) ([]*vcs.Branch, Response, error)
	ListTags_          func(repo RepositorySpec2, opt *RepositoryListTagsOptions) ([]*vcs.Tag, Response, error)
	ListBadges_        func(repo RepositorySpec) ([]*Badge, Response, error)
	ListCounters_      func(repo RepositorySpec) ([]*Counter, Response, error)
	ListAuthors_       func(repo RepositorySpec, opt *RepositoryListAuthorsOptions) ([]*AugmentedRepoAuthor, Response, error)
	ListClients_       func(repo RepositorySpec, opt *RepositoryListClientsOptions) ([]*AugmentedRepoClient, Response, error)
	ListDependencies_  func(repo RepositorySpec, opt *RepositoryListDependenciesOptions) ([]*AugmentedRepoDependency, Response, error)
	ListDependents_    func(repo RepositorySpec, opt *RepositoryListDependentsOptions) ([]*AugmentedRepoDependent, Response, error)
	ListByContributor_ func(person PersonSpec, opt *RepositoryListByContributorOptions) ([]*AugmentedRepoContribution, Response, error)
	ListByClient_      func(person PersonSpec, opt *RepositoryListByClientOptions) ([]*AugmentedRepoUsageByClient, Response, error)
	ListByRefdAuthor_  func(person PersonSpec, opt *RepositoryListByRefdAuthorOptions) ([]*AugmentedRepoUsageOfAuthor, Response, error)
}

var _ RepositoriesService = MockRepositoriesService{}

func (s MockRepositoriesService) Get(repo RepositorySpec, opt *RepositoryGetOptions) (*Repository, Response, error) {
	if s.Get_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.Get_(repo, opt)
}

func (s MockRepositoriesService) GetOrCreate(repo RepositorySpec, opt *RepositoryGetOptions) (*Repository, Response, error) {
	if s.GetOrCreate_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.GetOrCreate_(repo, opt)
}

func (s MockRepositoriesService) GetSettings(repo RepositorySpec) (*RepositorySettings, Response, error) {
	if s.GetSettings_ == nil {
		return nil, nil, nil
	}
	return s.GetSettings_(repo)
}

func (s MockRepositoriesService) UpdateSettings(repo RepositorySpec, settings RepositorySettings) (Response, error) {
	if s.UpdateSettings_ == nil {
		return nil, nil
	}
	return s.UpdateSettings_(repo, settings)
}

func (s MockRepositoriesService) RefreshProfile(repo RepositorySpec) (Response, error) {
	if s.RefreshProfile_ == nil {
		return nil, nil
	}
	return s.RefreshProfile_(repo)
}

func (s MockRepositoriesService) RefreshVCSData(repo RepositorySpec) (Response, error) {
	if s.RefreshVCSData_ == nil {
		return nil, nil
	}
	return s.RefreshVCSData_(repo)
}

func (s MockRepositoriesService) ComputeStats(repo RepositorySpec) (Response, error) {
	if s.ComputeStats_ == nil {
		return nil, nil
	}
	return s.ComputeStats_(repo)
}

func (s MockRepositoriesService) Create(newRepoSpec NewRepositorySpec) (*repo.Repository, Response, error) {
	if s.Create_ == nil {
		return nil, nil, nil
	}
	return s.Create_(newRepoSpec)
}

func (s MockRepositoriesService) GetReadme(repo RepositorySpec) (*vcsclient.TreeEntry, Response, error) {
	if s.GetReadme_ == nil {
		return nil, nil, nil
	}
	return s.GetReadme_(repo)
}

func (s MockRepositoriesService) List(opt *RepositoryListOptions) ([]*Repository, Response, error) {
	if s.List_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.List_(opt)
}

func (s MockRepositoriesService) ListBadges(repo RepositorySpec) ([]*Badge, Response, error) {
	if s.ListBadges_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListBadges_(repo)
}

func (s MockRepositoriesService) ListCounters(repo RepositorySpec) ([]*Counter, Response, error) {
	if s.ListCounters_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListCounters_(repo)
}

func (s MockRepositoriesService) ListAuthors(repo RepositorySpec, opt *RepositoryListAuthorsOptions) ([]*AugmentedRepoAuthor, Response, error) {
	if s.ListAuthors_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListAuthors_(repo, opt)
}

func (s MockRepositoriesService) ListClients(repo RepositorySpec, opt *RepositoryListClientsOptions) ([]*AugmentedRepoClient, Response, error) {
	if s.ListClients_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListClients_(repo, opt)
}

func (s MockRepositoriesService) ListDependencies(repo RepositorySpec, opt *RepositoryListDependenciesOptions) ([]*AugmentedRepoDependency, Response, error) {
	if s.ListDependencies_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListDependencies_(repo, opt)
}

func (s MockRepositoriesService) ListDependents(repo RepositorySpec, opt *RepositoryListDependentsOptions) ([]*AugmentedRepoDependent, Response, error) {
	if s.ListDependents_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListDependents_(repo, opt)
}

func (s MockRepositoriesService) ListByContributor(person PersonSpec, opt *RepositoryListByContributorOptions) ([]*AugmentedRepoContribution, Response, error) {
	if s.ListByContributor_ == nil {
		return nil, nil, nil
	}
	return s.ListByContributor_(person, opt)
}

func (s MockRepositoriesService) ListByClient(person PersonSpec, opt *RepositoryListByClientOptions) ([]*AugmentedRepoUsageByClient, Response, error) {
	if s.ListByClient_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListByClient_(person, opt)
}

func (s MockRepositoriesService) ListByRefdAuthor(person PersonSpec, opt *RepositoryListByRefdAuthorOptions) ([]*AugmentedRepoUsageOfAuthor, Response, error) {
	if s.ListByRefdAuthor_ == nil {
		return nil, &HTTPResponse{}, nil
	}
	return s.ListByRefdAuthor_(person, opt)
}

func (s MockRepositoriesService) ListCommits(repo RepositorySpec2, opt *RepositoryListCommitsOptions) ([]*Commit, Response, error) {
	if s.ListCommits_ == nil {
		return nil, nil, nil
	}
	return s.ListCommits_(repo, opt)
}

func (s MockRepositoriesService) GetCommit(rev RepoRevSpec, opt *RepositoryGetCommitOptions) (*Commit, Response, error) {
	if s.GetCommit_ == nil {
		return nil, nil, nil
	}
	return s.GetCommit_(rev, opt)
}

func (s MockRepositoriesService) CompareCommits(base RepoRevSpec, opt *RepositoryCompareCommitsOptions) (*CommitsComparison, Response, error) {
	if s.CompareCommits_ == nil {
		return nil, nil, nil
	}
	return s.CompareCommits_(base, opt)
}

func (s MockRepositoriesService) ListBranches(repo RepositorySpec2, opt *RepositoryListBranchesOptions) ([]*vcs.Branch, Response, error) {
	if s.ListBranches_ == nil {
		return nil, nil, nil
	}
	return s.ListBranches_(repo, opt)
}

func (s MockRepositoriesService) ListTags(repo RepositorySpec2, opt *RepositoryListTagsOptions) ([]*vcs.Tag, Response, error) {
	if s.ListTags_ == nil {
		return nil, nil, nil
	}
	return s.ListTags_(repo, opt)
}

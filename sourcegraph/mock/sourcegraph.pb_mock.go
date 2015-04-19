// generated by gen-mocks; DO NOT EDIT

package mock

import (
	"golang.org/x/net/context"
	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
)

type RepoGoodiesServer struct {
	ListBadges_   func(v0 context.Context, v1 *sourcegraph.RepoSpec) (*sourcegraph.BadgeList, error)
	ListCounters_ func(v0 context.Context, v1 *sourcegraph.RepoSpec) (*sourcegraph.CounterList, error)
}

func (s *RepoGoodiesServer) ListBadges(v0 context.Context, v1 *sourcegraph.RepoSpec) (*sourcegraph.BadgeList, error) {
	return s.ListBadges_(v0, v1)
}

func (s *RepoGoodiesServer) ListCounters(v0 context.Context, v1 *sourcegraph.RepoSpec) (*sourcegraph.CounterList, error) {
	return s.ListCounters_(v0, v1)
}

var _ sourcegraph.RepoGoodiesServer = (*RepoGoodiesServer)(nil)

type RepoStatusesServer struct {
	Create_      func(v0 context.Context, v1 *sourcegraph.RepoStatusesCreateOp) (*sourcegraph.RepoStatus, error)
	GetCombined_ func(v0 context.Context, v1 *sourcegraph.RepoRevSpec) (*sourcegraph.CombinedStatus, error)
}

func (s *RepoStatusesServer) Create(v0 context.Context, v1 *sourcegraph.RepoStatusesCreateOp) (*sourcegraph.RepoStatus, error) {
	return s.Create_(v0, v1)
}

func (s *RepoStatusesServer) GetCombined(v0 context.Context, v1 *sourcegraph.RepoRevSpec) (*sourcegraph.CombinedStatus, error) {
	return s.GetCombined_(v0, v1)
}

var _ sourcegraph.RepoStatusesServer = (*RepoStatusesServer)(nil)

type ReposServer struct {
	Get_       func(v0 context.Context, v1 *sourcegraph.RepoSpec) (*sourcegraph.Repo, error)
	List_      func(v0 context.Context, v1 *sourcegraph.RepoListOptions) (*sourcegraph.RepoList, error)
	GetReadme_ func(v0 context.Context, v1 *sourcegraph.RepoRevSpec) (*sourcegraph.Readme, error)
	Enable_    func(v0 context.Context, v1 *sourcegraph.RepoSpec) (*sourcegraph.Void, error)
	Disable_   func(v0 context.Context, v1 *sourcegraph.RepoSpec) (*sourcegraph.Void, error)
}

func (s *ReposServer) Get(v0 context.Context, v1 *sourcegraph.RepoSpec) (*sourcegraph.Repo, error) {
	return s.Get_(v0, v1)
}

func (s *ReposServer) List(v0 context.Context, v1 *sourcegraph.RepoListOptions) (*sourcegraph.RepoList, error) {
	return s.List_(v0, v1)
}

func (s *ReposServer) GetReadme(v0 context.Context, v1 *sourcegraph.RepoRevSpec) (*sourcegraph.Readme, error) {
	return s.GetReadme_(v0, v1)
}

func (s *ReposServer) Enable(v0 context.Context, v1 *sourcegraph.RepoSpec) (*sourcegraph.Void, error) {
	return s.Enable_(v0, v1)
}

func (s *ReposServer) Disable(v0 context.Context, v1 *sourcegraph.RepoSpec) (*sourcegraph.Void, error) {
	return s.Disable_(v0, v1)
}

var _ sourcegraph.ReposServer = (*ReposServer)(nil)

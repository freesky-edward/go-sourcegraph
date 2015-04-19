// generated by gen-mocks; DO NOT EDIT

package mock

import "sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"

type RepoStatusService struct {
	Create_      func(spec sourcegraph.RepoRevSpec, st sourcegraph.RepoStatus) (*sourcegraph.RepoStatus, error)
	GetCombined_ func(spec sourcegraph.RepoRevSpec) (*sourcegraph.CombinedStatus, error)
}

func (s *RepoStatusService) Create(spec sourcegraph.RepoRevSpec, st sourcegraph.RepoStatus) (*sourcegraph.RepoStatus, error) {
	return s.Create_(spec, st)
}

func (s *RepoStatusService) GetCombined(spec sourcegraph.RepoRevSpec) (*sourcegraph.CombinedStatus, error) {
	return s.GetCombined_(spec)
}

var _ sourcegraph.RepoStatusService = (*RepoStatusService)(nil)
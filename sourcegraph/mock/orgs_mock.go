// generated by gen-mocks; DO NOT EDIT

package mock

import "sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"

type OrgsService struct {
	Get_            func(org sourcegraph.OrgSpec) (*sourcegraph.Org, sourcegraph.Response, error)
	ListMembers_    func(org sourcegraph.OrgSpec, opt *sourcegraph.OrgListMembersOptions) ([]*sourcegraph.User, sourcegraph.Response, error)
	GetSettings_    func(org sourcegraph.OrgSpec) (*sourcegraph.OrgSettings, sourcegraph.Response, error)
	UpdateSettings_ func(org sourcegraph.OrgSpec, settings sourcegraph.OrgSettings) (sourcegraph.Response, error)
}

func (s *OrgsService) Get(org sourcegraph.OrgSpec) (*sourcegraph.Org, sourcegraph.Response, error) {
	return s.Get_(org)
}

func (s *OrgsService) ListMembers(org sourcegraph.OrgSpec, opt *sourcegraph.OrgListMembersOptions) ([]*sourcegraph.User, sourcegraph.Response, error) {
	return s.ListMembers_(org, opt)
}

func (s *OrgsService) GetSettings(org sourcegraph.OrgSpec) (*sourcegraph.OrgSettings, sourcegraph.Response, error) {
	return s.GetSettings_(org)
}

func (s *OrgsService) UpdateSettings(org sourcegraph.OrgSpec, settings sourcegraph.OrgSettings) (sourcegraph.Response, error) {
	return s.UpdateSettings_(org, settings)
}

var _ sourcegraph.OrgsService = (*OrgsService)(nil)

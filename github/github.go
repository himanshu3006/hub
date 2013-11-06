package github

import (
	"fmt"
	"github.com/jingweno/octokat"
	"github.com/octokit/go-octokit"
)

const (
	GitHubHost  string = "github.com"
	OAuthAppURL string = "http://owenou.com/gh"
)

type GitHub struct {
	Project *Project
	Config  *Config
}

func (gh *GitHub) PullRequest(id string) (pr *octokit.PullRequest, err error) {
	client := gh.octokit()
	prService, err := client.PullRequests(&octokit.PullRequestsURL, octokit.M{"owner": gh.Project.Owner, "repo": gh.Project.Name, "number": id})
	if err != nil {
		return
	}

	pr, result := prService.Get()
	if result.HasError() {
		err = result.Err
	}

	return
}

func (gh *GitHub) CreatePullRequest(base, head, title, body string) (url string, err error) {
	client := gh.octokit()
	prService, err := client.PullRequests(&octokit.PullRequestsURL, octokit.M{"owner": gh.Project.Owner, "repo": gh.Project.Name})
	if err != nil {
		return
	}

	params := octokit.PullRequestParams{Base: base, Head: head, Title: title, Body: body}
	pr, result := prService.Create(params)
	if result.HasError() {
		err = result.Err
	}

	url = pr.HTMLURL

	return
}

func (gh *GitHub) CreatePullRequestForIssue(base, head, issue string) (url string, err error) {
	client := gh.octokit()
	prService, err := client.PullRequests(&octokit.PullRequestsURL, octokit.M{"owner": gh.Project.Owner, "repo": gh.Project.Name})
	if err != nil {
		return
	}

	params := octokit.PullRequestForIssueParams{Base: base, Head: head, Issue: issue}
	pr, result := prService.Create(params)
	if result.HasError() {
		err = result.Err
	}

	url = pr.HTMLURL

	return
}

func (gh *GitHub) Repository(project Project) (repo *octokit.Repository, err error) {
	client := gh.octokit()
	repoService, err := client.Repositories(&octokit.RepositoryURL, octokit.M{"owner": project.Owner, "repo": project.Name})
	if err != nil {
		return
	}

	repo, result := repoService.Get()
	if result.HasError() {
		err = result.Err
	}

	return
}

// TODO: detach GitHub from Project
func (gh *GitHub) IsRepositoryExist(project Project) bool {
	repo, err := gh.Repository(project)

	return err == nil && repo != nil
}

func (gh *GitHub) CreateRepository(project Project, description, homepage string, isPrivate bool) (repo *octokit.Repository, err error) {
	var repoURL octokit.Hyperlink
	if project.Owner != gh.Config.FetchUser() {
		repoURL = octokit.OrgRepositoriesURL
	} else {
		repoURL = octokit.UserRepositoriesURL
	}

	client := gh.octokit()
	repoService, err := client.Repositories(&repoURL, octokit.M{"org": project.Owner})
	if err != nil {
		return
	}

	params := octokat.Repository{Name: project.Name, Description: description, Homepage: homepage, Private: isPrivate}
	repo, result := repoService.Create(params)
	if result.HasError() {
		err = result.Err
	}

	return
}

func (gh *GitHub) Releases() ([]octokat.Release, error) {
	client := gh.client()
	releases, err := client.Releases(gh.repo(), nil)
	if err != nil {
		return nil, err
	}

	return releases, nil
}

func (gh *GitHub) CiStatus(sha string) (*octokat.Status, error) {
	client := gh.client()
	statuses, err := client.Statuses(gh.repo(), sha, nil)
	if err != nil {
		return nil, err
	}

	if len(statuses) == 0 {
		return nil, nil
	}

	return &statuses[0], nil
}

func (gh *GitHub) ForkRepository(name, owner string, noRemote bool) (repo *octokit.Repository, err error) {
	config := gh.Config
	project := Project{Name: name, Owner: config.User}
	r, err := gh.Repository(project)
	if err == nil && r != nil {
		err = fmt.Errorf("Error creating fork: %s exists on %s", r.FullName, GitHubHost)
		return
	}

	client := gh.octokit()
	repoService, err := client.Repositories(&octokit.ForksURL, octokit.M{"owner": owner, "repo": name})
	repo, result := repoService.Create(nil)
	if result.HasError() {
		err = result.Err
	}

	return
}

func (gh *GitHub) ExpandRemoteUrl(owner, name string, isSSH bool) (url string) {
	project := gh.Project
	if owner == "origin" {
		config := gh.Config
		owner = config.FetchUser()
	}

	return project.GitURL(name, owner, isSSH)
}

func (gh *GitHub) repo() octokat.Repo {
	project := gh.Project
	return octokat.Repo{Name: project.Name, UserName: project.Owner}
}

func findOrCreateToken(user, password, twoFactorCode string) (string, error) {
	client := octokat.NewClient().WithLogin(user, password)
	options := &octokat.Options{}
	if twoFactorCode != "" {
		headers := octokat.Headers{"X-GitHub-OTP": twoFactorCode}
		options.Headers = headers
	}

	auths, err := client.Authorizations(options)
	if err != nil {
		return "", err
	}

	var token string
	for _, auth := range auths {
		if auth.NoteURL == OAuthAppURL {
			token = auth.Token
			break
		}
	}

	if token == "" {
		authParam := octokat.AuthorizationParams{}
		authParam.Scopes = append(authParam.Scopes, "repo")
		authParam.Note = "gh"
		authParam.NoteURL = OAuthAppURL
		options.Params = authParam

		auth, err := client.CreateAuthorization(options)
		if err != nil {
			return "", err
		}

		token = auth.Token
	}

	return token, nil
}

func (gh *GitHub) client() *octokat.Client {
	config := gh.Config
	config.FetchCredentials()

	return octokat.NewClient().WithToken(config.Token)
}

func (gh *GitHub) octokit() *octokit.Client {
	config := gh.Config
	config.FetchCredentials()
	tokenAuth := octokit.TokenAuth{AccessToken: config.Token}

	return octokit.NewClient(tokenAuth)
}

func New() *GitHub {
	project := CurrentProject()
	c := CurrentConfig()
	c.FetchUser()

	return &GitHub{project, c}
}

// TODO: detach project from GitHub
func NewWithoutProject() *GitHub {
	c := CurrentConfig()
	c.FetchUser()

	return &GitHub{nil, c}
}

func (gh *GitHub) Issues() ([]octokat.Issue, error) {
	client := gh.client()
	issues, err := client.Issues(gh.repo(), nil)
	if err != nil {
		return nil, err
	}

	return issues, nil
}

package internal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v6"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6/git"
	"github.com/muesli/termenv"
	"github.com/urfave/cli/v2"
)

const REF_PREFIX = "refs/heads/"

type PrWithDetails struct {
	Title        string
	Status       git.PullRequestStatus
	SourceBranch string
	TargetBranch string
	Author       string
	CreatedOn    azuredevops.Time
	IsDraft      bool
	Url          string
}

func (pr PrWithDetails) String() string {
	var details []string
	if time.Now().Add(-30 * 24 * time.Hour).Before(pr.CreatedOn.Time) {
		details = append(details, fmt.Sprintf("Was updated recently (%s)", pr.CreatedOn.Time))
	}
	if pr.Author != "" {
		details = append(details, fmt.Sprintf("Was created by (%s)", pr.Author))
	}
	if pr.SourceBranch != "" {
		details = append(details, fmt.Sprintf("From Source Branch (%s)", pr.SourceBranch))
	}
	if pr.TargetBranch != "" {
		details = append(details, fmt.Sprintf("To Target Branch (%s)", pr.TargetBranch))
	}
	if pr.Status != "" {
		details = append(details, fmt.Sprintf("Current status (%s)", pr.Status))
	}
	if pr.IsDraft {
		details = append(details, fmt.Sprintf("It is a draft!"))
	}
	if pr.Url != "" {
		details = append(details, fmt.Sprintf("URL to this PR (%s)", pr.Url))
	}

	var s string
	for _, d := range details {
		s += "    * " + d + "\n"
	}
	s += "\n"
	return termenv.String(s).Faint().Italic().String()
}

func NewPrWithDetail(prs *[]git.GitPullRequest) *[]PrWithDetails {
	var detailPr []PrWithDetails

	for _, pr := range *prs {
		detailPr = append(detailPr, PrWithDetails{
			Title:        *pr.Title,
			Status:       *pr.Status,
			SourceBranch: *pr.SourceRefName,
			TargetBranch: *pr.TargetRefName,
			Author:       *pr.CreatedBy.DisplayName,
			CreatedOn:    *pr.CreationDate,
			IsDraft:      *pr.IsDraft,
		})
	}

	return &detailPr
}

func GetAzureClient() func(c *cli.Context) error {
	return func(c *cli.Context) error {
		token := c.String("token")
		azurl := c.String("az-url")
		if token == "" || azurl == "" {
			return cli.NewExitError("token and base url required", 1)
		}
		connection := azuredevops.NewPatConnection(azurl, token)
		client, err := git.NewClient(c.Context, connection)

		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}

		if token == "" {
			return cli.NewExitError("missing azure token", 1)
		}
		c.Context = context.WithValue(c.Context, "client", client)
		return nil
	}
}

//List prs for the repository
func ListPrs() func(c *cli.Context) error {
	return func(c *cli.Context) error {
		client := (c.Context.Value("client")).(git.Client)
		repoName := c.String("name")
		repoDetails, err := client.GetRepositories(c.Context, git.GetRepositoriesArgs{})
		if repoDetails == nil {
			fmt.Printf("err: %v\n", err.Error())

			return cli.NewExitError("no repo found", -1)
		}
		var currRepo *git.GitRepository

		for _, v := range *repoDetails {
			if *v.Name == repoName {
				currRepo = &v
				break
			}
		}

		if currRepo == nil {
			if err != nil {
				fmt.Printf("err: %v\n", err.Error())
			}

			return cli.NewExitError("no repo found", -1)
		}
		includeLinks := true
		repoId := currRepo.Id.String()

		pullRequests, err := client.GetPullRequests(c.Context, git.GetPullRequestsArgs{
			RepositoryId: &repoId,
			SearchCriteria: &git.GitPullRequestSearchCriteria{
				Status:       &git.PullRequestStatusValues.Active,
				IncludeLinks: &includeLinks,
			},
		})

		if err != nil {
			return err
		}

		detailPrs := NewPrWithDetail(pullRequests)
		for _, v := range *detailPrs {
			fmt.Println(termenv.String("PR: " + v.Title).Bold())
			fmt.Fprintln(c.App.Writer, v)
		}

		return nil
	}
}

func CreatePr() func(c *cli.Context) error {
	return func(c *cli.Context) error {
		client := (c.Context.Value("client")).(git.Client)
		repoName := c.String("name")
		repoDetails, err := client.GetRepositories(c.Context, git.GetRepositoriesArgs{})
		if repoDetails == nil {
			fmt.Printf("err: %v\n", err.Error())

			return cli.NewExitError("no repo found", -1)
		}
		var currRepo *git.GitRepository

		for _, v := range *repoDetails {
			if *v.Name == repoName {
				currRepo = &v
			}
		}

		if currRepo == nil {
			fmt.Printf("err: %v\n", err.Error())

			return cli.NewExitError("no repo found", -1)
		}

		messageArg := strings.Split(c.String("message"), ";")

		targetBranch := c.String("target")

		if targetBranch == "" {
			targetBranch = *currRepo.DefaultBranch
		}

		sourceBranch := c.String("source")
		sourceBranch, targetBranch = getRefNameFromBranch(sourceBranch, targetBranch)
		prMessage := strings.Join(messageArg[1:], "")
		repoId := currRepo.Id.String()
		isDraft := c.Bool("draft")

		pullRequest, err := client.CreatePullRequest(c.Context, git.CreatePullRequestArgs{
			GitPullRequestToCreate: &git.GitPullRequest{
				Title:         &messageArg[0],
				Description:   &prMessage,
				TargetRefName: &targetBranch,
				SourceRefName: &sourceBranch,
				IsDraft:       &isDraft,
			},
			RepositoryId: &repoId,
		})

		if err != nil {
			return err
		}

		fmt.Println(termenv.String("Pull Request was created successfully").Bold().Italic().Background(termenv.ANSIGreen).Faint())

		prDetail := NewPrWithDetail(&[]git.GitPullRequest{*pullRequest})
		for _, v := range *prDetail {
			fmt.Println(termenv.String("PR: " + v.Title).Bold())
			fmt.Fprintln(c.App.Writer, v)
		}

		return nil
	}
}

func getRefNameFromBranch(sourceBranch string, targetBranch string) (string, string) {
	if !strings.HasPrefix(sourceBranch, REF_PREFIX) {
		sourceBranch = REF_PREFIX + sourceBranch
	}
	if !strings.HasPrefix(targetBranch, REF_PREFIX) {
		targetBranch = REF_PREFIX + targetBranch
	}
	return sourceBranch, targetBranch
}

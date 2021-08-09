package digest

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util/hint"
)

const (
	ProblemMimetype    = "application/problem+json; charset=utf-8"
	ProblemNamespace   = "https://github.com/spikeekips/mitum-currency/problems"
	DefaultProblemType = "others"
)

var (
	ProblemType = hint.Type("mitum-currency-problem")
	ProblemHint = hint.NewHint(ProblemType, "v0.0.1")
)

// Problem implements "Problem Details for HTTP
// APIs"<https://tools.ietf.org/html/rfc7807>.
type Problem struct {
	t      string // NOTE http problem type
	title  string
	detail string
	extra  map[string]interface{}
}

func NewProblem(t, title string) Problem {
	return Problem{t: t, title: title}
}

func NewProblemFromError(err error) Problem {
	return Problem{
		t:      DefaultProblemType,
		title:  fmt.Sprintf("%s", err),
		detail: fmt.Sprintf("%+v", err),
	}
}

func (Problem) Hint() hint.Hint {
	return ProblemHint
}

func (pr Problem) Error() string {
	return pr.title
}

func (pr Problem) SetTitle(title string) Problem {
	pr.title = title

	return pr
}

func (pr Problem) SetDetail(detail string) Problem {
	pr.detail = detail

	return pr
}

func (pr Problem) SetExtra(key string, value interface{}) Problem {
	if pr.extra == nil {
		pr.extra = map[string]interface{}{}
	}

	pr.extra[key] = value

	return pr
}

func makeProblemNamespace(t string) string {
	return fmt.Sprintf("%s/%s", ProblemNamespace, t)
}

func parseProblemNamespace(s string) (string, error) {
	if !strings.HasPrefix(s, ProblemNamespace) {
		return "", errors.Errorf("invalid problem namespace: %q", s)
	}
	return s[len(ProblemNamespace)+1:], nil
}

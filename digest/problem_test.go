package digest

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/stretchr/testify/suite"
)

type testProblem struct {
	suite.Suite
}

func (t *testProblem) TestNew() {
	pt := "showme"
	title := "killme"
	pr := NewProblem(pt, title)

	b, err := jsonenc.Marshal(pr)
	t.NoError(err)

	var m map[string]interface{}
	t.NoError(jsonenc.Unmarshal(b, &m))

	t.Contains(m["type"], pt)
	t.Equal(title, m["title"])
	t.Empty(m["detail"])
}

func (t *testProblem) TestExtra() {
	pt := "showme"
	title := "killme"
	pr := NewProblem(pt, title)
	pr = pr.SetExtra("a", []string{"1", "2"})

	b, err := jsonenc.Marshal(pr)
	t.NoError(err)

	var m map[string]interface{}
	t.NoError(jsonenc.Unmarshal(b, &m))

	t.Contains(m["type"], pt)
	t.Equal(title, m["title"])
	t.Empty(m["detail"])
	t.Equal([]interface{}{"1", "2"}, m["a"])
}

func (t *testProblem) TestFromError() {
	e := util.NewError("showme")
	pr := NewProblemFromError(e)

	b, err := jsonenc.Marshal(pr)
	t.NoError(err)

	var m map[string]interface{}
	t.NoError(jsonenc.Unmarshal(b, &m))

	t.Contains(m["type"], DefaultProblemType)
	t.Equal("showme", m["title"])
	t.Equal("showme", m["detail"])
}

func (t *testProblem) TestFromWrapedError() {
	e0 := errors.Errorf("showme")
	e := errors.Wrap(e0, "findme")
	pr := NewProblemFromError(e)

	b, err := jsonenc.Marshal(pr)
	t.NoError(err)

	var m map[string]interface{}
	t.NoError(jsonenc.Unmarshal(b, &m))

	t.Contains(m["type"], DefaultProblemType)
	t.Equal("findme: showme", m["title"])
	t.Contains(m["detail"], "findme")
	t.Contains(m["detail"], "problem_test.go")
}

func TestProblem(t *testing.T) {
	suite.Run(t, new(testProblem))
}

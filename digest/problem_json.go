package digest

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"golang.org/x/xerrors"
)

func (pr Problem) MarshalJSON() ([]byte, error) {
	extra := pr.extra
	if extra == nil {
		extra = map[string]interface{}{}
	}

	extra["_hint"] = pr.Hint()
	extra["type"] = makeProblemNamespace(pr.t)
	extra["title"] = pr.title
	if len(pr.detail) > 0 {
		extra["detail"] = pr.detail
	}

	return jsonenc.Marshal(extra)
}

func (pr *Problem) UnmarshalJSON(b []byte) error {
	var m map[string]interface{}
	if err := jsonenc.Unmarshal(b, &m); err != nil {
		return err
	}

	c := func(key string) (string, error) {
		if i, found := m[key]; !found {
			return "", xerrors.Errorf("%s not found", key)
		} else if s, ok := i.(string); !ok {
			return "", xerrors.Errorf("%s not string", key)
		} else {
			return s, nil
		}
	}

	if s, err := c("type"); err != nil {
		return err
	} else if n, err := parseProblemNamespace(s); err != nil {
		return err
	} else {
		pr.t = n
	}

	if s, err := c("title"); err != nil {
		return err
	} else {
		pr.title = s
	}
	if s, err := c("detail"); err != nil {
		return err
	} else {
		pr.detail = s
	}

	extra := map[string]interface{}{}
	for k, v := range m {
		switch k {
		case "_hint", "type", "title", "detail":
		default:
			extra[k] = v
		}
	}

	pr.extra = extra

	return nil
}

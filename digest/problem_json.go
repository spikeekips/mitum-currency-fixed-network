package digest

import (
	"github.com/pkg/errors"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
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
			return "", errors.Errorf("%s not found", key)
		} else if s, ok := i.(string); !ok {
			return "", errors.Errorf("%s not string", key)
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

	var err error
	pr.title, err = c("title")
	if err != nil {
		return err
	}

	pr.detail, err = c("detail")
	if err != nil {
		return err
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

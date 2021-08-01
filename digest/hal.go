package digest

import (
	"net/url"

	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/hint"
)

type Hal interface {
	RawInterface() []byte
	Interface() interface{}
	SetInterface(interface{}) Hal
	Self() HalLink
	SetSelf(HalLink) Hal
	Links() map[string]HalLink
	AddLink(rel string, link HalLink) Hal
	Extras() map[string]interface{}
	AddExtras(string, interface{}) Hal
}

var (
	BaseHalType = hint.Type("mitum-currency-hal")
	BaseHalHint = hint.NewHint(BaseHalType, "v0.0.1")
)

type BaseHal struct {
	ht     hint.Hint
	i      interface{}
	raw    []byte
	self   HalLink
	links  map[string]HalLink
	extras map[string]interface{}
}

func NewBaseHal(i interface{}, self HalLink) BaseHal {
	return BaseHal{
		ht:     BaseHalHint,
		i:      i,
		self:   self,
		links:  map[string]HalLink{},
		extras: map[string]interface{}{},
	}
}

func (BaseHal) Hint() hint.Hint {
	return BaseHalHint
}

func (hal BaseHal) Interface() interface{} {
	return hal.i
}

func (hal BaseHal) RawInterface() []byte {
	return hal.raw
}

func (hal BaseHal) SetInterface(i interface{}) Hal {
	hal.i = i

	return hal
}

func (hal BaseHal) Links() map[string]HalLink {
	if hal.links == nil {
		return map[string]HalLink{}
	}

	return hal.links
}

func (hal BaseHal) Extras() map[string]interface{} {
	return hal.extras
}

func (hal BaseHal) AddExtras(key string, value interface{}) Hal {
	if hal.extras == nil {
		hal.extras = map[string]interface{}{}
	}

	hal.extras[key] = value

	return hal
}

func (hal BaseHal) Self() HalLink {
	return hal.self
}

func (hal BaseHal) SetSelf(u HalLink) Hal {
	hal.self = u

	return hal
}

func (hal BaseHal) AddLink(rel string, link HalLink) Hal {
	if hal.links == nil {
		hal.links = map[string]HalLink{}
	}

	hal.links[rel] = link

	return hal
}

type HalLink struct {
	href       string
	properties map[string]interface{}
}

func NewHalLink(href string, properties map[string]interface{}) HalLink {
	return HalLink{href: href, properties: properties}
}

func (hl HalLink) Href() string {
	return hl.href
}

func (hl HalLink) URL() (*url.URL, error) {
	return network.ParseURL(hl.href, false)
}

func (hl HalLink) Properties() map[string]interface{} {
	return hl.properties
}

func (hl HalLink) SetTemplated() HalLink {
	if hl.properties == nil {
		hl.properties = map[string]interface{}{}
	}

	hl.properties["templated"] = true

	return hl
}

func (hl HalLink) SetProperty(key string, value interface{}) HalLink {
	if hl.properties == nil {
		hl.properties = map[string]interface{}{}
	}

	hl.properties[key] = value

	return hl
}

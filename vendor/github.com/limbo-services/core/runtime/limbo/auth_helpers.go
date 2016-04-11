package limbo

import (
	"sort"
	"strings"

	"github.com/limbo-services/protobuf/proto"
	"github.com/limbo-services/protobuf/protoc-gen-gogo/descriptor"
)

func GetScope(method *descriptor.MethodDescriptorProto) (string, bool) {
	if method.Options != nil {
		v, _ := proto.GetExtension(method.Options, E_Authz)
		if v != nil {
			scope := v.(*AuthzRule).Scope
			if scope != "" {
				return scope, true
			}
		}
	}

	return "", false
}

func (r *AuthnRule) SetDefaults() {
	if r == nil {
		return
	}
	if r.Caller == "" {
		r.Caller = "caller"
	}
}

func (r *AuthzRule) SetDefaults() {
	if r == nil {
		return
	}
	if r.Caller == "" {
		r.Caller = "caller"
	}
}

func (base *AuthnRule) Inherit(r *AuthnRule) *AuthnRule {
	if r == nil && base != nil {
		r = new(AuthnRule)
	}
	if base == nil {
		return r
	}

	if r.Caller == "" {
		r.Caller = base.Caller
	}

	r.Strategies = append(base.Strategies, r.Strategies...)
	m := make(map[string]bool, len(r.Strategies))
	for _, s := range r.Strategies {
		if strings.HasPrefix(s, "-") {
			m[s[1:]] = false
			continue
		}
		if strings.HasPrefix(s, "+") {
			m[s[1:]] = true
			continue
		}
		m[s] = true
	}
	r.Strategies = r.Strategies[:0]
	for s := range m {
		r.Strategies = append(r.Strategies, s)
	}
	sort.Strings(r.Strategies)

	return r
}

func (base *AuthzRule) Inherit(r *AuthzRule) *AuthzRule {
	if r == nil && base != nil {
		r = new(AuthzRule)
	}
	if base == nil {
		return r
	}

	if r.Caller == "" {
		r.Caller = base.Caller
	}
	if r.Context == "" {
		r.Context = base.Context
	}
	if r.Scope == "" {
		r.Scope = base.Scope
	}

	return r
}

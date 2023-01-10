/*
Copyright 2020 The KubeSphere Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rbac

import (
	"context"
	"fmt"

	"github.com/open-policy-agent/opa/rego"
	"go.uber.org/zap"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/kubeclipper/kubeclipper/pkg/authorization/authorizer"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	"github.com/kubeclipper/kubeclipper/pkg/models/iam"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sliceutil"
)

const (
	defaultRegoQuery    = "data.authz.allow"
	defaultRegoFileName = "authz.rego"
)

var (
	_ authorizer.Authorizer = (*Authorizer)(nil)
)

func NewAuthorizer(am iam.Operator, cm cluster.Operator) authorizer.Authorizer {
	return &Authorizer{am: am, cm: cm}
}

type authorizingVisitor struct {
	requestAttributes authorizer.Attributes

	allowed bool
	reason  string
	errors  []error
}

func (v *authorizingVisitor) visit(source fmt.Stringer, regoPolicy string, rule *rbacv1.PolicyRule, err error) bool {
	if regoPolicy != "" && regoPolicyAllows(v.requestAttributes, regoPolicy) {
		v.allowed = true
		v.reason = fmt.Sprintf("RBAC: allowed by %s", source.String())
		return false
	}
	if rule != nil && ruleAllows(v.requestAttributes, rule) {
		v.allowed = true
		v.reason = fmt.Sprintf("RBAC: allowed by %s", source.String())
		return false
	}
	if err != nil {
		v.errors = append(v.errors, err)
	}
	return true
}

type Authorizer struct {
	am iam.Operator
	cm cluster.Operator
}

func (a *Authorizer) Authorize(ctx context.Context, attributes authorizer.Attributes) (authorizer.Decision, string, error) {
	ruleCheckingVisitor := &authorizingVisitor{requestAttributes: attributes}

	a.visitRulesFor(ctx, attributes, ruleCheckingVisitor.visit)

	if ruleCheckingVisitor.allowed {
		return authorizer.DecisionAllow, ruleCheckingVisitor.reason, nil
	}

	var reason string
	if len(ruleCheckingVisitor.errors) > 0 {
		reason = fmt.Sprintf("RBAC: %v", utilerrors.NewAggregate(ruleCheckingVisitor.errors))
	}
	return authorizer.DecisionNoOpinion, reason, nil
}

func (a *Authorizer) visitRulesFor(ctx context.Context, requestAttributes authorizer.Attributes, visitor func(source fmt.Stringer, regoPolicy string, rule *rbacv1.PolicyRule, err error) bool) {
	if globalRoleBindings, err := a.am.ListRoleBindings(ctx, &query.Query{
		Pagination:      query.NoPagination(),
		ResourceVersion: "0",
		LabelSelector:   "",
		FieldSelector:   "",
	}); err != nil {
		if !visitor(nil, "", nil, err) {
			return
		}
	} else {
		sourceDescriber := &globalRoleBindingDescriber{}
		for _, globalRoleBinding := range globalRoleBindings.Items {
			subjectIndex, applies := appliesTo(requestAttributes.GetUser(), globalRoleBinding.Subjects, "")
			if !applies {
				continue
			}
			regoPolicy, rules, err := a.getRoleReferenceRules(ctx, globalRoleBinding.RoleRef)
			if err != nil {
				visitor(nil, "", nil, err)
				continue
			}
			sourceDescriber.binding = globalRoleBinding.DeepCopy()
			sourceDescriber.subject = &globalRoleBinding.Subjects[subjectIndex]
			if !visitor(sourceDescriber, regoPolicy, nil, nil) {
				return
			}
			for i := range rules {
				if !visitor(sourceDescriber, "", &rules[i], nil) {
					return
				}
			}
		}
	}
}

// TODO: add am interface rather than iam.Operator
func (a *Authorizer) getRoleReferenceRules(ctx context.Context, roleRef rbacv1.RoleRef) (regoPolicy string, rules []rbacv1.PolicyRule, err error) {
	empty := make([]rbacv1.PolicyRule, 0)
	switch roleRef.Kind {
	case common.ResourceKindGlobalRole:
		globalRole, err := a.am.GetRoleEx(ctx, roleRef.Name, "0")
		if err != nil {
			if errors.IsNotFound(err) {
				return "", empty, nil
			}
			return "", nil, err
		}
		return globalRole.Annotations[common.RegoOverrideAnnotation], globalRole.Rules, nil
	default:
		return "", nil, fmt.Errorf("unsupported role reference kind: %q", roleRef.Kind)
	}
}

type globalRoleBindingDescriber struct {
	binding *iamv1.GlobalRoleBinding
	subject *rbacv1.Subject
}

func (d *globalRoleBindingDescriber) String() string {
	return fmt.Sprintf("GlobalRoleBinding %q of %s %q to %s",
		d.binding.Name,
		d.binding.RoleRef.Kind,
		d.binding.RoleRef.Name,
		describeSubject(d.subject, ""),
	)
}

func describeSubject(s *rbacv1.Subject, bindingNamespace string) string {
	switch s.Kind {
	case rbacv1.ServiceAccountKind:
		if len(s.Namespace) > 0 {
			return fmt.Sprintf("%s %q", s.Kind, s.Name+"/"+s.Namespace)
		}
		return fmt.Sprintf("%s %q", s.Kind, s.Name+"/"+bindingNamespace)
	default:
		return fmt.Sprintf("%s %q", s.Kind, s.Name)
	}
}

func ruleAllows(requestAttributes authorizer.Attributes, rule *rbacv1.PolicyRule) bool {
	if requestAttributes.IsResourceRequest() {
		combinedResource := requestAttributes.GetResource()
		if len(requestAttributes.GetSubresource()) > 0 {
			combinedResource = requestAttributes.GetResource() + "/" + requestAttributes.GetSubresource()
		}

		return VerbMatches(rule, requestAttributes.GetVerb()) &&
			APIGroupMatches(rule, requestAttributes.GetAPIGroup()) &&
			ResourceMatches(rule, combinedResource, requestAttributes.GetSubresource()) &&
			ResourceNameMatches(rule, requestAttributes.GetName())
	}

	return VerbMatches(rule, requestAttributes.GetVerb()) &&
		NonResourceURLMatches(rule, requestAttributes.GetPath())
}

// appliesTo returns whether any of the bindingSubjects applies to the specified subject,
// and if true, the index of the first subject that applies
func appliesTo(user user.Info, bindingSubjects []rbacv1.Subject, namespace string) (int, bool) {
	for i, bindingSubject := range bindingSubjects {
		if appliesToUser(user, bindingSubject, namespace) {
			return i, true
		}
	}
	return 0, false
}

func appliesToUser(user user.Info, subject rbacv1.Subject, namespace string) bool {
	switch subject.Kind {
	case rbacv1.UserKind:
		return user.GetName() == subject.Name

	case rbacv1.GroupKind:
		return sliceutil.HasString(user.GetGroups(), subject.Name)

	default:
		return false
	}
}

func regoPolicyAllows(requestAttributes authorizer.Attributes, regoPolicy string) bool {
	q, err := rego.New(rego.Query(defaultRegoQuery), rego.Module(defaultRegoFileName, regoPolicy)).PrepareForEval(context.Background())
	if err != nil {
		logger.Warn("syntax error", zap.String("content", regoPolicy), zap.Error(err))
		return false
	}

	results, err := q.Eval(context.Background(), rego.EvalInput(requestAttributes))
	if err != nil {
		logger.Warn("syntax error", zap.String("content", regoPolicy), zap.Error(err))
		return false
	}
	if len(results) > 0 && results[0].Expressions[0].Value == true {
		return true
	}
	return false
}

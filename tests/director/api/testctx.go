package api

import (
	"context"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/kyma-incubator/compass/components/director/pkg/graphql"

	"github.com/kyma-incubator/compass/tests/director/pkg/gql"

	"github.com/sirupsen/logrus"

	"github.com/avast/retry-go"

	"github.com/kyma-incubator/compass/tests/director/pkg/jwtbuilder"
	gcli "github.com/machinebox/graphql"
	"github.com/pkg/errors"
)

const (
	emptyTenant = ""
	testTenant  = "foo"
)

var defaultTenant = "3e64ebae-38b5-46a0-b1ed-9ccee153a0ae"

var tenants = make(map[string]string)

var tc *testContext

func init() {
	var err error
	tc, err = newTestContext()
	if err != nil {
		panic(errors.Wrap(err, "while test context setup"))
	}

	setDefaultTenant()

	tc, err = newTestContext()
	if err != nil {
		panic(errors.Wrap(err, "while test context with internal tenant setup"))
	}
}

func setDefaultTenant() {
	request := gcli.NewRequest(
		`query {
				result: tenants {
				id
				name
				internalID
					}
				}`)

	output := []*graphql.Tenant{}
	err := tc.RunOperation(context.TODO(), request, &output)
	if err != nil {
		panic(errors.Wrap(err, "while getting default tenant"))
	}

	for _, v := range output {
		tenants[*v.Name] = v.InternalID
		if *v.Name == testTenant {
			defaultTenant = v.InternalID
		}
	}
}

// testContext contains dependencies that help executing tests
type testContext struct {
	graphqlizer       gql.Graphqlizer
	gqlFieldsProvider gql.GqlFieldsProvider
	currentScopes     []string
	cli               *gcli.Client
}

const defaultScopes = "runtime:write application:write tenant:read label_definition:write integration_system:write application:read runtime:read label_definition:read integration_system:read health_checks:read application_template:read application_template:write eventing:manage"

func newTestContext() (*testContext, error) {
	scopesStr := os.Getenv("ALL_SCOPES")
	if scopesStr == "" {
		scopesStr = defaultScopes
	}

	currentScopes := strings.Split(scopesStr, " ")

	bearerToken, err := jwtbuilder.Do(defaultTenant, currentScopes)
	if err != nil {
		return nil, errors.Wrap(err, "while building JWT token")
	}

	return &testContext{
		graphqlizer:       gql.Graphqlizer{},
		gqlFieldsProvider: gql.GqlFieldsProvider{},
		currentScopes:     currentScopes,
		cli:               gql.NewAuthorizedGraphQLClient(bearerToken),
	}, nil
}

func (tc *testContext) RunOperation(ctx context.Context, req *gcli.Request, resp interface{}) error {
	m := resultMapperFor(&resp)

	return tc.withRetryOnTemporaryConnectionProblems(func() error {
		return tc.cli.Run(ctx, req, &m)
	})
}

func (tc *testContext) withRetryOnTemporaryConnectionProblems(risky func() error) error {
	return retry.Do(risky, retry.Attempts(7), retry.Delay(time.Second), retry.OnRetry(func(n uint, err error) {
		logrus.WithField("component", "testContext").Warnf("OnRetry: attempts: %d, error: %v", n, err)

	}), retry.LastErrorOnly(true), retry.RetryIf(func(err error) bool {
		return strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "connection reset by peer")
	}))
}

func (tc *testContext) RunOperationWithCustomTenant(ctx context.Context, tenant string, req *gcli.Request, resp interface{}) error {
	return tc.runCustomOperation(ctx, tenant, tc.currentScopes, req, resp)
}

func (tc *testContext) RunOperationWithCustomScopes(ctx context.Context, scopes []string, req *gcli.Request, resp interface{}) error {
	return tc.runCustomOperation(ctx, defaultTenant, scopes, req, resp)
}

func (tc *testContext) RunOperationWithoutTenant(ctx context.Context, req *gcli.Request, resp interface{}) error {
	return tc.runCustomOperation(ctx, emptyTenant, tc.currentScopes, req, resp)
}

func (tc *testContext) runCustomOperation(ctx context.Context, tenant string, scopes []string, req *gcli.Request, resp interface{}) error {
	m := resultMapperFor(&resp)

	token, err := jwtbuilder.Do(tenant, scopes)
	if err != nil {
		return errors.Wrap(err, "while building JWT token")
	}

	cli := gql.NewAuthorizedGraphQLClient(token)
	return tc.withRetryOnTemporaryConnectionProblems(func() error { return cli.Run(ctx, req, &m) })
}

// resultMapperFor returns generic object that can be passed to Run method for storing response.
// In GraphQL, set `result` alias for your query
func resultMapperFor(target interface{}) genericGQLResponse {
	if reflect.ValueOf(target).Kind() != reflect.Ptr {
		panic("target has to be a pointer")
	}
	return genericGQLResponse{
		Result: target,
	}
}

type genericGQLResponse struct {
	Result interface{} `json:"result"`
}

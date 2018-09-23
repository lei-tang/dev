/*
Copyright 2015 The Kubernetes Authors.

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

/*
oidc implements the authenticator.Token interface using the OpenID Connect protocol.

	config := oidc.Options{
		IssuerURL:     "https://accounts.google.com",
		ClientID:      os.Getenv("GOOGLE_CLIENT_ID"),
		UsernameClaim: "email",
	}
	tokenAuthenticator, err := oidc.New(config)
*/
package oidc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/square/go-jose.v2"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	oidc "github.com/coreos/go-oidc"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/authentication/user"
	certutil "k8s.io/client-go/util/cert"
)

var (
	// synchronizeTokenIDVerifierForTest should be set to true to force a
	// wait until the token ID verifiers are ready.
	synchronizeTokenIDVerifierForTest = false
)

type Options struct {
	// IssuerURL is the URL the provider signs ID Tokens as. This will be the "iss"
	// field of all tokens produced by the provider and is used for configuration
	// discovery.
	//
	// The URL is usually the provider's URL without a path, for example
	// "https://accounts.google.com" or "https://login.salesforce.com".
	//
	// The provider must implement configuration discovery.
	// See: https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderConfig
	IssuerURL string

	// ClientID the JWT must be issued for, the "sub" field. This plugin only trusts a single
	// client to ensure the plugin can be used with public providers.
	//
	// The plugin supports the "authorized party" OpenID Connect claim, which allows
	// specialized providers to issue tokens to a client for a different client.
	// See: https://openid.net/specs/openid-connect-core-1_0.html#IDToken
	ClientID string

	// Path to a PEM encoded root certificate of the provider.
	CAFile string

	// UsernameClaim is the JWT field to use as the user's username.
	UsernameClaim string

	// UsernamePrefix, if specified, causes claims mapping to username to be prefix with
	// the provided value. A value "oidc:" would result in usernames like "oidc:john".
	UsernamePrefix string

	// GroupsClaim, if specified, causes the OIDCAuthenticator to try to populate the user's
	// groups with an ID Token field. If the GrouppClaim field is present in an ID Token the value
	// must be a string or list of strings.
	GroupsClaim string

	// GroupsPrefix, if specified, causes claims mapping to group names to be prefixed with the
	// value. A value "oidc:" would result in groups like "oidc:engineering" and "oidc:marketing".
	GroupsPrefix string

	// SupportedSigningAlgs sets the accepted set of JOSE signing algorithms that
	// can be used by the provider to sign tokens.
	//
	// https://tools.ietf.org/html/rfc7518#section-3.1
	//
	// This value defaults to RS256, the value recommended by the OpenID Connect
	// spec:
	//
	// https://openid.net/specs/openid-connect-core-1_0.html#IDTokenValidation
	SupportedSigningAlgs []string

	// RequiredClaims, if specified, causes the OIDCAuthenticator to verify that all the
	// required claims key value pairs are present in the ID Token.
	RequiredClaims map[string]string

	// now is used for testing. It defaults to time.Now.
	now func() time.Time
}

// initVerifier creates a new ID token verifier for the given configuration and issuer URL.  On success, calls setVerifier with the
// resulting verifier.
func initVerifier(ctx context.Context, config *oidc.Config, iss string) (*oidc.IDTokenVerifier, error) {
	glog.V(4).Infof("initVerifier: iss=%v, config=%+v", iss, config)
	provider, err := oidc.NewProvider(ctx, iss)
	if err != nil {
		return nil, fmt.Errorf("init verifier failed: %v", err)
	}
	return provider.Verifier(config), nil
}

// asyncIDTokenVerifier is an ID token verifier that allows async initialization
// of the issuer check.  Must be passed by reference as it wraps sync.Mutex.
type asyncIDTokenVerifier struct {
	m sync.Mutex

	// v is the ID token verifier initialized asynchronously.  It remains nil
	// up until it is eventually initialized.
	// Guarded by m
	v *oidc.IDTokenVerifier
}

// newAsyncIDTokenVerifier creates a new asynchronous token verifier.  The
// verifier is available immediately, but may remain uninitialized for some time
// after creation.
func newAsyncIDTokenVerifier(ctx context.Context, c *oidc.Config, iss string) *asyncIDTokenVerifier {
	t := &asyncIDTokenVerifier{}

	sync := make(chan struct{})
	// Polls indefinitely in an attempt to initialize the distributed claims
	// verifier, or until context canceled.
	initFn := func() (done bool, err error) {
		glog.V(4).Infof("oidc authenticator: attempting init: iss=%v", iss)
		v, err := initVerifier(ctx, c, iss)
		if err != nil {
			glog.Errorf("oidc authenticator: async token verifier for issuer: %q: %v", iss, err)
			return false, nil
		}
		t.m.Lock()
		defer t.m.Unlock()
		t.v = v
		close(sync)
		return true, nil
	}

	go func() {
		if done, _ := initFn(); !done {
			go wait.PollUntil(time.Second*10, initFn, ctx.Done())
		}
	}()

	if synchronizeTokenIDVerifierForTest {
		<-sync
	}

	return t
}

// verifier returns the underlying ID token verifier, or nil if one is not yet initialized.
func (a *asyncIDTokenVerifier) verifier() *oidc.IDTokenVerifier {
	a.m.Lock()
	defer a.m.Unlock()
	return a.v
}

type Authenticator struct {
	issuerURL string

	usernameClaim  string
	usernamePrefix string
	groupsClaim    string
	groupsPrefix   string
	requiredClaims map[string]string

	// Contains an *oidc.IDTokenVerifier. Do not access directly use the
	// idTokenVerifier method.
	verifier atomic.Value

	cancel context.CancelFunc

	// resolver is used to resolve distributed claims.
	resolver *claimResolver
}

func (a *Authenticator) setVerifier(v *oidc.IDTokenVerifier) {
	a.verifier.Store(v)
}

func (a *Authenticator) idTokenVerifier() (*oidc.IDTokenVerifier, bool) {
	if v := a.verifier.Load(); v != nil {
		return v.(*oidc.IDTokenVerifier), true
	}
	return nil, false
}

func (a *Authenticator) Close() {
	a.cancel()
}

func New(opts Options) (*Authenticator, error) {
	return newAuthenticator(opts, func(ctx context.Context, a *Authenticator, config *oidc.Config) {
		// Asynchronously attempt to initialize the authenticator. This enables
		// self-hosted providers, providers that run on top of Kubernetes itself.

		glog.Errorf("To call wait.PollUnitl()")
		go wait.PollUntil(time.Second*10, func() (done bool, err error) {
			glog.Errorf("Enter wait.PollUnitl()")
			provider, err := oidc.NewProvider(ctx, a.issuerURL)
			if err != nil {
				glog.Errorf("oidc authenticator: initializing plugin: %v", err)
				return false, nil
			}

			verifier := provider.Verifier(config)
			a.setVerifier(verifier)
			return true, nil
		}, ctx.Done())
	})
}

func NewAuthenticatorWithIssuerURL(opts Options) (*Authenticator, error) {
	return newAuthenticator(opts, func(ctx context.Context, a *Authenticator, config *oidc.Config) {
		glog.V(5).Infof("NewProvider(%v)", a.issuerURL)
		provider, err := oidc.NewProvider(ctx, a.issuerURL)
		if err == nil {
			verifier := provider.Verifier(config)
			a.setVerifier(verifier)
			glog.V(5).Infof("setVerifier(%+v)", verifier)
		} else {
			glog.Errorf("Failed to create a provider for %v: %v", provider, err)
		}
	})
}

func NewAuthenticatorWithPubKey(opts Options, pubKeys []*jose.JSONWebKey) (*Authenticator, error) {
	// Initialize the authenticator.
	a, err := newAuthenticator(opts, func(ctx context.Context, a *Authenticator, config *oidc.Config) {
		// Set the verifier to use the public key set instead of reading from a remote.
		a.setVerifier(oidc.NewVerifier(
			opts.IssuerURL,
			&StaticKeySet{keys: pubKeys},
			config,
		))
	})
	return a, err
}

// The verifier may need to be ready by SetSynchronizeTokenIDVerifier(true)
func SetSynchronizeTokenIDVerifier(sync bool) {
	synchronizeTokenIDVerifierForTest = sync
}

// StaticKeySet implements oidc.KeySet.
type StaticKeySet struct {
	keys []*jose.JSONWebKey
}

func (s *StaticKeySet) VerifySignature(ctx context.Context, jwt string) (payload []byte, err error) {
	jws, err := jose.ParseSigned(jwt)
	if err != nil {
		return nil, err
	}
	if len(jws.Signatures) == 0 {
		return nil, fmt.Errorf("jwt contained no signatures")
	}
	kid := jws.Signatures[0].Header.KeyID

	for _, key := range s.keys {
		if key.KeyID == kid {
			return jws.Verify(key)
		}
	}

	return nil, fmt.Errorf("no keys matches jwk keyid")
}

// whitelist of signing algorithms to ensure users don't mistakenly pass something
// goofy.
var allowedSigningAlgs = map[string]bool{
	oidc.RS256: true,
	oidc.RS384: true,
	oidc.RS512: true,
	oidc.ES256: true,
	oidc.ES384: true,
	oidc.ES512: true,
	oidc.PS256: true,
	oidc.PS384: true,
	oidc.PS512: true,
}

func newAuthenticator(opts Options, initVerifier func(ctx context.Context, a *Authenticator, config *oidc.Config)) (*Authenticator, error) {
	url, err := url.Parse(opts.IssuerURL)
	if err != nil {
		return nil, err
	}

	if url.Scheme != "https" {
		return nil, fmt.Errorf("'oidc-issuer-url' (%q) has invalid scheme (%q), require 'https'", opts.IssuerURL, url.Scheme)
	}

	if opts.UsernameClaim == "" {
		return nil, errors.New("no username claim provided")
	}

	supportedSigningAlgs := opts.SupportedSigningAlgs
	if len(supportedSigningAlgs) == 0 {
		// RS256 is the default recommended by OpenID Connect and an 'alg' value
		// providers are required to implement.
		supportedSigningAlgs = []string{oidc.RS256}
	}
	for _, alg := range supportedSigningAlgs {
		if !allowedSigningAlgs[alg] {
			return nil, fmt.Errorf("oidc: unsupported signing alg: %q", alg)
		}
	}

	var roots *x509.CertPool
	if opts.CAFile != "" {
		roots, err = certutil.NewPool(opts.CAFile)
		if err != nil {
			return nil, fmt.Errorf("Failed to read the CA file: %v", err)
		}
	} else {
		glog.Info("OIDC: No x509 certificates provided, will use host's root CA set")
	}

	// Copied from http.DefaultTransport.
	tr := net.SetTransportDefaults(&http.Transport{
		// According to golang's doc, if RootCAs is nil,
		// TLS uses the host's root CA set.
		TLSClientConfig: &tls.Config{RootCAs: roots},
	})

	client := &http.Client{Transport: tr, Timeout: 30 * time.Second}

	ctx, cancel := context.WithCancel(context.Background())
	ctx = oidc.ClientContext(ctx, client)

	now := opts.now
	if now == nil {
		now = time.Now
	}

	verifierConfig := &oidc.Config{
		ClientID:             opts.ClientID,
		SupportedSigningAlgs: supportedSigningAlgs,
		Now:                  now,
	}

	var resolver *claimResolver
	if opts.GroupsClaim != "" {
		glog.V(5).Infof("opts.GroupsClaim: %v", opts.GroupsClaim)
		glog.V(5).Infof("verifierConfig is: %+v", verifierConfig)
		resolver = newClaimResolver(opts.GroupsClaim, client, verifierConfig)
	}

	authenticator := &Authenticator{
		issuerURL:      opts.IssuerURL,
		usernameClaim:  opts.UsernameClaim,
		usernamePrefix: opts.UsernamePrefix,
		groupsClaim:    opts.GroupsClaim,
		groupsPrefix:   opts.GroupsPrefix,
		requiredClaims: opts.RequiredClaims,
		cancel:         cancel,
		resolver:       resolver,
	}

	initVerifier(ctx, authenticator, verifierConfig)
	return authenticator, nil
}

// untrustedIssuer extracts an untrusted "iss" claim from the given JWT token,
// or returns an error if the token can not be parsed.  Since the JWT is not
// verified, the returned issuer should not be trusted.
func untrustedIssuer(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("malformed token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("error decoding token: %v", err)
	}
	claims := struct {
		// WARNING: this JWT is not verified. Do not trust these claims.
		Issuer string `json:"iss"`
	}{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("while unmarshaling token: %v", err)
	}
	return claims.Issuer, nil
}

func hasCorrectIssuer(iss, tokenData string) bool {
	uiss, err := untrustedIssuer(tokenData)
	if err != nil {
		return false
	}
	if uiss != iss {
		return false
	}
	return true
}

// endpoint represents an OIDC distributed claims endpoint.
type endpoint struct {
	// URL to use to request the distributed claim.  This URL is expected to be
	// prefixed by one of the known issuer URLs.
	URL string `json:"endpoint,omitempty"`
	// AccessToken is the bearer token to use for access.  If empty, it is
	// not used.  Access token is optional per the OIDC distributed claims
	// specification.
	// See: http://openid.net/specs/openid-connect-core-1_0.html#DistributedExample
	AccessToken string `json:"access_token,omitempty"`
	// JWT is the container for aggregated claims.  Not supported at the moment.
	// See: http://openid.net/specs/openid-connect-core-1_0.html#AggregatedExample
	JWT string `json:"JWT,omitempty"`
}

// claimResolver expands distributed claims by calling respective claim source
// endpoints.
type claimResolver struct {
	// claim is the distributed claim that may be resolved.
	claim string

	// client is the to use for resolving distributed claims
	client *http.Client

	// config is the OIDC configuration used for resolving distributed claims.
	config *oidc.Config

	// verifierPerIssuer contains, for each issuer, the appropriate verifier to use
	// for this claim.  It is assumed that there will be very few entries in
	// this map.
	// Guarded by m.
	verifierPerIssuer map[string]*asyncIDTokenVerifier

	m sync.Mutex
}

// newClaimResolver creates a new resolver for distributed claims.
func newClaimResolver(claim string, client *http.Client, config *oidc.Config) *claimResolver {
	return &claimResolver{claim: claim, client: client, config: config, verifierPerIssuer: map[string]*asyncIDTokenVerifier{}}
}

// Verifier returns either the verifier for the specified issuer, or error.
func (r *claimResolver) Verifier(iss string) (*oidc.IDTokenVerifier, error) {
	r.m.Lock()
	av := r.verifierPerIssuer[iss]
	if av == nil {
		// This lazy init should normally be very quick.
		// TODO: Make this context cancelable.
		ctx := oidc.ClientContext(context.Background(), r.client)
		av = newAsyncIDTokenVerifier(ctx, r.config, iss)
		r.verifierPerIssuer[iss] = av
	}
	r.m.Unlock()

	v := av.verifier()
	if v == nil {
		return nil, fmt.Errorf("verifier not initialized for issuer: %q", iss)
	}
	return v, nil
}

// expand extracts the distributed claims from claim names and claim sources.
// The extracted claim value is pulled up into the supplied claims.
//
// Distributed claims are of the form as seen below, and are defined in the
// OIDC Connect Core 1.0, section 5.6.2.
// See: https://openid.net/specs/openid-connect-core-1_0.html#AggregatedDistributedClaims
//
// {
//   ... (other normal claims)...
//   "_claim_names": {
//     "groups": "src1"
//   },
//   "_claim_sources": {
//     "src1": {
//       "endpoint": "https://www.example.com",
//       "access_token": "f005ba11"
//     },
//   },
// }
func (r *claimResolver) expand(c claims) error {
	const (
		// The claim containing a map of endpoint references per claim.
		// OIDC Connect Core 1.0, section 5.6.2.
		claimNamesKey = "_claim_names"
		// The claim containing endpoint specifications.
		// OIDC Connect Core 1.0, section 5.6.2.
		claimSourcesKey = "_claim_sources"
	)

	glog.V(5).Infof("The resolver claim (i.e., r.claim) is: %v", r.claim)
	glog.V(5).Infof("claims is: %+v", c)

	_, ok := c[r.claim]
	if ok {
		// There already is a normal claim, skip resolving.
		return nil
	}
	names, ok := c[claimNamesKey]
	if !ok {
		// No _claim_names, no keys to look up.
		return nil
	}

	// map from claim name to source name
	claimToSource := map[string]string{}
	if err := json.Unmarshal([]byte(names), &claimToSource); err != nil {
		return fmt.Errorf("oidc: error parsing distributed claim names: %v", err)
	}
	glog.V(5).Infof("claimToSource map is: %+v", claimToSource)

	rawSources, ok := c[claimSourcesKey]
	if !ok {
		// Having _claim_names claim,  but no _claim_sources is not an expected
		// state.
		return fmt.Errorf("oidc: no claim sources")
	}

	// map from source name to source endpoint
	var sources map[string]endpoint
	if err := json.Unmarshal([]byte(rawSources), &sources); err != nil {
		// The claims sources claim is malformed, this is not an expected state.
		return fmt.Errorf("oidc: could not parse claim sources: %v", err)
	}

	glog.V(5).Infof("source name to source endpoint map is: %+v", sources)

	// find the source for the claim, e.g., groups
	src, ok := claimToSource[r.claim]
	if !ok {
		// No distributed claim present.
		return nil
	}
	// find the endpoint for the claim
	ep, ok := sources[src]
	if !ok {
		return fmt.Errorf("id token _claim_names contained a source %s missing in _claims_sources", src)
	}
	if ep.URL == "" {
		// This is maybe an aggregated claim (ep.JWT != "").
		return nil
	}
	// resolve the claim at remote endpoint
	return r.resolve(ep, c)
}

// resolve requests distributed claims from all endpoints passed in,
// and inserts the lookup results into allClaims.
func (r *claimResolver) resolve(endpoint endpoint, allClaims claims) error {
	glog.V(5).Infof("resolve the claims %+v at %+v", allClaims, endpoint)
	// get the claim JWT from remote endpoint
	// TODO: cache resolved claims.
	glog.V(5).Infof("getClaimJWT() will be called to get claim JWT")
	jwt, err := getClaimJWT(r.client, endpoint.URL, endpoint.AccessToken)
	if err != nil {
		return fmt.Errorf("while getting distributed claim %q: %v", r.claim, err)
	}
	untrustedIss, err := untrustedIssuer(jwt)
	if err != nil {
		return fmt.Errorf("getting untrusted issuer from endpoint %v failed for claim %q: %v", endpoint.URL, r.claim, err)
	}
	glog.V(5).Infof("claim JWT is issued by %v", untrustedIss)
	glog.V(5).Infof("create a IDTokenVerifier for %v", untrustedIss)
	v, err := r.Verifier(untrustedIss)
	if err != nil {
		return fmt.Errorf("verifying untrusted issuer %v failed: %v", untrustedIss, err)
	}
	// verify the claim JWT from remote endpoint
	t, err := v.Verify(context.Background(), jwt)
	if err != nil {
		return fmt.Errorf("verify distributed claim token: %v", err)
	}
	var distClaims claims
	if err := t.Claims(&distClaims); err != nil {
		return fmt.Errorf("could not parse distributed claims for claim %v: %v", r.claim, err)
	}
	glog.V(5).Infof("Verified distributed claims is: %+v", distClaims)
	value, ok := distClaims[r.claim]
	if !ok {
		return fmt.Errorf("jwt returned by distributed claim endpoint %s did not contain claim: %v", endpoint, r.claim)
	}
	glog.V(5).Infof("resolved claim name %v has value: %+v", r.claim, string(value))
	allClaims[r.claim] = value
	return nil
}

func (a *Authenticator) AuthenticateToken(token string) (user.Info, bool, error) {
	glog.V(5).Infof("------------------------------------------------------------")
	glog.V(5).Infof("Enter AuthenticateToken()")
	glog.V(5).Infof("Authenticator issuerURL: %v", a.issuerURL)
	glog.V(5).Infof("Token to authenticate is: %v", token)
	// Example token:
	//{
	//	"iss": "https://127.0.0.1:34445",
	//	"aud": "my-client",
	//	"username": "jane",
	//	"_claim_names": {
	//	  "groups": "src1"
	//  },
	//	"_claim_sources": {
	//	  "src1": {
	//		  "endpoint": "https://127.0.0.1:34445/groups",
	//			"access_token": "groups_token"
	//	  }
	//  },
	//	"exp": 1257897600
	//}
	if !hasCorrectIssuer(a.issuerURL, token) {
		return nil, false, nil
	}

	verifier, ok := a.idTokenVerifier()
	if !ok {
		return nil, false, fmt.Errorf("oidc: authenticator not initialized")
	}

	ctx := context.Background()
	glog.V(5).Infof("Verify the token ...")
	idToken, err := verifier.Verify(ctx, token)
	if err != nil {
		return nil, false, fmt.Errorf("oidc: verify token: %v", err)
	}
	glog.V(5).Infof("The idToken returned by Verify() is:")
	glog.V(5).Infof("%+v", *idToken)

	var c claims
	if err := idToken.Claims(&c); err != nil {
		return nil, false, fmt.Errorf("oidc: parse claims: %v", err)
	}
	glog.V(5).Infof("The idToken claims is:")
	glog.V(5).Infof("%+v", c)

	if a.resolver != nil {
		if err := a.resolver.expand(c); err != nil {
			return nil, false, fmt.Errorf("oidc: could not expand distributed claims: %v", err)
		}
	}

	var username string
	if err := c.unmarshalClaim(a.usernameClaim, &username); err != nil {
		return nil, false, fmt.Errorf("oidc: parse username claims %q: %v", a.usernameClaim, err)
	}

	if a.usernameClaim == "email" {
		// If the email_verified claim is present, ensure the email is valid.
		// https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims
		if hasEmailVerified := c.hasClaim("email_verified"); hasEmailVerified {
			var emailVerified bool
			if err := c.unmarshalClaim("email_verified", &emailVerified); err != nil {
				return nil, false, fmt.Errorf("oidc: parse 'email_verified' claim: %v", err)
			}

			// If the email_verified claim is present we have to verify it is set to `true`.
			if !emailVerified {
				return nil, false, fmt.Errorf("oidc: email not verified")
			}
		}
	}

	if a.usernamePrefix != "" {
		username = a.usernamePrefix + username
	}

	info := &user.DefaultInfo{Name: username}
	if a.groupsClaim != "" {
		if _, ok := c[a.groupsClaim]; ok {
			// Some admins want to use string claims like "role" as the group value.
			// Allow the group claim to be a single string instead of an array.
			//
			// See: https://github.com/kubernetes/kubernetes/issues/33290
			var groups stringOrArray
			if err := c.unmarshalClaim(a.groupsClaim, &groups); err != nil {
				return nil, false, fmt.Errorf("oidc: parse groups claim %q: %v", a.groupsClaim, err)
			}
			info.Groups = []string(groups)
		}
	}

	if a.groupsPrefix != "" {
		for i, group := range info.Groups {
			info.Groups[i] = a.groupsPrefix + group
		}
	}

	glog.V(5).Infof("a.requiredClaims are %+v", a.requiredClaims)

	// check to ensure all required claims are present in the ID token and have matching values.
	for claim, value := range a.requiredClaims {
		if !c.hasClaim(claim) {
			return nil, false, fmt.Errorf("oidc: required claim %s not present in ID token", claim)
		}
		glog.V(5).Infof("c has claim %v, value=", claim, value)

		// NOTE: Only string values are supported as valid required claim values.
		var claimValue string
		if err := c.unmarshalClaim(claim, &claimValue); err != nil {
			return nil, false, fmt.Errorf("oidc: parse claim %s: %v", claim, err)
		}
		if claimValue != value {
			return nil, false, fmt.Errorf("oidc: required claim %s value does not match. Got = %s, want = %s", claim, claimValue, value)
		}
	}

	glog.V(5).Infof("Exit AuthenticateToken()")
	glog.V(5).Infof("------------------------------------------------------------")
	return info, true, nil
}

// getClaimJWT gets a distributed claim JWT from url, using the supplied access
// token as bearer token.  If the access token is "", the authorization header
// will not be set.
// TODO: Allow passing in JSON hints to the IDP.
func getClaimJWT(client *http.Client, url, accessToken string) (string, error) {
	glog.V(5).Infof("getClaimJWT(): url=%v, accessToken=%v", url, accessToken)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// TODO: Allow passing request body with configurable information.
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("while calling %v: %v", url, err)
	}
	if accessToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", accessToken))
	}
	req = req.WithContext(ctx)
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	// Report non-OK status code as an error.
	if response.StatusCode < http.StatusOK || response.StatusCode > http.StatusIMUsed {
		return "", fmt.Errorf("error while getting distributed claim JWT: %v", response.Status)
	}
	defer response.Body.Close()
	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("could not decode distributed claim response")
	}
	glog.V(5).Infof("claim JWT response from remote endpoint is: %+v", string(responseBytes))
	return string(responseBytes), nil
}

type stringOrArray []string

func (s *stringOrArray) UnmarshalJSON(b []byte) error {
	var a []string
	if err := json.Unmarshal(b, &a); err == nil {
		*s = a
		return nil
	}
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	*s = []string{str}
	return nil
}

type claims map[string]json.RawMessage

func (c claims) unmarshalClaim(name string, v interface{}) error {
	val, ok := c[name]
	if !ok {
		return fmt.Errorf("claim not present")
	}
	return json.Unmarshal([]byte(val), v)
}

func (c claims) hasClaim(name string) bool {
	if _, ok := c[name]; !ok {
		return false
	}
	return true
}

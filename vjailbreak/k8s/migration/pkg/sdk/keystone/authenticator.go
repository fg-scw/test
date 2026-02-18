// Package keystone provides authentication and token management for OpenStack Keystone API.
// It includes various authenticator implementations with different caching strategies
// to improve performance and reduce API calls.
package keystone

import (
	"context"
	"encoding/json"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AuthOptions configures the behavior of the authentication process
type AuthOptions struct {
	// PropagateCacheErrors will cause failures in caching layers to be bubbled up
	PropagateCacheErrors bool
}

// Authenticator provides an abstraction layer over the regular keystone client
// to simplify getting and refreshing authentication tokens. It handles various
// caching strategies to improve performance and reduce API calls.
type Authenticator interface {
	Auth(ctx context.Context, opts ...AuthOptions) (AuthInfo, error)
	ResetCache() error
}

// BasicAuthenticator provides a simple implementation of the Authenticator interface
// that directly authenticates with Keystone without any caching
type BasicAuthenticator struct {
	client      Client
	credentials Credentials
}

var _ Authenticator = (*BasicAuthenticator)(nil)

// NewBasicTokenGenerator creates a new BasicAuthenticator instance
func NewBasicTokenGenerator(client Client, credentials Credentials) *BasicAuthenticator {
	return &BasicAuthenticator{client: client, credentials: credentials}
}

// Auth authenticates with Keystone using the configured credentials.
// It implements the Authenticator interface's Auth method by making a direct API call without caching.
func (b *BasicAuthenticator) Auth(ctx context.Context, _ ...AuthOptions) (AuthInfo, error) {
	return b.client.Auth(ctx, b.credentials)
}

// ResetCache implements the Authenticator interface's ResetCache method.
// For BasicAuthenticator, this is a no-op since no caching is performed.
func (b *BasicAuthenticator) ResetCache() error {
	// no cache
	return nil
}

// StaticTokenAuthenticator provides an implementation of the Authenticator interface
type StaticTokenAuthenticator struct {
	client Client
	token  string
}

var _ Authenticator = (*StaticTokenAuthenticator)(nil)

// NewStaticTokenAuthenticator creates a new StaticTokenAuthenticator instance.
func NewStaticTokenAuthenticator(client Client, token string) *StaticTokenAuthenticator {
	return &StaticTokenAuthenticator{client: client, token: token}
}

// Auth validates the static token with Keystone and returns its authentication information.
func (s *StaticTokenAuthenticator) Auth(ctx context.Context, _ ...AuthOptions) (AuthInfo, error) {
	tokenInfo, err := s.client.GetTokenInfo(ctx, s.token)
	if err != nil {
		return AuthInfo{}, err
	}
	return AuthInfo{
		Token:     s.token,
		UserID:    tokenInfo.Token.User.ID,
		ProjectID: tokenInfo.Token.Project.ID,
		ExpiresAt: tokenInfo.Token.ExpiresAt,
	}, nil
}

// ResetCache implements the Authenticator interface's ResetCache method.
func (s *StaticTokenAuthenticator) ResetCache() error {
	// no cache
	return nil
}

// CachedAuthenticator implements Authenticator with in-memory caching of authentication tokens
type CachedAuthenticator struct {
	authenticator Authenticator
	RenewBefore   time.Duration
	cached        *AuthInfo
}

var _ Authenticator = (*CachedAuthenticator)(nil)

// NewCachedAuthenticator creates a new CachedAuthenticator that wraps another authenticator
func NewCachedAuthenticator(authenticator Authenticator) *CachedAuthenticator {
	return &CachedAuthenticator{
		authenticator: authenticator,
		RenewBefore:   10 * time.Minute,
	}
}

// Auth authenticates with Keystone, using an in-memory cached token when available.
// It implements the Authenticator interface's Auth method with memory caching strategy.
func (c *CachedAuthenticator) Auth(ctx context.Context, _ ...AuthOptions) (AuthInfo, error) {
	if c.cached != nil && c.cached.ExpiresAt.After(time.Now().Add(c.RenewBefore)) {
		return *c.cached, nil
	}
	authInfo, err := c.authenticator.Auth(ctx)
	if err != nil {
		return AuthInfo{}, err
	}
	c.cached = &authInfo

	return authInfo, nil
}

// ResetCache clears the in-memory authentication token cache.
// This forces the next Auth call to perform a fresh authentication.
func (c *CachedAuthenticator) ResetCache() error {
	c.cached = nil
	return nil
}

// FileCachedAuthenticator implements Authenticator with file-based caching of authentication tokens
type FileCachedAuthenticator struct {
	authenticator Authenticator
	RenewBefore   time.Duration
	CachePath     string
}

var _ Authenticator = (*FileCachedAuthenticator)(nil)

// NewFileCachedAuthenticator creates a new FileCachedAuthenticator that wraps another authenticator
func NewFileCachedAuthenticator(authenticator Authenticator, cachePath string) *FileCachedAuthenticator {
	return &FileCachedAuthenticator{
		authenticator: authenticator,
		CachePath:     cachePath,
		RenewBefore:   10 * time.Minute,
	}
}

// Auth authenticates with Keystone, using a file-cached token when available.
// It implements the Authenticator interface's Auth method with file-based caching strategy.
func (fc *FileCachedAuthenticator) Auth(ctx context.Context, opts ...AuthOptions) (AuthInfo, error) {
	var authOpts AuthOptions
	if len(opts) > 0 {
		authOpts = opts[0]
	}

	cached, err := fc.readFromFile()
	if err == nil && cached != nil && cached.ExpiresAt.After(time.Now().Add(fc.RenewBefore)) {
		return *cached, nil
	}

	authInfo, err := fc.authenticator.Auth(ctx)
	if err != nil {
		return AuthInfo{}, err
	}

	err = fc.writeToFile(&authInfo)
	if err != nil && authOpts.PropagateCacheErrors {
		return AuthInfo{}, err
	}
	return authInfo, nil
}

// readFromFile reads the cached authentication token from the file.
func (fc *FileCachedAuthenticator) readFromFile() (*AuthInfo, error) {
	data, err := os.ReadFile(fc.CachePath)
	if err != nil {
		return nil, err
	}
	authInfo := &AuthInfo{}
	err = json.Unmarshal(data, authInfo)
	if err != nil {
		return nil, err
	}
	return authInfo, nil
}

// writeToFile writes the authentication token to the file.
func (fc *FileCachedAuthenticator) writeToFile(authInfo *AuthInfo) error {
	data, err := json.Marshal(authInfo)
	if err != nil {
		return err
	}
	return os.WriteFile(fc.CachePath, data, 0600)
}

// ResetCache deletes the file-based authentication token cache.
// This forces the next Auth call to perform a fresh authentication.
func (fc *FileCachedAuthenticator) ResetCache() error {
	return os.WriteFile(fc.CachePath, []byte{}, 0600)
}

// SecretCachedAuthenticator caches the token in a Secret
type SecretCachedAuthenticator struct {
	authenticator Authenticator
	Client        client.Client
	SecretRef     metav1.ObjectMeta
	RenewBefore   time.Duration
}

// NewSecretCachedAuthenticator creates a new SecretCachedAuthenticator that wraps another authenticator
func NewSecretCachedAuthenticator(client client.Client, secretRef metav1.ObjectMeta, authenticator Authenticator) *SecretCachedAuthenticator {
	return &SecretCachedAuthenticator{
		authenticator: authenticator,
		Client:        client,
		SecretRef:     secretRef,
		RenewBefore:   10 * time.Minute,
	}
}

var _ Authenticator = (*SecretCachedAuthenticator)(nil)

// Auth authenticates with Keystone, using a Kubernetes Secret-cached token when available.
// It implements the Authenticator interface's Auth method with Kubernetes Secret caching strategy.
func (sc *SecretCachedAuthenticator) Auth(ctx context.Context, opts ...AuthOptions) (AuthInfo, error) {
	var authOpts AuthOptions
	if len(opts) > 0 {
		authOpts = opts[0]
	}

	cached, err := sc.readFromSecret(ctx)
	if err != nil && authOpts.PropagateCacheErrors {
		return AuthInfo{}, err
	}
	if err == nil && cached != nil && cached.ExpiresAt.After(time.Now().Add(sc.RenewBefore)) {
		return *cached, nil
	}

	authInfo, err := sc.authenticator.Auth(ctx)
	if err != nil {
		return AuthInfo{}, err
	}

	err = sc.writeToSecret(ctx, &authInfo)
	if err != nil && authOpts.PropagateCacheErrors {
		return AuthInfo{}, err
	}
	return authInfo, nil
}

// readFromSecret reads the cached authentication token from the Kubernetes Secret.
func (sc *SecretCachedAuthenticator) readFromSecret(ctx context.Context) (*AuthInfo, error) {
	tokenSecret := &corev1.Secret{}
	err := sc.Client.Get(ctx, types.NamespacedName{
		Namespace: sc.SecretRef.Namespace,
		Name:      sc.SecretRef.Name,
	}, tokenSecret)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	expiresAt, err := time.Parse(time.RFC3339, string(tokenSecret.Data["expiresAt"]))
	if err != nil {
		return nil, err
	}
	return &AuthInfo{
		Token:     string(tokenSecret.Data["token"]),
		UserID:    string(tokenSecret.Data["userID"]),
		ProjectID: string(tokenSecret.Data["projectID"]),
		ExpiresAt: expiresAt,
	}, nil
}

// writeToSecret writes the authentication token to the Kubernetes Secret.
func (sc *SecretCachedAuthenticator) writeToSecret(ctx context.Context, authInfo *AuthInfo) error {
	tokenSecret := &corev1.Secret{
		ObjectMeta: sc.SecretRef,
		Data: map[string][]byte{
			"token":     []byte(authInfo.Token),
			"userID":    []byte(authInfo.UserID),
			"projectID": []byte(authInfo.ProjectID),
			"expiresAt": []byte(authInfo.ExpiresAt.Format(time.RFC3339)),
		},
		Immutable: ptr.To(true),
	}
	err := sc.Client.Update(ctx, tokenSecret)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return sc.Client.Create(ctx, tokenSecret)
}

// ResetCache removes the authentication token from the Kubernetes Secret cache.
// This forces the next Auth call to perform a fresh authentication.
func (sc *SecretCachedAuthenticator) ResetCache() error {
	tokenSecret := &corev1.Secret{
		ObjectMeta: sc.SecretRef,
	}
	return sc.Client.Delete(context.Background(), tokenSecret)
}

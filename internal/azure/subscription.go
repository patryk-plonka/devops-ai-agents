package azure

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions/v2"
)

// boundCredential disallows tenant/claims challenges that would widen the CLI login.
type boundCredential struct{ inner azcore.TokenCredential }

func (c boundCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if opts.TenantID != "" || opts.Claims != "" {
		return azcore.AccessToken{}, fail("login_required", "Azure requires renewed login; run az login externally for the intended tenant.", 1)
	}
	token, err := c.inner.GetToken(ctx, opts)
	if err != nil {
		return azcore.AccessToken{}, fail("login_required", "Azure CLI authentication failed; run az login externally for the intended tenant.", 1)
	}
	return token, nil
}

func subscriptionCredential(id string) (azcore.TokenCredential, error) {
	if !uuid.MatchString(id) {
		return nil, fail("invalid_scope", "A valid subscription UUID is required.", 2)
	}
	c, err := azidentity.NewAzureCLICredential(&azidentity.AzureCLICredentialOptions{Subscription: id})
	if err != nil {
		return nil, fail("login_required", "Cannot initialize Azure CLI authentication; sign in externally.", 1)
	}
	return boundCredential{c}, nil
}

// Validator accepts a transport/credential only as a Go testing boundary.
type Validator struct {
	Base       http.RoundTripper
	Credential func(string) (azcore.TokenCredential, error)
}

func (v Validator) Validate(ctx context.Context, account Account) error {
	ctx, cancel := context.WithTimeout(ctx, OperationTimeout)
	defer cancel()
	newCredential := v.Credential
	if newCredential == nil {
		newCredential = subscriptionCredential
	}
	credential, err := newCredential(account.ID)
	if err != nil {
		return err
	}
	options := &arm.ClientOptions{DisableRPRegistration: true}
	options.Transport = ReadTransport{Subscription: account.ID, Base: v.Base}
	options.Retry.MaxRetries = 3
	client, err := armsubscriptions.NewClient(credential, options)
	if err != nil {
		return fail("client_failure", "Cannot initialize subscription validation.", 1)
	}
	response, err := client.Get(ctx, account.ID, nil)
	if err != nil {
		var login *Failure
		if errors.As(err, &login) {
			return login
		}
		var re *azcore.ResponseError
		if errors.As(err, &re) {
			switch re.StatusCode {
			case 401:
				return fail("login_required", "Azure authentication expired or was rejected; run az login externally.", 1)
			case 403:
				return fail("subscription_access_denied", "Live access to the selected subscription was denied.", 1)
			case 404:
				return fail("subscription_inaccessible", "Selected subscription was not found or is inaccessible.", 1)
			}
		}
		return fail("subscription_validation_failed", "Live subscription validation failed; check login, network, and selected scope.", 1)
	}
	if response.SubscriptionID == nil || !strings.EqualFold(*response.SubscriptionID, account.ID) ||
		response.ID != nil && !strings.EqualFold(*response.ID, "/subscriptions/"+account.ID) ||
		response.TenantID != nil && !strings.EqualFold(*response.TenantID, account.TenantID) {
		return fail("scope_mismatch", "Azure returned inconsistent subscription identity; validation failed.", 1)
	}
	return nil
}

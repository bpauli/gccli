package garminauth

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// ticketRe extracts the service ticket from SSO response HTML.
var ticketRe = regexp.MustCompile(`embed\?ticket=([^"]+)"`)

// oauthConsumer holds the OAuth1 consumer credentials fetched from Garmin.
type oauthConsumer struct {
	ConsumerKey    string `json:"consumer_key"`
	ConsumerSecret string `json:"consumer_secret"`
}

// oauth2Response represents the JSON response from the OAuth2 token exchange.
type oauth2Response struct {
	TokenType             string `json:"token_type"`
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	ExpiresIn             int    `json:"expires_in"`
	RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
}

// oauthConsumerURL is the URL for fetching OAuth consumer credentials.
// It is a variable to allow overriding in tests.
var oauthConsumerURL = OAuthConsumerURL

// LoginHeadless performs a headless SSO login using email and password.
// It follows the Garmin SSO flow: embed -> signin -> credentials -> ticket -> OAuth1 -> OAuth2.
func LoginHeadless(ctx context.Context, email, password string, opts LoginOptions) (*Tokens, error) {
	ep := NewEndpoints(opts.domain())
	return loginHeadless(ctx, email, password, opts, ep)
}

// loginHeadless is the internal implementation that accepts explicit endpoints
// for testability.
func loginHeadless(ctx context.Context, email, password string, opts LoginOptions, ep Endpoints) (*Tokens, error) {
	client := opts.HTTPClient
	if client == nil {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, fmt.Errorf("create cookie jar: %w", err)
		}
		client = &http.Client{Jar: jar}
	}

	ssoEmbed := ep.SSOEmbed

	// Step 1: GET SSO embed page to establish cookies.
	embedParams := url.Values{
		"id":          {"gauth-widget"},
		"embedWidget": {"true"},
		"gauthHost":   {ep.SSOBase + "/sso"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		ssoEmbed+"?"+embedParams.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("create embed request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sso embed: %w", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	// Step 2: GET SSO signin page to extract CSRF token.
	signinParams := url.Values{
		"id":                              {"gauth-widget"},
		"embedWidget":                     {"true"},
		"gauthHost":                       {ssoEmbed},
		"service":                         {ssoEmbed},
		"source":                          {ssoEmbed},
		"redirectAfterAccountLoginUrl":    {ssoEmbed},
		"redirectAfterAccountCreationUrl": {ssoEmbed},
	}
	req, err = http.NewRequestWithContext(ctx, http.MethodGet,
		ep.SSOSignin+"?"+signinParams.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("create signin request: %w", err)
	}
	req.Header.Set("Referer", ssoEmbed)
	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sso signin page: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read signin page: %w", err)
	}

	csrf, err := getCSRFToken(string(body))
	if err != nil {
		return nil, fmt.Errorf("extract CSRF: %w", err)
	}

	// Step 3: POST credentials to SSO signin.
	formData := url.Values{
		"username": {email},
		"password": {password},
		"embed":    {"true"},
		"_csrf":    {csrf},
	}
	req, err = http.NewRequestWithContext(ctx, http.MethodPost,
		ep.SSOSignin+"?"+signinParams.Encode(),
		strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create credentials request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", ep.SSOSignin)
	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sso signin: %w", err)
	}
	body, err = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read signin response: %w", err)
	}

	// Check for MFA challenge.
	var ticket string
	if isMFARequired(string(body)) {
		// Extract CSRF token from the MFA page for the verification POST.
		mfaCSRF, csrfErr := getCSRFToken(string(body))
		if csrfErr != nil {
			return nil, fmt.Errorf("extract MFA CSRF: %w", csrfErr)
		}

		// Get the MFA code from options or interactive prompt.
		mfaCode, mfaErr := resolveMFACode(opts)
		if mfaErr != nil {
			return nil, mfaErr
		}

		// Submit the MFA code and get the service ticket.
		ticket, err = submitMFA(ctx, client, ep, mfaCSRF, mfaCode)
		if err != nil {
			return nil, fmt.Errorf("MFA verification: %w", err)
		}
	} else {
		// Extract service ticket from response.
		ticket, err = getTicket(string(body))
		if err != nil {
			title, _ := getTitle(string(body))
			if strings.Contains(strings.ToLower(title), "locked") {
				return nil, fmt.Errorf("account locked")
			}
			return nil, fmt.Errorf("authentication failed (title=%q): %w", title, err)
		}
	}

	// Step 4: Fetch OAuth consumer credentials.
	consumer, err := fetchOAuthConsumer(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("fetch oauth consumer: %w", err)
	}

	// Step 5: Exchange ticket for OAuth1 token.
	oauth1Token, oauth1Secret, mfaToken, err := exchangePreauthorized(
		ctx, client, ep, consumer, ticket, ep.SSOEmbed)
	if err != nil {
		return nil, fmt.Errorf("oauth1 exchange: %w", err)
	}

	// Step 6: Exchange OAuth1 for OAuth2 tokens.
	tokens, err := exchangeOAuth2(
		ctx, client, ep, consumer, oauth1Token, oauth1Secret, mfaToken)
	if err != nil {
		return nil, fmt.Errorf("oauth2 exchange: %w", err)
	}

	tokens.Domain = ep.domain()
	tokens.Email = email
	tokens.OAuth1Token = oauth1Token
	tokens.OAuth1Secret = oauth1Secret
	if mfaToken != "" {
		tokens.MFAToken = mfaToken
	}

	return tokens, nil
}

// domain returns the domain from the endpoint's SSOBase URL.
func (e Endpoints) domain() string {
	if strings.Contains(e.SSOBase, DomainChina) {
		return DomainChina
	}
	return DomainGlobal
}

// fetchOAuthConsumer retrieves the OAuth1 consumer key and secret.
func fetchOAuthConsumer(ctx context.Context, client *http.Client) (*oauthConsumer, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, oauthConsumerURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d fetching consumer credentials", resp.StatusCode)
	}

	var consumer oauthConsumer
	if err := json.NewDecoder(resp.Body).Decode(&consumer); err != nil {
		return nil, fmt.Errorf("decode consumer credentials: %w", err)
	}
	return &consumer, nil
}

// exchangePreauthorized exchanges a service ticket for OAuth1 tokens.
// loginURL must match the service URL the ticket was originally issued for.
func exchangePreauthorized(ctx context.Context, client *http.Client,
	ep Endpoints, consumer *oauthConsumer, ticket, loginURL string,
) (token, secret, mfaToken string, err error) {
	preAuthURL := ep.PreauthorizedURL(ticket, loginURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, preAuthURL, nil)
	if err != nil {
		return "", "", "", err
	}

	authHeader := signOAuth1(http.MethodGet, preAuthURL,
		consumer.ConsumerKey, consumer.ConsumerSecret, "", "", nil)
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("User-Agent", UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		return "", "", "", fmt.Errorf("parse response: %w", err)
	}

	token = values.Get("oauth_token")
	secret = values.Get("oauth_token_secret")
	mfaToken = values.Get("mfa_token")

	if token == "" || secret == "" {
		return "", "", "", fmt.Errorf("response missing oauth_token or oauth_token_secret")
	}

	return token, secret, mfaToken, nil
}

// exchangeOAuth2 exchanges OAuth1 credentials for OAuth2 access and refresh tokens.
func exchangeOAuth2(ctx context.Context, client *http.Client,
	ep Endpoints, consumer *oauthConsumer,
	oauth1Token, oauth1Secret, mfaToken string,
) (*Tokens, error) {
	exchangeURL := ep.ExchangeURL()

	var bodyParams url.Values
	if mfaToken != "" {
		bodyParams = url.Values{"mfa_token": {mfaToken}}
	}

	var bodyReader io.Reader
	var bodyStr string
	if bodyParams != nil {
		bodyStr = bodyParams.Encode()
		bodyReader = strings.NewReader(bodyStr)
	} else {
		bodyReader = strings.NewReader("")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, exchangeURL, bodyReader)
	if err != nil {
		return nil, err
	}

	authHeader := signOAuth1(http.MethodPost, exchangeURL,
		consumer.ConsumerKey, consumer.ConsumerSecret,
		oauth1Token, oauth1Secret, bodyParams)
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}

	var oauth2Resp oauth2Response
	if err := json.Unmarshal(body, &oauth2Resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &Tokens{
		OAuth2AccessToken:  oauth2Resp.AccessToken,
		OAuth2RefreshToken: oauth2Resp.RefreshToken,
		OAuth2ExpiresAt:    time.Now().Add(time.Duration(oauth2Resp.ExpiresIn) * time.Second),
	}, nil
}

// getCSRFToken extracts the _csrf input value from HTML.
func getCSRFToken(htmlBody string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlBody))
	if err != nil {
		return "", fmt.Errorf("parse HTML: %w", err)
	}

	var csrf string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if csrf != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "input" {
			var name, value string
			for _, a := range n.Attr {
				switch a.Key {
				case "name":
					name = a.Val
				case "value":
					value = a.Val
				}
			}
			if name == "_csrf" {
				csrf = value
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	if csrf == "" {
		return "", fmt.Errorf("CSRF token not found in HTML")
	}
	return csrf, nil
}

// getTitle extracts the <title> text from HTML.
func getTitle(htmlBody string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlBody))
	if err != nil {
		return "", err
	}

	var title string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if title != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "title" {
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				title = n.FirstChild.Data
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return title, nil
}

// getTicket extracts the service ticket from SSO response HTML.
func getTicket(htmlBody string) (string, error) {
	matches := ticketRe.FindStringSubmatch(htmlBody)
	if len(matches) < 2 {
		return "", fmt.Errorf("service ticket not found in response")
	}
	return matches[1], nil
}

// signOAuth1 builds an OAuth1 Authorization header using HMAC-SHA1 signing.
// bodyParams are included in the signature for POST requests with form-encoded bodies.
func signOAuth1(method, rawURL, consumerKey, consumerSecret, token, tokenSecret string, bodyParams url.Values) string {
	parsed, _ := url.Parse(rawURL)

	nonce := generateNonce()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	oauthParams := map[string]string{
		"oauth_consumer_key":     consumerKey,
		"oauth_nonce":            nonce,
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        timestamp,
		"oauth_version":          "1.0",
	}
	if token != "" {
		oauthParams["oauth_token"] = token
	}

	// Collect all params for the signature base string.
	allParams := url.Values{}
	for k, v := range oauthParams {
		allParams.Set(k, v)
	}
	for k, vs := range parsed.Query() {
		for _, v := range vs {
			allParams.Add(k, v)
		}
	}
	for k, vs := range bodyParams {
		for _, v := range vs {
			allParams.Add(k, v)
		}
	}

	// Build signature base string.
	baseURL := fmt.Sprintf("%s://%s%s", parsed.Scheme, parsed.Host, parsed.Path)
	paramStr := sortedPercentEncode(allParams)
	baseString := percentEncode(method) + "&" +
		percentEncode(baseURL) + "&" +
		percentEncode(paramStr)

	// Sign with HMAC-SHA1.
	signingKey := percentEncode(consumerSecret) + "&" + percentEncode(tokenSecret)
	mac := hmac.New(sha1.New, []byte(signingKey))
	mac.Write([]byte(baseString))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// Build Authorization header.
	oauthParams["oauth_signature"] = signature
	keys := make([]string, 0, len(oauthParams))
	for k := range oauthParams {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, k, percentEncode(oauthParams[k])))
	}
	return "OAuth " + strings.Join(parts, ", ")
}

// sortedPercentEncode returns percent-encoded parameters sorted by key then value.
func sortedPercentEncode(params url.Values) string {
	type kv struct{ key, value string }
	var pairs []kv
	for k, vs := range params {
		for _, v := range vs {
			pairs = append(pairs, kv{k, v})
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].key == pairs[j].key {
			return pairs[i].value < pairs[j].value
		}
		return pairs[i].key < pairs[j].key
	})

	parts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		parts = append(parts, percentEncode(p.key)+"="+percentEncode(p.value))
	}
	return strings.Join(parts, "&")
}

// percentEncode applies RFC 3986 percent-encoding as required by OAuth1.
func percentEncode(s string) string {
	var buf strings.Builder
	for _, b := range []byte(s) {
		if isUnreserved(b) {
			buf.WriteByte(b)
		} else {
			fmt.Fprintf(&buf, "%%%02X", b)
		}
	}
	return buf.String()
}

// isUnreserved reports whether a byte is an RFC 3986 unreserved character.
func isUnreserved(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') || c == '-' || c == '.' || c == '_' || c == '~'
}

// generateNonce returns a random string for OAuth1 nonce values.
func generateNonce() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

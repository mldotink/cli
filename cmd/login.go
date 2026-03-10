package cmd

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/mldotink/cli/internal/config"
	"github.com/spf13/cobra"
)

const oauthServerBase = "https://mcp.ml.ink"

func init() {
	loginCmd.Flags().Bool("global", true, "Save globally (~/.config/ink/config)")
	loginCmd.Flags().String("api-key", "", "Authenticate with an API key directly")
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Ink via browser OAuth or API key",
	Long:  "Log in via browser (default) or provide an API key directly.",
	Example: `# Browser login (opens browser, recommended)
ink login

# API key directly
ink login --api-key dk_live_abc123`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		global, _ := cmd.Flags().GetBool("global")
		apiKey, _ := cmd.Flags().GetString("api-key")

		var key string

		if apiKey != "" {
			if !strings.HasPrefix(apiKey, "dk_") {
				fatal("Invalid API key — keys start with dk_live_ or dk_test_")
			}
			key = apiKey
		} else {
			var method string
			err := huh.NewSelect[string]().
				Title("How would you like to authenticate?").
				Options(
					huh.NewOption("Log in with browser (recommended)", "browser"),
					huh.NewOption("Paste an API key", "apikey"),
				).
				Value(&method).
				Run()
			if err != nil {
				fatal("Login cancelled")
			}

			switch method {
			case "browser":
				k, err := oauthBrowserLogin()
				if err != nil {
					fatal(err.Error())
				}
				key = k
			case "apikey":
				fmt.Println()
				fmt.Println(dim.Render("  Create an API key at: ") + accent.Render("https://ml.ink/account/api-keys"))
				fmt.Println()
				var inputKey string
				err := huh.NewInput().
					Title("API key").
					Placeholder("dk_live_...").
					Value(&inputKey).
					Validate(func(s string) error {
						if !strings.HasPrefix(s, "dk_") {
							return fmt.Errorf("keys start with dk_live_ or dk_test_")
						}
						return nil
					}).
					Run()
				if err != nil {
					fatal("Login cancelled")
				}
				key = inputKey
			}
		}

		c := &config.Config{APIKey: key}
		var err error
		if global {
			err = config.SaveGlobal(c)
		} else {
			err = config.SaveLocal(c)
		}
		if err != nil {
			fatal(fmt.Sprintf("Failed to save: %v", err))
		}

		if global {
			success("Logged in — saved to ~/.config/ink/config")
		} else {
			success("Logged in — saved to .ink (project-local)")
		}
	},
}

func oauthBrowserLogin() (string, error) {
	// Generate PKCE code verifier (43-128 URL-safe chars)
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return "", fmt.Errorf("failed to generate PKCE verifier: %w", err)
	}
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// code_challenge = BASE64URL(SHA256(code_verifier))
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	// Generate state for CSRF protection
	stateBytes := make([]byte, 16)
	rand.Read(stateBytes)
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	// Start local server on random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to start local server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	// Register client
	regBody, _ := json.Marshal(map[string]any{
		"redirect_uris": []string{redirectURI},
		"client_name":   "ink-cli",
	})
	resp, err := http.Post(oauthServerBase+"/oauth/register", "application/json", strings.NewReader(string(regBody)))
	if err != nil {
		listener.Close()
		return "", fmt.Errorf("failed to register OAuth client: %w", err)
	}
	var regResult struct {
		ClientID string `json:"client_id"`
	}
	json.NewDecoder(resp.Body).Decode(&regResult)
	resp.Body.Close()
	clientID := regResult.ClientID

	// Build authorize URL
	authorizeURL, _ := url.Parse(oauthServerBase + "/oauth/authorize")
	q := authorizeURL.Query()
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	q.Set("response_type", "code")
	authorizeURL.RawQuery = q.Encode()

	// Channel to receive result
	result := make(chan oauthResult, 1)

	// Set up callback handler
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		returnedState := r.URL.Query().Get("state")
		code := r.URL.Query().Get("code")
		errMsg := r.URL.Query().Get("error")

		if errMsg != "" {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, callbackPage("Login failed", errMsg, true))
			sendOAuthResult(result, oauthResult{err: fmt.Errorf("OAuth error: %s", errMsg)})
			return
		}

		if returnedState != state {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, callbackPage("Login failed", "State mismatch — please try again.", true))
			sendOAuthResult(result, oauthResult{err: fmt.Errorf("OAuth state mismatch")})
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, callbackPage("You're logged in", "You can close this tab and return to the terminal.", false))
		sendOAuthResult(result, oauthResult{code: code})
	})

	server := &http.Server{Handler: mux}
	go server.Serve(listener)

	// Open browser
	fmt.Println()
	fmt.Println(dim.Render("  Opening browser to log in..."))
	fmt.Println(dim.Render("  If the browser doesn't open, visit:"))
	fmt.Println(dim.Render("  " + authorizeURL.String()))
	fmt.Println()

	openBrowser(authorizeURL.String())
	startManualOAuthFallback(result, state)

	// Wait for callback
	res := <-result
	server.Close()

	if res.err != nil {
		return "", res.err
	}

	// Exchange code for token
	tokenData := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {res.code},
		"code_verifier": {codeVerifier},
		"redirect_uri":  {redirectURI},
		"client_id":     {clientID},
	}

	tokenResp, err := http.PostForm(oauthServerBase+"/oauth/token", tokenData)
	if err != nil {
		return "", fmt.Errorf("failed to exchange code for token: %w", err)
	}
	defer tokenResp.Body.Close()

	var tokenResult struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	json.NewDecoder(tokenResp.Body).Decode(&tokenResult)

	if tokenResult.Error != "" {
		return "", fmt.Errorf("token exchange failed: %s — %s", tokenResult.Error, tokenResult.Description)
	}

	if tokenResult.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}

	return tokenResult.AccessToken, nil
}

type oauthResult struct {
	code string
	err  error
}

func sendOAuthResult(ch chan<- oauthResult, res oauthResult) {
	select {
	case ch <- res:
	default:
	}
}

func startManualOAuthFallback(result chan<- oauthResult, expectedState string) {
	go func() {
		time.Sleep(5 * time.Second)

		fmt.Println()
		fmt.Println(dim.Render("  If the browser lands on a 127.0.0.1 page that can't load, paste the full callback URL here."))
		fmt.Print(dim.Render("  Callback URL or code: "))

		reader := bufio.NewReader(os.Stdin)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}

			line = strings.TrimSpace(line)
			if line == "" {
				fmt.Print(dim.Render("  Callback URL or code: "))
				continue
			}

			res, err := parseOAuthCallbackInput(line, expectedState)
			if err != nil {
				fmt.Println()
				fmt.Println(red.Render("  ✕ ") + err.Error())
				fmt.Print(dim.Render("  Callback URL or code: "))
				continue
			}

			sendOAuthResult(result, res)
			return
		}
	}()
}

func parseOAuthCallbackInput(input, expectedState string) (oauthResult, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return oauthResult{}, fmt.Errorf("callback URL or code is required")
	}

	if !strings.Contains(input, "://") {
		return oauthResult{code: input}, nil
	}

	parsed, err := url.Parse(input)
	if err != nil {
		return oauthResult{}, fmt.Errorf("invalid callback URL")
	}

	q := parsed.Query()
	if errMsg := q.Get("error"); errMsg != "" {
		return oauthResult{}, fmt.Errorf("OAuth error: %s", errMsg)
	}

	code := q.Get("code")
	if code == "" {
		return oauthResult{}, fmt.Errorf("callback URL is missing code")
	}

	returnedState := q.Get("state")
	if returnedState != "" && returnedState != expectedState {
		return oauthResult{}, fmt.Errorf("callback URL state does not match this login attempt")
	}

	return oauthResult{code: code}, nil
}

func callbackPage(title, message string, isError bool) string {
	bg := "#0a0a0a"
	color := "#e0e0e0"
	accent := "#06B6D4"
	if isError {
		accent = "#ef4444"
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Ink — %s</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{background:%s;color:%s;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh}
.card{text-align:center;max-width:420px;padding:48px}
h1{font-size:28px;font-weight:600;margin-bottom:12px;color:%s}
p{font-size:16px;line-height:1.5;opacity:0.7}
</style></head>
<body><div class="card"><h1>%s</h1><p>%s</p></div></body></html>`, title, bg, color, accent, title, message)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Start()
	}
}

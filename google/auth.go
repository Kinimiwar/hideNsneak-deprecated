package google

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/appengine/v1"
	compute "google.golang.org/api/compute/v1"
)

const jsonTemplate = `{"installed":{"client_id":"{{.ClientID}}","project_id":"{{.Project}}","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://accounts.google.com/o/oauth2/token","auth_provider_x509_cert_url":"https://www.googleapis.com/oauth2/v1/certs","client_secret":"{{.Secret}}","redirect_uris":["urn:ietf:wg:oauth:2.0:oob","http://localhost"]}}`

type Authentication struct {
	cachePath string // the path to the token cache file on disk
	ClientID  string
	Secret    string
	Project   string
	token     *oauth2.Token // the oauth token cached in the cache file
}

// Token returns the authentication token by first looking on disk for the
// cache, and if it doesn't exist by executing the authentication.
func (auth *Authentication) Token() (*oauth2.Token, error) {
	// Load the token from the cache if it doesn't exist.
	if auth.token == nil {
		if err := auth.Load(""); err != nil {
			// If we cannot load the token from disk, authenticate
			if err != nil {
				if err = auth.Authenticate(); err != nil {
					return nil, err
				}
			}
		}
	}

	// Return the token
	return auth.token, nil
}

func Prompt(input string) (string, error) {
	fmt.Print(input)
	reader := bufio.NewReader(os.Stdin)
	userInput, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	userInput = strings.TrimSpace(userInput)
	return userInput, nil
}

// Authenticate runs an interactive authentication on the command line,
// prompting the user to open a brower page and enter an authorization code.
// It will then fetch an token via OAuth and cache it as credentials. Note
// that this method will overwrite any previously cached token.
func (auth *Authentication) Authenticate() error {
	// Load and create the OAuth2 Configuration
	config, err := auth.Config()
	if err != nil {
		return err
	}

	// Compute the URL for the authoerization
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	// Notify the user of the web browser.
	fmt.Println("In order to authenticate, use a browser to authorize hideNsneak with Google")

	// Open the web browser
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", authURL).Start()
	case "windows", "darwin":
		err = exec.Command("open", authURL).Start()
	default:
		err = fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// If we couldn't open the web browser, prompt the user to do it manually.
	if err != nil {
		fmt.Printf("Copy and paste the following link: \n%s\n\n", authURL)
	}

	// Prompt for the authorization code
	code, err := Prompt("enter authorization code: ")
	if err != nil {
		return fmt.Errorf("unable to read authorization code %v", err)
	}

	// Perform the exchange for the token
	token, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		return fmt.Errorf("unable to retrieve token from web %v", err)
	}

	// Cache the token to disk
	auth.token = token
	auth.Save("")

	return nil
}

// Config loads the client_secret.json from the ConfigPath. It is used both to
// create the client for requests as well as to perform authentication.
func (auth *Authentication) Config() (*oauth2.Config, error) {
	var data bytes.Buffer
	parsedJSON, _ := template.New("client_secret").Parse(jsonTemplate)

	err := parsedJSON.Execute(&data, auth)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret template: %v", err)
	}

	config, err := google.ConfigFromJSON(data.Bytes(), compute.CloudPlatformScope, appengine.AppengineAdminScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	return config, nil
}

// CachePath computes the path to the credential token file, creating the
// directory if necessary and stores it in the authentication struct.
func (auth *Authentication) CachePath() (string, error) {
	if auth.cachePath == "" {
		// Get the user to look up the user's home directory
		usr, err := user.Current()
		if err != nil {
			return "", err
		}

		// Get the hidden credentials directory, making sure it's created
		cacheDir := filepath.Join(usr.HomeDir, ".hideNsneak/auth")
		os.MkdirAll(cacheDir, 0700)

		// Determine the path to the token cache file
		cacheFile := url.QueryEscape("credentials.json")
		auth.cachePath = filepath.Join(cacheDir, cacheFile)
	}

	return auth.cachePath, nil
}

// Load the token from the specified path (and saves the path to the struct).
// If path is an empty string then it will load the token from the default
// cache path in the home directory. This method returns an error if the token
// cannot be loaded from the file.
func (auth *Authentication) Load(path string) error {
	var err error

	// Get the default cache path or save the specified path
	if path == "" {
		path, err = auth.CachePath()
		if err != nil {
			return err
		}
	} else {
		auth.cachePath = path
	}

	// Open the file at the path
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("could not open cache file at %s: %v", path, err)
	}
	os.Chmod(path, 0500)
	defer f.Close()

	// Decode the JSON token cache
	auth.token = new(oauth2.Token)
	if err := json.NewDecoder(f).Decode(auth.token); err != nil {
		return fmt.Errorf("could not decode token in cache file at %s: %v", path, err)
	}

	return nil
}

// Save the token to the specified path (and save the path to the struct).
// If the path is empty, then it will save the path to the current CachePath.
func (auth *Authentication) Save(path string) error {
	var err error

	// Get the default cache path or save the specified path
	if path == "" {
		path, err = auth.CachePath()
		if err != nil {
			return err
		}
	} else {
		auth.cachePath = path
	}

	// Open the file for writing
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %v", err)
	}
	defer f.Close()

	// Encode the token and write to disk
	if err := json.NewEncoder(f).Encode(auth.token); err != nil {
		return fmt.Errorf("could not encode oauth token: %v", err)
	}

	return nil
}

// Delete the token file at the given path in order to force a
// reauthentication. This method also saves the given path, or if the path is
// empty, then it will compute the default CachePath. This method will not
// return an error on failure (e.g. if the file does not exist).
func (auth *Authentication) Delete(path string) {
	// Get the default cache path or save the specified path
	if path == "" {
		path, _ = auth.CachePath()
	} else {
		auth.cachePath = path
	}

	//  Delete the file at the cache path if it exists
	os.Remove(path)
}

func computeAuth(auth Authentication) *compute.Service {
	// Load the configuration from client_secret.json
	config, err := auth.Config()
	if err != nil {
		log.Fatal(err.Error())
	}

	// Load the token from the cache or force authentication
	token, err := auth.Token()
	if err != nil {
		log.Fatal(err.Error())
	}

	// Create the API client with a background context.
	ctx := context.Background()
	client := config.Client(ctx, token)

	// Create the google compute engine service
	service, err := compute.New(client)
	if err != nil {
		log.Fatal("could not create the google calendar service")
	}
	return service
}

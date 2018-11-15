package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/BurntSushi/toml"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

const redirectURI = "http://localhost:8080/callback"
const html = `
<html>
	<head>
	<script>window.setTimeout(function(){window.close()}, 2500)</script>
	</head>
	<body>Success</body>
</html>`

var (
	authenticator = spotify.NewAuthenticator(redirectURI,
		spotify.ScopePlaylistReadPrivate,
	)
	authChannel = make(chan authResult)
	server      = http.Server{Addr: ":8080"}
)

type authResult struct {
	Client spotify.Client
	Token  *oauth2.Token
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := authenticator.Token("", r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	client := authenticator.NewClient(tok)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, html)
	auth := authResult{client, tok}
	authChannel <- auth
	go func() {
		time.Sleep(10 * time.Second)
		server.Shutdown(context.Background())
	}()
}

func handleOauth(config SpotifyCreds) {
	authenticator.SetAuthInfo(config.ClientID, config.ClientSecret)
	http.HandleFunc("/callback", completeAuth)
	url := authenticator.AuthURL("")
	fmt.Println(url)

	server.ListenAndServe()
}

// Persistance

// SpotifyCreds stores client id and secret
type SpotifyCreds struct {
	ClientID     string
	ClientSecret string
}

// Tokens stores oauth data
type Tokens struct {
	AccessToken  string `toml:"access_token"`
	RefreshToken string `toml:"refresh_token"`
	TokenType    string `toml:"token_type"`
	Expiry       int64
}

const tokenFile string = "~/.config/artist_graph/tokens.toml"

func expandPath(path string) string {
	fullPath, err := homedir.Expand(path)
	if err != nil {
		fmt.Println("Unable to locate home directory")
		os.Exit(1)
	}
	return fullPath
}

func getTokens() Tokens {
	var tokens Tokens
	path := expandPath(tokenFile)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Tokens{}
	}
	if _, err := toml.DecodeFile(path, &tokens); err != nil {
		fmt.Println("Error loading token file:\n", err)
		os.Exit(1)
	}
	return tokens
}

func getCreds() SpotifyCreds {
	id, set := os.LookupEnv("SPOTIFY_ID")
	if !set {
		log.Fatal("$SPOTIFY_ID not set\n")
	}
	secret, set := os.LookupEnv("SPOTIFY_SECRET")
	if !set {
		log.Fatal("$SPOTIFY_SECRET not set\n")
	}

	return SpotifyCreds{id, secret}
}

func putTokens(tokens Tokens) error {
	tokenPath := expandPath(tokenFile)
	os.MkdirAll(path.Dir(expandPath(tokenPath)), os.ModePerm)
	f, err := os.Create(tokenPath)
	if err != nil {
		fmt.Println("Unable to write tokens:", err)
		return err
	}
	defer f.Close()
	encoder := toml.NewEncoder(f)
	return encoder.Encode(tokens)
}

func clientFromRefresh(tokens Tokens, config SpotifyCreds) spotify.Client {
	authenticator.SetAuthInfo(config.ClientID, config.ClientSecret)
	token := new(oauth2.Token)
	token.AccessToken = tokens.AccessToken
	token.RefreshToken = tokens.RefreshToken
	token.Expiry = time.Unix(tokens.Expiry, 0)
	token.TokenType = tokens.TokenType
	return authenticator.NewClient(token)
}

// Creates a spotify client
func getClient() spotify.Client {
	tokens := getTokens()
	config := getCreds()
	var client spotify.Client
	if tokens.RefreshToken != "" {
		client = clientFromRefresh(tokens, config)
	} else {
		go handleOauth(config)
		auth := <-authChannel
		tokens = Tokens{
			auth.Token.AccessToken,
			auth.Token.RefreshToken,
			auth.Token.TokenType,
			auth.Token.Expiry.Unix(),
		}
		putTokens(tokens)
		client = auth.Client
	}

	return client
}

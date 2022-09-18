package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"f360_email_archive_file_dl/settings"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
func fileExists(fileName string) bool {
	_, err := os.Stat(fileName)
	return err == nil
}
func getUniqueFileName(fileName string) string {
	newFileName := fileName
	fileNameWithoutExt := fileName[:len(fileName)-len(filepath.Ext(fileName))]
	for i := 2; fileExists(newFileName); i++ {
		newFileName = fmt.Sprintf("%s%d%s", fileNameWithoutExt, i, filepath.Ext(fileName))
	}
	return newFileName
}

func downloadFile(url string, dirName string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Headerからファイル名を取得
	_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
	fileName := fmt.Sprintf("%s/%s", dirName, params["filename"])

	//　ファイル名の重複を回避
	newFileName := getUniqueFileName(fileName)

	// ファイル作成
	if err := os.MkdirAll(dirName, 0777); err != nil {
		return "", err
	}
	file, err := os.Create(newFileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)

	return newFileName, nil
}

func main() {
	ctx := context.Background()
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	user := "me"
	// メールからURLを取得する
	var urlList []string

	// 検索条件に一致するメールのIDを取得
	res, err := srv.Users.Messages.List(user).Q(settings.SearchText).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve labels: %v", err)
	}
	if len(res.Messages) == 0 {
		fmt.Println("No messages found.")
		return
	}

	// IDからメールの本文を取得
	for _, message := range res.Messages {
		//fmt.Printf("- %s\n", message.Id)
		messageWithRaw, err := srv.Users.Messages.Get(user, message.Id).Format("raw").Do()
		if err != nil {
			log.Fatalf("Unable to retrieve message: %v", err)
		}
		decodedMsg, _ := base64.URLEncoding.DecodeString(messageWithRaw.Raw)
		decodedMsgStr := string(decodedMsg)
		urlBegin := strings.Index(decodedMsgStr, settings.UrlPrefix)
		if urlBegin != -1 {
			var urlEnd int
			for urlEnd = urlBegin; urlEnd < len(decodedMsgStr); urlEnd++ {
				if decodedMsgStr[urlEnd] == settings.UrlSuffix {
					break
				}
			}
			urlList = append(urlList, decodedMsgStr[urlBegin:urlEnd])
		}
	}
	fmt.Printf("Detected %d URLs", len(urlList))
	for _, url := range urlList {
		fmt.Printf("- %s\n", url)
	}

	// ダウンロードするかを尋ねる
	var answer string
	for {
		fmt.Println("Do you want to download?(y/N)")
		_, err = fmt.Scan(&answer)
		if err != nil {
			log.Fatalln(err)
			return
		}
		switch answer {
		case "y":
			goto C
		case "Y":
			goto C
		case "yes":
			goto C
		case "n":
			return
		case "N":
			return
		case "no":
			return
		case "":
			return

		}
	}
	// 脱出用のラベル
C:
	fmt.Println("File download")
	for _, url := range urlList {
		fmt.Printf("- Start download a file from %s\n", url)
		fileName, err := downloadFile(url, settings.DownloadFolderName)
		if err != nil {
			log.Fatalf("%v", err)
			return
		}
		fmt.Printf("- File download finished: %s\n", fileName)
		fmt.Printf("   wait 5 seconds...\n")
		time.Sleep(5 * time.Second)
	}
}

package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil" // ioutil.ReadFile is deprecated, use io or os instead
	"math/rand"
	"net/smtp"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type SenderSettings struct {
	EmailAddr string `json:"email"`
	Login     string `json:"login"`
	Password  string `json:"password"`
	Host      string `json:"host"`
	Port      string `json:"port"`
}

//yahoo add app here: https://login.yahoo.com/myaccount/security
//read https://kinsta.com/blog/yahoo-smtp-settings/

type AppSettings struct {
	Sender SenderSettings `json:"sender"`
}

var appSettings AppSettings
var prodSettingsFile string = "emailer-prod.json"
var testSettingsFile string = "emailer-test.json"

type MailData struct {
	Receivers   []string
	Subject     string
	Body        string
	IsHtml      bool
	Attachments map[string]string
}

func getAppRoot() string {
	//wd, err := os.Getwd()
	//ex, err := os.Executable()
	exeDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	return exeDir
}
func loadConfig() error {
	appRoot := getAppRoot()
	os.Chdir(appRoot)
	fileName := path.Join(appRoot, testSettingsFile)
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		//fmt.Printf("os.Stat(%s): %v\n", fileName, err)
		fileName = path.Join(getAppRoot(), prodSettingsFile)
		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			return fmt.Errorf("File %s: %v\n", fileName, err)
		}
	}
	// open file in read-write mode
	fp, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("File %s: %v\n", fileName, err)
	}
	defer fp.Close()
	cfgReader := bufio.NewReader(fp)
	jsonBytes, err := io.ReadAll(cfgReader)
	appSettings = AppSettings{}
	err = json.Unmarshal(jsonBytes, &appSettings)
	if err != nil {
		return fmt.Errorf("File %s JSON: %v\n", fileName, err)
	}
	return nil
}

func chunkString(content string, chunkSize int, delimiter string) string {
	var chunks []string
	for chunkSize < len(content) {
		chunks = append(chunks, content[:chunkSize])
		content = content[chunkSize:]
	}
	return strings.Join(append(chunks, content), delimiter)
}

func randomString(n int) string {
	letterBytes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func sendEmail(mail MailData) error {
	var buf bytes.Buffer
	buf.Reset()
	boundary := randomString(20)
	block := ""
	chunkSize := 76

	//HEADER
	block = strings.Join([]string{
		"MIME-version: 1.0",
		fmt.Sprintf(`Content-Type: multipart/mixed; boundary="%s"`, boundary),
		fmt.Sprintf("From: %s", appSettings.Sender.EmailAddr),
		fmt.Sprintf("To: %s", strings.Join(mail.Receivers, ",")),
		fmt.Sprintf("Subject: %s", mail.Subject),
		"",
		"",
	}, "\r\n")
	buf.WriteString(block)

	//MESSAGE
	mimeType := "text/plain"
	if mail.IsHtml {
		mimeType = "text/html"
	}
	block = strings.Join([]string{
		fmt.Sprintf("--%s", boundary),
		fmt.Sprintf("Content-Type: %s; charset=\"UTF-8\"", mimeType),
		"",
		fmt.Sprintf("%s", mail.Body),
	}, "\r\n") + "\r\n"
	//fmt.Println(block)
	buf.WriteString(block)

	//ATTACHMENTS
	var fileContent, encodedContent []byte
	var err error
	for fileName, filePath := range mail.Attachments {
		//attachment header
		block = strings.Join([]string{
			fmt.Sprintf("--%s", boundary),
			"Content-Type: application/octet-stream",
			"Content-Transfer-Encoding: base64",
			fmt.Sprintf(`Content-Disposition: attachment; filename="%s"`, fileName),
			fmt.Sprintf(`Content-ID: "%s"`, fileName),
			"",
		}, "\r\n") + "\r\n"
		buf.WriteString(block)
		//fmt.Println(block + "\n...")
		//file content
		fileContent, err = ioutil.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("%s error: %w", filePath, err)
		}
		encodedContent = make([]byte, base64.StdEncoding.EncodedLen(len(fileContent)))
		base64.StdEncoding.Encode(encodedContent, fileContent)
		//chunkString() is fixing "500 Line length exceeded. See RFC 2821 #4.5.3.1."
		buf.WriteString(chunkString(string(encodedContent), chunkSize, "\r\n") + "\r\n")
	}
	//fmt.Printf("\n--%s--\n", boundary)
	buf.WriteString(fmt.Sprintf("--%s--", boundary))

	fmt.Printf("\nsending letter...\n")
	auth := smtp.PlainAuth("", appSettings.Sender.Login, appSettings.Sender.Password, appSettings.Sender.Host)
	//fmt.Printf("\n%s\n", buf.String())
	//return fmt.Errorf("break point")
	err = smtp.SendMail(appSettings.Sender.Host+":"+appSettings.Sender.Port, auth, appSettings.Sender.EmailAddr, mail.Receivers, buf.Bytes())
	return err
}

func main() {
	err := loadConfig()
	if err != nil {
		fmt.Println("Configuration not found:", err)
		return
	}
	if len(os.Args) < 4 { //email, subject and html-file are required
		fmt.Println("args:\n 1: receiver-email1;receiver-email2;...\n 2: subject")
		fmt.Println(" 3: html-file-path\n 4: [attachement1-name;attachment1-path ...]")
		return
	}
	if appSettings.Sender.EmailAddr == "" || appSettings.Sender.Host == "" {
		fmt.Println("Sender SMTP configuration: missing email or host")
		return
	}
	if appSettings.Sender.Login == "" || appSettings.Sender.Password == "" {
		fmt.Println("Sender SMTP configuration: missing login or password")
		return
	}
	receiverEmails := strings.Split(os.Args[1], ";")
	subjectText := os.Args[2]
	messagePath := os.Args[3]
	messageContent, err := ioutil.ReadFile(messagePath)
	if err != nil {
		fmt.Printf("%s error: %v", messagePath, err)
		return
	}
	mailData := MailData{
		Receivers:   receiverEmails,
		Subject:     subjectText,
		Body:        string(messageContent),
		IsHtml:      true,
		Attachments: map[string]string{},
	}
	var finfo os.FileInfo
	namePathPair := []string{}
	fPath := ""
	fName := ""
	for i := 4; i < len(os.Args); i++ {
		namePathPair = strings.Split(os.Args[i], ";")
		if len(namePathPair) != 2 {
			fmt.Printf("Attachment '%s' must have display name and file path separated with ';'\n", os.Args[i])
			return
		}
		fName = namePathPair[0]
		fPath = namePathPair[1]
		finfo, err = os.Stat(fPath)
		if os.IsNotExist(err) {
			fmt.Printf("Attachment '%s' (%s) not found\n", fPath, fName)
			return
		}
		if finfo.IsDir() {
			fmt.Printf("Attachment '%s' (%s) is directory. Must be a file.\n", fPath, fName)
			return
		}
		mailData.Attachments[fName] = fPath
	}

	err = sendEmail(mailData)
	if err != nil {
		fmt.Println("sendEmail:", err)
	} else {
		fmt.Println("Email sent successfully")
		err = os.Remove(messagePath)
		if err != nil {
			fmt.Println("os.Remove:", err)
		}
	}
}
